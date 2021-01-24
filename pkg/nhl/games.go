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

type LiveGame struct {
	ID       int    `json:"gamePk"`
	Link     string `json:"link"`
	GameTime time.Time
	GameData *struct {
		DateTime *struct {
			DateTime string `json:"datetime,omitempty"`
		} `json:"dateTime,omitempty"`
		Status *struct {
			AbstractGameState string `json:"abstractGameState,omitempty"`
			DetailedState     string `json:"detailedState,omitempty"`
		} `json:"status,omitempty"`
	} `json:"gameData,omitempty"`
	LiveData *struct {
		Linescore *struct {
			Teams *struct {
				Home *GameTeam `json:"home"`
				Away *GameTeam `json:"away"`
			} `json:"teams"`
			CurrentPeriod              int    `json:"currentPeriod,omitempty"`
			CurrentPeriodTimeRemaining string `json:"currentPeriodTimeRemaining,omitempty"`
		} `json:"linescore,omitempty"`
	} `json:"liveData"`
}

type GameTeam struct {
	Score int `json:"score,omitempty"`
	Team  *struct {
		ID           int    `json:"id"`
		Abbreviation string `json:"abbreviation,omitempty"`
		Name         string `json:"name,omitempty"`
	} `json:"team,omitempty"`
	Goals       int  `json:"goals,omitempty"`
	ShotsOnGoal int  `json:"shotsOnGoal,omitempty"`
	PowerPlay   bool `json:"powerPlay,omitempty"`
}

type games struct {
	Dates []*struct {
		Games []*Game `json:"games"`
	} `json:"dates"`
}

func timeFromGameTime(gameTime string) (time.Time, error) {
	t, err := time.Parse("2006-01-02T15:04:05Z", gameTime)
	if err != nil {
		return time.Time{}, err
	}

	t = t.Local()

	return t, nil
}

func GetLiveGame(ctx context.Context, link string) (*LiveGame, error) {
	uri := fmt.Sprintf("%s/%s", LinkBase, link)

	req, err := http.NewRequest("GET", uri, nil)
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
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var game *LiveGame

	if err := json.Unmarshal(body, &game); err != nil {
		return nil, fmt.Errorf("failed to unmarshal LiveGame: %w", err)
	}

	t, err := timeFromGameTime(game.GameData.DateTime.DateTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse game time: %w", err)
	}

	game.GameTime = t

	return game, nil
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
			t, err := timeFromGameTime(g.GameDate)
			if err != nil {
				return nil, fmt.Errorf("failed to set GameTime: %w", err)
			}
			g.GameTime = t
			retGames = append(retGames, g)
		}
	}

	return retGames, nil
}
