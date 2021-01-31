package nhl

import (
	"context"
	"fmt"
	"image"
	"time"

	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/sportboard"
	"github.com/robbydyer/sports/pkg/util"
)

const (
	baseURL  = "https://statsapi.web.nhl.com/api/v1/"
	linkBase = "https://statsapi.web.nhl.com"
	// DateFormat ...
	DateFormat   = "2006-01-02"
	logoCacheDir = "/tmp/sportsmatrix_logos/nhl"
)

// ALL is a list of all teams in the league
var ALL = []string{ANA, ARI, BOS, BUF, CAR, CBJ, CGY, CHI, COL, DAL, DET, EDM, FLA, LAK, MIN, MTL, NJD, NSH, NYI, NYR, OTT, PHI, PIT, SJS, STL, TBL, TOR, VAN, VGK, WPG, WSH}

// NHL implements sportboard.API
type NHL struct {
	teams           []*Team
	games           map[string][]*Game
	logos           map[string]*logo.Logo
	logoSourceCache map[string]image.Image
	log             *log.Logger
}

// New ...
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

	c := cron.New()
	if _, err := c.AddFunc("0 5 * * *", n.cacheClear); err != nil {
		return nil, fmt.Errorf("failed to set cron job for cacheClear: %w", err)
	}
	c.Start()

	return n, nil
}

// CacheClear ...
func (n *NHL) cacheClear() {
	for k := range n.games {
		delete(n.games, k)
	}
	if err := n.UpdateGames(context.Background(), util.Today().Format(DateFormat)); err != nil {
		n.log.Errorf("failed to get today's games: %s", err.Error())
	}
}

func (n *NHL) HTTPPathPrefix() string {
	return "nhl"
}

// AllTeamAbbreviations ...
func (n *NHL) AllTeamAbbreviations() []string {
	return ALL
}

// GetTeams ...
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

// TeamFromAbbreviation ...
func (n *NHL) TeamFromAbbreviation(ctx context.Context, abbreviation string) (sportboard.Team, error) {
	for _, t := range n.teams {
		if t.Abbreviation == abbreviation {
			return t, nil
		}
	}

	return nil, fmt.Errorf("could not find team '%s'", abbreviation)
}

// GetScheduledGames ...
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

// DateStr ...
func (n *NHL) DateStr(d time.Time) string {
	return d.Format(DateFormat)
}

// League ...
func (n *NHL) League() string {
	return "NHL"
}

// UpdateTeams ...
func (n *NHL) UpdateTeams(ctx context.Context) error {
	teamList, err := GetTeams(ctx)
	if err != nil {
		return err
	}

	n.teams = teamList

	return nil
}

// UpdateGames ...
func (n *NHL) UpdateGames(ctx context.Context, dateStr string) error {
	games, err := getGames(ctx, dateStr)
	if err != nil {
		return err
	}

	n.games[dateStr] = games

	return nil
}
