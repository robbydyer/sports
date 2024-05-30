package nhl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	sportboard "github.com/robbydyer/sports/internal/board/sport"
	"github.com/robbydyer/sports/internal/espn"
	"github.com/robbydyer/sports/internal/logo"
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

	c := cron.New()
	if _, err := c.AddFunc("0 5 * * *", func() { n.CacheClear(ctx) }); err != nil {
		return n, fmt.Errorf("failed to set cron job for cacheClear: %w", err)
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
	if err := n.UpdateTeams(ctx); err != nil {
		n.log.Error("failed to update teams", zap.Error(err))
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

// GetWatchTeams returns a list of team ID's
func (n *NHL) GetWatchTeams(teams []string, season string) []string {
	if len(n.teams) < 1 {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := n.UpdateTeams(ctx); err != nil {
			n.log.Error("failed to update nhl teams",
				zap.Error(err),
			)
			return []string{}
		}
	}
	watch := make(map[string]struct{})
	for _, t := range teams {
		if t == "ALL" {
			n.log.Info("setting NHL watch teams to ALL teams")
			ids := []string{}
			for _, t := range n.teams {
				ids = append(ids, t.GetID())
			}
			return ids
		}

	INNER:
		for _, team := range n.teams {
			if team.Division != nil && team.Division.Abbreviation == t {
				watch[team.GetID()] = struct{}{}
				continue INNER
			}
			if team.GetAbbreviation() == t {
				watch[team.GetID()] = struct{}{}
			}
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

// TeamFromID ...
func (n *NHL) TeamFromID(ctx context.Context, id string) (sportboard.Team, error) {
	for _, t := range n.teams {
		if t.GetID() == id {
			return t, nil
		}
	}

	return nil, fmt.Errorf("could not find team '%s'", id)
}

// GetScheduledGames ...
func (n *NHL) GetScheduledGames(ctx context.Context, dates []time.Time) ([]sportboard.Game, error) {
	var gList []sportboard.Game

	for _, date := range dates {
		dateStr := n.DateStr(date)
		games, ok := n.games[dateStr]
		if !ok || len(games) == 0 {
			if err := n.UpdateGames(ctx, dateStr); err != nil {
				return nil, err
			}
		}

		for _, g := range games {
			gList = append(gList, g)
		}
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

// LeagueShortName ...
func (n *NHL) LeagueShortName() string {
	return "NHL"
}

// UpdateTeams ...
func (n *NHL) UpdateTeams(ctx context.Context) error {
	teamList, err := n.getTeams(ctx)
	if err != nil {
		return err
	}

	n.teams = teamList

	return nil
}

// UpdateGames ...
func (n *NHL) UpdateGames(ctx context.Context, dateStr string) error {
	n.Lock()
	defer n.Unlock()
	games, err := getGames(ctx, dateStr)
	if err != nil {
		return err
	}

	n.games[dateStr] = games

	return nil
}

// TeamRecord ...
func (n *NHL) TeamRecord(ctx context.Context, team sportboard.Team, season string) string {
	return ""
}

// TeamRank ...
func (n *NHL) TeamRank(ctx context.Context, team sportboard.Team, season string) string {
	return ""
}

func (n *NHL) HomeSideSwap() bool {
	return false
}

// GetSeason gets the season identifier based on a date, i.e. 20202021
func GetSeason(day time.Time) string {
	year := day.Year()

	if day.Month() > 6 {
		return fmt.Sprintf("%d%d", year, year+1)
	}

	return fmt.Sprintf("%d%d", year-1, year)
}
