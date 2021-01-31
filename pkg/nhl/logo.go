package nhl

import (
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"strings"

	yaml "github.com/ghodss/yaml"
	"github.com/markbates/pkger"

	"github.com/robbydyer/sports/pkg/logo"
)

// GetLogo ...
func (n *NHL) GetLogo(logoKey string, logoConf *logo.Config, bounds image.Rectangle) (*logo.Logo, error) {
	fullLogoKey := fmt.Sprintf("%s_%dx%d", logoKey, bounds.Dx(), bounds.Dy())
	l, ok := n.logos[fullLogoKey]
	if ok {
		return l, nil
	}

	sources, err := n.logoSources()
	if err != nil {
		return nil, err
	}

	l, err = GetLogo(logoKey, logoConf, bounds, sources)
	if err != nil {
		return nil, err
	}

	n.logos[fullLogoKey] = l

	return n.logos[fullLogoKey], nil
}

// GetLogo is a generic logo getter. Useful for testing
func GetLogo(logoKey string, logoConf *logo.Config, bounds image.Rectangle, logoSources map[string]image.Image) (*logo.Logo, error) {
	p := strings.Split(logoKey, "_")
	if len(p) < 2 {
		return nil, fmt.Errorf("invalid logo key '%s'", logoConf.Abbrev)
	}
	teamAbbrev := p[0]

	fullLogoKey := fmt.Sprintf("%s_%dx%d", logoKey, bounds.Dx(), bounds.Dy())

	logoAsset := fmt.Sprintf("github.com/robbydyer/sports:/pkg/nhl/assets/logopos_%dx%d.yaml",
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

	if _, ok := logoSources[teamAbbrev]; !ok {
		return nil, fmt.Errorf("did not find logo source for %s", teamAbbrev)
	}

	if logoConf != nil {
		fmt.Printf("using provided config for logo %s\n", fullLogoKey)
		l := logo.New(teamAbbrev, logoSources[teamAbbrev], logoCacheDir, bounds, logoConf)

		return l, nil
	}

	fmt.Printf("using default config for logo %s\n", fullLogoKey)
	// Use defaults for this logo
	for _, defConf := range defaultPos {
		if defConf.Abbrev == logoKey {
			fmt.Printf("default logo config for %s:\n%d, %d zoom %f\n",
				logoKey,
				defConf.Pt.X,
				defConf.Pt.Y,
				defConf.Pt.Zoom,
			)
			l := logo.New(teamAbbrev, logoSources[teamAbbrev], logoCacheDir, bounds, defConf)
			return l, nil
		}
	}

	return nil, fmt.Errorf("could not find logo config for %s", logoKey)
}

func (n *NHL) logoSources() (map[string]image.Image, error) {
	if len(n.logoSourceCache) == len(ALL) {
		return n.logoSourceCache, nil
	}

	for _, t := range ALL {
		f, err := pkger.Open(fmt.Sprintf("github.com/robbydyer/sports:/pkg/nhl/assets/logos/%s.png", t))
		if err != nil {
			return nil, fmt.Errorf("failed to locate logo asset: %w", err)
		}
		defer f.Close()

		i, err := png.Decode(f)
		if err != nil {
			return nil, err
		}

		n.logoSourceCache[t] = i
	}

	return n.logoSourceCache, nil
}

func includes() {
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logopos_64x32.yaml")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/ANA.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/ARI.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/BOS.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/BUF.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/CAR.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/CBJ.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/CGY.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/CHI.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/COL.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/DAL.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/DET.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/EDM.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/FLA.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/LAK.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/MIN.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/MTL.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/NJD.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/NSH.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/NYI.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/NYR.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/OTT.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/PHI.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/PIT.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/SJS.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/STL.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/TBL.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/TOR.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/VAN.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/VGK.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/WPG.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhl/assets/logos/WSH.png")
}
