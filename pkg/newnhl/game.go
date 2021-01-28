package newnhl

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/robbydyer/sports/pkg/sportboard"
)

type Game struct {
	ID    int    `json:"gamePk"`
	Link  string `json:"link"`
	Teams *struct {
		Away *GameTeam `json:"away"`
		Home *GameTeam `json:"home"`
	} `json:"teams"`
	GameTime time.Time
	GameDate string `json:"gameDate"`
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
		} `json:"linescore"`
	} `json:"liveData"`
}

type GameTeam struct {
	Score       int   `json:"score,omitempty"`
	Team        *Team `json:"team"`
	Goals       int   `json:"goals,omitempty"`
	ShotsOnGoal int   `json:"shotsOnGoal,omitempty"`
	PowerPlay   bool  `json:"powerPlay,omitempty"`
}

type games struct {
	Dates []*struct {
		Games []*Game `json:"games"`
	} `json:"dates"`
}

func (n *NHL) Games(dateStr string) ([]*Game, error) {
	games, ok := n.games[dateStr]
	if !ok || len(games) == 0 {
		return nil, fmt.Errorf("no games for date %s", dateStr)
	}

	return games, nil
}

func (g *Game) GetID() int {
	return g.ID
}
func (g *Game) GetLink() (string, error) {
	return g.Link, nil
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
		g.LiveData.Linescore.Teams.Home.Team.score = g.LiveData.Linescore.Teams.Home.Goals

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

		g.LiveData.Linescore.Teams.Away.Team.score = g.LiveData.Linescore.Teams.Away.Goals
		return g.LiveData.Linescore.Teams.Away.Team, nil
	}

	return nil, fmt.Errorf("could not locate home team in Game")
}
func (g *Game) GetQuarter() (int, error) {
	return 0, nil
}
func (g *Game) GetClock() (string, error) {
	return "0:0", nil
}

func (g *Game) GetUpdate(ctx context.Context) (sportboard.Game, error) {
	return GetLiveGame(ctx, g.Link)
}

func timeFromGameTime(gameTime string) (time.Time, error) {
	t, err := time.Parse("2006-01-02T15:04:05Z", gameTime)
	if err != nil {
		return time.Time{}, err
	}

	t = t.Local()

	return t, nil
}

func GetLiveGame(ctx context.Context, link string) (sportboard.Game, error) {
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
