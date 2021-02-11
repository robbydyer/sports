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
	fullLogoKey := fmt.Sprintf("%s_%dx%d", logoKey, bounds.Dx(), bounds.Dy())
	l, ok := n.logos[fullLogoKey]
	if ok {
		return l, nil
	}

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

	n.logos[fullLogoKey] = l

	return n.logos[fullLogoKey], nil
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
		l := logo.New(teamAbbrev, logoSources[teamAbbrev], logoCacheDir, bounds, logoConf)

		return l, nil
	}

	if defaultConfigs == nil || len(*defaultConfigs) < 1 {
		dat, err := assets.ReadFile(fmt.Sprintf("assets/logopos_%dx%d.yaml", bounds.Dx(), bounds.Dy()))
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(dat, &defaultConfigs); err != nil {
			return nil, err
		}
	}

	// Use defaults for this logo
	for _, defConf := range *defaultConfigs {
		if defConf.Abbrev == logoKey {
			l := logo.New(teamAbbrev, logoSources[teamAbbrev], logoCacheDir, bounds, defConf)
			return l, nil
		}
	}

	return nil, fmt.Errorf("could not find logo config for %s", logoKey)
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
