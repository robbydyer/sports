package mlb

import (
	"context"
	// embed
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

//go:embed assets/divisions.json
var divisionAPIData []byte

type teams struct {
	Teams []*Team `json:"teams"`
}

// Team implements a sportboard.Team
type Team struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Link         string `json:"link,omitempty"`
	DivisionData struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Link string `json:"link"`
	} `json:"division"`
	Division *division
	Runs     int
	Roster   []*Player
}

type rosterData struct {
	Roster []*Player `json:"roster"`
}

type divisionData struct {
	Divisions []*division `json:"divisions"`
}

type division struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	NameShort    string `json:"nameShort"`
	Abbreviation string `json:"abbreviation"`
}

// GetID ...
func (t *Team) GetID() int {
	return t.ID
}

// GetName ...
func (t *Team) GetName() string {
	return t.Name
}

// GetAbbreviation ...
func (t *Team) GetAbbreviation() string {
	return t.Abbreviation
}

// GetDisplayName ...
func (t *Team) GetDisplayName() string {
	return t.Name
}

// ConferenceName ...
func (t *Team) ConferenceName() string {
	if t.Division != nil {
		return t.Division.Abbreviation
	}
	return ""
}

// Score ...
func (t *Team) Score() int {
	return t.Runs
}

// GetTeams ...
func GetTeams(ctx context.Context) ([]*Team, error) {
	uri, err := url.Parse(fmt.Sprintf("%s/v1/teams", baseURL))
	if err != nil {
		return nil, err
	}
	yr := strconv.Itoa(time.Now().Year())
	v := uri.Query()
	v.Set("season", yr)
	v.Set("leagueIds", "103,104")
	v.Set("sportId", "1")

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

	var teams *teams

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &teams); err != nil {
		return nil, err
	}

	var d *divisionData

	if err := json.Unmarshal(divisionAPIData, &d); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MLB divisions: %w", err)
	}

OUTER:
	for _, team := range teams.Teams {
		for _, div := range d.Divisions {
			if div.ID == team.DivisionData.ID {
				team.Division = div
				continue OUTER
			}
		}
	}

	for _, team := range teams.Teams {
		if err := team.setRoster(ctx); err != nil {
			return nil, err
		}
	}

	return teams.Teams, nil
}

func (t *Team) setRoster(ctx context.Context) error {
	uri, err := url.Parse(fmt.Sprintf("%s/%s/roster", linkBase, t.Link))
	if err != nil {
		return err
	}

	yr := strconv.Itoa(time.Now().Year())
	v := uri.Query()
	v.Set("season", yr)

	uri.RawQuery = v.Encode()

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	client := http.DefaultClient

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var dat *rosterData

	if err := json.Unmarshal(body, &dat); err != nil {
		return err
	}

	t.Roster = dat.Roster

	return nil
}
