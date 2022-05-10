package matrix

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
	PreLoad(*MatrixScene)
	Play(ctx context.Context, startInterval time.Duration, interval <-chan time.Duration) error
}

type MatrixScene struct {
	Points []MatrixPoint
	// Index is the ordering of each scene
	Index int
}

type MatrixPoint struct {
	X     int
	Y     int
	Color color.Color
}
