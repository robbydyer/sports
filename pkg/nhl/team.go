package nhl

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type Teams map[int]*Team

type teams struct {
	Teams []*Team `json:"teams"`
}
type Team struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	Abbreviation     string `json:"abbreviation"`
	NextGameSchedule *struct {
		Dates []*struct {
			Games []*Game `json:"games"`
		} `json:"dates"`
	} `json:"nextGameSchedule"`
}

type NextGame struct {
}

func (t *Team) setGameTimes() error {
	for _, d := range t.NextGameSchedule.Dates {
		for _, g := range d.Games {
			var err error
			g.GameTime, err = time.Parse("2006-01-02T15:04:05Z", g.GameDate)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func GetTeams(ctx context.Context) (Teams, error) {
	uri := fmt.Sprintf("%s/teams?expand=team.stats,team.schedule.previous,team.schedule.next", BaseURL)
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

	var teams *teams

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response body: %w", err)
	}

	if err := json.Unmarshal(body, &teams); err != nil {
		return nil, fmt.Errorf("failed to unmarshal NHL API response: %w", err)
	}

	teamList := make(map[int]*Team)
	for _, t := range teams.Teams {
		// Set game time to a time.Time
		if err := t.setGameTimes(); err != nil {
			return nil, fmt.Errorf("failed to set GameTime: %w", err)
		}

		teamList[t.ID] = t
	}

	return teamList, nil
}
