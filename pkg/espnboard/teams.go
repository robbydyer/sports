package espnboard

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	// embed
	_ "embed"

	"go.uber.org/zap"
)

//go:embed assets
var assets embed.FS

// Conference ...
type Conference struct {
	Name         string
	Abbreviation string
}

// Team implements sportboard.Team
type Team struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Abbreviation string  `json:"abbreviation"`
	Color        string  `json:"color"`
	Logos        []*Logo `json:"logos"`
	Points       string  `json:"score"`
	LogoURL      string  `json:"logo"`
	Conference   *Conference
	IsHome       bool
	rank         int
	record       string
	sync.Mutex
}

// Logo ...
type Logo struct {
	Href  string `json:"href"`
	Width int    `json:"width"`
	Heigh int    `json:"height"`
}

type teamData struct {
	Sports []struct {
		Leagues []struct {
			Teams []struct {
				Team *Team `json:"team"`
			} `json:"teams"`
		} `json:"leagues"`
	} `json:"sports"`
	Groups []struct {
		// This is the Conference abbreviation
		Abbreviation string `json:"abbreviation"`
		Children     []struct {
			// Division abbreviation
			Abbreviation string  `json:"abbreviation"`
			Name         string  `json:"name"`
			Teams        []*Team `json:"teams"`
		} `json:"children"`
	} `json:"groups"`
}

type teamDetails struct {
	Team struct {
		ID           string `json:"id"`
		Abbreviation string `json:"abbreviation"`
		Color        string `json:"color"`
		Rank         int    `json:"rank"`
		Record       struct {
			Items []struct {
				Description string `json:"description"`
				Type        string `json:"type"`
				Summary     string `json:"summary"`
			}
		}
	}
}

// GetTeams reads team data sourced via http://site.api.espn.com/apis/site/v2/sports/football/nfl/groups
func (e *ESPNBoard) getTeams(ctx context.Context) ([]*Team, error) {
	if len(e.teams) > 1 {
		e.log.Debug("returning cached ESPN teams", zap.Int("num teams", len(e.teams)))
		return e.teams, nil
	}

	assetFile := fmt.Sprintf("%s_%s_teams.json", e.leaguer.Sport(), e.leaguer.League())

	dat, err := assets.ReadFile(filepath.Join("assets", assetFile))
	if err != nil {
		e.log.Info("pulling team info from API",
			zap.String("sport", e.leaguer.Sport()),
			zap.String("league", e.leaguer.League()),
		)
		dat, err = pullTeams(ctx, e.leaguer.Sport(), e.leaguer.League())
		if err != nil {
			return nil, err
		}
	} else {
		e.log.Info("pulling team info from assets",
			zap.String("sport", e.leaguer.Sport()),
			zap.String("league", e.leaguer.League()),
			zap.String("file", assetFile),
		)
	}

	var d *teamData

	if err := json.Unmarshal(dat, &d); err != nil {
		return nil, err
	}

	var teams []*Team
	for _, sport := range d.Sports {
		for _, league := range sport.Leagues {
			for _, t := range league.Teams {
				teams = append(teams, t.Team)
			}
		}
	}

	for _, group := range d.Groups {
		conf := group.Abbreviation
		for _, c := range group.Children {
			division := c.Abbreviation
			for _, team := range c.Teams {
				conf := &Conference{
					Name:         c.Name,
					Abbreviation: fmt.Sprintf("%s_%s", conf, division),
				}
				team.Conference = conf
				teams = append(teams, team)
			}
		}
	}

	for _, team := range teams {
		if err := team.setDetails(ctx, e.leaguer.Sport(), e.leaguer.League(), e.log); err != nil {
			return nil, err
		}
	}

	return teams, nil
}

func (t *Team) setDetails(ctx context.Context, sport string, league string, log *zap.Logger) error {
	t.Lock()
	defer t.Unlock()
	if t.record != "" {
		return nil
	}

	uri := fmt.Sprintf("http://site.api.espn.com/apis/site/v2/sports/%s/%s/teams/%s", sport, league, t.ID)

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return err
	}

	client := http.DefaultClient

	req = req.WithContext(ctx)

	log.Info("fetching team data", zap.String("team", t.Abbreviation))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var details *teamDetails

	if err := json.Unmarshal(body, &details); err != nil {
		return err
	}

	t.rank = details.Team.Rank

	for _, i := range details.Team.Record.Items {
		if strings.ToLower(i.Type) != "total" {
			continue
		}

		log.Debug("setting team record", zap.String("team", t.Abbreviation), zap.String("record", i.Summary))
		t.record = i.Summary
		return nil
	}
	log.Error("did not find record for team", zap.String("team", t.Abbreviation))

	return nil
}

// GetID ...
func (t *Team) GetID() int {
	id, err := strconv.Atoi(t.ID)
	if err != nil {
		return 0
	}

	return id
}

// GetName ...
func (t *Team) GetName() string {
	return t.Name
}

// GetAbbreviation ...
func (t *Team) GetAbbreviation() string {
	return t.Abbreviation
}

// Score ...
func (t *Team) Score() int {
	p, _ := strconv.Atoi(t.Points)

	return p
}

func pullTeams(ctx context.Context, sport string, league string) ([]byte, error) {
	uri, err := url.Parse(fmt.Sprintf("http://site.api.espn.com/apis/site/v2/sports/%s", teamEndpoint(sport, league)))
	if err != nil {
		return nil, err
	}

	v := uri.Query()
	v.Set("limit", "400")

	uri.RawQuery = v.Encode()

	req, err := http.NewRequest("GET", uri.String(), nil)
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

	return ioutil.ReadAll(resp.Body)
}

func teamEndpoint(sport string, league string) string {
	switch league {
	case "mens-college-basketball":
		return "basketball/mens-college-basketball/groups"
	}
	return fmt.Sprintf("%s/%s/teams", sport, league)
}
