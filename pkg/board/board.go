package board

import (
	"context"

	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
)

// Board is the interface to implement for displaying on the matrix
type Board interface {
	Name() string
	Render(ctx context.Context, matrix rgb.Matrix) error
	HasPriority() bool
	Enabled() bool
	Cleanup()
}
