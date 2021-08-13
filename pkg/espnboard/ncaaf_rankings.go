package espnboard

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

var preferedPolls = []string{"cfp", "ap", "usa"}

type ncaafRankingsData struct {
	Rankings []*struct {
		Type   string `json:"type"`
		Season *struct {
			Year int `json:"year"`
		} `json:"season"`
		Ranks []*ncaafRanks `json:"ranks"`
	}
}

type ncaafRanks struct {
	Current  int    `json:"current"`
	Previous int    `json:"previous"`
	Record   string `json:"recordSummary"`
	Team     *struct {
		Abbreviation string `json:"abbreviation"`
	} `json:"team"`
}

func (n *ncaaf) setRankings(ctx context.Context, e *ESPNBoard, season string, teams []*Team) error {
	// For NCAAF we set rankings for all teams at once, so we don't need to run this more
	// than once per day
	if e.ranksSet.Load() {
		return nil
	}
	e.Lock()
	defer e.Unlock()

	uri, err := url.Parse("https://site.api.espn.com/apis/site/v2/sports/football/college-football/rankings")
	if err != nil {
		return err
	}

	if season != "" {
		v := uri.Query()
		v.Set("season", season)
		uri.RawQuery = v.Encode()
	}

	e.log.Info("getting NCAAF rankings",
		zap.String("season", season),
		zap.String("url", uri.String()),
	)

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return err
	}
	client := http.DefaultClient

	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data *ncaafRankingsData

	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	ranks := n.getRanks(data)

RANK:
	for _, rank := range ranks {
		for _, t := range e.teams {
			if err := t.setDetails(ctx, season, e.leaguer.APIPath(), e.log); err != nil {
				e.log.Error("failed to set NCAAF team details",
					zap.Error(err),
				)
			}
			if t.Abbreviation == rank.Team.Abbreviation {
				e.log.Debug("setting NCAAF rank",
					zap.String("team", t.Abbreviation),
					zap.Int("rank", rank.Current),
					zap.String("record", rank.Record),
				)
				t.Lock()
				t.rank = rank.Current
				t.record = rank.Record
				t.Unlock()
				continue RANK
			}
		}
	}

	e.ranksSet.Store(true)

	return nil
}

func (n *ncaaf) getRanks(data *ncaafRankingsData) []*ncaafRanks {
	prefIndex := 0
	thisYear := n.latestRankingsYear(data)

	for {
		if prefIndex > len(preferedPolls) {
			return nil
		}
	INNER:
		for _, ranking := range data.Rankings {
			if ranking.Type != preferedPolls[prefIndex] || ranking.Season.Year != thisYear {
				continue INNER
			}
			return ranking.Ranks
		}
		prefIndex++
	}
}

func (n *ncaaf) latestRankingsYear(data *ncaafRankingsData) int {
	year := 0
	for _, r := range data.Rankings {
		if r.Season.Year > year {
			year = r.Season.Year
		}
	}
	if year == 0 {
		return time.Now().Year()
	}
	return year
}
