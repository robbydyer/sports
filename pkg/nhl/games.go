package nhl

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type Game struct {
	ID    int    `json:"gamePk"`
	Link  string `json:"link"`
	Teams struct {
		Away *GameTeam `json:"away"`
		Home *GameTeam `json:"home"`
	} `json:"teams"`
	GameTime time.Time
	GameDate string `json:"gameDate"`
}

type GameTeam struct {
	Score int `json:"score"`
	Team  *struct {
		Id int `json:"id"`
	} `json:"team"`
}

type games struct {
	Dates []*struct {
		Games []*Game `json:"games"`
	} `json:"dates"`
}

func (g *Game) setGameTime() error {
	var err error
	g.GameTime, err = time.Parse("2006-01-02T15:04:05Z", g.GameDate)
	if err != nil {
		return err
	}

	return nil
}

func getGames(ctx context.Context, dateStr string) ([]*Game, error) {
	if err := validateDateStr(dateStr); err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("%s/schedule?date=%s&expand=schedule.linescore", BaseURL, dateStr)
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	client := http.DefaultClient

	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get games for '%s': %w", dateStr, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var gameList *games

	if err := json.Unmarshal(body, &gameList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal games list: %w", err)
	}

	var retGames []*Game

	for _, d := range gameList.Dates {
		for _, g := range d.Games {
			if err := g.setGameTime(); err != nil {
				return nil, fmt.Errorf("failed to set GameTime: %w", err)
			}
			retGames = append(retGames, g)
		}
	}

	return retGames, nil
}
