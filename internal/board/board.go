package board

import (
	"context"

	rgb "github.com/robbydyer/rgbmatrix-rpi"
)

type Board interface {
	Name() string
	Render(ctx context.Context, matrix rgb.Matrix) error
	HasPriority() bool
	Cleanup()
}
