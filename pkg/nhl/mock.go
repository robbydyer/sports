package nhl

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	yaml "github.com/ghodss/yaml"
	"github.com/markbates/pkger"
	//"github.com/gobuffalo/packr/v2"
)

type MockNHLAPI struct {
	GameList map[string][]*Game
	Teams    map[int]*Team
}

func (m *MockNHLAPI) UpdateTeams(ctx context.Context) error {
	return nil
}
func (m *MockNHLAPI) UpdateGames(ctx context.Context, dateStr string) error {
	return nil
}
func (m *MockNHLAPI) TeamFromAbbreviation(abbrev string) (*Team, error) {
	for _, t := range m.Teams {
		if t.Abbreviation == abbrev {
			return t, nil
		}
	}

	return nil, fmt.Errorf("could not find team with abbreviation '%s'", abbrev)
}
func (m *MockNHLAPI) Games(dateStr string) ([]*Game, error) {
	games, ok := m.GameList[dateStr]
	if !ok {
		return nil, fmt.Errorf("games not found for %s", dateStr)
	}

	return games, nil
}

func MockLiveGameGetter(ctx context.Context, link string) (*LiveGame, error) {
	f, err := pkger.Open("github.com/robbydyer/sports:/pkg/nhl/assets/mock_livegames.yaml")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dat, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var gameList []*LiveGame

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

func NewMockAPI() (*MockNHLAPI, error) {
	today := Today()
	// Load Teams
	f, err := pkger.Open("github.com/robbydyer/sports:/pkg/nhl/assets/mock_teams.yaml")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dat, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var teamList []*Team

	if err := yaml.Unmarshal(dat, &teamList); err != nil {
		return nil, err
	}

	// Load Games
	gamef, err := pkger.Open("github.com/robbydyer/sports:/pkg/nhl/assets/mock_games.yaml")
	if err != nil {
		return nil, err
	}
	defer gamef.Close()

	dat, err = ioutil.ReadAll(gamef)
	if err != nil {
		return nil, err
	}

	var gameList []*Game

	if err := yaml.Unmarshal(dat, &gameList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mock yaml: %w", err)
	}

	m := &MockNHLAPI{
		GameList: map[string][]*Game{
			today: gameList,
		},
		Teams: make(map[int]*Team),
	}

	for _, t := range teamList {
		m.Teams[t.ID] = t
	}

	return m, nil
}
