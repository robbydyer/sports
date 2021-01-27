package nhl

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

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
	f, err := pkger.Open("github.com/robbydyer/sports:/pkg/nhl/assets/mock_livegames.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dat, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var gameList []*LiveGame

	if err := json.Unmarshal(dat, &gameList); err != nil {
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
	f, err := pkger.Open("github.com/robbydyer/sports:/pkg/nhl/assets/mock_games.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dat, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var gameList []*Game

	if err := json.Unmarshal(dat, &gameList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mock yaml: %w", err)
	}

	m := &MockNHLAPI{
		GameList: map[string][]*Game{
			today: gameList,
		},
		Teams: map[int]*Team{
			2: {
				ID:           2,
				Name:         "New York Islanders",
				Abbreviation: "NYI",
			},
			1: {
				ID:           1,
				Name:         "New Jersey Devils",
				Abbreviation: "NJD",
			},
			3: {
				ID:           3,
				Name:         "New York Rangers",
				Abbreviation: "NYR",
			},
			4: {
				ID:           4,
				Name:         "Philadelphia Flyers",
				Abbreviation: "PHI",
			},
			29: {
				ID:           29,
				Name:         "Columbus Blue Jackets",
				Abbreviation: "CBJ",
			},
			30: {
				ID:           30,
				Name:         "Minnesota Wild",
				Abbreviation: "MIN",
			},
		},
	}

	return m, nil
}
