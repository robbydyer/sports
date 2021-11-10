package espnboard

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/sportboard"
)

// DateFormat for getting games
const DateFormat = "20060102"

type rankSetter func(ctx context.Context, e *ESPNBoard, season string, teams []*Team) error

// Leaguer ...
type Leaguer interface {
	League() string
	APIPath() string
	TeamEndpoints() []string
	HTTPPathPrefix() string
}

// ESPNBoard ...
type ESPNBoard struct {
	leaguer          Leaguer
	rankSetter       rankSetter
	recordSetter     rankSetter
	log              *zap.Logger
	teams            []*Team
	games            map[string][]*Game
	logos            map[string]*logo.Logo
	logoConfOnce     map[string]struct{}
	defaultLogoConf  *[]*logo.Config
	allTeamIDs       []string
	logoLock         sync.RWMutex
	logoLockers      map[string]*sync.Mutex
	conferenceNames  map[string]struct{}
	ranksSet         *atomic.Bool
	rankSorted       *atomic.Bool
	lastScheduleCall map[string]*time.Time
	gameLock         sync.Mutex
	offSeasonLock    sync.Mutex
	offSeason        map[string]bool
	sync.Mutex
}

func (e *ESPNBoard) logoCacheDir() (string, error) {
	cacheDir := fmt.Sprintf("/tmp/sportsmatrix_logos/%s", e.leaguer.APIPath())
	if _, err := os.Stat(cacheDir); err != nil {
		if os.IsNotExist(err) {
			return cacheDir, os.MkdirAll(cacheDir, 0755)
		}
	}
	return cacheDir, nil
}

// New ...
func New(ctx context.Context, leaguer Leaguer, logger *zap.Logger, r rankSetter, rec rankSetter) (*ESPNBoard, error) {
	e := &ESPNBoard{
		leaguer:          leaguer,
		log:              logger,
		games:            make(map[string][]*Game),
		logos:            make(map[string]*logo.Logo),
		logoConfOnce:     make(map[string]struct{}),
		defaultLogoConf:  &[]*logo.Config{},
		logoLockers:      make(map[string]*sync.Mutex),
		conferenceNames:  make(map[string]struct{}),
		rankSetter:       r,
		recordSetter:     rec,
		rankSorted:       atomic.NewBool(false),
		ranksSet:         atomic.NewBool(false),
		lastScheduleCall: make(map[string]*time.Time),
		offSeason:        make(map[string]bool),
	}

	c := cron.New()
	if _, err := c.AddFunc("0 5 * * *", func() { e.CacheClear(context.Background()) }); err != nil {
		return e, fmt.Errorf("failed to set cron job for cacheClear: %w", err)
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
	e.allTeamIDs = []string{}
	e.teams = nil
	e.rankSorted.Store(false)
	e.ranksSet.Store(false)
	if _, err := e.GetTeams(ctx); err != nil {
		e.log.Error("failed to get teams after cache clear", zap.Error(err))
	}
	for k := range e.offSeason {
		delete(e.offSeason, k)
	}
}

// GetTeams ...
func (e *ESPNBoard) GetTeams(ctx context.Context) ([]sportboard.Team, error) {
	var err error
	e.teams, err = e.getTeams(ctx)
	if err != nil {
		return nil, err
	}

	var tList []sportboard.Team

	for _, t := range e.teams {
		e.allTeamIDs = append(e.allTeamIDs, t.ID)
		tList = append(tList, t)

		if t.Conference != nil {
			e.conferenceNames[t.Conference.Abbreviation] = struct{}{}
		}
	}

	return tList, nil
}

// TeamFromID ...
func (e *ESPNBoard) TeamFromID(ctx context.Context, id string) (sportboard.Team, error) {
	if len(e.teams) < 1 {
		if _, err := e.GetTeams(ctx); err != nil {
			return nil, err
		}
	}
	for _, t := range e.teams {
		if t.ID == id {
			return t, nil
		}
	}

	return nil, fmt.Errorf("could not find team '%s'", id)
}

// GetScheduledGames ...
func (e *ESPNBoard) GetScheduledGames(ctx context.Context, dates []time.Time) ([]sportboard.Game, error) {
	var gList []sportboard.Game

	e.offSeasonLock.Lock()
	defer e.offSeasonLock.Unlock()

DATES:
	for _, date := range dates {
		t := TimeToGameDateStr(date)

		off, ok := e.offSeason[t]
		if ok && off {
			e.log.Debug("skipping offseason date",
				zap.String("date", t),
				zap.String("league", e.League()),
			)
			continue DATES
		}

		games, ok := e.games[t]
		if !ok || len(games) == 0 {
			e.log.Info("updating games from API",
				zap.String("league", e.League()),
			)
			if err := e.UpdateGames(ctx, t); err != nil {
				return nil, err
			}
		}

		games, ok = e.games[t]
		if !ok {
			return nil, fmt.Errorf("failed to update games")
		}

		if len(games) < 1 {
			e.offSeason[t] = true
		} else {
			e.offSeason[t] = false
		}

		for _, g := range games {
			gList = append(gList, g)
		}
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
	return e.leaguer.HTTPPathPrefix()
}

// GetWatchTeams returns a list of team ID's
func (e *ESPNBoard) GetWatchTeams(teams []string, season string) []string {
	if len(e.teams) < 1 {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if _, err := e.GetTeams(ctx); err != nil {
			e.log.Error("failed to get teams",
				zap.Error(err),
				zap.String("league", e.League()),
			)
		}
	}
	if len(teams) == 0 {
		e.log.Info("setting ESPNBoard watch teams to ALL teams")
		return e.allTeamIDs
	}

	confs := make([]string, len(e.conferenceNames))
	i := 0
	for k := range e.conferenceNames {
		confs[i] = k
		i++
	}
	e.log.Debug("getting watch teams",
		zap.String("league", e.leaguer.League()),
		zap.Strings("conferences", confs),
		zap.Strings("input", teams),
	)

	watch := make(map[string]struct{})

OUTER:
	for _, t := range teams {
		if t == "ALL" {
			e.log.Info("setting ESPNBoard watch teams to ALL teams")
			return e.allTeamIDs
		}
		if strings.HasPrefix(t, "TOP") {
			fields := strings.Split(t, "TOP")
			if len(fields) < 2 {
				continue OUTER
			}
			top, err := strconv.Atoi(fields[1])
			if err != nil {
				e.log.Error("failed to convert TOP rank",
					zap.Error(err),
				)
			}
			for _, a := range e.teamsInRank(top, season) {
				watch[a.GetID()] = struct{}{}
			}
			continue OUTER
		}
		for _, team := range e.teams {
			if team.GetAbbreviation() == t {
				watch[team.GetID()] = struct{}{}
			}
		}
		for _, team := range e.TeamsInConference(t) {
			watch[team.GetID()] = struct{}{}
		}
	}

	ret := make([]string, len(watch))
	i = 0
	for k := range watch {
		ret[i] = k
		i++
	}

	return ret
}

// TeamsInConference ...
func (e *ESPNBoard) TeamsInConference(conference string) []*Team {
	if len(e.teams) < 1 {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if _, err := e.GetTeams(ctx); err != nil {
			e.log.Error("failed to get teams",
				zap.Error(err),
				zap.String("league", e.League()),
			)
			return nil
		}
	}
	conference = strings.ToLower(conference)
	found := false
	for c := range e.conferenceNames {
		if strings.Contains(strings.ToLower(c), conference) {
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	e.log.Debug("checking conference for teams", zap.String("conference", conference))
	ret := []*Team{}

	for _, team := range e.teams {
		if team.Conference == nil {
			continue
		}
		if strings.Contains(conference, strings.ToLower(team.Conference.Abbreviation)) || strings.Contains(strings.ToLower(team.Conference.Abbreviation), conference) {
			ret = append(ret, team)
		}
	}

	return ret
}

// teamsInRank grabs all teams within the top X number of rankings
func (e *ESPNBoard) teamsInRank(top int, season string) []*Team {
	if len(e.teams) < 1 {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if _, err := e.GetTeams(ctx); err != nil {
			e.log.Error("failed to get teams",
				zap.Error(err),
				zap.String("league", e.League()),
			)
			return nil
		}
	}
	if top < 1 {
		return []*Team{}
	}
	e.log.Debug("fetching teams in rank",
		zap.Int("top", top),
		zap.String("league", e.League()),
	)
	teams := []*Team{}
	if !e.rankSorted.Load() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := e.rankSetter(ctx, e, season, e.teams); err != nil {
			e.log.Error("failed to set team rankings",
				zap.Error(err),
				zap.String("league", e.League()),
			)
		}
		sort.SliceStable(e.teams, func(i, j int) bool {
			return e.teams[i].rank < e.teams[j].rank
		})
		e.rankSorted.Store(true)
	}

	index := 1
	for _, t := range e.teams {
		if index > top {
			break
		}
		if t.rank != 0 {
			teams = append(teams, t)
			index++
		}
	}

	return teams
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
func (e *ESPNBoard) TeamRank(ctx context.Context, team sportboard.Team, season string) string {
	if len(e.teams) < 1 {
		if _, err := e.GetTeams(ctx); err != nil {
			e.log.Error("failed to get teams",
				zap.Error(err),
				zap.String("league", e.League()),
			)
			return ""
		}
	}
	var realTeam *Team
	for _, t := range e.teams {
		if t.ID == team.GetID() {
			realTeam = t
			break
		}
	}

	if realTeam == nil {
		return ""
	}

	if realTeam.rank > 0 {
		return strconv.Itoa(realTeam.rank)
	}

	if err := e.rankSetter(ctx, e, season, []*Team{realTeam}); err != nil {
		e.log.Error("failed to set team details", zap.Error(err))
	}

	if realTeam.rank == 0 {
		return ""
	}

	return strconv.Itoa(realTeam.rank)
}

// TeamRecord ...
func (e *ESPNBoard) TeamRecord(ctx context.Context, team sportboard.Team, season string) string {
	if len(e.teams) < 1 {
		if _, err := e.GetTeams(ctx); err != nil {
			e.log.Error("failed to get teams",
				zap.Error(err),
				zap.String("league", e.League()),
			)
			return ""
		}
	}
	var realTeam *Team
	for _, t := range e.teams {
		if t.ID == team.GetID() {
			realTeam = t
			break
		}
	}

	if realTeam == nil {
		return ""
	}

	if err := e.recordSetter(ctx, e, season, []*Team{realTeam}); err != nil {
		e.log.Error("failed to set team details", zap.Error(err))
	}

	return realTeam.record
}
