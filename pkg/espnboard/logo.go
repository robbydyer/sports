package espnboard

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/rgbrender"
	"github.com/robbydyer/sports/pkg/util"
)

func (e *ESPNBoard) getLogoCache(logoKey string) (*logo.Logo, error) {
	e.logoLock.RLock()
	defer e.logoLock.RUnlock()

	l, ok := e.logos[logoKey]
	if ok {
		return l, nil
	}

	return l, fmt.Errorf("no cache for logo %s", logoKey)
}

func (e *ESPNBoard) setLogoCache(logoKey string, l *logo.Logo) {
	e.logoLock.Lock()
	defer e.logoLock.Unlock()

	e.logos[logoKey] = l
}

// GetLogo ...
func (e *ESPNBoard) GetLogo(ctx context.Context, logoKey string, logoConf *logo.Config, bounds image.Rectangle) (*logo.Logo, error) {
	logoKey = fmt.Sprintf("%s_%s", e.leaguer.HTTPPathPrefix(), logoKey)
	if l, err := e.getLogoCache(logoKey); err == nil {
		return l, nil
	}

	cacheDir, err := e.logoCacheDir()
	if err != nil {
		return nil, err
	}

	// A logoKey should be LEAGUE_TEAM_HOME|AWAY_XxY, ie. ncaam_1234_HOME_64x32
	p := strings.Split(logoKey, "_")
	if len(p) < 4 {
		return nil, fmt.Errorf("invalid logo key")
	}

	teamID := p[1]
	dimKey := p[3]

	e.logoLock.Lock()
	_, ok := e.logoConfOnce[dimKey]
	if !ok {
		e.log.Debug("loading default logo configs",
			zap.Int("x", bounds.Dx()),
			zap.Int("y", bounds.Dy()),
		)
		if err := e.loadDefaultLogoConfigs(bounds); err != nil {
			// Log the error, but don't return. We'll just use defaults
			e.log.Warn("no defaults defined for logos", zap.String("league", e.leaguer.League()))
		}
		e.logoConfOnce[dimKey] = struct{}{}
	}
	e.logoLock.Unlock()

	var l *logo.Logo
	defer e.setLogoCache(logoKey, l)

	logoGetter := func(ctx context.Context) (image.Image, error) {
		return e.GetLogoSource(ctx, teamID, logoSearch(teamID))
	}

	if logoConf != nil {
		l = logo.New(logoKey, logoGetter, cacheDir, bounds, logoConf)

		return l, nil
	}

	for _, d := range *e.defaultLogoConf {
		if d.Abbrev == logoKey {
			l = logo.New(logoKey, logoGetter, cacheDir, bounds, d)

			return l, nil
		}
	}

	c := &logo.Config{
		Abbrev: logoKey,
		XSize:  bounds.Dx(),
		YSize:  bounds.Dy(),
		Pt: &logo.Pt{
			X:    0,
			Y:    0,
			Zoom: 1,
		},
	}

	*e.defaultLogoConf = append(*e.defaultLogoConf, c)

	l = logo.New(logoKey, logoGetter, cacheDir, bounds, c)

	return l, nil
}

func (e *ESPNBoard) loadDefaultLogoConfigs(bounds image.Rectangle) error {
	dat, err := assets.ReadFile(fmt.Sprintf("assets/logopos_%dx%d.yaml", bounds.Dx(), bounds.Dy()))
	if err != nil {
		return err
	}

	var confs []*logo.Config
	if err := yaml.Unmarshal(dat, &confs); err != nil {
		return err
	}
	*e.defaultLogoConf = append(*e.defaultLogoConf, confs...)

	return nil
}

func logoSearch(teamID string) string {
	switch teamID {
	case "2294":
		// IOWA
		return "dark"
	}

	return "scoreboard"
}

// GetLogoSource ...
func (e *ESPNBoard) GetLogoSource(ctx context.Context, teamID string, logoURLSearch string) (image.Image, error) {
	e.Lock()
	l, ok := e.logoLockers[teamID]
	if !ok {
		e.logoLockers[teamID] = &sync.Mutex{}
		l = e.logoLockers[teamID]
	}
	e.Unlock()
	l.Lock()
	defer l.Unlock()

	cacheDir, err := e.logoCacheDir()
	if err != nil {
		return nil, err
	}

	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s_%s.png", e.leaguer.HTTPPathPrefix(), teamID))

	if _, err := os.Stat(cacheFile); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to detect if logo cache file exists: %w", err)
		}
	} else {
		e.log.Debug("reading source logo from cache",
			zap.String("league", e.leaguer.League()),
			zap.String("team", teamID),
		)
		r, err := os.Open(cacheFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open logo cache file: %w", err)
		}
		return png.Decode(r)
	}

	teams, err := e.getTeams(ctx)
	if err != nil || len(teams) < 1 {
		return nil, fmt.Errorf("failed to fetch team info for logo: %w", err)
	}
	e.log.Debug("looking for logo source",
		zap.String("league", e.leaguer.League()),
		zap.String("team", teamID),
		zap.Int("num teams", len(teams)),
	)

	var i image.Image
	foundSource := false
OUTER:
	for _, team := range teams {
		if team.ID != teamID {
			continue OUTER
		}

		e.log.Debug("looking for logo source",
			zap.String("league", e.leaguer.League()),
			zap.String("team", teamID),
			zap.Int("num logos", len(team.Logos)),
		)

		var defaultHref sync.Once
		href := ""
		dHref := ""
		foundStr := false

		for _, logo := range team.Logos {
			defaultHref.Do(func() {
				e.log.Debug("logo default ONCE",
					zap.String("href", logo.Href),
				)
				dHref = logo.Href
			})
			href = logo.Href
			if logoURLSearch != "" {
				if strings.Contains(logo.Href, logoURLSearch) {
					foundStr = true
					break
				}
				continue
			} else {
				foundStr = true
				break
			}
		}

		if !foundStr {
			href = dHref
		}

		e.log.Debug("pulling logo from API",
			zap.String("URL", href),
			zap.String("league", e.leaguer.League()),
			zap.String("team", teamID),
		)
		i, err = util.PullPng(ctx, href)
		if err != nil || i == nil {
			return nil, fmt.Errorf("failed to retrieve logo from API for %s %s: %w", teamID, href, err)
		}
		foundSource = true
	}

	if !foundSource {
		return nil, fmt.Errorf("failed to determine source URL for logo %s", teamID)
	}

	e.log.Debug("saving source logo to cache",
		zap.String("league", e.leaguer.League()),
		zap.String("team", teamID),
	)
	if err := rgbrender.SavePng(i, cacheFile); err != nil {
		e.log.Error("failed to save logo to cache", zap.Error(err))
		_ = os.Remove(cacheFile)
	}

	return i, nil
}
