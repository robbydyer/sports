package espnboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/sportboard"
)

var overUnderRegex = regexp.MustCompile(`^([A-Z]+)\s+([-]{0,1}[0-9]+[\.0-9]*)`)

type schedule struct {
	Events []*event `json:"events"`
}

type event struct {
	ID           string  `json:"id"`
	Date         string  `json:"date"`
	Status       *status `json:"status"`
	Competitions []struct {
		Competitors []struct {
			HomeAway string `json:"homeAway"`
			Team     *Team  `json:"team"`
			Score    string `json:"score"`
		}
		Odds []*Odds `json:"odds"`
	} `json:"competitions"`
}

// Odds represents a game's betting odds
type Odds struct {
	Provider *struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Priority int    `json:"priority"`
	} `json:"provider"`
	Details   string  `json:"details"`
	OverUnder float64 `json:"overUnder"`
}

// Game ...
type Game struct {
	ID       string
	Home     *Team
	Away     *Team
	GameTime time.Time
	status   *status
	leaguer  Leaguer
	odds     []*Odds
}

type status struct {
	DisplayClock string `json:"displayClock"`
	Period       int    `json:"period"`
	Type         struct {
		Name        string `json:"name"`
		Completed   *bool  `json:"completed"`
		Description string `json:"description"`
		State       string `json:"state"`
	} `json:"type"`
}

// GetID ...
func (g *Game) GetID() int {
	id, _ := strconv.Atoi(g.ID)
	return id
}

// GetLink ...
func (g *Game) GetLink() (string, error) {
	return "", nil
}

// IsLive ...
func (g *Game) IsLive() (bool, error) {
	complete, err := g.IsComplete()
	if err != nil {
		return false, err
	}
	if g.status.Period > 0 && !complete {
		return true, nil
	}

	return false, nil
}

// IsComplete ...
func (g *Game) IsComplete() (bool, error) {
	if strings.Contains(g.status.Type.Name, "FINAL") && g.status.Type.Completed != nil && *g.status.Type.Completed {
		return true, nil
	}

	return false, nil
}

// IsPostponed ...
func (g *Game) IsPostponed() (bool, error) {
	n := strings.ToLower(g.status.Type.Name)
	d := strings.ToLower(g.status.Type.Description)

	if strings.Contains(n, "postponed") || strings.Contains(d, "postponed") || strings.Contains(n, "canceled") || strings.Contains(d, "canceled") {
		return true, nil
	}

	return false, nil
}

// HomeTeam ...
func (g *Game) HomeTeam() (sportboard.Team, error) {
	return g.Home, nil
}

// AwayTeam ...
func (g *Game) AwayTeam() (sportboard.Team, error) {
	return g.Away, nil
}

// GetQuarter ...
func (g *Game) GetQuarter() (string, error) {
	return strconv.Itoa(g.status.Period), nil
}

// GetClock ...
func (g *Game) GetClock() (string, error) {
	return g.status.DisplayClock, nil
}

// GetUpdate ...
func (g *Game) GetUpdate(ctx context.Context) (sportboard.Game, error) {
	uri, err := url.Parse(
		fmt.Sprintf("http://site.api.espn.com/apis/site/v2/sports/%s/scoreboard/%s", g.leaguer.APIPath(), g.ID),
	)
	if err != nil {
		return nil, err
	}

	v := uri.Query()
	v.Set("lang", "en")
	v.Set("region", "us")

	uri.RawQuery = v.Encode()

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	client := http.DefaultClient

	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to GET games: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var event *event

	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal game JSON: %w", err)
	}

	return gameFromEvent(event)
}

// GetStartTime ...
func (g *Game) GetStartTime(ctx context.Context) (time.Time, error) {
	return g.GameTime, nil
}

// GetOdds ...
func (g *Game) GetOdds() (string, string, error) {
	if len(g.odds) == 0 {
		return "", "", fmt.Errorf("no odds for game %d", g.GetID())
	}

	for _, odd := range g.odds {
		if odd.Provider.Priority == 1 || odd.Provider.Priority == 0 {
			return extractOverUnder(odd.Details)
		}
	}

	return extractOverUnder(g.odds[0].Details)
}

func extractOverUnder(details string) (string, string, error) {
	match := overUnderRegex.FindStringSubmatch(details)
	if len(match) < 3 {
		return "", "", fmt.Errorf("no match found")
	}

	return match[1], match[2], nil
}

// GetGames gets the games for a given date
func (e *ESPNBoard) GetGames(ctx context.Context, dateStr string) ([]*Game, error) {
	// http://site.api.espn.com/apis/site/v2/sports/football/nfl/scoreboard?lang=en&region=us&limit=500&dates=20191121&groups=50
	uri, err := url.Parse(fmt.Sprintf("http://site.api.espn.com/apis/site/v2/sports/%s/scoreboard", e.leaguer.APIPath()))
	if err != nil {
		return nil, err
	}

	v := uri.Query()
	v.Set("lang", "en")
	v.Set("region", "us")
	v.Set("dates", dateStr)

	if e.leaguer.League() == "mens-college-basketball" {
		v.Set("limit", "500")
		v.Set("groups", "50")
	}

	uri.RawQuery = v.Encode()

	e.log.Debug("fetching games",
		zap.String("date", dateStr),
		zap.String("league", e.leaguer.League()),
		zap.String("uri", uri.String()),
	)

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	client := http.DefaultClient

	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to GET games: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var schedule *schedule

	if err := json.Unmarshal(body, &schedule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal game JSON: %w", err)
	}

	var games []*Game
	for _, event := range schedule.Events {
		game, err := gameFromEvent(event)
		if err != nil {
			return nil, err
		}

		game.leaguer = e.leaguer

		games = append(games, game)
	}

	return games, nil
}

func gameFromEvent(event *event) (*Game, error) {
	t, err := timeFromGameTime(event.Date)
	if err != nil {
		return nil, err
	}
	game := &Game{
		ID:       event.ID,
		GameTime: t,
		status:   event.Status,
	}
	for _, comp := range event.Competitions {
		game.odds = append(game.odds, comp.Odds...)
		for _, team := range comp.Competitors {
			if strings.ToLower(team.HomeAway) == "home" {
				game.Home = team.Team
				game.Home.Points = team.Score
			} else {
				game.Away = team.Team
				game.Away.Points = team.Score
			}
		}
	}

	return game, nil
}

// TimeToGameDateStr converts a time.Time into the date string format the API expects
func TimeToGameDateStr(t time.Time) string {
	return t.Format(DateFormat)
}

func timeFromGameTime(gameTime string) (time.Time, error) {
	t, err := time.Parse("2006-01-02T15:04Z", gameTime)
	if err != nil {
		return time.Time{}, err
	}

	t = t.Local()

	return t, nil
}
