package pga

import (
	"context"
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"

	"github.com/robbydyer/sports/pkg/statboard"
)

// Player ...
type Player struct {
	ID      string `json:"id"`
	Athlete struct {
		DisplayName string `json:"displayName"`
		Flag        struct {
			Href string `json:"href"`
		} `json:"flag"`
	}
	Status struct {
		TeeTime  string `json:"teeTime"`
		Hole     int    `json:"hole"`
		Thru     int    `json:"thru"`
		Position struct {
			DisplayName string `json:"displayName"`
			ID          string `json:"id"`
		}
	} `json:"status"`
	Score struct {
		DisplayValue string `json:"displayValue"`
	} `json:"score"`
	SortOrder int `json:"sortOrder"`
}

// SortByScore sorts players by score
func SortByScore(players []statboard.Player) []statboard.Player {
	sort.SliceStable(players, func(i, j int) bool {
		posI := players[i].GetStat("sort")
		posJ := players[j].GetStat("sort")
		pI, ierr := strconv.Atoi(posI)
		pJ, jerr := strconv.Atoi(posJ)
		if ierr != nil || jerr != nil {
			return posI < posJ
		}
		return pI < pJ
	})
	return players
}

// FirstName ...
func (p *Player) FirstName() string {
	parts := strings.Fields(p.Athlete.DisplayName)
	if len(parts) < 1 {
		return ""
	}
	return parts[0]
}

// LastName ...
func (p *Player) LastName() string {
	parts := strings.Fields(p.Athlete.DisplayName)
	if len(parts) < 1 {
		return ""
	}
	return strings.Join(parts[1:], " ")
}

// GetStat ...
func (p *Player) GetStat(stat string) string {
	switch strings.ToLower(stat) {
	case "teetime":
		return p.Status.TeeTime
	case "hole":
		return fmt.Sprint(p.Status.Thru)
	case "score":
		return p.Score.DisplayValue
	case "position":
		return p.Status.Position.DisplayName
	case "sort":
		return fmt.Sprint(p.SortOrder)
	}

	return ""
}

// StatColor ...
func (p *Player) StatColor(stat string) color.Color {
	switch strings.ToLower(stat) {
	case "score":
		score := p.GetStat("score")
		i, err := strconv.Atoi(score)
		if err != nil {
			return color.White
		}
		if i == 0 {
			return color.White
		}
		if i < 0 {
			return color.RGBA{255, 0, 0, 255}
		}
		return color.RGBA{0, 255, 0, 255}
	}
	return color.White
}

// Position returns nothing
func (p *Player) Position() string {
	return ""
}

// GetCategory ...
func (p *Player) GetCategory() string {
	return "player"
}

// UpdateStats does nothing, as stats are collected on calls to ListPlayers
func (p *Player) UpdateStats(ctx context.Context) error {
	return nil
}
