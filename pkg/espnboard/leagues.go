package espnboard

import (
	"context"
	"path/filepath"

	"go.uber.org/zap"
)

type nfl struct{}

func (n *nfl) League() string {
	return "NFL"
}

func (n *nfl) APIPath() string {
	return "football/nfl"
}

func (n *nfl) TeamEndpoints() []string {
	return []string{filepath.Join(n.APIPath(), "teams")}
}

func (n *nfl) HTTPPathPrefix() string {
	return "nfl"
}

// NewNFL ...
func NewNFL(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	return New(ctx, &nfl{}, logger)
}

type ncaam struct{}

func (n *ncaam) League() string {
	return "NCAA Basketball"
}

func (n *ncaam) APIPath() string {
	return "basketball/mens-college-basketball"
}

func (n *ncaam) TeamEndpoints() []string {
	return []string{
		filepath.Join(n.APIPath(), "groups"),
		filepath.Join(n.APIPath(), "teams"),
	}
}

func (n *ncaam) HTTPPathPrefix() string {
	return "ncaam"
}

// NewNCAAMensBasketball ...
func NewNCAAMensBasketball(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	return New(ctx, &ncaam{}, logger)
}
