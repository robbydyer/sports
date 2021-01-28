package mlb

import (
	"context"
	"time"

	"github.com/robbydyer/sports/pkg/sportboard"
)

type Game struct {
	ID int
}

func (g *Game) GetID() int {
	return g.ID
}
func (g *Game) GetLink() (string, error) {
	return "", nil
}
func (g *Game) IsLive() (bool, error) {
	return true, nil
}
func (g *Game) IsComplete() (bool, error) {
	return false, nil
}
func (g *Game) HomeTeam() (sportboard.Team, error) {
	return nil, nil
}
func (g *Game) AwayTeam() (sportboard.Team, error) {
	return nil, nil
}

// GetQuarter returns the inning
func (g *Game) GetQuarter() (int, error) {
	return 1, nil
}

func (g *Game) GetClock() (string, error) {
	return "", nil
}
func (g *Game) GetUpdate(ctx context.Context) (sportboard.Game, error) {
	return nil, nil
}
func (g *Game) GetStartTime(ctx context.Context) (time.Time, error) {
	return time.Now().Local(), nil
}
