package nhl

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/statboard"
	"github.com/robbydyer/sports/pkg/util"
)

const (
	skater = "skater"
	goalie = "goalie"
)

var statShortNames = map[string]string{
	"goals":              "G",
	"assists":            "A",
	"shots":              "S",
	"games":              "GP",
	"pim":                "PIM",
	"plusMinus":          "+/-",
	"hits":               "HIT",
	"record":             "W/L",
	"savePercentage":     "SV%",
	"goalAgainstAverage": "GAA",
	"shutouts":           "SO",
}

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
		Splits []struct {
			Season string       `json:"season"`
			Stat   *playerStats `json:"stat"`
		}
	}
}

type playerStats struct {
	Assists            int     `json:"assists"`
	Goals              int     `json:"goals"`
	Shots              int     `json:"shots"`
	Games              int     `json:"games"`
	Hits               int     `json:"hits"`
	PlusMinus          int     `json:"plusMinus"`
	Pim                int     `json:"pim"`
	Wins               int     `json:"wins"`
	Losses             int     `json:"losses"`
	SavePercentage     float64 `json:"savePercentage"`
	GoalAgainstAverage float64 `json:"goalAgainstAverage"`
	Shutouts           int     `json:"shutouts"`
}

// PlayerCategories returns the possible categories a player falls into
func (n *NHL) PlayerCategories() []string {
	return []string{
		skater,
		goalie,
	}
}

// FindPlayer ...
func (n *NHL) FindPlayer(ctx context.Context, first string, last string) (statboard.Player, error) {
	full := fmt.Sprintf("%s %s", strings.ToLower(first), strings.ToLower(last))

	for _, team := range n.teams {
		for _, p := range team.Roster.Roster {
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
func (n *NHL) ListPlayers(ctx context.Context, teamAbbreviation string) ([]statboard.Player, error) {
	var players []statboard.Player
	for _, team := range n.teams {
		if team.Abbreviation != teamAbbreviation {
			continue
		}

	INNER:
		for _, p := range team.Roster.Roster {
			if p.Stats == nil {
				if err := p.setStats(ctx); err != nil {
					n.log.Error("could not find stats for player", zap.Error(err))
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
func (n *NHL) GetPlayer(ctx context.Context, id string) (statboard.Player, error) {
	intID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, err
	}
	for _, team := range n.teams {
		for _, player := range team.Roster.Roster {
			if player.Person.ID == intID {
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

// GetCategory returns the player's catgeory: skater or goalie
func (p *Player) GetCategory() string {
	if strings.ToLower(p.PlayerPosition.Name) == "goalie" {
		return goalie
	}

	return skater
}

func (p *Player) setStats(ctx context.Context) error {
	//"https://statsapi.web.nhl.com/api/v1/people/8478445/stats?stats=statsSingleSeason&season=20202021"
	uri, err := url.Parse(fmt.Sprintf("https://statsapi.web.nhl.com/api/v1/people/%d/stats", p.Person.ID))
	if err != nil {
		return err
	}

	v := uri.Query()
	v.Set("stats", "statsSingleSeason")
	v.Set("season", GetSeason(util.Today()))

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
		for _, s := range all.Splits {
			p.Stats = s.Stat
			return nil
		}
	}

	return fmt.Errorf("could not find stats for player: %d %s", p.Person.ID, p.Person.FullName)
}

// AvailableStats ...
func (n *NHL) AvailableStats(ctx context.Context, category string) ([]string, error) {
	if category == goalie {
		return []string{
			"record",
			"savePercentage",
			"goalAgainstAverage",
			"shutouts",
		}, nil
	}
	return []string{
		"goals",
		"assists",
		"shots",
		"pim",
		"hits",
		"games",
		"plusMinus",
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
	case "assists":
		return fmt.Sprint(p.Stats.Assists)
	case "goals":
		return fmt.Sprint(p.Stats.Goals)
	case "shots":
		return fmt.Sprint(p.Stats.Shots)
	case "games":
		return fmt.Sprint(p.Stats.Games)
	case "hits":
		return fmt.Sprint(p.Stats.Hits)
	case "plusminus":
		return fmt.Sprint(p.Stats.PlusMinus)
	case "pim":
		return fmt.Sprint(p.Stats.Pim)
	case "record":
		return fmt.Sprintf("%d-%d", p.Stats.Wins, p.Stats.Losses)
	case "savepercentage":
		return fmt.Sprint(p.Stats.SavePercentage)
	case "goalagainstaverage":
		return fmt.Sprint(p.Stats.GoalAgainstAverage)
	case "shutouts":
		return fmt.Sprint(p.Stats.Shutouts)
	}

	return "?"
}

// StatShortName returns a short name representation of the stat, if any
func (n *NHL) StatShortName(stat string) string {
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
