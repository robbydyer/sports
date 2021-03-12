package mlb

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/statboard"
)

const (
	hitter  = "hitter"
	pitcher = "pitcher"
)

var statShortNames = map[string]string{}

// Player ...
type Player struct {
	Person struct {
		ID       int64  `json:"id"`
		FullName string `json:"fullName"`
		Link     string `json:"link"`
	} `json:"person"`
	PlayerPosition *struct {
		Abbreviation string `json:"abbreviation"`
		Name         string `json:"name"`
	} `json:"position"`
	Stats *playerStats
}

type playerStatData struct {
	Stats []struct {
		Type struct {
			DisplayName string `json:"displayName"`
		} `json:"type"`
		Group struct {
			DisplayName string `json:"displayName"`
		} `json:"group"`
		Splits []struct {
			Stat *playerStats `json:"stat"`
		}
	}
}

type playerStats struct {
	Average  string `json:"avg"`
	HomeRuns int    `json:"homeRuns"`
	RBI      int    `json:"rbi"`
	OPS      string `json:"ops"`
	ERA      string `json:"era"`
	Wins     int    `json:"wins"`
	Losses   int    `json:"losses"`
	Saves    int    `json:"saves"`
}

// PlayerCategories returns the possible categories a player falls into
func (m *MLB) PlayerCategories() []string {
	return []string{
		hitter,
		pitcher,
	}
}

// FindPlayer ...
func (m *MLB) FindPlayer(ctx context.Context, first string, last string) (statboard.Player, error) {
	full := fmt.Sprintf("%s %s", strings.ToLower(first), strings.ToLower(last))

	for _, team := range m.teams {
		for _, p := range team.Roster {
			if full == strings.ToLower(p.Person.FullName) {
				if p.Stats == nil {
					if err := p.setStats(ctx); err != nil {
						return nil, err
					}
				}
				return p, nil
			}
		}
	}

	return nil, fmt.Errorf("could not find player '%s %s'", first, last)
}

// ListPlayers ...
func (m *MLB) ListPlayers(ctx context.Context, teamAbbreviation string) ([]statboard.Player, error) {
	var players []statboard.Player
	for _, team := range m.teams {
		if team.Abbreviation != teamAbbreviation {
			continue
		}

	INNER:
		for _, p := range team.Roster {
			if p.Stats == nil {
				if err := p.setStats(ctx); err != nil {
					m.log.Error("could not find stats for player", zap.Error(err))
					continue INNER
				}
			}
			players = append(players, p)
		}

		return players, nil
	}

	return nil, fmt.Errorf("could not find team '%s'", teamAbbreviation)
}

// GetPlayer ...
func (m *MLB) GetPlayer(ctx context.Context, id string) (statboard.Player, error) {
	return nil, nil
}

// UpdatePlayer ...
func (m *MLB) UpdatePlayer(ctx context.Context, id string) (statboard.Player, error) {
	return nil, nil
}

// GetCategory returns the player's catgeory: skater or goalie
func (p *Player) GetCategory() string {
	switch strings.ToLower(p.PlayerPosition.Name) {
	case "p":
		return pitcher
	}

	return hitter
}

func (p *Player) setStats(ctx context.Context) error {
	//"https://statsapi.web.nhl.com/api/v1/people/8478445/stats?stats=statsSingleSeason&season=20202021"
	uri, err := url.Parse(fmt.Sprintf("%s/v1/people/%d", baseURL, p.Person.ID))
	if err != nil {
		return err
	}

	v := uri.Query()
	// TODO: Change this to "season" for type
	v.Set("hydrate", "stats=(group=[hitting,pitching,fielding],type=career),currentTeam")

	uri.RawQuery = v.Encode()

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	client := http.DefaultClient

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var pStat *playerStatData

	if err := json.Unmarshal(body, &pStat); err != nil {
		return err
	}

	for _, all := range pStat.Stats {
		cat := p.GetCategory()
		if cat == "hitter" && all.Group.DisplayName != "hitting" {
			continue
		}
		if cat == "pitcher" && all.Group.DisplayName != "pitching" {
			continue
		}
		for _, s := range all.Splits {
			p.Stats = s.Stat
			return nil
		}
	}

	return fmt.Errorf("could not find stats for player: %d %s", p.Person.ID, p.Person.FullName)
}

// AvailableStats ...
func (m *MLB) AvailableStats(ctx context.Context, category string) ([]string, error) {
	if category == pitcher {
		return []string{}, nil
	}
	return []string{}, nil
}

// Position ...
func (p *Player) Position() string {
	if p.PlayerPosition == nil {
		return ""
	}
	return p.PlayerPosition.Abbreviation
}

// GetStat ...
func (p *Player) GetStat(stat string) string {
	if p.Stats == nil {
		return ""
	}
	switch strings.ToLower(stat) {
	}

	return "?"
}

// StatShortName returns a short name representation of the stat, if any
func (m *MLB) StatShortName(stat string) string {
	s, ok := statShortNames[stat]
	if ok {
		return s
	}

	return stat
}

// FirstName ...
func (p *Player) FirstName() string {
	parts := strings.Fields(p.Person.FullName)
	if len(parts) > 0 {
		return parts[0]
	}

	return p.Person.FullName
}

// LastName ...
func (p *Player) LastName() string {
	parts := strings.Fields(p.Person.FullName)
	if len(parts) > 0 {
		return strings.Join(parts[1:], " ")
	}
	return p.Person.FullName
}
