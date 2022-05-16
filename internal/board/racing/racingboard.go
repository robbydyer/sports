package racingboard

import (
	"context"
	"fmt"
	"image"
	"net/http"
	"strings"
	"time"

	"github.com/twitchtv/twirp"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/enabler"
	"github.com/robbydyer/sports/internal/logo"
	"github.com/robbydyer/sports/internal/rgbrender"
	scrcnvs "github.com/robbydyer/sports/internal/scrollcanvas"
	"github.com/robbydyer/sports/internal/twirphelpers"
	"github.com/robbydyer/sports/internal/util"

	pb "github.com/robbydyer/sports/internal/proto/racingboard"
)

// RacingBoard implements board.Board
type RacingBoard struct {
	config         *Config
	api            API
	log            *zap.Logger
	scheduleWriter *rgbrender.TextWriter
	leagueLogo     *logo.Logo
	events         []*Event
	rpcServer      pb.TwirpServer
	boardCtx       context.Context
	boardCancel    context.CancelFunc
	enabler        board.Enabler
}

// Todayer is a func that returns a string representing a date
// that will be used for determining "Today's" games.
// This is useful in testing what past days looked like
type Todayer func() []time.Time

// Config ...
type Config struct {
	TodayFunc          Todayer
	boardDelay         time.Duration
	scrollDelay        time.Duration
	StartEnabled       *atomic.Bool `json:"enabled"`
	BoardDelay         string       `json:"boardDelay"`
	ScrollMode         *atomic.Bool `json:"scrollMode"`
	ScrollDelay        string       `json:"scrollDelay"`
	OnTimes            []string     `json:"onTimes"`
	OffTimes           []string     `json:"offTimes"`
	TightScrollPadding int          `json:"tightScrollPadding"`
}

// API ...
type API interface {
	LeagueShortName() string
	GetLogo(ctx context.Context, bounds image.Rectangle) (*logo.Logo, error)
	GetScheduledEvents(ctx context.Context) ([]*Event, error)
	HTTPPathPrefix() string
}

// Event ...
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

	if c.StartEnabled == nil {
		c.StartEnabled = atomic.NewBool(false)
	}
	if c.ScrollMode == nil {
		c.ScrollMode = atomic.NewBool(false)
	}
	if c.ScrollDelay != "" {
		d, err := time.ParseDuration(c.ScrollDelay)
		if err != nil {
			c.scrollDelay = scrcnvs.DefaultScrollDelay
		}
		c.scrollDelay = d
	} else {
		c.scrollDelay = scrcnvs.DefaultScrollDelay
	}
}

// New ...
func New(api API, logger *zap.Logger, config *Config) (*RacingBoard, error) {
	s := &RacingBoard{
		config:  config,
		api:     api,
		log:     logger,
		enabler: enabler.New(),
	}

	if config.StartEnabled.Load() {
		s.enabler.Enable()
	}

	s.log.Info("Register Racing Board",
		zap.String("league", api.LeagueShortName()),
		zap.String("board name", s.Name()),
	)

	if s.config.boardDelay < 10*time.Second {
		s.log.Warn("cannot set sportboard delay below 10 sec")
		s.config.boardDelay = 10 * time.Second
	}

	if s.config.TodayFunc == nil {
		s.config.TodayFunc = util.Today
	}

	if err := util.SetCrons(config.OnTimes, func() {
		s.log.Info("sportboard turning on")
		s.Enabler().Enable()
	}); err != nil {
		return nil, err
	}
	if err := util.SetCrons(config.OffTimes, func() {
		s.log.Info("racingboard turning off")
		s.Enabler().Disable()
	}); err != nil {
		return nil, err
	}
	if err := util.SetCrons([]string{"0 4 * * *"}, s.cacheClear); err != nil {
		return nil, err
	}

	svr := &Server{
		board: s,
	}
	prfx := s.api.HTTPPathPrefix()
	if !strings.HasPrefix(prfx, "/") {
		prfx = fmt.Sprintf("/%s", prfx)
	}
	s.rpcServer = pb.NewRacingServer(svr,
		twirp.WithServerPathPrefix(prfx),
		twirp.ChainHooks(
			twirphelpers.GetDefaultHooks(s, s.log),
		),
	)

	return s, nil
}

func (s *RacingBoard) cacheClear() {
	s.events = []*Event{}
	s.leagueLogo = nil
}

// Name ...
func (s *RacingBoard) Name() string {
	return s.api.HTTPPathPrefix()
}

func (s *RacingBoard) Enabler() board.Enabler {
	return s.enabler
}

// InBetween ...
func (s *RacingBoard) InBetween() bool {
	return false
}

// ScrollMode ...
func (s *RacingBoard) ScrollMode() bool {
	return s.config.ScrollMode.Load()
}

// HasPriority ...
func (s *RacingBoard) HasPriority() bool {
	return false
}

// GetHTTPHandlers ...
func (s *RacingBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return nil, nil
}

// GetRPCHandler ...
func (s *RacingBoard) GetRPCHandler() (string, http.Handler) {
	return s.rpcServer.PathPrefix(), s.rpcServer
}
