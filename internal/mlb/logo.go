package mlb

import (
	"context"
	"embed"
	"fmt"
	"image"
	"strings"

	yaml "github.com/ghodss/yaml"

	"github.com/robbydyer/sports/internal/logo"
)

//go:embed assets
var assets embed.FS

func (m *MLB) getLogoCache(logoKey string) (*logo.Logo, error) {
	m.logoLock.RLock()
	defer m.logoLock.RUnlock()

	l, ok := m.logos[logoKey]
	if ok {
		return l, nil
	}

	return l, fmt.Errorf("no cache for logo %s", logoKey)
}

func (m *MLB) setLogoCache(logoKey string, l *logo.Logo) {
	m.logoLock.Lock()
	defer m.logoLock.Unlock()

	m.logos[logoKey] = l
}

// GetLogo ...
func (m *MLB) GetLogo(ctx context.Context, logoKey string, logoConf *logo.Config, bounds image.Rectangle) (*logo.Logo, error) {
	var l *logo.Logo
	var err error

	defer func() {
		if l != nil {
			l.SetLogger(m.log)
			m.setLogoCache(logoKey, l)
		}
	}()

	l, err = m.getLogoCache(logoKey)
	if err == nil {
		return l, nil
	}

	if m.defaultLogoConf == nil {
		m.defaultLogoConf = &[]*logo.Config{}
	}

	p := strings.Split(logoKey, "_")
	if len(p) < 2 {
		return nil, fmt.Errorf("invalid logo key '%s'", logoConf.Abbrev)
	}
	teamAbbrev := p[0]

	logoGetter := func(ctx context.Context) (image.Image, error) {
		return m.espnAPI.GetLogo(ctx, "baseball", "mlb", sportsAPIToESPN(teamAbbrev), logoSearch(teamAbbrev))
	}

	if logoConf != nil {
		l = logo.New(logoKey, logoGetter, logoCacheDir, bounds, logoConf)

		return l, nil
	}

	for _, d := range *m.defaultLogoConf {
		if d.Abbrev == logoKey {
			l = logo.New(logoKey, logoGetter, logoCacheDir, bounds, d)
			return l, nil
		}
	}

	dat, err := assets.ReadFile(fmt.Sprintf("assets/logopos_%dx%d.yaml", bounds.Dx(), bounds.Dy()))
	if err != nil {
		*m.defaultLogoConf = append(*m.defaultLogoConf,
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
	} else {
		if err := yaml.Unmarshal(dat, &m.defaultLogoConf); err != nil {
			return nil, err
		}
	}

	for _, d := range *m.defaultLogoConf {
		if d.Abbrev == logoKey {
			l = logo.New(logoKey, logoGetter, logoCacheDir, bounds, d)
			return l, nil
		}
	}

	return nil, fmt.Errorf("could not get logo for %s", teamAbbrev)
}

func sportsAPIToESPN(abbrev string) string {
	switch abbrev {
	case "CWS":
		return "CHW"
	}
	return abbrev
}

func logoSearch(team string) string {
	return "scoreboard"
}
