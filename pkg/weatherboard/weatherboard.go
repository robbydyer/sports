package weatherboard

import (
	"context"
	"fmt"
	"image"
	"net/http"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

// WeatherBoard displays weather
type WeatherBoard struct {
	config      *Config
	api         API
	log         *zap.Logger
	enablerLock sync.Mutex
	cancelBoard chan struct{}
	bigWriter   *rgbrender.TextWriter
	smallWriter *rgbrender.TextWriter
	sync.Mutex
}

// Config for a WeatherBoard
type Config struct {
	boardDelay         time.Duration
	updateInterval     time.Duration
	scrollDelay        time.Duration
	Enabled            *atomic.Bool `json:"enabled"`
	BoardDelay         string       `json:"boardDelay"`
	UpdateInterval     string       `json:"updateInterval"`
	ScrollMode         *atomic.Bool `json:"scrollMode"`
	TightScrollPadding int          `json:"tightScrollPadding"`
	ScrollDelay        string       `json:"scrollDelay"`
	CityID             string       `json:"cityID"`
	APIKey             string       `json:"apiKey"`
}

// Forecast ...
type Forecast struct {
	Temperature float64
	TempUnit    string
	Icon        image.Image
}

// API interface for getting stock data
type API interface {
	CurrentForecast(ctx context.Context, cityID string) (*Forecast, error)
}

// SetDefaults ...
func (c *Config) SetDefaults() {
	if c.Enabled == nil {
		c.Enabled = atomic.NewBool(false)
	}
	if c.ScrollMode == nil {
		c.ScrollMode = atomic.NewBool(false)
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
func New(api API, cfg *Config, log *zap.Logger) (*WeatherBoard, error) {
	s := &WeatherBoard{
		config:      cfg,
		api:         api,
		log:         log,
		cancelBoard: make(chan struct{}),
	}

	return s, nil
}

func (w *WeatherBoard) cacheClear() {
	//s.api.CacheClear()
}

// Enabled ...
func (w *WeatherBoard) Enabled() bool {
	return w.config.Enabled.Load()
}

// Enable ...
func (w *WeatherBoard) Enable() {
	w.config.Enabled.Store(true)
}

// Disable ...
func (w *WeatherBoard) Disable() {
	w.config.Enabled.Store(false)
}

// Name ...
func (w *WeatherBoard) Name() string {
	return "Weather"
}

func (w *WeatherBoard) enablerCancel(ctx context.Context, cancel context.CancelFunc) {
	w.enablerLock.Lock()
	defer w.enablerLock.Unlock()
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.cancelBoard:
			cancel()
			return
		case <-ticker.C:
			if !w.config.Enabled.Load() {
				cancel()
				return
			}
		}
	}
}

// Render ...
func (w *WeatherBoard) Render(ctx context.Context, canvas board.Canvas) error {
	boardCtx, boardCancel := context.WithCancel(ctx)
	defer boardCancel()

	go w.enablerCancel(boardCtx, boardCancel)

	var scrollCanvas *rgbmatrix.ScrollCanvas
	if canvas.Scrollable() && w.config.ScrollMode.Load() {
		base, ok := canvas.(*rgbmatrix.ScrollCanvas)
		if !ok {
			return fmt.Errorf("wat")
		}

		var err error
		scrollCanvas, err = rgbmatrix.NewScrollCanvas(base.Matrix, w.log)
		if err != nil {
			return fmt.Errorf("failed to get tight scroll canvas: %w", err)
		}
		scrollCanvas.SetScrollDirection(rgbmatrix.RightToLeft)
	}

	if err := w.drawForecast(boardCtx, canvas); err != nil {
		return err
	}

	if canvas.Scrollable() && scrollCanvas != nil {
		scrollCanvas.Merge(w.config.TightScrollPadding)
		return scrollCanvas.Render(boardCtx)
	}

	if err := canvas.Render(boardCtx); err != nil {
		return err
	}

	select {
	case <-boardCtx.Done():
		return context.Canceled
	case <-time.After(w.config.boardDelay):
		return nil
	}
}

// GetHTTPHandlers ...
func (w *WeatherBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return []*board.HTTPHandler{
		{
			Path: "/weather/enable",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				w.log.Info("enabling board", zap.String("board", w.Name()))
				w.Enable()
			},
		},
		{
			Path: "/weather/disable",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				w.log.Info("disabling board", zap.String("board", w.Name()))
				select {
				case w.cancelBoard <- struct{}{}:
				default:
				}
				w.Disable()
				w.cacheClear()
			},
		},
		{
			Path: "/weather/status",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				w.log.Debug("get board status", zap.String("board", w.Name()))
				wrtr.Header().Set("Content-Type", "text/plain")
				if w.Enabled() {
					_, _ = wrtr.Write([]byte("true"))
					return
				}
				_, _ = wrtr.Write([]byte("false"))
			},
		},
		{
			Path: "/weather/scrollon",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				select {
				case w.cancelBoard <- struct{}{}:
				default:
				}
				w.config.ScrollMode.Store(true)
				w.cacheClear()
			},
		},
		{
			Path: "/weather/scrolloff",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				w.config.ScrollMode.Store(false)
				select {
				case w.cancelBoard <- struct{}{}:
				default:
				}
				w.cacheClear()
			},
		},
		{
			Path: "/weather/scrollstatus",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				w.log.Debug("get board scroll status", zap.String("board", w.Name()))
				wrtr.Header().Set("Content-Type", "text/plain")
				if w.config.ScrollMode.Load() {
					_, _ = wrtr.Write([]byte("true"))
					return
				}
				_, _ = wrtr.Write([]byte("false"))
			},
		},
		{
			Path: "/weather/clearcache",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				select {
				case w.cancelBoard <- struct{}{}:
				default:
				}
				w.cacheClear()
			},
		},
	}, nil
}

// ScrollMode ...
func (w *WeatherBoard) ScrollMode() bool {
	return w.config.ScrollMode.Load()
}
