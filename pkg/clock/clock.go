package clock

import (
	"context"

	"github.com/robbydyer/sports/pkg/board"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
)

// Board implements board.Board
type basicClock struct{}

func New() (board.Board, error) {
	b := &basicClock{}

	return b, nil
}

func (b *basicClock) Name() string {
	return "Basic Clock"
}

func (b *basicClock) Enabled() bool {
	return true
}

func (b *basicClock) Cleanup() {}

func (b *basicClock) Render(ctx context.Context, matrix rgb.Matrix) error {
	cv := rgb.NewCanvas(matrix)
	cv.Clear()

	// rgbrender.DrawRectangle(cv, )

	return nil
}

func (b *basicClock) HasPriority() bool {
	return false
}
