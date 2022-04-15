package pga

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/robfig/cron/v3"

	statboard "github.com/robbydyer/sports/internal/board/stat"
)

const leaderboardURL = "https://site.web.api.espn.com/apis/site/v2/sports/golf/leaderboard?league=pga"

var rawDatReg = regexp.MustCompile(`\s*([0-9]{1,2})\.{0,1}\s+([a-zA-Z]{1}[a-zA-Z\.\s/]+[a-zA-Z]{1})\s+(-{0,1}[0-9]{1,2})\s+([0-9F]{1,2})`)

// PGA ...
type PGA struct {
	log            *zap.Logger
	players        []*Player
	updateInterval time.Duration
	lastUpdate     time.Time
}

type eventDat struct {
	Events []struct {
		ShortName    string `json:"shortName"`
		Competitions []struct {
			DataFormat  string    `json:"dataFormat"`
			RawData     string    `json:"rawData"`
			Competitors []*Player `json:"competitors"`
		} `json:"competitions"`
	}
}

// New ...
func New(logger *zap.Logger, updateInterval time.Duration) (*PGA, error) {
	p := &PGA{
		log:            logger,
		updateInterval: updateInterval,
		lastUpdate:     time.Now().Add(-1 * updateInterval),
	}

	c := cron.New()

	if _, err := c.AddFunc("0 4 * * *", p.cacheClear); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *PGA) cacheClear() {
	p.players = []*Player{}
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
		"teetime",
	}, nil
}

// StatShortName ...
func (p *PGA) StatShortName(stat string) string {
	return ""
}

// ListPlayers ...
func (p *PGA) ListPlayers(ctx context.Context, teamAbbreviation string) ([]statboard.Player, error) {
	if len(p.players) < 1 || time.Since(p.lastUpdate) > p.updateInterval {
		var err error
		p.log.Debug("updating PGA leaderboard",
			zap.Duration("update interval", p.updateInterval),
			zap.Duration("since last update", time.Since(p.lastUpdate)),
		)
		p.players, err = p.updatePlayers(ctx)
		if err != nil {
			return nil, err
		}
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

	if len(dat.Events) < 1 || len(dat.Events[0].Competitions) < 1 {
		return nil, fmt.Errorf("could not find players")
	}

	comp := dat.Events[0].Competitions[0]
	if len(comp.Competitors) > 1 {
		p.lastUpdate = time.Now()
		return comp.Competitors, nil
	}

	if strings.Contains(comp.DataFormat, "RAW") && comp.RawData != "" {
		p.lastUpdate = time.Now()
		return p.parseRaw(comp.RawData)
	}

	return nil, fmt.Errorf("could not find players")
}

func (p *PGA) parseRaw(data string) ([]*Player, error) {
	fields := strings.Split(data, "\n")

	players := []*Player{}

	for _, f := range fields {
		f = strings.TrimSpace(f)
		p.log.Debug("parsing raw PGA data", zap.String("data", f))
		matches := rawDatReg.FindSubmatch([]byte(f))

		if len(matches) < 5 {
			p.log.Debug("not enough matches for PGA raw", zap.ByteStrings("matches", matches))
			continue
		}
		pos := string(matches[1])

		name := string(matches[2])

		score := string(matches[3])

		thru, err := strconv.Atoi(string(matches[4]))
		if err != nil {
			thru = 18
		}

		p.log.Debug("PGA raw data",
			zap.ByteStrings("matches", matches),
			zap.String("pos", pos),
			zap.String("name", name),
			zap.String("score", score),
			zap.Int("thru", thru),
		)

		newP := &Player{}
		newP.Status.Thru = thru
		newP.Status.Position.DisplayName = pos
		newP.Status.Position.ID = pos
		newP.Score.DisplayValue = score
		newP.Athlete.DisplayName = "X " + name

		players = append(players, newP)
	}

	return players, nil
}
