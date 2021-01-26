package nhlboard

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"

	"github.com/markbates/pkger"
	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/rgbrender"
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

type logoInfo struct {
	logo      *logo.Logo
	xPosition int
	yPosition int
}

func getLogos() (map[string]*logoInfo, error) {

	l := make(map[string]*logoInfo)

	sources, err := logoSources()
	if err != nil {
		return nil, err
	}

	l[ANA+"_HOME"] = &logoInfo{
		logo:      logo.New(ANA, sources[ANA], logoCacheDir, 0.8),
		xPosition: -22,
		yPosition: 3,
	}
	l[ANA+"_AWAY"] = &logoInfo{
		logo:      logo.New(ANA, sources[ANA], logoCacheDir, 0.8),
		xPosition: 7,
		yPosition: 3,
	}

	l[ARI+"_HOME"] = &logoInfo{
		logo:      logo.New(ARI, sources[ARI], logoCacheDir, 1),
		xPosition: -14,
		yPosition: 0,
	}
	l[ARI+"_AWAY"] = &logoInfo{
		logo:      logo.New(ARI, sources[ARI], logoCacheDir, 1),
		xPosition: 3,
		yPosition: 0,
	}

	l[BOS+"_HOME"] = &logoInfo{
		logo:      logo.New(BOS, sources[BOS], logoCacheDir, 1),
		xPosition: -4,
		yPosition: 0,
	}
	l[BOS+"_AWAY"] = &logoInfo{
		logo:      logo.New(BOS, sources[BOS], logoCacheDir, 1),
		xPosition: 4,
		yPosition: 0,
	}

	l[BUF+"_HOME"] = &logoInfo{
		logo:      logo.New(BUF, sources[BUF], logoCacheDir, 1),
		xPosition: -4,
		yPosition: 0,
	}
	l[BUF+"_AWAY"] = &logoInfo{
		logo:      logo.New(BUF, sources[BUF], logoCacheDir, 1),
		xPosition: 4,
		yPosition: 0,
	}

	l[CGY+"_HOME"] = &logoInfo{
		logo:      logo.New(CGY, sources[CGY], logoCacheDir, 1),
		xPosition: -16,
		yPosition: 0,
	}
	l[CGY+"_AWAY"] = &logoInfo{
		logo:      logo.New(CGY, sources[CGY], logoCacheDir, 1),
		xPosition: 5,
		yPosition: 0,
	}

	l[CAR+"_HOME"] = &logoInfo{
		logo:      logo.New(CAR, sources[CAR], logoCacheDir, 1),
		xPosition: -16,
		yPosition: 0,
	}
	l[CAR+"_AWAY"] = &logoInfo{
		logo:      logo.New(CAR, sources[CAR], logoCacheDir, 1),
		xPosition: 15,
		yPosition: 0,
	}

	l[CHI+"_HOME"] = &logoInfo{
		logo:      logo.New(CHI, sources[CHI], logoCacheDir, 1),
		xPosition: -4,
		yPosition: 0,
	}
	l[CHI+"_AWAY"] = &logoInfo{
		logo:      logo.New(CHI, sources[CHI], logoCacheDir, 1),
		xPosition: 20,
		yPosition: 0,
	}

	l[COL+"_HOME"] = &logoInfo{
		logo:      logo.New(COL, sources[COL], logoCacheDir, 1),
		xPosition: -5,
		yPosition: 0,
	}
	l[COL+"_AWAY"] = &logoInfo{
		logo:      logo.New(COL, sources[COL], logoCacheDir, 1),
		xPosition: 5,
		yPosition: 0,
	}

	l[CBJ+"_HOME"] = &logoInfo{
		logo:      logo.New(CBJ, sources[CBJ], logoCacheDir, 1),
		xPosition: -15,
		yPosition: 0,
	}
	l[CBJ+"_AWAY"] = &logoInfo{
		logo:      logo.New(CBJ, sources[CBJ], logoCacheDir, 1),
		xPosition: 8,
		yPosition: 0,
	}

	l[DAL+"_HOME"] = &logoInfo{
		logo:      logo.New(DAL, sources[DAL], logoCacheDir, 1),
		xPosition: -10,
		yPosition: 0,
	}
	l[DAL+"_AWAY"] = &logoInfo{
		logo:      logo.New(DAL, sources[DAL], logoCacheDir, 1),
		xPosition: 12,
		yPosition: 0,
	}

	l[DET+"_HOME"] = &logoInfo{
		logo:      logo.New(DET, sources[DET], logoCacheDir, 1),
		xPosition: -5,
		yPosition: 0,
	}
	l[DET+"_AWAY"] = &logoInfo{
		logo:      logo.New(DET, sources[DET], logoCacheDir, 1),
		xPosition: 24,
		yPosition: 0,
	}

	l[EDM+"_HOME"] = &logoInfo{
		logo:      logo.New(EDM, sources[EDM], logoCacheDir, 1),
		xPosition: -3,
		yPosition: 0,
	}
	l[EDM+"_AWAY"] = &logoInfo{
		logo:      logo.New(EDM, sources[EDM], logoCacheDir, 1),
		xPosition: 3,
		yPosition: 0,
	}

	l[FLA+"_HOME"] = &logoInfo{
		logo:      logo.New(FLA, sources[FLA], logoCacheDir, 1),
		xPosition: -3,
		yPosition: 0,
	}
	l[FLA+"_AWAY"] = &logoInfo{
		logo:      logo.New(FLA, sources[FLA], logoCacheDir, 1),
		xPosition: 3,
		yPosition: 0,
	}

	l[LAK+"_HOME"] = &logoInfo{
		logo:      logo.New(LAK, sources[LAK], logoCacheDir, 0.9),
		xPosition: -1,
		yPosition: 2,
	}
	l[LAK+"_AWAY"] = &logoInfo{
		logo:      logo.New(LAK, sources[LAK], logoCacheDir, 0.9),
		xPosition: 1,
		yPosition: 2,
	}

	l[MIN+"_HOME"] = &logoInfo{
		logo:      logo.New(MIN, sources[MIN], logoCacheDir, 1),
		xPosition: -20,
		yPosition: 0,
	}
	l[MIN+"_AWAY"] = &logoInfo{
		logo:      logo.New(MIN, sources[MIN], logoCacheDir, 1),
		xPosition: 13,
		yPosition: 0,
	}

	l[MTL+"_HOME"] = &logoInfo{
		logo:      logo.New(MTL, sources[MTL], logoCacheDir, 0.9),
		xPosition: -3,
		yPosition: 2,
	}
	l[MTL+"_AWAY"] = &logoInfo{
		logo:      logo.New(MTL, sources[MTL], logoCacheDir, 0.9),
		xPosition: 3,
		yPosition: 2,
	}

	l[NJD+"_HOME"] = &logoInfo{
		logo:      logo.New(NJD, sources[NJD], logoCacheDir, 1),
		xPosition: -4,
		yPosition: 0,
	}
	l[NJD+"_AWAY"] = &logoInfo{
		logo:      logo.New(NJD, sources[NJD], logoCacheDir, 1),
		xPosition: 4,
		yPosition: 0,
	}

	l[NSH+"_HOME"] = &logoInfo{
		logo:      logo.New(NSH, sources[NSH], logoCacheDir, 1),
		xPosition: -4,
		yPosition: 0,
	}
	l[NSH+"_AWAY"] = &logoInfo{
		logo:      logo.New(NSH, sources[NSH], logoCacheDir, 1),
		xPosition: 4,
		yPosition: 0,
	}

	l[NYI+"_HOME"] = &logoInfo{
		logo:      logo.New(NYI, sources[NYI], logoCacheDir, 1),
		xPosition: -3,
		yPosition: 0,
	}
	l[NYI+"_AWAY"] = &logoInfo{
		logo:      logo.New(NYI, sources[NYI], logoCacheDir, 1),
		xPosition: 3,
		yPosition: 0,
	}

	l[NYR+"_HOME"] = &logoInfo{
		logo:      logo.New(NYR, sources[NYR], logoCacheDir, 1),
		xPosition: -3,
		yPosition: 0,
	}
	l[NYR+"_AWAY"] = &logoInfo{
		logo:      logo.New(NYR, sources[NYR], logoCacheDir, 1),
		xPosition: 3,
		yPosition: 0,
	}

	l[OTT+"_HOME"] = &logoInfo{
		logo:      logo.New(OTT, sources[OTT], logoCacheDir, 1),
		xPosition: -22,
		yPosition: 0,
	}
	l[OTT+"_AWAY"] = &logoInfo{
		logo:      logo.New(OTT, sources[OTT], logoCacheDir, 1),
		xPosition: 3,
		yPosition: 0,
	}

	l[PHI+"_HOME"] = &logoInfo{
		logo:      logo.New(PHI, sources[PHI], logoCacheDir, 0.9),
		xPosition: -21,
		yPosition: 2,
	}
	l[PHI+"_AWAY"] = &logoInfo{
		logo:      logo.New(PHI, sources[PHI], logoCacheDir, 0.9),
		xPosition: 8,
		yPosition: 2,
	}

	l[PIT+"_HOME"] = &logoInfo{
		logo:      logo.New(PIT, sources[PIT], logoCacheDir, 1),
		xPosition: -3,
		yPosition: 0,
	}
	l[PIT+"_AWAY"] = &logoInfo{
		logo:      logo.New(PIT, sources[PIT], logoCacheDir, 1),
		xPosition: 18,
		yPosition: 0,
	}

	l[SJS+"_HOME"] = &logoInfo{
		logo:      logo.New(SJS, sources[SJS], logoCacheDir, 1),
		xPosition: -16,
		yPosition: 0,
	}
	l[SJS+"_AWAY"] = &logoInfo{
		logo:      logo.New(SJS, sources[SJS], logoCacheDir, 1),
		xPosition: 7,
		yPosition: 0,
	}

	l[STL+"_HOME"] = &logoInfo{
		logo:      logo.New(STL, sources[STL], logoCacheDir, 1),
		xPosition: -4,
		yPosition: 0,
	}
	l[STL+"_AWAY"] = &logoInfo{
		logo:      logo.New(STL, sources[STL], logoCacheDir, 1),
		xPosition: 20,
		yPosition: 0,
	}

	l[TBL+"_HOME"] = &logoInfo{
		logo:      logo.New(TBL, sources[TBL], logoCacheDir, 1),
		xPosition: -3,
		yPosition: 0,
	}
	l[TBL+"_AWAY"] = &logoInfo{
		logo:      logo.New(TBL, sources[TBL], logoCacheDir, 1),
		xPosition: 5,
		yPosition: 0,
	}

	l[TOR+"_HOME"] = &logoInfo{
		logo:      logo.New(TOR, sources[TOR], logoCacheDir, 1),
		xPosition: -4,
		yPosition: 0,
	}
	l[TOR+"_AWAY"] = &logoInfo{
		logo:      logo.New(TOR, sources[TOR], logoCacheDir, 1),
		xPosition: 4,
		yPosition: 0,
	}

	l[VAN+"_HOME"] = &logoInfo{
		logo:      logo.New(VAN, sources[VAN], logoCacheDir, 1),
		xPosition: -17,
		yPosition: 0,
	}
	l[VAN+"_AWAY"] = &logoInfo{
		logo:      logo.New(VAN, sources[VAN], logoCacheDir, 1),
		xPosition: 5,
		yPosition: 0,
	}

	l[VGK+"_HOME"] = &logoInfo{
		logo:      logo.New(VGK, sources[VGK], logoCacheDir, 1),
		xPosition: -2,
		yPosition: 0,
	}
	l[VGK+"_AWAY"] = &logoInfo{
		logo:      logo.New(VGK, sources[VGK], logoCacheDir, 1),
		xPosition: 2,
		yPosition: 0,
	}

	l[WSH+"_HOME"] = &logoInfo{
		logo:      logo.New(WSH, sources[WSH], logoCacheDir, 1.2),
		xPosition: -5,
		yPosition: 3,
	}
	l[WSH+"_AWAY"] = &logoInfo{
		logo:      logo.New(WSH, sources[WSH], logoCacheDir, 1.2),
		xPosition: 40,
		yPosition: -3,
	}

	l[WPG+"_HOME"] = &logoInfo{
		logo:      logo.New(WPG, sources[WPG], logoCacheDir, 1),
		xPosition: -2,
		yPosition: 0,
	}
	l[WPG+"_AWAY"] = &logoInfo{
		logo:      logo.New(WPG, sources[WPG], logoCacheDir, 1),
		xPosition: 2,
		yPosition: 0,
	}

	return l, nil
}

func (c *Config) setDefaultPositions() {
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

func (n *nhlBoards) logoShift(key string) (image.Rectangle, error) {
	if _, ok := n.logos[key]; !ok {
		return image.Rectangle{}, fmt.Errorf("logo for key %s not found", key)
	}

	// Away teams are on the right side
	if strings.Contains(key, "AWAY") {
		return rgbrender.ShiftedSize(
			n.logos[key].xPosition+(n.matrixBounds.Dx()/2),
			n.logos[key].yPosition,
			n.matrixBounds,
		), nil
	}
	return rgbrender.ShiftedSize(
		n.logos[key].xPosition,
		n.logos[key].yPosition,
		n.matrixBounds,
	), nil
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
