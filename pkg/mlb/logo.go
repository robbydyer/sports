package mlb

import (
	"context"
	"embed"
	"fmt"
	"image"
	"image/png"
	"strings"

	yaml "github.com/ghodss/yaml"

	"github.com/robbydyer/sports/pkg/logo"
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
	if l, err := m.getLogoCache(logoKey); err == nil {
		return l, nil
	}

	if m.defaultLogoConf == nil {
		m.defaultLogoConf = &[]*logo.Config{}
	}

	l, err := GetLogo(ctx, logoKey, logoConf, bounds, m.defaultLogoConf)
	if err != nil {
		return nil, err
	}

	l.SetLogger(m.log)

	m.setLogoCache(logoKey, l)

	return m.getLogoCache(logoKey)
}

// GetLogo is a generic function that can be used outside the scope of an MLB type. Useful for testing
func GetLogo(ctx context.Context, logoKey string, logoConf *logo.Config, bounds image.Rectangle, defaultConfigs *[]*logo.Config) (*logo.Logo, error) {
	p := strings.Split(logoKey, "_")
	if len(p) < 2 {
		return nil, fmt.Errorf("invalid logo key '%s'", logoConf.Abbrev)
	}
	teamAbbrev := p[0]

	logoGetter := func(ctx context.Context) (image.Image, error) {
		return logoSource(ctx, teamAbbrev)
	}

	if logoConf != nil {
		l := logo.New(logoKey, logoGetter, logoCacheDir, bounds, logoConf)

		return l, nil
	}

	for _, d := range *defaultConfigs {
		if d.Abbrev == logoKey {
			l := logo.New(logoKey, logoGetter, logoCacheDir, bounds, d)
			return l, nil
		}
	}

	dat, err := assets.ReadFile(fmt.Sprintf("assets/logopos_%dx%d.yaml", bounds.Dx(), bounds.Dy()))
	if err != nil {
		*defaultConfigs = append(*defaultConfigs,
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
		if err := yaml.Unmarshal(dat, &defaultConfigs); err != nil {
			return nil, err
		}
	}

	for _, d := range *defaultConfigs {
		if d.Abbrev == logoKey {
			l := logo.New(logoKey, logoGetter, logoCacheDir, bounds, d)
			return l, nil
		}
	}

	return nil, fmt.Errorf("failed to prepare logo")
}

func logoSource(ctx context.Context, abbreviation string) (image.Image, error) {
	f, err := assets.Open(fmt.Sprintf("assets/logos/%s.png", abbreviation))
	if err != nil {
		return nil, fmt.Errorf("failed to get logo for %s: %w", abbreviation, err)
	}
	defer f.Close()

	i, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode logo %s: %w", abbreviation, err)
	}

	return i, nil
}
