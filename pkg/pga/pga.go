package pga

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/robbydyer/sports/pkg/statboard"
)

const leaderboardURL = "https://site.web.api.espn.com/apis/site/v2/sports/golf/leaderboard?league=pga"

// PGA ...
type PGA struct {
	players []*Player
}

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
	if err := p.updatePlayers(ctx); err != nil {
		return nil, err
	}

	players := []statboard.Player{}
	for _, player := range p.players {
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

func (p *PGA) updatePlayers(ctx context.Context) error {
	req, err := http.NewRequest("GET", leaderboardURL, nil)
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

	var dat *eventDat

	if err := json.Unmarshal(body, &dat); err != nil {
		return err
	}

	for _, event := range dat.Events {
		for _, comp := range event.Competitions {
			p.players = comp.Competitors
		}
	}

	return nil
}
