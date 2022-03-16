package statboard

import (
	"context"
	"fmt"
	"image/color"
	"image/draw"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robfig/cron/v3"
	"github.com/twitchtv/twirp"

	pb "github.com/robbydyer/sports/internal/proto/basicboard"
	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
	"github.com/robbydyer/sports/pkg/twirphelpers"
)

var defaultUpdateInterval = 5 * time.Minute

// StatBoard ...
type StatBoard struct {
	config              *Config
	log                 *zap.Logger
	api                 API
	writers             map[string]*rgbrender.TextWriter
	sorter              Sorter
	withTitleRow        bool
	withPrefixCol       bool
	lastUpdate          time.Time
	cancelBoard         chan struct{}
	rpcServer           pb.TwirpServer
	stateChangeNotifier board.StateChangeNotifier
	sync.Mutex
}

// Config ...
type Config struct {
	boardDelay      time.Duration
	updateInterval  time.Duration
	BoardDelay      string              `json:"boardDelay"`
	Enabled         *atomic.Bool        `json:"enabled"`
	Players         []string            `json:"players"`
	Teams           []string            `json:"teams"`
	StatOverride    map[string][]string `json:"statOverride"`
	LimitPlayers    int                 `json:"limitPlayers"`
	UpdateInterval  string              `json:"updateInterval"`
	OnTimes         []string            `json:"onTimes"`
	OffTimes        []string            `json:"offTimes"`
	ScrollMode      *atomic.Bool        `json:"scrollMode"`
	Horizontal      *atomic.Bool        `json:"horizontal"`
	HorizontalLimit int                 `json:"horizontalLimit"`
}

// OptionFunc provides options to the StatBoard that are not exposed in a Config
type OptionFunc func(s *StatBoard) error

// Sorter sorts the ordering of a Player list for the stat board
type Sorter func(players []Player) []Player

// StringMeasurer measures the width of strings as they would be written to a canvas
type StringMeasurer interface {
	MeasureStrings(canvas draw.Image, strs []string) ([]int, error)
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
	StatColor(stat string) color.Color
	Position() string
	GetCategory() string
	UpdateStats(ctx context.Context) error
	PrefixCol() string
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
		} else {
			c.boardDelay = d
		}
	} else {
		c.boardDelay = 0 * time.Second
	}

	if c.UpdateInterval != "" {
		d, err := time.ParseDuration(c.UpdateInterval)
		if err != nil {
			c.updateInterval = defaultUpdateInterval
		} else {
			c.updateInterval = d
		}
	} else {
		c.updateInterval = defaultUpdateInterval
	}

	if c.StatOverride == nil {
		c.StatOverride = make(map[string][]string)
	}

	if c.ScrollMode == nil {
		c.ScrollMode = atomic.NewBool(false)
	}

	if c.Horizontal == nil {
		c.Horizontal = atomic.NewBool(false)
	}
	if c.HorizontalLimit == 0 {
		c.HorizontalLimit = 7
	}
}

// New ...
func New(ctx context.Context, api API, config *Config, logger *zap.Logger, opts ...OptionFunc) (*StatBoard, error) {
	s := &StatBoard{
		config:        config,
		log:           logger,
		api:           api,
		writers:       make(map[string]*rgbrender.TextWriter),
		withTitleRow:  true,
		withPrefixCol: false,
		cancelBoard:   make(chan struct{}),
	}

	for _, f := range opts {
		if err := f(s); err != nil {
			return nil, err
		}
	}

	if s.sorter == nil {
		s.sorter = defaultSorter
	}

	if len(config.OnTimes) > 1 || len(config.OffTimes) > 1 {
		c := cron.New()

		for _, off := range config.OffTimes {
			s.log.Info("statboard will be scheduled to turn off",
				zap.String("turn off", off),
				zap.String("league", s.api.LeagueShortName()),
			)
			_, err := c.AddFunc(off, func() {
				s.log.Warn("statboard turning off",
					zap.String("league", s.api.LeagueShortName()),
				)
				s.Disable()
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add cron for statboard off time: %w", err)
			}
		}
		for _, on := range config.OnTimes {
			s.log.Info("statboard will be scheduled to turn off",
				zap.String("turn on", on),
				zap.String("league", s.api.LeagueShortName()),
			)
			_, err := c.AddFunc(on, func() {
				s.log.Warn("statboard turning on",
					zap.String("league", s.api.LeagueShortName()),
				)
				s.Enable()
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add cron for statboard on time: %w", err)
			}
		}
		c.Start()
	}

	svr := &Server{
		board: s,
	}
	prfx := s.api.HTTPPathPrefix()
	if !strings.HasPrefix(prfx, "/") {
		prfx = fmt.Sprintf("/%s", prfx)
	}
	prfx = fmt.Sprintf("/stat%s", prfx)

	s.log.Info("registering RPC server for Statboard",
		zap.String("league", s.api.LeagueShortName()),
		zap.String("prefix", prfx),
	)
	s.rpcServer = pb.NewBasicBoardServer(svr,
		twirp.WithServerPathPrefix(prfx),
		twirp.ChainHooks(
			twirphelpers.GetDefaultHooks(s, s.log),
		),
	)

	return s, nil
}

func defaultSorter(players []Player) []Player {
	sort.SliceStable(players, func(i, j int) bool {
		return strings.ToLower(players[i].LastName()) < strings.ToLower(players[j].LastName())
	})

	return players
}

// Enabled ...
func (s *StatBoard) Enabled() bool {
	return s.config.Enabled.Load()
}

// Enable ...
func (s *StatBoard) Enable() bool {
	if s.config.Enabled.CAS(false, true) {
		if s.stateChangeNotifier != nil {
			s.stateChangeNotifier()
		}
		return true
	}
	return false
}

// InBetween ...
func (s *StatBoard) InBetween() bool {
	return false
}

// Disable ..
func (s *StatBoard) Disable() bool {
	if s.config.Enabled.CAS(true, false) {
		if s.stateChangeNotifier != nil {
			s.stateChangeNotifier()
		}
		return true
	}
	return false
}

// Name ...
func (s *StatBoard) Name() string {
	return fmt.Sprintf("StatBoard: %s", s.api.LeagueShortName())
}

// Clear ...
func (s *StatBoard) Clear() error {
	return nil
}

// Close ...
func (s *StatBoard) Close() error {
	return nil
}

// ScrollMode ...
func (s *StatBoard) ScrollMode() bool {
	return s.config.ScrollMode.Load()
}

// SetStateChangeNotifier ...
func (s *StatBoard) SetStateChangeNotifier(st board.StateChangeNotifier) {
	s.stateChangeNotifier = st
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

// WithPrefixCol enables/disables the prefix column in the statboard
func WithPrefixCol(with bool) OptionFunc {
	return func(s *StatBoard) error {
		s.withPrefixCol = with
		return nil
	}
}
