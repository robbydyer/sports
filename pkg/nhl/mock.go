package nhl

import (
	"context"
	"fmt"
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

/*
func NewMockAPI() *MockNHLAPI {
	today := Today()
	return &MockNHLAPI{
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
		},
		GameList: map[string][]*Game{
			today: {
				{
					ID:   1,
					Link: "1",
					Teams: {
						Away: &GameTeam{
							Team: {
								ID:           2,
								Abbreviation: "NYI",
								Name:         "New York Islanders",
							},
							Goals: 9,
						},
						Home: &GameTeam{
							Team: {
								ID:           2,
								Abbreviation: "NJD",
								Name:         "New Jersey Devils",
							},
							Goals: 0,
						},
					},
				},
			},
		},
	}
}

*/
