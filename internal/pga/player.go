package pga

import (
	"context"
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"time"

	statboard "github.com/robbydyer/sports/internal/board/stat"
)

// Player ...
type Player struct {
	ID      string   `json:"id"`
	Athlete athlete  `json:"athlete"`
	Team    *athlete `json:"team"`
	Status  struct {
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
	SortOrder  int `json:"sortOrder"`
	Statistics []*struct {
		Name         string `json:"name"`
		DisplayValue string `json:"displayValue"`
	} `json:"statistics"`
}

type athlete struct {
	DisplayName string `json:"displayName"`
	Flag        struct {
		Href string `json:"href"`
	} `json:"flag"`
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
	if p.Team != nil {
		// This is probably the Zuric classic
		return ""
	}
	parts := strings.Fields(p.Athlete.DisplayName)
	if len(parts) < 1 {
		return ""
	}
	return parts[0]
}

// LastName ...
func (p *Player) LastName() string {
	if p.Team != nil {
		// This is probably the Zuric classic
		return p.Team.DisplayName
	}
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
		t, err := time.Parse("2006-01-02T15:04Z", p.Status.TeeTime)
		if err != nil {
			return p.Status.TeeTime
		}
		if time.Until(t) < 0 {
			return ""
		}
		return t.Local().Format("03:04PM")
	case "hole":
		return fmt.Sprint(p.Status.Thru)
	case "score":
		if p.Statistics != nil {
			for _, stat := range p.Statistics {
				if stat.Name == "scoreToPar" && stat.DisplayValue != "" {
					return stat.DisplayValue
				}
			}
		}
		return p.Score.DisplayValue
	case "position":
		return strings.TrimLeft(p.Status.Position.DisplayName, "T")
	case "sort":
		return fmt.Sprint(p.SortOrder)
	}

	return ""
}

// PrefixCol returns the col before the player's name, leaderboard position
func (p *Player) PrefixCol() string {
	return strings.TrimLeft(p.Status.Position.DisplayName, "T")
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
