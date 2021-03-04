package nhl

import (
	"context"
	"embed"
	"fmt"
	"image"
	"strings"

	"github.com/robbydyer/sports/pkg/logo"
)

//go:embed assets
var assets embed.FS

func (n *NHL) getLogoCache(logoKey string) (*logo.Logo, error) {
	n.logoLock.RLock()
	defer n.logoLock.RUnlock()

	l, ok := n.logos[logoKey]
	if ok {
		return l, nil
	}

	return nil, fmt.Errorf("no cache for logo %s", logoKey)
}

func (n *NHL) setLogoCache(logoKey string, l *logo.Logo) {
	n.logoLock.Lock()
	defer n.logoLock.Unlock()

	n.logos[logoKey] = l
}

// GetLogo ...
func (n *NHL) GetLogo(ctx context.Context, logoKey string, logoConf *logo.Config, bounds image.Rectangle) (*logo.Logo, error) {
	var l *logo.Logo
	var err error

	defer func() {
		if l != nil {
			l.SetLogger(n.log)
			n.setLogoCache(logoKey, l)
		}
	}()

	l, err = n.getLogoCache(logoKey)
	if err == nil {
		return l, nil
	}

	if n.defaultLogoConf == nil {
		n.defaultLogoConf = &[]*logo.Config{}
	}

	p := strings.Split(logoKey, "_")
	if len(p) < 2 {
		return nil, fmt.Errorf("invalid logo key '%s'", logoConf.Abbrev)
	}
	teamAbbrev := p[0]

	logoGetter := func(ctx context.Context) (image.Image, error) {
		return n.espnAPI.GetLogo(ctx, "hockey", "nhl", sportsAPIToESPN(teamAbbrev), logoSearch(teamAbbrev))
	}

	if logoConf != nil {
		l = logo.New(logoKey, logoGetter, logoCacheDir, bounds, logoConf)

		return l, nil
	}

	for _, d := range *n.defaultLogoConf {
		if d.Abbrev == logoKey {
			l := logo.New(logoKey, logoGetter, logoCacheDir, bounds, d)
			return l, nil
		}
	}

	*n.defaultLogoConf = append(*n.defaultLogoConf,
		&logo.Config{
			Abbrev: logoKey,
			XSize:  bounds.Dx(),
			YSize:  bounds.Dy(),
			Pt: &logo.Pt{
				X:    0,
				Y:    0,
				Zoom: 1,
			},
		},
	)

	for _, d := range *n.defaultLogoConf {
		if d.Abbrev == logoKey {
			l = logo.New(logoKey, logoGetter, logoCacheDir, bounds, d)
			return l, nil
		}
	}

	return nil, fmt.Errorf("failed to prepare logo")
}

func sportsAPIToESPN(sportsAPIAbbreviation string) string {
	switch sportsAPIAbbreviation {
	case "NJD":
		return "NJ"
	case "LAK":
		return "LA"
	case "SJS":
		return "SJ"
	case "TBL":
		return "TB"
	case "VGK":
		return "VGS"
	}

	return sportsAPIAbbreviation
}

func logoSearch(team string) string {
	return "scoreboard"
}
