package espnboard

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// GetLeaguer ...
func GetLeaguer(league string) (Leaguer, error) {
	switch strings.Trim(strings.ToLower(league), " ") {
	case "nfl":
		return &nfl{}, nil
	case "mlb":
		return &mlb{}, nil
	case "ncaaf":
		return &ncaaf{}, nil
	case "ncaam":
		return &ncaam{}, nil
	case "epl":
		return &epl{}, nil
	case "nhl":
		return &nhl{}, nil
	case "mls":
		return &mls{}, nil
	case "nba":
		return &nba{}, nil
	case "dfl":
		return &dfl{}, nil
	case "dfb":
		return &dfb{}, nil
	case "uefa":
		return &uefa{}, nil
	case "fifa":
		return &fifa{}, nil
	}

	return nil, fmt.Errorf("invalid league '%s'", league)
}

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

func (n *nfl) HeadlinePath() string {
	return "football/nfl/news"
}

func (n *nfl) HomeSideSwap() bool {
	return false
}

func (n *nfl) SetScoreboardQuery(v url.Values) {
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

func (n *ncaam) HeadlinePath() string {
	return fmt.Sprintf("%s/news", n.APIPath())
}

func (n *ncaam) HomeSideSwap() bool {
	return false
}

func (n *ncaam) SetScoreboardQuery(v url.Values) {
	v.Set("groups", "50")
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

func (n *nba) HeadlinePath() string {
	return fmt.Sprintf("%s/news", n.APIPath())
}

func (n *nba) HomeSideSwap() bool {
	return false
}

func (n *nba) SetScoreboardQuery(v url.Values) {
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

func (n *mls) HeadlinePath() string {
	return fmt.Sprintf("%s/news", n.APIPath())
}

func (n *mls) HomeSideSwap() bool {
	return true
}

func (n *mls) SetScoreboardQuery(v url.Values) {
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

func (n *nhl) HeadlinePath() string {
	return fmt.Sprintf("%s/news", n.APIPath())
}

func (n *nhl) HomeSideSwap() bool {
	return false
}

func (n *nhl) SetScoreboardQuery(v url.Values) {
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

func (n *mlb) HeadlinePath() string {
	return fmt.Sprintf("%s/news", n.APIPath())
}

func (n *mlb) HomeSideSwap() bool {
	return false
}

func (n *mlb) SetScoreboardQuery(v url.Values) {
}

// NewMLB ...
func NewMLB(ctx context.Context, logger *zap.Logger, opts ...Option) (*ESPNBoard, error) {
	return New(ctx, &mlb{}, logger, defaultRankSetter, defaultRankSetter, opts...)
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

func (n *ncaaf) HeadlinePath() string {
	return fmt.Sprintf("%s/news", n.APIPath())
}

func (n *ncaaf) HomeSideSwap() bool {
	return false
}

// NewNCAAF ...
func NewNCAAF(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	n := &ncaaf{}
	return New(ctx, n, logger, n.setRankings, n.setRecords)
}

func (n *ncaaf) SetScoreboardQuery(v url.Values) {
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
		filepath.Join(n.APIPath(), "teams"),
	}
}

func (n *epl) HTTPPathPrefix() string {
	return "epl"
}

func (n *epl) HeadlinePath() string {
	return fmt.Sprintf("%s/news", n.APIPath())
}

func (n *epl) HomeSideSwap() bool {
	return true
}

func (n *epl) SetScoreboardQuery(v url.Values) {
}

// NewDFL ...
func NewDFL(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	return New(ctx, &dfl{}, logger, defaultRankSetter, defaultRankSetter)
}

type dfl struct{}

func (n *dfl) League() string {
	return "DFL"
}

func (n *dfl) APIPath() string {
	return "soccer/ger.1"
}

func (n *dfl) TeamEndpoints() []string {
	return []string{
		filepath.Join(n.APIPath(), "teams"),
	}
}

func (n *dfl) HTTPPathPrefix() string {
	return "dfl"
}

func (n *dfl) HeadlinePath() string {
	return fmt.Sprintf("%s/news", n.APIPath())
}

func (n *dfl) HomeSideSwap() bool {
	return true
}

func (n *dfl) SetScoreboardQuery(v url.Values) {
}

// NewDFB ...
func NewDFB(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	return New(ctx, &dfb{}, logger, defaultRankSetter, defaultRankSetter)
}

type dfb struct{}

func (n *dfb) League() string {
	return "DFB"
}

func (n *dfb) APIPath() string {
	return "soccer/ger.dfb_pokal"
}

func (n *dfb) TeamEndpoints() []string {
	return []string{
		filepath.Join(n.APIPath(), "teams"),
	}
}

func (n *dfb) HTTPPathPrefix() string {
	return "dfb"
}

func (n *dfb) HeadlinePath() string {
	return fmt.Sprintf("%s/news", n.APIPath())
}

func (n *dfb) HomeSideSwap() bool {
	return true
}

func (n *dfb) SetScoreboardQuery(v url.Values) {
}

// NewUEFA ...
func NewUEFA(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	return New(ctx, &uefa{}, logger, defaultRankSetter, defaultRankSetter)
}

type uefa struct{}

func (n *uefa) League() string {
	return "UEFA"
}

func (n *uefa) APIPath() string {
	return "soccer/uefa.champions"
}

func (n *uefa) TeamEndpoints() []string {
	return []string{
		filepath.Join(n.APIPath(), "teams"),
	}
}

func (n *uefa) HTTPPathPrefix() string {
	return "uefa"
}

func (n *uefa) HeadlinePath() string {
	return fmt.Sprintf("%s/news", n.APIPath())
}

func (n *uefa) HomeSideSwap() bool {
	return true
}

func (n *uefa) SetScoreboardQuery(v url.Values) {
}

// NewFIFA ...
func NewFIFA(ctx context.Context, logger *zap.Logger) (*ESPNBoard, error) {
	return New(ctx, &fifa{}, logger, defaultRankSetter, defaultRankSetter)
}

type fifa struct{}

func (n *fifa) League() string {
	return "FIFA"
}

func (n *fifa) APIPath() string {
	return "soccer/fifa.world"
}

func (n *fifa) TeamEndpoints() []string {
	return []string{
		filepath.Join(n.APIPath(), "teams"),
	}
}

func (n *fifa) HTTPPathPrefix() string {
	return "fifa"
}

func (n *fifa) HeadlinePath() string {
	return fmt.Sprintf("%s/news", n.APIPath())
}

func (n *fifa) HomeSideSwap() bool {
	return true
}

func (n *fifa) SetScoreboardQuery(v url.Values) {
}
