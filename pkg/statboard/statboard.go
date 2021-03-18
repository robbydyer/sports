package statboard

import (
	"context"
	"image/color"
	"sort"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/rgbrender"
)

// StatBoard ...
type StatBoard struct {
	config       *Config
	log          *zap.Logger
	api          API
	writers      map[string]*rgbrender.TextWriter
	sorter       Sorter
	withTitleRow bool
	sync.Mutex
}

// Config ...
type Config struct {
	boardDelay   time.Duration
	BoardDelay   string              `json:"boardDelay"`
	Enabled      *atomic.Bool        `json:"enabled"`
	Players      []string            `json:"players"`
	Teams        []string            `json:"teams"`
	StatOverride map[string][]string `json:"statOverride"`
	LimitPlayers int                 `json:"limitPlayers"`
}

type OptionFunc func(s *StatBoard) error

type Sorter func(players []Player) []Player

// API ...
type API interface {
	FindPlayer(ctx context.Context, firstName string, lastName string) (Player, error)
	GetPlayer(ctx context.Context, id string) (Player, error)
	AvailableStats(ctx context.Context, playerCategory string) ([]string, error)
	StatShortName(stat string) string
	ListPlayers(ctx context.Context, teamAbbreviation string) ([]Player, error)
	LeagueShortName() string
	HTTPPathPrefix() string
	PlayerCategories() []string
}

// Player ...
type Player interface {
	FirstName(shorten bool) string
	LastName(shorten bool) string
	GetStat(stat string) string
	StatColor(stat string) color.Color
	Position() string
	GetCategory() string
	UpdateStats(ctx context.Context) error
}

// SetDefaults ...
func (c *Config) SetDefaults() {
	if c.Enabled == nil {
		c.Enabled = atomic.NewBool(false)
	}
	if c.BoardDelay != "" {
		d, err := time.ParseDuration(c.BoardDelay)
		if err != nil {
			c.boardDelay = 0 * time.Second
		}
		c.boardDelay = d
	} else {
		c.boardDelay = 0 * time.Second
	}

	if c.StatOverride == nil {
		c.StatOverride = make(map[string][]string)
	}
}

// New ...
func New(ctx context.Context, api API, config *Config, logger *zap.Logger, opts ...OptionFunc) (*StatBoard, error) {
	s := &StatBoard{
		config:       config,
		log:          logger,
		api:          api,
		writers:      make(map[string]*rgbrender.TextWriter),
		withTitleRow: true,
	}

	for _, f := range opts {
		if err := f(s); err != nil {
			return nil, err
		}
	}

	if s.sorter == nil {
		s.sorter = defaultSorter
	}

	return s, nil
}

func defaultSorter(players []Player) []Player {
	sort.SliceStable(players, func(i, j int) bool {
		return players[i].LastName(false) < players[j].LastName(false)
	})

	return players
}

// Enabled ...
func (s *StatBoard) Enabled() bool {
	return s.config.Enabled.Load()
}

// Enable ...
func (s *StatBoard) Enable() {
	s.config.Enabled.Store(true)
}

// Disable ..
func (s *StatBoard) Disable() {
	s.config.Enabled.Store(false)
}

// Name ...
func (s *StatBoard) Name() string {
	return "StatBoard"
}

// Clear ...
func (s *StatBoard) Clear() error {
	return nil
}

// Close ...
func (s *StatBoard) Close() error {
	return nil
}

// WithSorter ...
func WithSorter(sorter Sorter) OptionFunc {
	return func(s *StatBoard) error {
		s.sorter = sorter
		return nil
	}
}

// WithTitleRow enables/disables the stats title row
func WithTitleRow(with bool) OptionFunc {
	return func(s *StatBoard) error {
		s.withTitleRow = with
		return nil
	}
}
