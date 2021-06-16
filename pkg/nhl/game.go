package nhl

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

// LiveGameGetter retrieves a live game from a game link
type LiveGameGetter func(ctx context.Context, link string) (sportboard.Game, error)

// Game implements sportboard.Game
type Game struct {
	GameGetter LiveGameGetter
	ID         int    `json:"gamePk"`
	Link       string `json:"link"`
	Teams      *struct {
		Away *gameTeam `json:"away"`
		Home *gameTeam `json:"home"`
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
				Home *gameTeam `json:"home"`
				Away *gameTeam `json:"away"`
			} `json:"teams"`
			CurrentPeriod              int    `json:"currentPeriod,omitempty"`
			CurrentPeriodOrdinal       string `json:"currentPeriodOrdinal"`
			CurrentPeriodTimeRemaining string `json:"currentPeriodTimeRemaining,omitempty"`
		} `json:"linescore"`
	} `json:"liveData"`
}

type gameTeam struct {
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

// Games ...
func (n *NHL) Games(dateStr string) ([]*Game, error) {
	games, ok := n.games[dateStr]
	if !ok || len(games) == 0 {
		return nil, fmt.Errorf("no games for date %s", dateStr)
	}

	return games, nil
}

// GetStartTime ...
func (g *Game) GetStartTime(ctx context.Context) (time.Time, error) {
	return g.GameTime, nil
}

// GetID ...
func (g *Game) GetID() int {
	return g.ID
}

// GetLink ...
func (g *Game) GetLink() (string, error) {
	return g.Link, nil
}

// IsLive ...
func (g *Game) IsLive() (bool, error) {
	complete, err := g.IsComplete()
	if err != nil {
		return false, err
	}
	if complete {
		return false, nil
	}
	if g.LiveData != nil && g.LiveData.Linescore != nil && g.LiveData.Linescore.CurrentPeriod > 0 {
		return true, nil
	}
	return false, nil
}

// IsComplete ...
func (g *Game) IsComplete() (bool, error) {
	if g.GameData != nil &&
		g.GameData.Status != nil &&
		strings.Contains(strings.ToLower(g.GameData.Status.AbstractGameState), "final") {
		return true, nil
	}
	if g.LiveData != nil &&
		g.LiveData.Linescore != nil &&
		strings.Contains(strings.ToLower(g.LiveData.Linescore.CurrentPeriodTimeRemaining), "final") {
		return true, nil
	}
	return false, nil
}

// IsPostponed ...
func (g *Game) IsPostponed() (bool, error) {
	if g.GameData != nil &&
		g.GameData.Status != nil &&
		strings.Contains(strings.ToLower(g.GameData.Status.DetailedState), "postponed") {
		return true, nil
	}

	return false, nil
}

// HomeTeam ...
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

// AwayTeam ...
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

// GetQuarter ...
func (g *Game) GetQuarter() (string, error) {
	if g.LiveData != nil && g.LiveData.Linescore != nil {
		return g.LiveData.Linescore.CurrentPeriodOrdinal, nil
	}

	return "", nil
}

// GetClock ...
func (g *Game) GetClock() (string, error) {
	if g.LiveData != nil && g.LiveData.Linescore != nil {
		return g.LiveData.Linescore.CurrentPeriodTimeRemaining, nil
	}
	return "00:00", nil
}

// GetUpdate ...
func (g *Game) GetUpdate(ctx context.Context) (sportboard.Game, error) {
	if g.GameGetter == nil {
		g.GameGetter = GetLiveGame
	}
	return g.GameGetter(ctx, g.Link)
}

// GetOdds ...
func (g *Game) GetOdds() (string, string, error) {
	return "", "", fmt.Errorf("not implemented")
}

func timeFromGameTime(gameTime string) (time.Time, error) {
	t, err := time.Parse("2006-01-02T15:04:05Z", gameTime)
	if err != nil {
		return time.Time{}, err
	}

	t = t.Local()

	return t, nil
}

// GetLiveGame is a LiveGameGetter
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
	uri := fmt.Sprintf("%s/schedule?date=%s&expand=schedule.linescore", baseURL, dateStr)
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
