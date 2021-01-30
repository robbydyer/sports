package nhl

import (
	"context"
	"fmt"
	"image"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/sportboard"
	"github.com/robbydyer/sports/pkg/util"
)

const (
	BaseURL      = "https://statsapi.web.nhl.com/api/v1/"
	LinkBase     = "https://statsapi.web.nhl.com"
	DateFormat   = "2006-01-02"
	logoCacheDir = "/tmp/sportsmatrix_logos/nhl"
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

type NHL struct {
	teams           []*Team
	games           map[string][]*Game
	logos           map[string]*logo.Logo
	logoSourceCache map[string]image.Image
	log             *log.Logger
}

func New(ctx context.Context, logger *log.Logger) (*NHL, error) {
	n := &NHL{
		games:           make(map[string][]*Game),
		logos:           make(map[string]*logo.Logo),
		logoSourceCache: make(map[string]image.Image),
		log:             logger,
	}

	if err := n.UpdateTeams(ctx); err != nil {
		return nil, err
	}

	if err := n.UpdateGames(ctx, util.Today().Format(DateFormat)); err != nil {
		return nil, fmt.Errorf("failed to get today's games: %w", err)
	}

	return n, nil
}

func (n *NHL) AllTeamAbbreviations() []string {
	return ALL
}

func (n *NHL) GetTeams(ctx context.Context) ([]sportboard.Team, error) {
	if n.teams == nil {
		if err := n.UpdateTeams(ctx); err != nil {
			return nil, err
		}
	}

	var tList []sportboard.Team

	for _, t := range n.teams {
		tList = append(tList, t)
	}

	return tList, nil
}
func (n *NHL) TeamFromAbbreviation(ctx context.Context, abbreviation string) (sportboard.Team, error) {
	for _, t := range n.teams {
		if t.Abbreviation == abbreviation {
			return t, nil
		}
	}

	return nil, fmt.Errorf("could not find team '%s'", abbreviation)
}

func (n *NHL) GetScheduledGames(ctx context.Context, date time.Time) ([]sportboard.Game, error) {
	dateStr := n.DateStr(date)
	games, ok := n.games[dateStr]
	if !ok || len(games) == 0 {
		if err := n.UpdateGames(ctx, dateStr); err != nil {
			return nil, err
		}
	}

	var gList []sportboard.Game

	for _, g := range games {
		gList = append(gList, g)
	}

	return gList, nil
}

func (n *NHL) DateStr(d time.Time) string {
	return d.Format(DateFormat)
}
func (n *NHL) League() string {
	return "NHL"
}

func (n *NHL) UpdateTeams(ctx context.Context) error {
	teamList, err := GetTeams(ctx)
	if err != nil {
		return err
	}

	n.teams = teamList

	return nil
}

func (n *NHL) UpdateGames(ctx context.Context, dateStr string) error {
	games, err := getGames(ctx, dateStr)
	if err != nil {
		return err
	}

	n.games[dateStr] = games

	return nil
}
