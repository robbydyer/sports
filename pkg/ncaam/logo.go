package ncaam

import (
	"context"
	"embed"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/robbydyer/sports/pkg/logo"
)

//go:embed assets
var assets embed.FS

// GetLogo ...
func (n *NcaaM) GetLogo(ctx context.Context, logoKey string, logoConf *logo.Config, bounds image.Rectangle) (*logo.Logo, error) {
	n.Lock()
	if l, ok := n.logos[logoKey]; ok {
		n.Unlock()
		return l, nil
	}
	n.Unlock()

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
			n.log.Error("no defaults defined for NCAAM logos")
		}
		n.logoConfOnce[dimKey] = struct{}{}
	}

	t, err := n.TeamFromAbbreviation(ctx, teamAbbrev)
	if err != nil {
		return nil, err
	}

	team, ok := t.(*Team)
	if !ok {
		return nil, fmt.Errorf("failed to convert sportboard.Team to ncaam.Team")
	}

	src, err := n.logoSource(ctx, team)
	if err != nil {
		return nil, fmt.Errorf("could not find logo source for %s: %w", teamAbbrev, err)
	}

	var l *logo.Logo
	defer func() {
		n.Lock()
		n.logos[logoKey] = l
		n.Unlock()
	}()

	if logoConf != nil {
		l = logo.New(logoKey, src, logoCacheDir, bounds, logoConf)

		return l, nil
	}

	for _, d := range *n.defaultLogoConf {
		if d.Abbrev == logoKey {
			l = logo.New(logoKey, src, logoCacheDir, bounds, d)

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

	l = logo.New(logoKey, src, logoCacheDir, bounds, c)

	return l, nil
}

func (n *NcaaM) loadDefaultLogoConfigs(bounds image.Rectangle) error {
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

func (n *NcaaM) logoSource(ctx context.Context, team *Team) (image.Image, error) {
	if team.LogoURL != "" {
		return pullPng(ctx, team.LogoURL)
	}

	for _, logo := range team.Logos {
		if logo.Href != "" {
			return pullPng(ctx, logo.Href)
		}
	}

	return nil, fmt.Errorf("no logo URL defined for team %s", team.Abbreviation)
}

func pullPng(ctx context.Context, url string) (image.Image, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	client := http.DefaultClient

	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return png.Decode(resp.Body)
}
