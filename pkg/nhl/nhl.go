package nhl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/espn"
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
	logoLock        sync.RWMutex
	log             *zap.Logger
	defaultLogoConf *[]*logo.Config
	espnAPI         *espn.ESPN
	sync.Mutex
}

// New ...
func New(ctx context.Context, logger *zap.Logger) (*NHL, error) {
	n := &NHL{
		games:   make(map[string][]*Game),
		logos:   make(map[string]*logo.Logo),
		log:     logger,
		espnAPI: espn.New(logger),
	}

	if err := n.UpdateTeams(ctx); err != nil {
		return nil, err
	}

	if err := n.UpdateGames(ctx, util.Today().Format(DateFormat)); err != nil {
		return nil, fmt.Errorf("failed to get today's games: %w", err)
	}

	c := cron.New()
	if _, err := c.AddFunc("0 5 * * *", func() { n.CacheClear(context.Background()) }); err != nil {
		return nil, fmt.Errorf("failed to set cron job for cacheClear: %w", err)
	}
	c.Start()

	return n, nil
}

// CacheClear ...
func (n *NHL) CacheClear(ctx context.Context) {
	n.log.Warn("clearing NHL cache")
	for k := range n.games {
		delete(n.games, k)
	}
	if err := n.UpdateGames(context.Background(), util.Today().Format(DateFormat)); err != nil {
		n.log.Error("failed to get today's games", zap.Error(err))
	}
	for k := range n.logos {
		delete(n.logos, k)
	}
	_ = n.espnAPI.ClearCache()
}

// HTTPPathPrefix returns the prefix of HTTP calls to this board. i.e. /nhl/foo
func (n *NHL) HTTPPathPrefix() string {
	return "nhl"
}

// AllTeamAbbreviations ...
func (n *NHL) AllTeamAbbreviations() []string {
	return ALL
}

// GetWatchTeams ...
func (n *NHL) GetWatchTeams(teams []string) []string {
	watch := make(map[string]struct{})
	for _, t := range teams {
		if t == "ALL" {
			n.log.Info("setting NHL watch teams to ALL teams")
			return n.AllTeamAbbreviations()
		}
		isDiv := false
	INNER:
		for _, team := range n.teams {
			if team.Division == nil {
				continue INNER
			}
			if team.Division.Abbreviation == t {
				watch[team.Abbreviation] = struct{}{}
				isDiv = true
			}
		}
		if !isDiv {
			watch[t] = struct{}{}
		}
	}

	var w []string

	for t := range watch {
		w = append(w, t)
	}

	return w
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

// TeamRecord ...
func (n *NHL) TeamRecord(ctx context.Context, team sportboard.Team) string {
	return ""
}

// TeamRank ...
func (n *NHL) TeamRank(ctx context.Context, team sportboard.Team) string {
	return ""
}
