package rgbmatrix

import (
	"context"
	"image/color"
	"time"
)

// Matrix is an interface that represent any RGB matrix, very useful for testing
type Matrix interface {
	Geometry() (width, height int)
	At(x int, y int) color.Color
	Set(x int, y int, c color.Color)
	// Apply([]color.Color) error
	Render() error
	Close() error
	SetBrightness(brightness int)
	PreLoad([]MatrixPoint)
	Play(ctx context.Context, interval time.Duration) error
}

type MatrixPoint struct {
	X     int
	Y     int
	Color color.Color
}
