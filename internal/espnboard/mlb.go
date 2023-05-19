package espnboard

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/robbydyer/sports/internal/mlblive"
)

func (g *Game) GetRunners(ctx context.Context) (*mlblive.Runners, error) {
	if g.Situation != nil {
		return &mlblive.Runners{
			First:  g.Situation.OnFirst,
			Second: g.Situation.OnSecond,
			Third:  g.Situation.OnThird,
		}, nil
	}

	return nil, fmt.Errorf("could not get runner info")
}

func (g *Game) GetCount(ctx context.Context) (string, error) {
	if g.Situation != nil {
		return fmt.Sprintf("%d-%d", g.Situation.Balls, g.Situation.Strikes), nil
	}

	return "", fmt.Errorf("could not get count")
}

func (g *Game) GetInningState(ctx context.Context) (*mlblive.InningState, error) {
	topBott, err := g.GetClock()
	if err != nil {
		return nil, err
	}
	top := strings.EqualFold(topBott, "top")

	inn, err := g.GetQuarter()
	if err != nil {
		return nil, err
	}

	outs := 0
	if g.Situation != nil {
		outs = g.Situation.Outs
	}

	return &mlblive.InningState{
		Number: inn,
		IsTop:  top,
		Outs:   outs,
	}, nil
}

func (g *Game) GetHomeScore(ctx context.Context) (int, error) {
	if g.Home != nil {
		return strconv.Atoi(g.Home.Points)
	}

	return 0, fmt.Errorf("could not get home team score")
}

func (g *Game) GetAwayScore(ctx context.Context) (int, error) {
	if g.Away != nil {
		return strconv.Atoi(g.Away.Points)
	}

	return 0, fmt.Errorf("could not get away team score")
}
