package mlb

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Team struct {
	Abbreviation string `json:"name_abbrev"`
	Name         string `json:"name_display_full"`
}

type teamsAPI struct {
	TeamsAllSeason *struct {
		QueryResults *struct {
			Row []*Team `json:"row"`
		} `json:"queryResults"`
	} `json:"teams_all_season"`
}

func GetTeams(ctx context.Context) ([]*Team, error) {
	uri := fmt.Sprintf("%s/sport_code='mlb'&sort_order=name_asc&season=%s", apiBase, "2020")

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	defer resp.Body.Close()

	var teams *teamsAPI

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &teams); err != nil {
		return nil, err
	}

	var teamList []*Team

	for _, t := range teams.TeamsAllSeason.QueryResults.Row {
		teamList = append(teamList, t)
	}

	return teamList, nil
}
