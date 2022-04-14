package nhl

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/util"
)

//go:embed assets/divisions.json
var divisionAPIData []byte

//go:embed assets
var assets embed.FS

// Team implements sportboard.Team
type Team struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	Abbreviation     string `json:"abbreviation"`
	NextGameSchedule *struct {
		Dates []*struct {
			Games []*Game `json:"games"`
		} `json:"dates"`
	} `json:"nextGameSchedule,omitempty"`
	DivisionData struct {
		ID int `json:"id"`
	} `json:"division"`
	Roster struct {
		Roster []*Player `json:"roster"`
	} `json:"roster"`
	Division *division
	score    int
}

type divisionData struct {
	Divisions []*division `json:"divisions"`
}

type division struct {
	ID           int    `json:"id"`
	Abbreviation string `json:"abbreviation"`
}

type teams struct {
	Teams []*Team `json:"teams"`
}

// GetID ...
func (t *Team) GetID() string {
	return fmt.Sprintf("%d", t.ID)
}

// GetName ...
func (t *Team) GetName() string {
	return t.Name
}

// GetDisplayName ...
func (t *Team) GetDisplayName() string {
	return t.Name
}

// GetAbbreviation ...
func (t *Team) GetAbbreviation() string {
	return t.Abbreviation
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
	return t.score
}

func (t *Team) setGameTimes() error {
	if t.NextGameSchedule == nil {
		return nil
	}
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

// GetTeams ...
func (n *NHL) getTeams(ctx context.Context) ([]*Team, error) {
	var body []byte
	var teams *teams
	var err error

	body, err = n.getTeamAPIData(ctx)
	if err != nil {
		var err2 error
		n.log.Error("failed to get team data from API",
			zap.String("league", n.LeagueShortName()),
			zap.Error(err),
		)
		body, err2 = assets.ReadFile(filepath.Join("assets", "teams.json"))
		if err2 != nil {
			return nil, fmt.Errorf("failed to get team data: %s: %w", err.Error(), err2)
		}
	}

	if err := json.Unmarshal(body, &teams); err != nil {
		return nil, fmt.Errorf("failed to unmarshal NHL API response: %w", err)
	}

	for _, t := range teams.Teams {
		// Set game time to a time.Time
		if err := t.setGameTimes(); err != nil {
			return nil, fmt.Errorf("failed to set GameTime: %w", err)
		}
	}

	var d *divisionData

	if err := json.Unmarshal(divisionAPIData, &d); err != nil {
		return nil, fmt.Errorf("failed to unmarshal NHL divisions: %w", err)
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

	return teams.Teams, nil
}

func (n *NHL) getTeamAPIData(ctx context.Context) ([]byte, error) {
	season := ""
	if d := util.Today(); len(d) > 0 {
		season = fmt.Sprintf("season=%s", GetSeason(d[0]))
		n.log.Debug("nhl today season",
			zap.String("util.Today", d[0].String()),
		)
	} else {
		n.log.Error("failed to determine today",
			zap.String("league", n.LeagueShortName()),
		)
	}
	uri := fmt.Sprintf("%s/teams?expand=team.roster,team.schedule.next&%s", baseURL, season)
	n.log.Debug("fetching team data from API",
		zap.String("league", n.LeagueShortName()),
		zap.String("url", uri),
	)
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response body: %w", err)
	}

	return body, nil
}
