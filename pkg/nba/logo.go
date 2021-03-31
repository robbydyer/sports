package nba

import (
	"context"
	"embed"
	"fmt"
	"image"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/robbydyer/sports/pkg/logo"
)

//go:embed assets
var assets embed.FS

func (n *NBA) getLogoCache(logoKey string) (*logo.Logo, error) {
	n.logoLock.RLock()
	defer n.logoLock.RUnlock()

	l, ok := n.logos[logoKey]
	if ok {
		return l, nil
	}

	return l, fmt.Errorf("no cache for logo %s", logoKey)
}

func (n *NBA) setLogoCache(logoKey string, l *logo.Logo) {
	n.logoLock.Lock()
	defer n.logoLock.Unlock()

	n.logos[logoKey] = l
}

// GetLogo ...
func (n *NBA) GetLogo(ctx context.Context, logoKey string, logoConf *logo.Config, bounds image.Rectangle) (*logo.Logo, error) {
	if l, err := n.getLogoCache(logoKey); err == nil {
		return l, nil
	}

	// A logoKey should be TEAM_HOME|AWAY_XxY, ie. ALA_HOME_64x32
	p := strings.Split(logoKey, "_")
	if len(p) < 3 {
		return nil, fmt.Errorf("invalid logo key")
	}

	teamAbbrev := p[0]
	dimKey := p[2]

	_, ok := n.logoConfOnce[dimKey]
	if !ok {
		n.log.Debug("loading default logo configs",
			zap.Int("x", bounds.Dx()),
			zap.Int("y", bounds.Dy()),
		)
		if err := n.loadDefaultLogoConfigs(bounds); err != nil {
			// Log the error, but don't return. We'll just use defaults
			n.log.Warn("no defaults defined for NBA logos")
		}
		n.logoConfOnce[dimKey] = struct{}{}
	}

	var l *logo.Logo
	defer n.setLogoCache(logoKey, l)

	logoGetter := func(ctx context.Context) (image.Image, error) {
		return n.espnAPI.GetLogo(ctx, "basketball", "nba", teamAbbrev, logoSearch(teamAbbrev))
	}

	if logoConf != nil {
		l = logo.New(logoKey, logoGetter, logoCacheDir, bounds, logoConf)

		return l, nil
	}

	for _, d := range *n.defaultLogoConf {
		if d.Abbrev == logoKey {
			l = logo.New(logoKey, logoGetter, logoCacheDir, bounds, d)

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

	*n.defaultLogoConf = append(*n.defaultLogoConf, c)

	l = logo.New(logoKey, logoGetter, logoCacheDir, bounds, c)

	return l, nil
}

func (n *NBA) loadDefaultLogoConfigs(bounds image.Rectangle) error {
	dat, err := assets.ReadFile(fmt.Sprintf("assets/logopos_%dx%d.yaml", bounds.Dx(), bounds.Dy()))
	if err != nil {
		return err
	}

	var confs []*logo.Config
	if err := yaml.Unmarshal(dat, &confs); err != nil {
		return err
	}
	*n.defaultLogoConf = append(*n.defaultLogoConf, confs...)

	return nil
}

func logoSearch(team string) string {
	switch team {
	case "IOWA":
		return "dark"
	}

	return "scoreboard"
}
