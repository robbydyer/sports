package pga

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/robbydyer/sports/pkg/statboard"
)

const leaderboardURL = "https://site.web.api.espn.com/apis/site/v2/sports/golf/leaderboard?league=pga"

// PGA ...
type PGA struct{}

type eventDat struct {
	Events []struct {
		ShortName    string `json:"shortName"`
		Competitions []struct {
			Competitors []*Player `json:"competitors"`
		} `json:"competitions"`
	}
}

// New ...
func New() (*PGA, error) {
	return &PGA{}, nil
}

// FindPlayer ...
func (p *PGA) FindPlayer(ctx context.Context, firstName string, lastName string) (statboard.Player, error) {
	return nil, nil
}

// GetPlayer ...
func (p *PGA) GetPlayer(ctx context.Context, id string) (statboard.Player, error) {
	return nil, nil
}

// AvailableStats ...
func (p *PGA) AvailableStats(ctx context.Context, playerCategory string) ([]string, error) {
	return []string{
		"score",
		"hole",
		"position",
	}, nil
}

// StatShortName ...
func (p *PGA) StatShortName(stat string) string {
	return ""
}

// ListPlayers ...
func (p *PGA) ListPlayers(ctx context.Context, teamAbbreviation string) ([]statboard.Player, error) {
	plyrs, err := p.updatePlayers(ctx)
	if err != nil {
		return nil, err
	}

	players := []statboard.Player{}
	for _, player := range plyrs {
		players = append(players, player)
	}
	return players, nil
}

// LeagueShortName ...
func (p *PGA) LeagueShortName() string {
	return "PGA"
}

// HTTPPathPrefix ...
func (p *PGA) HTTPPathPrefix() string {
	return "pga"
}

// PlayerCategories ...
func (p *PGA) PlayerCategories() []string {
	return []string{"player"}
}

func (p *PGA) updatePlayers(ctx context.Context) ([]*Player, error) {
	req, err := http.NewRequest("GET", leaderboardURL, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	client := http.DefaultClient

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var dat *eventDat

	if err := json.Unmarshal(body, &dat); err != nil {
		return nil, err
	}

	for _, event := range dat.Events {
		for _, comp := range event.Competitions {
			return comp.Competitors, nil
		}
	}

	return nil, fmt.Errorf("could not find players")
}
