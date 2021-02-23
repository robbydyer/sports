package ncaam

import (
	"context"
	"encoding/json"
	"strconv"

	// embed
	_ "embed"
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
}

// Logo ...
type Logo struct {
	Href  string `json:"href"`
	Width int    `json:"width"`
	Heigh int    `json:"height"`
}

// GetTeams reads team data sourced via http://site.api.espn.com/apis/site/v2/sports/basketball/mens-college-basketball/groups
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
