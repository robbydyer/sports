package espnboard

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/sportboard"
	"github.com/robbydyer/sports/pkg/util"
)

// DateFormat for getting games
const DateFormat = "20060102"

// Leaguer ...
type Leaguer interface {
	Sport() string
	League() string
}

// ESPNBoard ...
type ESPNBoard struct {
	leaguer         Leaguer
	log             *zap.Logger
	teams           []*Team
	games           map[string][]*Game
	logos           map[string]*logo.Logo
	logoConfOnce    map[string]struct{}
	defaultLogoConf *[]*logo.Config
	allTeams        []string
	logoLock        sync.RWMutex
	logoLockers     map[string]*sync.Mutex
	conferenceNames map[string]struct{}
	sync.Mutex
}

func (e *ESPNBoard) logoCacheDir() (string, error) {
	cacheDir := fmt.Sprintf("tmp/sportsmatrix_logos/%s_%s", e.leaguer.Sport(), e.leaguer.League())
	if _, err := os.Stat(cacheDir); err != nil {
		if os.IsNotExist(err) {
			return cacheDir, os.MkdirAll(cacheDir, 0755)
		}
	}
	return cacheDir, nil
}

// New ...
func New(ctx context.Context, leaguer Leaguer, logger *zap.Logger) (*ESPNBoard, error) {
	e := &ESPNBoard{
		leaguer:         leaguer,
		log:             logger,
		games:           make(map[string][]*Game),
		logos:           make(map[string]*logo.Logo),
		logoConfOnce:    make(map[string]struct{}),
		defaultLogoConf: &[]*logo.Config{},
		logoLockers:     make(map[string]*sync.Mutex),
		conferenceNames: make(map[string]struct{}),
	}

	if _, err := e.GetTeams(ctx); err != nil {
		return nil, err
	}

	if err := e.UpdateGames(ctx, util.Today().Format(DateFormat)); err != nil {
		return nil, err
	}

	c := cron.New()
	if _, err := c.AddFunc("0 5 * * *", func() { e.CacheClear(context.Background()) }); err != nil {
		return nil, fmt.Errorf("failed to set cron job for cacheClear: %w", err)
	}
	c.Start()

	return e, nil
}

// CacheClear ...
func (e *ESPNBoard) CacheClear(ctx context.Context) {
	e.log.Warn("clearing ESPNBoard cache")
	for k := range e.games {
		delete(e.games, k)
	}
	for k := range e.logos {
		delete(e.logos, k)
	}
	e.teams = nil
	if _, err := e.GetTeams(ctx); err != nil {
		e.log.Error("failed to get teams after cache clear", zap.Error(err))
	}
	if err := e.UpdateGames(ctx, util.Today().Format(DateFormat)); err != nil {
		e.log.Error("failed to get today's games", zap.Error(err))
	}
}

// GetTeams ...
func (e *ESPNBoard) GetTeams(ctx context.Context) ([]sportboard.Team, error) {
	if e.teams == nil {
		var err error
		e.teams, err = e.getTeams(ctx)
		if err != nil {
			return nil, err
		}
	}

	var tList []sportboard.Team

	for _, t := range e.teams {
		e.allTeams = append(e.allTeams, t.Abbreviation)
		tList = append(tList, t)

		if t.Conference != nil {
			e.conferenceNames[t.Conference.Abbreviation] = struct{}{}
		}
	}

	return tList, nil
}

// TeamFromAbbreviation ...
func (e *ESPNBoard) TeamFromAbbreviation(ctx context.Context, abbreviation string) (sportboard.Team, error) {
	for _, t := range e.teams {
		if t.Abbreviation == abbreviation {
			return t, nil
		}
	}

	return nil, fmt.Errorf("could not find team '%s'", abbreviation)
}

// GetScheduledGames ...
func (e *ESPNBoard) GetScheduledGames(ctx context.Context, date time.Time) ([]sportboard.Game, error) {
	t := TimeToGameDateStr(date)
	games, ok := e.games[t]
	if !ok || len(games) == 0 {
		if err := e.UpdateGames(ctx, t); err != nil {
			return nil, err
		}
	}

	games, ok = e.games[t]
	if !ok {
		return nil, fmt.Errorf("failed to update games")
	}

	var gList []sportboard.Game

	for _, g := range games {
		gList = append(gList, g)
	}

	return gList, nil
}

// DateStr ...
func (e *ESPNBoard) DateStr(d time.Time) string {
	return d.Format(DateFormat)
}

// League ...
func (e *ESPNBoard) League() string {
	return strings.ToUpper(e.leaguer.League())
}

// HTTPPathPrefix ...
func (e *ESPNBoard) HTTPPathPrefix() string {
	return strings.ToLower(e.leaguer.League())
}

// AllTeamAbbreviations ...
func (e *ESPNBoard) AllTeamAbbreviations() []string {
	return e.allTeams
}

// GetWatchTeams ...
func (e *ESPNBoard) GetWatchTeams(teams []string) []string {
	watch := make(map[string]struct{})
	if len(teams) == 0 {
		teams = append(teams, "ALL")
	}
OUTER:
	for _, t := range teams {
		if t == "ALL" {
			e.log.Info("setting ESPNBoard watch teams to ALL teams")
			return e.AllTeamAbbreviations()
		}
		confTeams := e.TeamsInConference(t)
		if len(confTeams) > 0 {
			for _, team := range confTeams {
				watch[team] = struct{}{}
			}
			continue OUTER
		}
	}

	ret := []string{}
	for k := range watch {
		ret = append(ret, k)
	}

	return ret
}

// TeamsInConference ...
func (e *ESPNBoard) TeamsInConference(conference string) []string {
	conference = strings.ToLower(conference)
	ret := []string{}
	for _, team := range e.teams {
		if strings.Contains(strings.ToLower(team.Conference.Abbreviation), conference) {
			ret = append(ret, team.Abbreviation)
		}
	}

	return ret
}

// UpdateGames ...
func (e *ESPNBoard) UpdateGames(ctx context.Context, dateStr string) error {
	games, err := e.GetGames(ctx, dateStr)
	if err != nil {
		return err
	}

	e.Lock()
	defer e.Unlock()

	e.games[dateStr] = games

	return nil
}

// TeamRank ...
func (e *ESPNBoard) TeamRank(ctx context.Context, team sportboard.Team) string {
	var realTeam *Team
	for _, t := range e.teams {
		if t.Abbreviation == team.GetAbbreviation() {
			realTeam = t
			break
		}
	}

	if realTeam == nil {
		return ""
	}

	if realTeam.rank < 1 {
		return ""
	}

	return strconv.Itoa(realTeam.rank)
}

// TeamRecord ...
func (e *ESPNBoard) TeamRecord(ctx context.Context, team sportboard.Team) string {
	var realTeam *Team
	for _, t := range e.teams {
		if t.Abbreviation == team.GetAbbreviation() {
			realTeam = t
			break
		}
	}

	if realTeam == nil {
		return ""
	}

	return realTeam.record
}
