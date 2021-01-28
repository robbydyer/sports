package nhlboard

import (
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"os"
	"strings"

	yaml "github.com/ghodss/yaml"
	"github.com/markbates/pkger"
	"github.com/robbydyer/sports/pkg/logo"
)

const (
	logoCacheDir = "/tmp/sportsmatrix_logos"
	ANA          = "ANA"
	ARI          = "ARI"
	BOS          = "BOS"
	BUF          = "BUF"
	CAR          = "CAR"
	CBJ          = "CBJ"
	CGY          = "CGY"
	CHI          = "CHI"
	COL          = "COL"
	DAL          = "DAL"
	DET          = "DET"
	EDM          = "EDM"
	FLA          = "FLA"
	LAK          = "LAK"
	MIN          = "MIN"
	MTL          = "MTL"
	NJD          = "NJD"
	NSH          = "NSH"
	NYI          = "NYI"
	NYR          = "NYR"
	OTT          = "OTT"
	PHI          = "PHI"
	PIT          = "PIT"
	SJS          = "SJS"
	STL          = "STL"
	TBL          = "TBL"
	TOR          = "TOR"
	VAN          = "VAN"
	VGK          = "VGK"
	WPG          = "WPG"
	WSH          = "WSH"
)

var ALL = []string{ANA, ARI, BOS, BUF, CAR, CBJ, CGY, CHI, COL, DAL, DET, EDM, FLA, LAK, MIN, MTL, NJD, NSH, NYI, NYR, OTT, PHI, PIT, SJS, STL, TBL, TOR, VAN, VGK, WPG, WSH}

type pt struct {
	X    int     `json:"xShift"`
	Y    int     `json:"yShift"`
	Zoom float64 `json:"zoom"`
}
type logoConfig struct {
	Abbrev string `json:"abbrev"`
	Pt     *pt    `json:"pt"`
}

type logoInfo struct {
	logo      *logo.Logo
	xPosition int
	yPosition int
}

func (b *nhlBoards) setLogoInfo() error {
	fmt.Println("Using builtin logo positions")
	// Tell pkger to load known assets
	_ = pkger.Include("github.com/robbydyer/sports:/pkg/nhlboard/assets/logopos_64x32.yaml")

	logoAsset := fmt.Sprintf("github.com/robbydyer/sports:/pkg/nhlboard/assets/logopos_%dx%d.yaml",
		b.matrixBounds.Dx(),
		b.matrixBounds.Dy(),
	)
	f, err := pkger.Open(logoAsset)
	if err != nil {
		return fmt.Errorf("could not load logoposition asset %s: %w", logoAsset, err)
	}
	defer f.Close()

	dat, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	var defaultPos []*logoConfig

	if err := yaml.Unmarshal(dat, &defaultPos); err != nil {
		return err
	}

	b.logos = make(map[string]*logoInfo)

	sources, err := logoSources()
	if err != nil {
		return err
	}

	for _, lConf := range b.config.LogoPosition {
		if lConf.Abbrev == "" {
			return fmt.Errorf("invalid configuration for LogoPosition: abbrev required (TEAM_[HOME|AWAY)")
		}
		parts := strings.Split(lConf.Abbrev, "_")
		if len(parts) != 2 {
			return fmt.Errorf("unexpected logo config key '%s'", lConf.Abbrev)
		}
		l := logo.New(parts[0], sources[parts[0]], logoCacheDir, nil)

		b.logos[lConf.Abbrev] = &logoInfo{
			logo:      l,
			xPosition: lConf.Pt.X,
			yPosition: lConf.Pt.Y,
		}
		fmt.Printf("Defining logo for %s: SHIFT %d, %d zoom: %f\n",
			lConf.Abbrev,
			lConf.Pt.X,
			lConf.Pt.Y,
			lConf.Pt.Zoom,
		)
	}

	// Set defaults
	for _, lConf := range defaultPos {
		if _, ok := b.logos[lConf.Abbrev]; !ok {
			parts := strings.Split(lConf.Abbrev, "_")
			if len(parts) != 2 {
				return fmt.Errorf("unexpected logo config key '%s'", lConf.Abbrev)
			}
			l := logo.New(parts[0], sources[parts[0]], logoCacheDir, nil)
			b.logos[lConf.Abbrev] = &logoInfo{
				logo:      l,
				xPosition: lConf.Pt.X,
				yPosition: lConf.Pt.Y,
			}
			fmt.Printf("Defining logo FROM DEFAULTS for %s: SHIFT %d, %d zoom: %f\n",
				lConf.Abbrev,
				lConf.Pt.X,
				lConf.Pt.Y,
				lConf.Pt.Zoom,
			)
		}
	}

	if len(b.logos) != len(ALL)*2 {
		return fmt.Errorf("logo configuration mismatch: %d out of %d expected", len(b.logos), len(ALL)*2)
	}

	return nil
}

func logoSources() (map[string]image.Image, error) {
	sources := make(map[string]image.Image)
	for _, t := range ALL {
		f, err := pkger.Open(fmt.Sprintf("github.com/robbydyer/sports:/assets/logos/%s/%s.png", t, t))
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

func imageRootDir() (string, error) {
	d := "/tmp/sportsmatrix"
	if err := os.MkdirAll(d, 0755); err != nil {
		return "", err
	}
	return d, nil
	/*
		u, err := user.Current()
		if err != nil {
			return "", err
		}
		return u.HomeDir, nil
	*/
}
