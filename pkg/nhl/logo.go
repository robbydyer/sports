package nhl

import (
	"context"
	"embed"
	"fmt"
	"image"
	"image/png"
	"strings"

	yaml "github.com/ghodss/yaml"
	"github.com/hashicorp/go-multierror"

	"github.com/robbydyer/sports/pkg/logo"
)

//go:embed assets
var assets embed.FS

// GetLogo ...
func (n *NHL) GetLogo(ctx context.Context, logoKey string, logoConf *logo.Config, bounds image.Rectangle) (*logo.Logo, error) {
	l, ok := n.logos[logoKey]
	if ok {
		return l, nil
	}

	n.Lock()
	defer n.Unlock()

	sources, err := n.logoSources(ctx)
	if err != nil {
		return nil, err
	}

	if n.defaultLogoConf == nil {
		n.defaultLogoConf = &[]*logo.Config{}
	}

	l, err = GetLogo(logoKey, logoConf, bounds, sources, n.defaultLogoConf)
	if err != nil {
		return nil, err
	}

	l.SetLogger(n.log)

	n.logos[logoKey] = l

	return n.logos[logoKey], nil
}

// GetLogo is a generic logo getter. Useful for testing
func GetLogo(logoKey string, logoConf *logo.Config, bounds image.Rectangle, logoSources map[string]image.Image, defaultConfigs *[]*logo.Config) (*logo.Logo, error) {
	p := strings.Split(logoKey, "_")
	if len(p) < 2 {
		return nil, fmt.Errorf("invalid logo key '%s'", logoConf.Abbrev)
	}
	teamAbbrev := p[0]

	if _, ok := logoSources[teamAbbrev]; !ok {
		return nil, fmt.Errorf("did not find logo source for %s", teamAbbrev)
	}

	if logoConf != nil {
		l := logo.New(logoKey, logoSources[teamAbbrev], logoCacheDir, bounds, logoConf)

		return l, nil
	}

	for _, d := range *defaultConfigs {
		if d.Abbrev == logoKey {
			l := logo.New(logoKey, logoSources[teamAbbrev], logoCacheDir, bounds, d)
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
			l := logo.New(logoKey, logoSources[teamAbbrev], logoCacheDir, bounds, d)
			return l, nil
		}
	}

	return nil, fmt.Errorf("failed to prepare logo")
}

func (n *NHL) logoSources(ctx context.Context) (map[string]image.Image, error) {
	if len(n.logoSourceCache) == len(ALL) {
		return n.logoSourceCache, nil
	}

	errs := &multierror.Error{}

	for _, t := range ALL {
		select {
		case <-ctx.Done():
			return nil, context.Canceled
		default:
		}
		f, err := assets.Open(fmt.Sprintf("assets/logos/%s.png", t))
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("failed to get logo for %s", t))
			continue
		}
		defer f.Close()

		i, err := png.Decode(f)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("failed to decode logo for %s", t))
			continue
		}

		n.logoSourceCache[t] = i
	}

	return n.logoSourceCache, errs.ErrorOrNil()
}
