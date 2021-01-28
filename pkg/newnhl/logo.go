package newnhl

import (
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"strings"

	"github.com/markbates/pkger"
	"github.com/robbydyer/sports/pkg/logo"
	"gopkg.in/yaml.v2"
)

func includes() {
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/logopos_64x32.yaml")
}

func (n *NHL) GetLogo(logoKey string, logoConf *logo.Config, bounds image.Rectangle) (*logo.Logo, error) {
	p := strings.Split(logoKey, "_")
	if len(p) < 2 {
		return nil, fmt.Errorf("invalid logo key '%s'", logoConf.Abbrev)
	}
	teamAbbrev := p[0]

	fullLogoKey := fmt.Sprintf("%s_%dx%d", logoKey, bounds.Dx(), bounds.Dy())

	l, ok := n.logos[fullLogoKey]
	if ok {
		return l, nil
	}

	logoAsset := fmt.Sprintf("github.com/robbydyer/sports:/pkg/newnhl/assets/logopos_%dx%d.yaml",
		bounds.Dx(),
		bounds.Dy(),
	)
	f, err := pkger.Open(logoAsset)
	if err != nil {
		return nil, fmt.Errorf("could not load logoposition asset %s: %w", logoAsset, err)
	}
	defer f.Close()

	dat, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var defaultPos []*logo.Config

	if err := yaml.Unmarshal(dat, &defaultPos); err != nil {
		return nil, err
	}

	sources, err := logoSources()
	if err != nil {
		return nil, err
	}

	if logoConf != nil {
		n.logos[fullLogoKey] = logo.New(teamAbbrev, sources[teamAbbrev], logoCacheDir, logoConf)

		return n.logos[fullLogoKey], nil
	}

	// Use defaults for this logo
	for _, defConf := range defaultPos {
		if defConf.Abbrev == logoKey {
			n.logos[fullLogoKey] = logo.New(teamAbbrev, sources[teamAbbrev], logoCacheDir, defConf)
			return n.logos[fullLogoKey], nil
		}
	}

	return nil, fmt.Errorf("could not find logo config for %s", logoKey)
}

func logoSources() (map[string]image.Image, error) {
	sources := make(map[string]image.Image)
	for _, t := range ALL {
		f, err := pkger.Open(fmt.Sprintf("github.com/robbydyer/sports:/pkg/newnhl/assets/logos/%s.png", t))
		if err != nil {
			return nil, fmt.Errorf("failed to locate logo asset: %w", err)
		}
		defer f.Close()

		i, err := png.Decode(f)
		if err != nil {
			return nil, err
		}

		sources[t] = i
	}

	return sources, nil
}
