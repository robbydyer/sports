package nhlboard

import (
	"context"
	"time"

	rgb "github.com/robbydyer/rgbmatrix-rpi"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/pkg/nhl"
)

type scoreBoard struct {
	api *nhl.Nhl
}

func New(ctx context.Context) ([]board.Board, error) {
	var err error

	var boards []board.Board

	b := &scoreBoard{}

	b.api, err = nhl.New(ctx)
	if err != nil {
		return nil, err
	}

	boards = append(boards, b)

	return boards, nil
}

func (b *scoreBoard) Name() string {
	return "NHL Scoreboard"
}

func (b *scoreBoard) Cleanup() {}

func (b *scoreBoard) Render(ctx context.Context, matrix rgb.Matrix, rotationDelay time.Duration) error {
	return nil
}

func (b *scoreBoard) HasPriority() bool {
	return false
}
