package mlb

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
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
	People []struct {
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
	} `json:"people"`
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

		m.log.Debug("fetching MLB player stats for team", zap.String("team", team.Abbreviation))

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
	intID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, err
	}
	for _, team := range m.teams {
		for _, player := range team.Roster {
			if player.Person.ID == intID {
				if player.Stats == nil {
					if err := player.setStats(ctx); err != nil {
						return nil, err
					}
				}
				return player, nil
			}
		}
	}

	return nil, fmt.Errorf("could not find player")
}

// UpdateStats ...
func (p *Player) UpdateStats(ctx context.Context) error {
	if err := p.setStats(ctx); err != nil {
		return err
	}

	return nil
}

// GetCategory returns the player's catgeory: pitcher or hitter
func (p *Player) GetCategory() string {
	switch strings.ToLower(p.PlayerPosition.Name) {
	case "pitcher":
		return pitcher
	}

	return hitter
}

func (p *Player) setStats(ctx context.Context) error {
	uri, err := url.Parse(fmt.Sprintf("%s/v1/people/%d", baseURL, p.Person.ID))
	if err != nil {
		return fmt.Errorf("failed to parse URI: %w", err)
	}

	v := uri.Query()
	// TODO: Change this to "season" for type
	v.Set("hydrate", "stats(group=[hitting,pitching,fielding],type=career)")
	v.Set("currentTeam", "")

	uri.RawQuery = v.Encode()

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	client := http.DefaultClient

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var pStat *playerStatData

	if err := json.Unmarshal(body, &pStat); err != nil {
		return fmt.Errorf("failed to unmarshal request: %w", err)
	}

	for _, person := range pStat.People {
		for _, all := range person.Stats {
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
	}

	return fmt.Errorf("could not find stats for player: %d %s", p.Person.ID, p.Person.FullName)
}

// AvailableStats ...
func (m *MLB) AvailableStats(ctx context.Context, category string) ([]string, error) {
	if category == pitcher {
		return []string{
			"wins",
			"losses",
			"saves",
			"era",
		}, nil
	}
	return []string{
		"avg",
		"homeRuns",
		"rbi",
		"ops",
	}, nil
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
	case "avg":
		return p.Stats.Average
	case "homeruns":
		return fmt.Sprint(p.Stats.HomeRuns)
	case "rbi":
		return fmt.Sprint(p.Stats.RBI)
	case "ops":
		return p.Stats.OPS
	case "era":
		return p.Stats.ERA
	case "wins":
		return fmt.Sprint(p.Stats.Wins)
	case "losses":
		return fmt.Sprint(p.Stats.Losses)
	case "saves":
		return fmt.Sprint(p.Stats.Saves)
	}

	return "?"
}

// StatColor ...
func (p *Player) StatColor(stat string) color.Color {
	return color.White
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
