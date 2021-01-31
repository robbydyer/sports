package mlb

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type teams struct {
	Teams []*Team `json:"teams"`
}

type Team struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Link         string `json:"link,omitempty"`
	Runs         int
}

func (t *Team) GetID() int {
	return t.ID
}

func (t *Team) GetName() string {
	return t.Name
}

func (t *Team) GetAbbreviation() string {
	return t.Abbreviation
}

func (t *Team) Score() int {
	return 0
}

func GetTeams(ctx context.Context) ([]*Team, error) {
	uri, err := url.Parse(fmt.Sprintf("%s/v1/teams", BaseURL))
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

	return teams.Teams, nil
}
