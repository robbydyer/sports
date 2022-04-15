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

	"github.com/twitchtv/twirp"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/enabler"
	pb "github.com/robbydyer/sports/internal/proto/basicboard"
	"github.com/robbydyer/sports/internal/rgbrender"
	"github.com/robbydyer/sports/internal/twirphelpers"
	"github.com/robbydyer/sports/internal/util"
)

var defaultUpdateInterval = 5 * time.Minute

// StatBoard ...
type StatBoard struct {
	config        *Config
	log           *zap.Logger
	api           API
	writers       map[string]*rgbrender.TextWriter
	sorter        Sorter
	withTitleRow  bool
	withPrefixCol bool
	lastUpdate    time.Time
	cancelBoard   chan struct{}
	rpcServer     pb.TwirpServer
	enabler       board.Enabler
	sync.Mutex
}

// Config ...
type Config struct {
	boardDelay      time.Duration
	updateInterval  time.Duration
	BoardDelay      string              `json:"boardDelay"`
	StartEnabled    *atomic.Bool        `json:"enabled"`
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
	if c.StartEnabled == nil {
		c.StartEnabled = atomic.NewBool(false)
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
		enabler:       enabler.New(),
	}

	if config.StartEnabled.Load() {
		s.enabler.Enable()
	}

	for _, f := range opts {
		if err := f(s); err != nil {
			return nil, err
		}
	}

	if s.sorter == nil {
		s.sorter = defaultSorter
	}

	if err := util.SetCrons(config.OnTimes, func() {
		s.log.Warn("statboard turning on",
			zap.String("league", s.api.LeagueShortName()),
		)
		s.Enabler().Enable()
	}); err != nil {
		return nil, err
	}
	if err := util.SetCrons(config.OffTimes, func() {
		s.log.Warn("statboard turning off",
			zap.String("league", s.api.LeagueShortName()),
		)
		s.Enabler().Disable()
	}); err != nil {
		return nil, err
	}

	svr := &Server{
		board: s,
	}
	prfx := s.api.HTTPPathPrefix()
	if !strings.HasPrefix(prfx, "/") {
		prfx = fmt.Sprintf("/%s", prfx)
	}
	prfx = fmt.Sprintf("/stat%s", prfx)

	s.rpcServer = pb.NewBasicBoardServer(svr,
		twirp.WithServerPathPrefix(prfx),
		twirp.ChainHooks(
			twirphelpers.GetDefaultHooks(s, s.log),
		),
	)
	s.log.Info("registering RPC server for Statboard",
		zap.String("league", s.api.LeagueShortName()),
		zap.String("prefix", s.rpcServer.PathPrefix()),
	)

	return s, nil
}

func defaultSorter(players []Player) []Player {
	sort.SliceStable(players, func(i, j int) bool {
		return strings.ToLower(players[i].LastName()) < strings.ToLower(players[j].LastName())
	})

	return players
}

func (s *StatBoard) Enabler() board.Enabler {
	return s.enabler
}

// InBetween ...
func (s *StatBoard) InBetween() bool {
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
