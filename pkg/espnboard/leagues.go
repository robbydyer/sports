package espnboard

import (
	"context"

	"go.uber.org/zap"
)

type nfl struct{}

func (n *nfl) Sport() string {
	return "football"
}

func (n *nfl) League() string {
	return "nfl"
}

// NewNFL ...
func NewNFL(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	return New(ctx, &nfl{}, logger)
}
