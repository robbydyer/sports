package stockboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"net/http"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robfig/cron/v3"
	"github.com/twitchtv/twirp"

	pb "github.com/robbydyer/sports/internal/proto/basicboard"
	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
	"github.com/robbydyer/sports/pkg/twirphelpers"
)

var (
	red        = color.RGBA{255, 0, 0, 255}
	green      = color.RGBA{0, 255, 0, 255}
	lightGreen = color.NRGBA{0, 255, 0, 50}
	lightRed   = color.NRGBA{255, 0, 0, 50}
)

// StockBoard displays stocks
type StockBoard struct {
	config       *Config
	api          API
	log          *zap.Logger
	symbolWriter *rgbrender.TextWriter
	priceWriter  *rgbrender.TextWriter
	enablerLock  sync.Mutex
	cancelBoard  chan struct{}
	rpcServer    pb.TwirpServer
	logos        map[string]*logo.Logo
	logoLock     sync.Mutex
	sync.Mutex
}

// Config for a StockBoard
type Config struct {
	boardDelay         time.Duration
	updateInterval     time.Duration
	scrollDelay        time.Duration
	adjustedResolution int
	Enabled            *atomic.Bool `json:"enabled"`
	Symbols            []string     `json:"symbols"`
	ChartResolution    int          `json:"chartResolution"`
	BoardDelay         string       `json:"boardDelay"`
	UpdateInterval     string       `json:"updateInterval"`
	ScrollMode         *atomic.Bool `json:"scrollMode"`
	TightScrollPadding int          `json:"tightScrollPadding"`
	ScrollDelay        string       `json:"scrollDelay"`
	OnTimes            []string     `json:"onTimes"`
	OffTimes           []string     `json:"offTimes"`
	UseLogos           *atomic.Bool `json:"useLogos"`
	MaxChartWidthRatio float64      `json:"maxChartWidthRatio"`
}

// Price represents a price of a stock at a particular time
type Price struct {
	Time  time.Time
	Price float64
}

// Stock ...
type Stock struct {
	Symbol    string
	OpenPrice float64
	Price     float64
	Prices    []*Price
	Change    float64
}

// API interface for getting stock data
type API interface {
	Get(ctx context.Context, symbols []string, interval time.Duration) ([]*Stock, error)
	TradingOpen() (time.Time, error)
	TradingClose() (time.Time, error)
	CacheClear()
}

// SetDefaults ...
func (c *Config) SetDefaults() {
	if c.Enabled == nil {
		c.Enabled = atomic.NewBool(false)
	}
	if c.ScrollMode == nil {
		c.ScrollMode = atomic.NewBool(false)
	}
	if c.ChartResolution <= 0 {
		c.ChartResolution = 4
	}
	if c.BoardDelay != "" {
		d, err := time.ParseDuration(c.BoardDelay)
		if err != nil {
			c.boardDelay = 10 * time.Second
		} else {
			c.boardDelay = d
		}
	} else {
		c.boardDelay = 10 * time.Second
	}

	if c.UpdateInterval != "" {
		d, err := time.ParseDuration(c.UpdateInterval)
		if err != nil {
			c.updateInterval = 5 * time.Minute
		} else {
			c.updateInterval = d
		}
	} else {
		c.updateInterval = 5 * time.Minute
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

	if c.UseLogos == nil {
		c.UseLogos = atomic.NewBool(false)
	}

	if c.MaxChartWidthRatio == 0 || c.MaxChartWidthRatio > 1 {
		c.MaxChartWidthRatio = 1
	}
}

// New ...
func New(api API, config *Config, log *zap.Logger) (*StockBoard, error) {
	s := &StockBoard{
		config:      config,
		api:         api,
		log:         log,
		cancelBoard: make(chan struct{}),
		logos:       make(map[string]*logo.Logo),
	}

	svr := &Server{
		board: s,
	}
	s.rpcServer = pb.NewBasicBoardServer(svr,
		twirp.WithServerPathPrefix("/stocks"),
		twirp.ChainHooks(
			twirphelpers.GetDefaultHooks(s, s.log),
		),
	)

	if len(config.OffTimes) > 0 || len(config.OnTimes) > 0 {
		c := cron.New()
		for _, on := range config.OnTimes {
			s.log.Info("stockboard will be schedule to turn on",
				zap.String("turn on", on),
			)
			_, err := c.AddFunc(on, func() {
				s.log.Info("stockboard turning on")
				s.Enable()
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add cron for stockboard: %w", err)
			}
		}

		for _, off := range config.OffTimes {
			s.log.Info("stockboard will be schedule to turn off",
				zap.String("turn on", off),
			)
			_, err := c.AddFunc(off, func() {
				s.log.Info("stockboard turning off")
				s.Disable()
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add cron for stockboard: %w", err)
			}
		}

		c.Start()
	}

	return s, nil
}

func (s *StockBoard) cacheClear() {
	s.api.CacheClear()
}

// Enabled ...
func (s *StockBoard) Enabled() bool {
	return s.config.Enabled.Load()
}

// Enable ...
func (s *StockBoard) Enable() {
	s.config.Enabled.Store(true)
}

// InBetween ...
func (s *StockBoard) InBetween() bool {
	return false
}

// Disable ...
func (s *StockBoard) Disable() {
	s.config.Enabled.Store(false)
}

// Name ...
func (s *StockBoard) Name() string {
	return "Stocks"
}

func (s *StockBoard) enablerCancel(ctx context.Context, cancel context.CancelFunc) {
	s.enablerLock.Lock()
	defer s.enablerLock.Unlock()
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.cancelBoard:
			cancel()
			return
		case <-ticker.C:
			if !s.config.Enabled.Load() {
				cancel()
				return
			}
		}
	}
}

func (s *StockBoard) Render(ctx context.Context, canvas board.Canvas) error {
	c, err := s.render(ctx, canvas)
	if err != nil {
		return err
	}
	if c != nil {
		return c.Render(ctx)
	}

	return nil
}

// ScrollRender ...
func (s *StockBoard) ScrollRender(ctx context.Context, canvas board.Canvas, padding int) (board.Canvas, error) {
	origScrollMode := s.config.ScrollMode.Load()
	origPad := s.config.TightScrollPadding
	defer func() {
		s.config.ScrollMode.Store(origScrollMode)
		s.config.TightScrollPadding = origPad
	}()

	s.config.ScrollMode.Store(true)
	s.config.TightScrollPadding = padding

	return s.render(ctx, canvas)
}

// Render ...
func (s *StockBoard) render(ctx context.Context, canvas board.Canvas) (board.Canvas, error) {
	boardCtx, boardCancel := context.WithCancel(ctx)
	defer boardCancel()

	go s.enablerCancel(boardCtx, boardCancel)

	s.log.Debug("fetching stock info",
		zap.Strings("stocks", s.config.Symbols),
		zap.String("update interval str", s.config.updateInterval.String()),
		zap.Duration("update interval", s.config.updateInterval),
	)
	stocks, err := s.api.Get(boardCtx, s.config.Symbols, s.config.updateInterval)
	if err != nil {
		return nil, err
	}

	var scrollCanvas *rgbmatrix.ScrollCanvas
	if canvas.Scrollable() && s.config.ScrollMode.Load() {
		base, ok := canvas.(*rgbmatrix.ScrollCanvas)
		if !ok {
			return nil, fmt.Errorf("wat")
		}

		var err error
		scrollCanvas, err = rgbmatrix.NewScrollCanvas(base.Matrix, s.log)
		if err != nil {
			return nil, fmt.Errorf("failed to get tight scroll canvas: %w", err)
		}
		scrollCanvas.SetScrollDirection(rgbmatrix.RightToLeft)
	}

STOCK:
	for _, stock := range stocks {
		if err := s.renderStock(boardCtx, stock, canvas); err != nil {
			s.log.Error("failed to render stock",
				zap.Error(err),
			)
			continue STOCK
		}

		if scrollCanvas != nil && s.config.ScrollMode.Load() {
			scrollCanvas.AddCanvas(canvas)
			draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Over)
			continue STOCK
		}

		if err := canvas.Render(boardCtx); err != nil {
			s.log.Error("failed to render stock board",
				zap.Error(err),
			)
			continue STOCK
		}

		if !s.config.ScrollMode.Load() {
			select {
			case <-boardCtx.Done():
				return nil, context.Canceled
			case <-time.After(s.config.boardDelay):
			}
		}
	}

	if canvas.Scrollable() && scrollCanvas != nil {
		scrollCanvas.Merge(s.config.TightScrollPadding)
		return scrollCanvas, nil
	}

	return nil, nil
}

// GetHTTPHandlers ...
func (s *StockBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return []*board.HTTPHandler{
		{
			Path: "/stocks/enable",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Info("enabling board", zap.String("board", s.Name()))
				s.Enable()
			},
		},
		{
			Path: "/stocks/disable",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Info("disabling board", zap.String("board", s.Name()))
				select {
				case s.cancelBoard <- struct{}{}:
				default:
				}
				s.Disable()
				s.cacheClear()
			},
		},
		{
			Path: "/stocks/status",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Debug("get board status", zap.String("board", s.Name()))
				w.Header().Set("Content-Type", "text/plain")
				if s.Enabled() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
		{
			Path: "/stocks/scrollon",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				select {
				case s.cancelBoard <- struct{}{}:
				default:
				}
				s.config.ScrollMode.Store(true)
				s.cacheClear()
			},
		},
		{
			Path: "/stocks/scrolloff",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.config.ScrollMode.Store(false)
				select {
				case s.cancelBoard <- struct{}{}:
				default:
				}
				s.cacheClear()
			},
		},
		{
			Path: "/stocks/scrollstatus",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Debug("get board scroll status", zap.String("board", s.Name()))
				w.Header().Set("Content-Type", "text/plain")
				if s.config.ScrollMode.Load() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
		{
			Path: "/stocks/clearcache",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				select {
				case s.cancelBoard <- struct{}{}:
				default:
				}
				s.cacheClear()
			},
		},
	}, nil
}

// ScrollMode ...
func (s *StockBoard) ScrollMode() bool {
	return s.config.ScrollMode.Load()
}
