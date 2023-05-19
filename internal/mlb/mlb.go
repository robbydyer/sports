package mlb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/robfig/cron/v3"

	sportboard "github.com/robbydyer/sports/internal/board/sport"
	"github.com/robbydyer/sports/internal/espn"
	"github.com/robbydyer/sports/internal/logo"
)

const (
	baseURL      = "https://statsapi.mlb.com/api"
	linkBase     = "https://statsapi.mlb.com"
	logoCacheDir = "/tmp/sportsmatrix_logos/mlb"

	// DateFormat is the game schedule format for querying a particular day from the API
	DateFormat = "2006-01-02"
)

// DefaultLogoConfigs contains default logo alignedment configurations
type DefaultLogoConfigs *[]*logo.Config

// MLB implements a sportboard.API
type MLB struct {
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
func New(ctx context.Context, logger *zap.Logger) (*MLB, error) {
	m := &MLB{
		games:   make(map[string][]*Game),
		logos:   make(map[string]*logo.Logo),
		log:     logger,
		espnAPI: espn.New(logger),
	}

	c := cron.New()
	if _, err := c.AddFunc("0 5 * * *", func() { m.CacheClear(context.Background()) }); err != nil {
		return m, fmt.Errorf("failed to set cron job for cacheClear: %w", err)
	}
	c.Start()

	return m, nil
}

// CacheClear ...
func (m *MLB) CacheClear(ctx context.Context) {
	m.log.Warn("clearing MLB cache")
	for k := range m.games {
		delete(m.games, k)
	}
	for k := range m.logos {
		delete(m.logos, k)
	}
	_ = m.espnAPI.ClearCache()
}

// HTTPPathPrefix returns the path prefix for the HTTP handlers for this board
func (m *MLB) HTTPPathPrefix() string {
	return "mlb"
}

// GetTeams ...
func (m *MLB) GetTeams(ctx context.Context) ([]sportboard.Team, error) {
	if m.teams == nil {
		if err := m.UpdateTeams(ctx); err != nil {
			return nil, err
		}
	}

	var tList []sportboard.Team

	for _, t := range m.teams {
		m.log.Debug("got team", zap.String("team", t.Abbreviation), zap.Int("num players", len(t.Roster)))
		tList = append(tList, t)
	}

	return tList, nil
}

// TeamFromID ...
func (m *MLB) TeamFromID(ctx context.Context, id string) (sportboard.Team, error) {
	if len(m.teams) < 1 {
		if _, err := m.GetTeams(ctx); err != nil {
			return nil, err
		}
	}
	for _, t := range m.teams {
		if t.GetID() == id {
			return t, nil
		}
	}

	return nil, fmt.Errorf("could not find team '%s'", id)
}

// GetScheduledGames ...
func (m *MLB) GetScheduledGames(ctx context.Context, dates []time.Time) ([]sportboard.Game, error) {
	var gList []sportboard.Game

	for _, date := range dates {
		dateStr := m.DateStr(date)
		games, ok := m.games[dateStr]
		if !ok || len(games) == 0 {
			if err := m.UpdateGames(ctx, dateStr); err != nil {
				return nil, err
			}
		}

		games, ok = m.games[dateStr]
		if !ok {
			return nil, fmt.Errorf("failed to update games")
		}

		for _, g := range games {
			gList = append(gList, g)
		}
	}

	return gList, nil
}

// DateStr ...
func (m *MLB) DateStr(d time.Time) string {
	return d.Format(DateFormat)
}

// League ...
func (m *MLB) League() string {
	return "MLB"
}

// LeagueShortName ...
func (m *MLB) LeagueShortName() string {
	return "MLB"
}

// AllTeamAbbreviations returns a list of all teams in the league
func (m *MLB) AllTeamAbbreviations() []string {
	return ALL
}

// GetWatchTeams parses 'ALL' or divisions and adds teams accordingly
func (m *MLB) GetWatchTeams(teams []string, season string) []string {
	if len(m.teams) < 1 {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if _, err := m.GetTeams(ctx); err != nil {
			m.log.Error("failed to get mlb teams",
				zap.Error(err),
			)
		}
	}
	watch := make(map[string]struct{})
	for _, t := range teams {
		if t == "ALL" {
			m.log.Info("setting NHL watch teams to ALL teams")
			ids := []string{}
			for _, t := range m.teams {
				ids = append(ids, t.GetID())
			}
			return ids
		}

	INNER:
		for _, team := range m.teams {
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

// UpdateTeams ...
func (m *MLB) UpdateTeams(ctx context.Context) error {
	teamList, err := GetTeams(ctx)
	if err != nil {
		return err
	}

	m.teams = teamList

	return nil
}

// UpdateGames updates scheduled games
func (m *MLB) UpdateGames(ctx context.Context, dateStr string) error {
	m.Lock()
	defer m.Unlock()
	games, err := getGames(ctx, dateStr)
	if err != nil {
		return err
	}

	m.games[dateStr] = games

	return nil
}

// TeamRecord ...
func (m *MLB) TeamRecord(ctx context.Context, team sportboard.Team, season string) string {
	return ""
}

// TeamRank ...
func (m *MLB) TeamRank(ctx context.Context, team sportboard.Team, season string) string {
	return ""
}

func (m *MLB) HomeSideSwap() bool {
	return false
}

// GetSeason gets the season identifier based on a date, i.e. 2020
func GetSeason(day time.Time) string {
	return fmt.Sprint(day.Year())
}
