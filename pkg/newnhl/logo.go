package newnhl

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

func includes() {
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/logopos_64x32.yaml")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/ANA.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/ARI.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/BOS.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/BUF.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/CAR.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/CBJ.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/CGY.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/CHI.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/COL.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/DAL.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/DET.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/EDM.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/FLA.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/LAK.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/MIN.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/MTL.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/NJD.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/NSH.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/NYI.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/NYR.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/OTT.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/PHI.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/PIT.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/SJS.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/STL.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/TBL.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/TOR.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/VAN.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/VGK.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/WPG.png")
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/WSH.png")
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

	sources, err := n.logoSources()
	if err != nil {
		return nil, err
	}

	if _, ok := sources[teamAbbrev]; !ok {
		return nil, fmt.Errorf("did not find logo source for %s", teamAbbrev)
	}

	if logoConf != nil {
		fmt.Printf("using provided config for logo %s\n", fullLogoKey)
		n.logos[fullLogoKey] = logo.New(teamAbbrev, sources[teamAbbrev], logoCacheDir, bounds, logoConf)

		return n.logos[fullLogoKey], nil
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
			n.logos[fullLogoKey] = logo.New(teamAbbrev, sources[teamAbbrev], logoCacheDir, bounds, defConf)
			return n.logos[fullLogoKey], nil
		}
	}

	return nil, fmt.Errorf("could not find logo config for %s", logoKey)
}

func (n *NHL) logoSources() (map[string]image.Image, error) {
	if len(n.logoSourceCache) == len(ALL) {
		return n.logoSourceCache, nil
	}

	for _, t := range ALL {
		f, err := pkger.Open(fmt.Sprintf("github.com/robbydyer/sports:/pkg/newnhl/assets/nhl/%s.png", t))
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
