package clock

import (
	"context"

	rgb "github.com/robbydyer/rgbmatrix-rpi"

	"github.com/robbydyer/sports/internal/board"
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

func (b *basicClock) Cleanup() {}

func (b *basicClock) Render(ctx context.Context, matrix rgb.Matrix) error {
	cv := rgb.NewCanvas(matrix)
	cv.Clear()

	//rgbrender.DrawRectangle(cv, )

	return nil
}

func (b *basicClock) HasPriority() bool {
	return false
}
