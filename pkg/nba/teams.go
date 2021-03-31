package nba

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"

	// embed
	_ "embed"

	"go.uber.org/zap"
)

//go:embed assets/teams.json
var teamAsset []byte

type teams struct {
	Groups []struct {
		Children []*Conference `json:"children"`
	} `json:"groups"`
}

// Conference ...
type Conference struct {
	Name         string  `json:"name"`
	Abbreviation string  `json:"abbreviation"`
	Teams        []*Team `json:"teams"`
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

// GetTeams reads team data sourced via http://site.api.espn.com/apis/site/v2/sports/basketball/nba/groups
func GetTeams(ctx context.Context) ([]*Team, error) {
	var dat *teams

	if err := json.Unmarshal(teamAsset, &dat); err != nil {
		return nil, err
	}

	var teams []*Team

	for _, div := range dat.Groups {
		for _, conference := range div.Children {
			for _, team := range conference.Teams {
				team.Conference = conference
				teams = append(teams, team)
			}
		}
	}

	return teams, nil
}

func (t *Team) setDetails(ctx context.Context, log *zap.Logger) error {
	t.Lock()
	defer t.Unlock()
	if t.record != "" {
		return nil
	}

	uri := fmt.Sprintf("http://site.api.espn.com/apis/site/v2/sports/basketball/nba/teams/%s", t.ID)

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
