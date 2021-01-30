package mlb

import (
	"context"
	"image"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/sportboard"
	"github.com/robbydyer/sports/pkg/util"
)

const (
	BaseURL      = "https://statsapi.mlb.com/api"
	LinkBase     = "https://statsapi.mlb.com"
	logoCacheDir = "/tmp/sportsmatrix_logos/mlb"
	DateFormat   = "2006-01-02"
	ATL          = "ATL"
)

var ALL = []string{ATL}

type MLB struct {
	teams           []*Team
	games           map[string][]*Game
	logos           map[string]*logo.Logo
	logoSourceCache map[string]image.Image
	log             *log.Logger
}

func New(ctx context.Context, logger *log.Logger) (*MLB, error) {
	m := &MLB{
		games:           make(map[string][]*Game),
		logos:           make(map[string]*logo.Logo),
		logoSourceCache: make(map[string]image.Image),
		log:             logger,
	}

	if err := m.UpdateTeams(ctx); err != nil {
		return nil, err
	}

	if err := m.UpdateGames(ctx, util.Today().Format(DateFormat)); err != nil {
		return nil, err
	}

	return m, nil
}

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
func (m *MLB) AllTeamAbbreviations() []string {
	return ALL
}

func (m *MLB) UpdateTeams(ctx context.Context) error {
	teamList, err := GetTeams(ctx)
	if err != nil {
		return err
	}

	m.teams = teamList

	return nil
}

func (m *MLB) UpdateGames(ctx context.Context, dateStr string) error {
	return nil
}
