package ncaam

import (
	"context"
	"fmt"
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

const logoCacheDir = "/tmp/sportsmatrix_logos/ncaam"

// DateFormat for getting games
const DateFormat = "20060102"

// Conferences ...
var Conferences = []string{
	"a-sun",
	"aac",
	"acc",
	"aeast",
	"atl10",
	"big10",
	"big12",
	"bige",
	"bigw",
	"bsky",
	"bsou",
	"col",
	"hor",
	"ivy",
	"maac",
	"meac",
	"midam",
	"mvc",
	"mwest",
	"neast",
	"ovc",
	"pac12",
	"pat",
	"sec",
	"usa",
}

// NcaaM implements sportboard.API
type NcaaM struct {
	log             *zap.Logger
	teams           []*Team
	games           map[string][]*Game
	logos           map[string]*logo.Logo
	logoLock        sync.RWMutex
	defaultLogoConf *[]*logo.Config
	logoConfOnce    map[string]struct{}
	allTeams        []string
	sync.Mutex
}

// New ...
func New(ctx context.Context, logger *zap.Logger) (*NcaaM, error) {
	n := &NcaaM{
		log:             logger,
		logos:           make(map[string]*logo.Logo),
		logoConfOnce:    make(map[string]struct{}),
		games:           make(map[string][]*Game),
		defaultLogoConf: &[]*logo.Config{},
	}

	if _, err := n.GetTeams(ctx); err != nil {
		return nil, err
	}

	if err := n.UpdateGames(ctx, util.Today().Format(DateFormat)); err != nil {
		return nil, err
	}

	c := cron.New()
	if _, err := c.AddFunc("0 5 * * *", func() { n.CacheClear(context.Background()) }); err != nil {
		return nil, fmt.Errorf("failed to set cron job for cacheClear: %w", err)
	}
	c.Start()

	return n, nil
}

// CacheClear ...
func (n *NcaaM) CacheClear(ctx context.Context) {
	n.log.Warn("clearing NCAAM cache")
	for k := range n.games {
		delete(n.games, k)
	}
	for k := range n.logos {
		delete(n.logos, k)
	}
	n.teams = nil
	if _, err := n.GetTeams(ctx); err != nil {
		n.log.Error("failed to get teams after cache clear", zap.Error(err))
	}
	if err := n.UpdateGames(ctx, util.Today().Format(DateFormat)); err != nil {
		n.log.Error("failed to get today's games", zap.Error(err))
	}
}

// GetTeams ...
func (n *NcaaM) GetTeams(ctx context.Context) ([]sportboard.Team, error) {
	if n.teams == nil {
		var err error
		n.teams, err = GetTeams(ctx)
		if err != nil {
			return nil, err
		}
	}

	var tList []sportboard.Team

	for _, t := range n.teams {
		n.allTeams = append(n.allTeams, t.Abbreviation)
		tList = append(tList, t)
	}

	return tList, nil
}

// TeamFromAbbreviation ...
func (n *NcaaM) TeamFromAbbreviation(ctx context.Context, abbreviation string) (sportboard.Team, error) {
	for _, t := range n.teams {
		if t.Abbreviation == abbreviation {
			return t, nil
		}
	}

	return nil, fmt.Errorf("could not find team '%s'", abbreviation)
}

// GetScheduledGames ...
func (n *NcaaM) GetScheduledGames(ctx context.Context, date time.Time) ([]sportboard.Game, error) {
	t := TimeToGameDateStr(date)
	games, ok := n.games[t]
	if !ok || len(games) == 0 {
		if err := n.UpdateGames(ctx, t); err != nil {
			return nil, err
		}
	}

	games, ok = n.games[t]
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
func (n *NcaaM) DateStr(d time.Time) string {
	return d.Format(DateFormat)
}

// League ...
func (n *NcaaM) League() string {
	return "NCAA Men's Basketball"
}

// HTTPPathPrefix ...
func (n *NcaaM) HTTPPathPrefix() string {
	return "ncaam"
}

// AllTeamAbbreviations ...
func (n *NcaaM) AllTeamAbbreviations() []string {
	return n.allTeams
}

// GetWatchTeams ...
func (n *NcaaM) GetWatchTeams(teams []string) []string {
	watch := []string{}
OUTER:
	for _, t := range teams {
		if t == "ALL" {
			n.log.Info("setting NCAAM watch teams to ALL teams")
			return n.AllTeamAbbreviations()
		}
		for _, conf := range Conferences {
			if strings.ToLower(t) == conf {
				n.log.Info("adding teams to watchlist from conference", zap.String("conference", conf))
				watch = append(watch, n.TeamsInConference(conf)...)
				continue OUTER
			}
		}
		watch = append(watch, t)
	}

	return watch
}

// TeamsInConference ...
func (n *NcaaM) TeamsInConference(conference string) []string {
	ret := []string{}
	for _, team := range n.teams {
		if team.Conference.Abbreviation == conference {
			ret = append(ret, team.Abbreviation)
		}
	}

	return ret
}

// UpdateGames ...
func (n *NcaaM) UpdateGames(ctx context.Context, dateStr string) error {
	games, err := GetGames(ctx, dateStr)
	if err != nil {
		return err
	}

	n.Lock()
	defer n.Unlock()

	n.games[dateStr] = games

	return nil
}

// TeamRank ...
func (n *NcaaM) TeamRank(ctx context.Context, team sportboard.Team) string {
	var realTeam *Team
	for _, t := range n.teams {
		if t.Abbreviation == team.GetAbbreviation() {
			realTeam = t
			break
		}
	}

	if realTeam == nil {
		return ""
	}

	if err := realTeam.setDetails(ctx, n.log); err != nil {
		n.log.Error("failed to set team data", zap.Error(err), zap.String("team", team.GetAbbreviation()))
		return ""
	}

	if realTeam.rank < 1 {
		return ""
	}

	return strconv.Itoa(realTeam.rank)
}

// TeamRecord ...
func (n *NcaaM) TeamRecord(ctx context.Context, team sportboard.Team) string {
	var realTeam *Team
	for _, t := range n.teams {
		if t.Abbreviation == team.GetAbbreviation() {
			realTeam = t
			break
		}
	}

	if realTeam == nil {
		return ""
	}

	if err := realTeam.setDetails(ctx, n.log); err != nil {
		n.log.Error("failed to set team data", zap.Error(err), zap.String("team", team.GetAbbreviation()))
		return ""
	}

	return realTeam.record
}
