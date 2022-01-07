package racingboard

import (
	"context"
	"fmt"
	"image"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/util"
)

// RacingBoard implements board.Board
type RacingBoard struct {
	config *Config
	api    API
	log    *zap.Logger
}

// Todayer is a func that returns a string representing a date
// that will be used for determining "Today's" games.
// This is useful in testing what past days looked like
type Todayer func() []time.Time

// Config ...
type Config struct {
	TodayFunc   Todayer
	boardDelay  time.Duration
	scrollDelay time.Duration
	Enabled     *atomic.Bool `json:"enabled"`
	BoardDelay  string       `json:"boardDelay"`
	ScrollMode  *atomic.Bool `json:"scrollMode"`
	ScrollDelay string       `json:"scrollDelay"`
	OnTimes     []string     `json:"onTimes"`
	OffTimes    []string     `json:"offTimes"`
}

// API ...
type API interface {
	GetLogo(ctx context.Context) (*logo.Logo, error)
	GetScheduledEvents(ctx context.Context) ([]*Event, error)
}

type Event struct {
	Date time.Time
	Name string
}

// SetDefaults sets config defaults
func (c *Config) SetDefaults() {
	if c.BoardDelay != "" {
		d, err := time.ParseDuration(c.BoardDelay)
		if err != nil {
			c.boardDelay = 10 * time.Second
		}
		c.boardDelay = d
	} else {
		c.boardDelay = 10 * time.Second
	}

	if c.Enabled == nil {
		c.Enabled = atomic.NewBool(false)
	}
	if c.ScrollMode == nil {
		c.ScrollMode = atomic.NewBool(false)
	}
	if c.ScrollDelay != "" {
		d, err := time.ParseDuration(c.ScrollDelay)
		if err != nil {
			c.scrollDelay = rgbmatrix.DefaultScrollDelay
		}
		c.scrollDelay = d
	} else {
		c.scrollDelay = rgbmatrix.DefaultScrollDelay
	}
}

// New ...
func New(ctx context.Context, api API, bounds image.Rectangle, logger *zap.Logger, config *Config) (*RacingBoard, error) {
	s := &RacingBoard{
		config: config,
		api:    api,
		log:    logger,
	}

	if s.config.boardDelay < 10*time.Second {
		s.log.Warn("cannot set sportboard delay below 10 sec")
		s.config.boardDelay = 10 * time.Second
	}

	if s.config.TodayFunc == nil {
		s.config.TodayFunc = util.Today
	}

	c := cron.New()

	for _, on := range config.OnTimes {
		s.log.Info("racingboard will be schedule to turn on",
			zap.String("turn on", on),
		)
		_, err := c.AddFunc(on, func() {
			s.log.Info("sportboard turning on")
			s.Enable()
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add cron for sportboard: %w", err)
		}
	}

	for _, off := range config.OffTimes {
		s.log.Info("racingboard will be schedule to turn off",
			zap.String("turn on", off),
		)
		_, err := c.AddFunc(off, func() {
			s.log.Info("racingboard turning off")
			s.Disable()
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add cron for sportboard: %w", err)
		}
	}

	c.Start()

	return s, nil
}

// Name ...
func (s *RacingBoard) Name() string {
	return "RacingBoard"
}

// Enabled ...
func (s *RacingBoard) Enabled() bool {
	return s.config.Enabled.Load()
}

// Enable ...
func (s *RacingBoard) Enable() {
	s.config.Enabled.Store(true)
}

// InBetween ...
func (s *RacingBoard) InBetween() bool {
	return false
}

// Disable ...
func (s *RacingBoard) Disable() {
	s.config.Enabled.Store(false)
}

// ScrollMode ...
func (s *RacingBoard) ScrollMode() bool {
	return s.config.ScrollMode.Load()
}

// HasPriority ...
func (s *RacingBoard) HasPriority() bool {
	return false
}
