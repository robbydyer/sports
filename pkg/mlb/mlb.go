package mlb

import (
	"context"
	"image"
	"time"

	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/sportboard"
)

const (
	ATL = "ATL"
)

var ALL = []string{ATL}

type MLB struct{}

func (m *MLB) GetTeams(ctx context.Context) ([]sportboard.Team, error) {
	return nil, nil
}
func (m *MLB) TeamFromAbbreviation(ctx context.Context, abbreviation string) (sportboard.Team, error) {
	return nil, nil
}
func (m *MLB) GetScheduledGames(ctx context.Context, date time.Time) ([]sportboard.Game, error) {
	return nil, nil
}
func (m *MLB) DateStr(d time.Time) string {
	return ""
}
func (m *MLB) League() string {
	return "MLB"
}
func (m *MLB) GetLogo(logoKey string, logoConf *logo.Config, bounds image.Rectangle) (*logo.Logo, error) {
	return nil, nil
}
func (m *MLB) AllTeamAbbreviations() []string {
	return ALL
}
