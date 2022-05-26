package espnboard

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	sportboard "github.com/robbydyer/sports/internal/board/sport"
	"github.com/robbydyer/sports/internal/rgbrender"
)

var (
	overUnderRegex   = regexp.MustCompile(`^([A-Z]+)\s+([-]{0,1}[0-9]+[\.0-9]*)`)
	mLB              = "MLB"
	scheduleAPILimit = 30 * time.Second
)

type schedule struct {
	Events []*event `json:"events"`
}

type baseballSituation struct {
	Balls    int  `json:"balls"`
	Strikes  int  `json:"strikes"`
	OnFirst  bool `json:"onFirst"`
	OnSecond bool `json:"onSecond"`
	OnThird  bool `json:"onThird"`
	Outs     int  `json:"outs"`
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
		Odds      []*Odds            `json:"odds"`
		Situation *baseballSituation `json:"situation"`
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
	espnBoard *ESPNBoard
	ID        string
	Home      *Team
	Away      *Team
	GameTime  time.Time
	status    *status
	leaguer   Leaguer
	odds      []*Odds
	Situation *baseballSituation
}

type status struct {
	DisplayClock string `json:"displayClock"`
	Period       int    `json:"period"`
	Type         struct {
		Name        string `json:"name"`
		Completed   *bool  `json:"completed"`
		Description string `json:"description"`
		State       string `json:"state"`
		ShortDetail string `json:"shortDetail"`
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
	if complete {
		return false, nil
	}
	if time.Until(g.GameTime).Minutes() > 0 {
		return false, nil
	}
	if g.status.Period > 0 {
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

func (g *Game) HomeAbbrev() string {
	return g.Home.Abbreviation
}

func (g *Game) AwayAbbrev() string {
	return g.Away.Abbreviation
}

func (g *Game) HomeColor() (*color.RGBA, *color.RGBA, error) {
	if g.Home != nil {
		r, gr, b, err := rgbrender.HexToRGB(g.Home.Color)
		if err != nil {
			return nil, nil, err
		}
		r2, g2, b2, err := rgbrender.HexToRGB(g.Home.AlternateColor)
		if err != nil {
			return nil, nil, err
		}
		return &color.RGBA{
				R: r,
				G: gr,
				B: b,
				A: 255,
			}, &color.RGBA{
				R: r2,
				B: b2,
				G: g2,
				A: 255,
			}, nil
	}

	return nil, nil, fmt.Errorf("failed to get home team color")
}

func (g *Game) AwayColor() (*color.RGBA, *color.RGBA, error) {
	if g.Away != nil {
		r, gr, b, err := rgbrender.HexToRGB(g.Away.Color)
		if err != nil {
			return nil, nil, err
		}
		r2, g2, b2, err := rgbrender.HexToRGB(g.Away.AlternateColor)
		if err != nil {
			return nil, nil, err
		}
		return &color.RGBA{
				R: r,
				G: gr,
				B: b,
				A: 255,
			}, &color.RGBA{
				R: r2,
				G: g2,
				B: b2,
				A: 255,
			}, nil
	}

	return nil, nil, fmt.Errorf("failed to get home team color")
}

// GetQuarter ...
func (g *Game) GetQuarter() (string, error) {
	if g.leaguer != nil && g.leaguer.League() == mLB {
		if g.status.Type.ShortDetail != "" {
			parts := strings.Fields(g.status.Type.ShortDetail)
			if len(parts) > 1 {
				return parts[1], nil
			}
		}
	}
	return strconv.Itoa(g.status.Period), nil
}

// GetClock ...
func (g *Game) GetClock() (string, error) {
	if g.leaguer != nil && g.leaguer.League() == mLB {
		if g.status.Type.ShortDetail != "" {
			parts := strings.Fields(g.status.Type.ShortDetail)
			if len(parts) > 0 {
				return parts[0], nil
			}
		}
	}
	return g.status.DisplayClock, nil
}

func (g *Game) getMockUpdate() (sportboard.Game, error) {
	var event *event

	if g.espnBoard == nil {
		return nil, fmt.Errorf("no mock data present")
	}
	mockDat, ok := g.espnBoard.mockLiveGames[g.ID]
	if !ok {
		return nil, fmt.Errorf("no mock data for game %s", g.ID)
	}

	if err := json.Unmarshal(mockDat, &event); err != nil {
		return nil, err
	}

	newG, err := gameFromEvent(event, g.espnBoard)
	if err != nil {
		return nil, err
	}
	newG.leaguer = g.leaguer

	if len(newG.odds) < 1 {
		newG.odds = append(newG.odds, g.odds...)
	}

	return newG, nil
}

// GetUpdate ...
func (g *Game) GetUpdate(ctx context.Context) (sportboard.Game, error) {
	if g.espnBoard != nil && len(g.espnBoard.mockLiveGames) > 0 {
		return g.getMockUpdate()
	}

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

	newG, err := gameFromEvent(event, g.espnBoard)
	if err != nil {
		return nil, err
	}
	newG.leaguer = g.leaguer

	if len(newG.odds) < 1 {
		newG.odds = append(newG.odds, g.odds...)
	}

	return newG, nil
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

func (e *ESPNBoard) getMockGames() ([]*Game, error) {
	if e.mockSchedule == nil {
		return nil, fmt.Errorf("missing mock schedule data")
	}
	var schedule *schedule

	if err := json.Unmarshal(e.mockSchedule, &schedule); err != nil {
		return nil, err
	}

	var games []*Game
	for _, event := range schedule.Events {
		game, err := gameFromEvent(event, e)
		if err != nil {
			return nil, err
		}

		game.leaguer = e.leaguer

		games = append(games, game)
	}

	return games, nil
}

// GetGames gets the games for a given date
func (e *ESPNBoard) GetGames(ctx context.Context, dateStr string) ([]*Game, error) {
	e.gameLock.Lock()
	defer e.gameLock.Unlock()

	if e.mockSchedule != nil {
		return e.getMockGames()
	}

	t, ok := e.lastScheduleCall[dateStr]
	if !ok || t == nil {
		t := time.Now().Local()
		e.lastScheduleCall[dateStr] = &t
	} else {
		// Make sure we don't hammer this API if it has failures
		if time.Now().Local().Before(t.Add(scheduleAPILimit)) {
			e.log.Error("tried calling ESPN scoreboard API too quickly",
				zap.Time("last call", *t),
				zap.String("date", dateStr),
				zap.String("league", e.League()),
			)
			return nil, fmt.Errorf("called scoreboard API too quickly")
		}
	}

	// http://site.api.espn.com/apis/site/v2/sports/football/nfl/scoreboard?lang=en&region=us&limit=500&dates=20191121&groups=50
	uri, err := url.Parse(fmt.Sprintf("http://site.api.espn.com/apis/site/v2/sports/%s/scoreboard", e.leaguer.APIPath()))
	if err != nil {
		return nil, err
	}

	v := uri.Query()
	v.Set("lang", "en")
	v.Set("region", "us")
	v.Set("dates", dateStr)
	v.Set("limit", "500")

	if e.leaguer.League() == "mens-college-basketball" {
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
		game, err := gameFromEvent(event, e)
		if err != nil {
			return nil, err
		}

		game.leaguer = e.leaguer

		games = append(games, game)
	}

	now := time.Now().Local()
	e.lastScheduleCall[dateStr] = &now

	return games, nil
}

func gameFromEvent(event *event, b *ESPNBoard) (*Game, error) {
	t, err := timeFromGameTime(event.Date)
	if err != nil {
		return nil, err
	}
	game := &Game{
		espnBoard: b,
		ID:        event.ID,
		GameTime:  t,
		status:    event.Status,
	}
	for _, comp := range event.Competitions {
		game.odds = append(game.odds, comp.Odds...)
		game.Situation = comp.Situation
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
