package statboard

import (
	"context"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/rgbrender"
)

// StatBoard ...
type StatBoard struct {
	config  *Config
	log     *zap.Logger
	api     API
	writers map[string]*rgbrender.TextWriter
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
}

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
	FirstName() string
	LastName() string
	GetStat(stat string) string
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
func New(ctx context.Context, api API, config *Config, logger *zap.Logger) (*StatBoard, error) {
	s := &StatBoard{
		config:  config,
		log:     logger,
		api:     api,
		writers: make(map[string]*rgbrender.TextWriter),
	}

	return s, nil
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
