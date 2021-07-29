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

	"go.uber.org/atomic"
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
	hasDetail    *atomic.Bool
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	DisplayName  string  `json:"displayName"`
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
		e.log.Debug("returning cached ESPN teams",
			zap.Int("num teams", len(e.teams)),
			zap.String("league", e.leaguer.League()),
		)
		return e.teams, nil
	}

	assetFiles := []string{
		fmt.Sprintf("%s_teams.json", e.leaguer.HTTPPathPrefix()),
		fmt.Sprintf("%s_groups.json", e.leaguer.HTTPPathPrefix()),
	}

	teams := []*Team{}

	foundAsset := false
	for _, assetFile := range assetFiles {
		dat, err := assets.ReadFile(filepath.Join("assets", assetFile))
		if err != nil {
			continue
		}
		e.log.Info("pulling team info from assets",
			zap.String("league", e.leaguer.League()),
			zap.String("file", assetFile),
		)
		t, err := parseTeamData(dat)
		if err != nil {
			return nil, err
		}
		teams = append(teams, t...)
		foundAsset = true
	}

	if foundAsset {
		return teams, nil
	}

	e.log.Info("pulling team info from API",
		zap.String("league", e.leaguer.League()),
	)
	for _, endpoint := range e.leaguer.TeamEndpoints() {
		dat, err := pullTeams(ctx, endpoint)
		if err != nil {
			return nil, err
		}
		t, err := parseTeamData(dat)
		if err != nil {
			return nil, err
		}
		teams = append(teams, t...)
	}

	return teams, nil
}

func parseTeamData(dat []byte) ([]*Team, error) {
	teamSet := make(map[string]*Team)
	var d *teamData

	if err := json.Unmarshal(dat, &d); err != nil {
		return nil, err
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
				teamSet[team.Abbreviation] = team
			}
		}
	}

	var teams []*Team
	for _, sport := range d.Sports {
		for _, league := range sport.Leagues {
			for _, t := range league.Teams {
				teamSet[t.Team.Abbreviation] = t.Team
			}
		}
	}

	for _, team := range teamSet {
		team.hasDetail = atomic.NewBool(false)
		teams = append(teams, team)
	}

	return teams, nil
}

// setDetails sets info about a team's record, rank, etc
func (t *Team) setDetails(ctx context.Context, apiPath string, log *zap.Logger) error {
	t.Lock()
	defer t.Unlock()

	if t.hasDetail.Load() {
		return nil
	}

	uri := fmt.Sprintf("http://site.api.espn.com/apis/site/v2/sports/%s/teams/%s", apiPath, t.ID)

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

	defer t.hasDetail.Store(true)

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

// GetDisplayName ...
func (t *Team) GetDisplayName() string {
	return t.DisplayName
}

// GetAbbreviation ...
func (t *Team) GetAbbreviation() string {
	return t.Abbreviation
}

// ConferenceName ...
func (t *Team) ConferenceName() string {
	if t.Conference != nil {
		return t.Conference.Abbreviation
	}

	return ""
}

// Score ...
func (t *Team) Score() int {
	p, _ := strconv.Atoi(t.Points)

	return p
}

func pullTeams(ctx context.Context, endpoint string) ([]byte, error) {
	uri, err := url.Parse(fmt.Sprintf("http://site.api.espn.com/apis/site/v2/sports/%s", endpoint))
	if err != nil {
		return nil, err
	}

	v := uri.Query()
	v.Set("limit", "500")

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
