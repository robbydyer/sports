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

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
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
	sync.Mutex
}

// Config for a StockBoard
type Config struct {
	boardDelay         time.Duration
	updateInterval     time.Duration
	scrollDelay        time.Duration
	Enabled            *atomic.Bool `json:"enabled"`
	Symbols            []string     `json:"symbols"`
	ChartResolution    int          `json:"chartResolution"`
	BoardDelay         string       `json:"boardDelay"`
	UpdateInterval     string       `json:"updateInterval"`
	ScrollMode         *atomic.Bool `json:"scrollMode"`
	TightScrollPadding int          `json:"tightScrollPadding"`
	ScrollDelay        string       `json:"scrollDelay"`
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
}

// New ...
func New(api API, cfg *Config, log *zap.Logger) (*StockBoard, error) {
	s := &StockBoard{
		config:      cfg,
		api:         api,
		log:         log,
		cancelBoard: make(chan struct{}),
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

// Render ...
func (s *StockBoard) Render(ctx context.Context, canvas board.Canvas) error {
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
		return err
	}

	var scrollCanvas *rgbmatrix.ScrollCanvas
	if canvas.Scrollable() && s.config.ScrollMode.Load() {
		base, ok := canvas.(*rgbmatrix.ScrollCanvas)
		if !ok {
			return fmt.Errorf("wat")
		}

		var err error
		scrollCanvas, err = rgbmatrix.NewScrollCanvas(base.Matrix, s.log)
		if err != nil {
			return fmt.Errorf("failed to get tight scroll canvas: %w", err)
		}
		scrollCanvas.SetScrollDirection(rgbmatrix.RightToLeft)
	}

STOCK:
	for _, stock := range stocks {
		if err := s.renderStock(boardCtx, stock, canvas); err != nil {
			s.log.Error("failed to render stock",
				zap.Error(err),
			)
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
				return context.Canceled
			case <-time.After(s.config.boardDelay):
			}
		}
	}

	if canvas.Scrollable() && scrollCanvas != nil {
		scrollCanvas.Merge(s.config.TightScrollPadding)
		return scrollCanvas.Render(boardCtx)
	}

	return nil
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
