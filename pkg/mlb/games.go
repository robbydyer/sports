package mlb

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/robbydyer/sports/pkg/sportboard"
)

// LiveGameGetter is a func used to retrieve an updated sportboard.Game
type LiveGameGetter func(ctx context.Context, link string) (sportboard.Game, error)

type schedule struct {
	Dates []*struct {
		Games []*Game `json:"games"`
	} `json:"dates"`
}

// Game ...
type Game struct {
	GameGetter LiveGameGetter
	ID         int    `json:"gamePk"`
	Link       string `json:"link"`
	Teams      *struct {
		Home *gameTeam `json:"home"`
		Away *gameTeam `json:"away"`
	} `json:"teams"`
	GameTime time.Time
	GameDate string `json:"gameDate"`
	GameData *struct {
		DateTime *struct {
			DateTime string `json:"dateTime,omitempty"`
		} `json:"datetime,omitempty"`
		Status *struct {
			AbstractGameState string `json:"abstractGameState,omitempty"`
			DetailedState     string `json:"detailedState,omitempty"`
			StatusCode        string `json:"statusCode,omitempty"`
		} `json:"status,omitempty"`
	} `json:"gameData,omitempty"`
	LiveData *struct {
		Linescore *struct {
			CurrentInning        int    `json:"currentInning"`
			CurrentInningOrdinal string `json:"currentInningOrdinal"`
			InningState          string `json:"inningState"`
			Teams                *struct {
				Home *gameTeam `json:"home"`
				Away *gameTeam `json:"away"`
			} `json:"teams"`
		} `json:"linescore"`
	} `json:"liveData"`
}

type gameTeam struct {
	Score    int   `json:"score,omitempty"`
	Runs     int   `json:"runs,omitempty"`
	Team     *Team `json:"team"`
	IsWinner bool  `json:"isWinner"`
}

func (g *Game) GetID() int {
	return g.ID
}

func (g *Game) GetLink() (string, error) {
	return "", nil
}

func (g *Game) IsLive() (bool, error) {
	return true, nil
}

func (g *Game) IsComplete() (bool, error) {
	return false, nil
}

func (g *Game) HomeTeam() (sportboard.Team, error) {
	if g.Teams != nil && g.Teams.Home != nil && g.Teams.Home.Team != nil {
		return g.Teams.Home.Team, nil
	}

	if g.LiveData != nil &&
		g.LiveData.Linescore != nil &&
		g.LiveData.Linescore.Teams != nil &&
		g.LiveData.Linescore.Teams.Home != nil {
		g.LiveData.Linescore.Teams.Home.Team.Runs = g.LiveData.Linescore.Teams.Home.Runs

		return g.LiveData.Linescore.Teams.Home.Team, nil
	}

	return nil, fmt.Errorf("could not locate home team in Game")
}

func (g *Game) AwayTeam() (sportboard.Team, error) {
	if g.Teams != nil && g.Teams.Away != nil && g.Teams.Away.Team != nil {
		return g.Teams.Away.Team, nil
	}

	if g.LiveData != nil &&
		g.LiveData.Linescore != nil &&
		g.LiveData.Linescore.Teams != nil &&
		g.LiveData.Linescore.Teams.Away != nil {
		g.LiveData.Linescore.Teams.Away.Team.Runs = g.LiveData.Linescore.Teams.Away.Runs

		return g.LiveData.Linescore.Teams.Away.Team, nil
	}

	return nil, fmt.Errorf("could not locate home team in Game")
}

// GetQuarter returns the inning
func (g *Game) GetQuarter() (string, error) {
	if g.LiveData != nil && g.LiveData.Linescore != nil {
		return g.LiveData.Linescore.CurrentInningOrdinal, nil
	}

	return "", fmt.Errorf("could not determine inning")
}

// GetClock represent bottom or top of inning
func (g *Game) GetClock() (string, error) {
	if g.LiveData != nil && g.LiveData.Linescore != nil {
		if strings.Contains(strings.ToLower(g.LiveData.Linescore.InningState), "bot") {
			return "BOT", nil
		}
		return "TOP", nil
	}
	return "", nil
}

// GetUpdate ...
func (g *Game) GetUpdate(ctx context.Context) (sportboard.Game, error) {
	if g.GameGetter == nil {
		g.GameGetter = GetLiveGame
	}
	return g.GameGetter(ctx, g.Link)
}

// GetStartTime ...
func (g *Game) GetStartTime(ctx context.Context) (time.Time, error) {
	return g.GameTime, nil
}

// GetLiveGame ...
func GetLiveGame(ctx context.Context, link string) (sportboard.Game, error) {
	uri := fmt.Sprintf("%s/%s", linkBase, link)

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

	var game *Game

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
	uri := fmt.Sprintf("%s/v1/schedule?date=%s", baseURL, dateStr)

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

	var gameList *schedule

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

func timeFromGameTime(gameTime string) (time.Time, error) {
	t, err := time.Parse("2006-01-02T15:04:05Z", gameTime)
	if err != nil {
		return time.Time{}, err
	}

	t = t.Local()

	return t, nil
}
