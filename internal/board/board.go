package board

import (
	"context"
	"time"

	rgb "github.com/robbydyer/rgbmatrix-rpi"
)

type Board interface {
	Name() string
	Render(ctx context.Context, matrix rgb.Matrix, rotationDelay time.Duration) error
	HasPriority() bool
	Cleanup()
}
