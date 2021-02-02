package nhl

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"time"

	yaml "github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"

	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/sportboard"
	"github.com/robbydyer/sports/pkg/util"
)

// MockNHLAPI implements sportboard.API. Used for testing
type MockNHLAPI struct {
	teams           []*Team
	games           map[string][]*Game
	logos           map[string]*logo.Logo
	logoSourceCache map[string]image.Image
	log             *log.Logger
}

// HTTPPathPrefix ...
func (m *MockNHLAPI) HTTPPathPrefix() string {
	return "nhl"
}

// GetTeams ...
func (m *MockNHLAPI) GetTeams(ctx context.Context) ([]sportboard.Team, error) {
	var tList []sportboard.Team

	for _, t := range m.teams {
		tList = append(tList, t)
	}

	return tList, nil
}

// GetScheduledGames ...
func (m *MockNHLAPI) GetScheduledGames(ctx context.Context, date time.Time) ([]sportboard.Game, error) {
	dateStr := m.DateStr(date)
	var gList []sportboard.Game

	for _, g := range m.games[dateStr] {
		gList = append(gList, g)
	}

	return gList, nil
}

// DateStr ...
func (m *MockNHLAPI) DateStr(d time.Time) string {
	return d.Format(DateFormat)
}

// League ...
func (m *MockNHLAPI) League() string {
	return "Fake NHL"
}

// GetLogo ...
func (m *MockNHLAPI) GetLogo(logoKey string, logoConf *logo.Config, bounds image.Rectangle) (*logo.Logo, error) {
	fullLogoKey := fmt.Sprintf("%s_%dx%d", logoKey, bounds.Dx(), bounds.Dy())
	l, ok := m.logos[fullLogoKey]
	if ok {
		return l, nil
	}

	sources, err := m.logoSources()
	if err != nil {
		return nil, err
	}

	l, err = GetLogo(logoKey, logoConf, bounds, sources)
	if err != nil {
		return nil, err
	}

	m.logos[fullLogoKey] = l

	return l, nil
}

func (m *MockNHLAPI) logoSources() (map[string]image.Image, error) {
	if len(m.logoSourceCache) == len(ALL) {
		return m.logoSourceCache, nil
	}

	for _, t := range ALL {
		f, err := assets.Open(fmt.Sprintf("assets/logos/%s.png", t))
		if err != nil {
			return nil, err
		}
		defer f.Close()

		i, err := png.Decode(f)
		if err != nil {
			return nil, err
		}

		m.logoSourceCache[t] = i
	}

	return m.logoSourceCache, nil
}

// AllTeamAbbreviations ...
func (m *MockNHLAPI) AllTeamAbbreviations() []string {
	return ALL
}

// UpdateTeams ...
func (m *MockNHLAPI) UpdateTeams(ctx context.Context) error {
	return nil
}

// UpdateGames ...
func (m *MockNHLAPI) UpdateGames(ctx context.Context, dateStr string) error {
	return nil
}

// TeamFromAbbreviation ...
func (m *MockNHLAPI) TeamFromAbbreviation(ctx context.Context, abbrev string) (sportboard.Team, error) {
	for _, t := range m.teams {
		if t.Abbreviation == abbrev {
			return t, nil
		}
	}

	return nil, fmt.Errorf("could not find team with abbreviation '%s'", abbrev)
}

// MockLiveGameGetter implements nhl.LiveGameGetter
func MockLiveGameGetter(ctx context.Context, link string) (sportboard.Game, error) {
	dat, err := assets.ReadFile("assets/mock_livegames.yaml")
	if err != nil {
		return nil, err
	}

	var gameList []*Game

	if err := yaml.Unmarshal(dat, &gameList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal live game mock yaml: %w", err)
	}

	for _, liveGame := range gameList {
		if liveGame.Link == link {
			liveGame.GameTime = time.Now().Local()
			return liveGame, nil
		}
	}

	return nil, fmt.Errorf("could not locate live game with Link '%s'", link)
}

// New ...
func NewMock(logger *log.Logger) (*MockNHLAPI, error) {
	// Load Teams
	dat, err := assets.ReadFile("assets/mock_teams.yaml")
	if err != nil {
		return nil, err
	}

	var teamList []*Team

	if err := yaml.Unmarshal(dat, &teamList); err != nil {
		return nil, err
	}

	// Load Games
	dat, err = assets.ReadFile("assets/mock_games.yaml")
	if err != nil {
		return nil, err
	}

	var gameList []*Game

	if err := yaml.Unmarshal(dat, &gameList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mock yaml: %w", err)
	}

	for _, g := range gameList {
		g.GameGetter = MockLiveGameGetter
	}

	today := util.Today().Format(DateFormat)
	m := &MockNHLAPI{
		games: map[string][]*Game{
			today: gameList,
		},
		teams:           teamList,
		logos:           make(map[string]*logo.Logo),
		logoSourceCache: make(map[string]image.Image),
		log:             logger,
	}

	return m, nil
}
