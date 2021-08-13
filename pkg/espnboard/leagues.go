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
	return []string{filepath.Join(n.APIPath(), "groups")}
}

func (n *nfl) HTTPPathPrefix() string {
	return "nfl"
}

// NewNFL ...
func NewNFL(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	return New(ctx, &nfl{}, logger, defaultRankSetter, defaultRankSetter)
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
	return New(ctx, &ncaam{}, logger, defaultRankSetter, defaultRankSetter)
}

type nba struct{}

func (n *nba) League() string {
	return "NBA"
}

func (n *nba) APIPath() string {
	return "basketball/nba"
}

func (n *nba) TeamEndpoints() []string {
	return []string{filepath.Join(n.APIPath(), "groups")}
}

func (n *nba) HTTPPathPrefix() string {
	return "nba"
}

// NewNBA ...
func NewNBA(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	return New(ctx, &nba{}, logger, defaultRankSetter, defaultRankSetter)
}

type mls struct{}

func (n *mls) League() string {
	return "MLS"
}

func (n *mls) APIPath() string {
	return "soccer/usa.1"
}

func (n *mls) TeamEndpoints() []string {
	return []string{filepath.Join(n.APIPath(), "teams")}
}

func (n *mls) HTTPPathPrefix() string {
	return "mls"
}

// NewMLS ...
func NewMLS(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	return New(ctx, &mls{}, logger, defaultRankSetter, defaultRankSetter)
}

type nhl struct{}

func (n *nhl) League() string {
	return "NHL"
}

func (n *nhl) APIPath() string {
	return "hockey/nhl"
}

func (n *nhl) TeamEndpoints() []string {
	return []string{
		filepath.Join(n.APIPath(), "groups"),
		filepath.Join(n.APIPath(), "teams"),
	}
}

func (n *nhl) HTTPPathPrefix() string {
	return "nhl"
}

// NewNHL ...
func NewNHL(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	return New(ctx, &nhl{}, logger, defaultRankSetter, defaultRankSetter)
}

type mlb struct{}

func (n *mlb) League() string {
	return "MLB"
}

func (n *mlb) APIPath() string {
	return "baseball/mlb"
}

func (n *mlb) TeamEndpoints() []string {
	return []string{
		filepath.Join(n.APIPath(), "groups"),
		filepath.Join(n.APIPath(), "teams"),
	}
}

func (n *mlb) HTTPPathPrefix() string {
	return "mlb"
}

// NewMLB ...
func NewMLB(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	return New(ctx, &mlb{}, logger, defaultRankSetter, defaultRankSetter)
}

type ncaaf struct{}

func (n *ncaaf) League() string {
	return "NCAAF"
}

func (n *ncaaf) APIPath() string {
	return "football/college-football"
}

func (n *ncaaf) TeamEndpoints() []string {
	return []string{
		filepath.Join(n.APIPath(), "teams"),
		// TODO: Group endpoint is different for NCAAF. It does not
		// contain conference data
		// filepath.Join(n.APIPath(), "groups"),
	}
}

func (n *ncaaf) HTTPPathPrefix() string {
	return "ncaaf"
}

// NewNCAAF ...
func NewNCAAF(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	n := &ncaaf{}
	return New(ctx, n, logger, n.setRankings, n.setRecords)
}

// NewEPL ...
func NewEPL(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	return New(ctx, &epl{}, logger, defaultRankSetter, defaultRankSetter)
}

type epl struct{}

func (n *epl) League() string {
	return "EPL"
}

func (n *epl) APIPath() string {
	return "soccer/eng.1"
}

func (n *epl) TeamEndpoints() []string {
	return []string{
		filepath.Join(n.APIPath(), "groups"),
		filepath.Join(n.APIPath(), "teams"),
	}
}

func (n *epl) HTTPPathPrefix() string {
	return "epl"
}
