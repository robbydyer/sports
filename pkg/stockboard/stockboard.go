package stockboard

import (
	"context"
	"image/color"
	"net/http"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

var (
	red   = color.RGBA{255, 0, 0, 255}
	green = color.RGBA{0, 255, 0, 255}
)

// StockBoard displays stocks
type StockBoard struct {
	config       *Config
	api          API
	log          *zap.Logger
	symbolWriter *rgbrender.TextWriter
	priceWriter  *rgbrender.TextWriter
	enablerLock  sync.Mutex
	sync.Mutex
}

// Config for a StockBoard
type Config struct {
	boardDelay      time.Duration
	updateInterval  time.Duration
	Enabled         *atomic.Bool `json:"enabled"`
	Symbols         []string     `json:"symbols"`
	ChartResolution int          `json:"chartResolution"`
	BoardDelay      string       `json:"boardDelay"`
	UpdateInterval  string       `json:"updateInterval"`
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
	CacheClear()
}

// SetDefaults ...
func (c *Config) SetDefaults() {
	if c.Enabled == nil {
		c.Enabled = atomic.NewBool(false)
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
}

// New ...
func New(api API, cfg *Config, log *zap.Logger) (*StockBoard, error) {
	s := &StockBoard{
		config: cfg,
		api:    api,
		log:    log,
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

	for _, stock := range stocks {
		if err := s.renderStock(boardCtx, stock, canvas); err != nil {
			s.log.Error("failed to render stock",
				zap.Error(err),
			)
		}
		select {
		case <-boardCtx.Done():
			return context.Canceled
		case <-time.After(s.config.boardDelay):
		}
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
	}, nil
}

// ScrollMode ...
func (s *StockBoard) ScrollMode() bool {
	return false
}
