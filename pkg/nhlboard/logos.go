package nhlboard

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"

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
	/*
		var dat []byte
				if _, err := os.Stat(logoPosFile); err != nil {
					fmt.Println("Using builtin logo positions")
					f, err := pkger.Open("github.com/robbydyer/sports:/pkg/nhlboard/logo_position.yaml")
					if err != nil {
						return nil, err
					}
					defer f.Close()

					dat, err = ioutil.ReadAll(f)
					if err != nil {
						return nil, err
					}
				} else {
					var err error
					dat, err = ioutil.ReadFile(logoPosFile)
					if err != nil {
						return nil, err
					}
				}
			var err error
			dat, err = ioutil.ReadFile(logoPosFile)
			if err != nil {
				return err
			}
	*/
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
		l := logo.New(parts[0], sources[parts[0]], logoCacheDir, lConf.Pt.Zoom)

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
