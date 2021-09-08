package weatherboard

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
	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

// WeatherBoard displays weather
type WeatherBoard struct {
	config      *Config
	api         API
	log         *zap.Logger
	enablerLock sync.Mutex
	iconLock    sync.Mutex
	iconCache   map[string]*logo.Logo
	cancelBoard chan struct{}
	bigWriter   *rgbrender.TextWriter
	smallWriter *rgbrender.TextWriter
	sync.Mutex
}

// Config for a WeatherBoard
type Config struct {
	boardDelay         time.Duration
	scrollDelay        time.Duration
	Enabled            *atomic.Bool `json:"enabled"`
	BoardDelay         string       `json:"boardDelay"`
	ScrollMode         *atomic.Bool `json:"scrollMode"`
	TightScrollPadding int          `json:"tightScrollPadding"`
	ScrollDelay        string       `json:"scrollDelay"`
	ZipCode            string       `json:"zipCode"`
	Country            string       `json:"country"`
	APIKey             string       `json:"apiKey"`
	CurrentForecast    *atomic.Bool `json:"currentForecast"`
	HourlyForecast     *atomic.Bool `json:"hourlyForecast"`
	DailyForecast      *atomic.Bool `json:"dailyForecast"`
	DailyNumber        int          `json:"dailyNumber"`
	HourlyNumber       int          `json:"hourlyNumber"`
}

// Forecast ...
type Forecast struct {
	Time         time.Time
	Temperature  *float64
	HighTemp     *float64
	LowTemp      *float64
	Humidity     int
	TempUnit     string
	Icon         *logo.Logo
	IconCode     string
	IsHourly     bool
	PrecipChance *int
}

// API interface for getting stock data
type API interface {
	CurrentForecast(ctx context.Context, zipCode string, country string, bounds image.Rectangle) (*Forecast, error)
	DailyForecasts(ctx context.Context, zipCode string, country string, bounds image.Rectangle) ([]*Forecast, error)
	HourlyForecasts(ctx context.Context, zipCode string, country string, bounds image.Rectangle) ([]*Forecast, error)
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
	if c.CurrentForecast == nil {
		c.CurrentForecast = atomic.NewBool(false)
	}
	if c.HourlyForecast == nil {
		c.HourlyForecast = atomic.NewBool(false)
	}
	if c.DailyForecast == nil {
		c.DailyForecast = atomic.NewBool(false)
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

	if c.ScrollDelay != "" {
		d, err := time.ParseDuration(c.ScrollDelay)
		if err != nil {
			c.scrollDelay = rgbmatrix.DefaultScrollDelay
		}
		c.scrollDelay = d
	} else {
		c.scrollDelay = rgbmatrix.DefaultScrollDelay
	}

	if c.DailyNumber == 0 {
		c.DailyNumber = 3
	}

	if c.HourlyNumber == 0 {
		c.HourlyNumber = 3
	}
}

// New ...
func New(api API, cfg *Config, log *zap.Logger) (*WeatherBoard, error) {
	s := &WeatherBoard{
		config:      cfg,
		api:         api,
		log:         log,
		cancelBoard: make(chan struct{}),
		iconCache:   make(map[string]*logo.Logo),
	}

	return s, nil
}

func (w *WeatherBoard) cacheClear() {
	w.api.CacheClear()
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
	if !w.config.Enabled.Load() {
		w.log.Warn("skipping disabled board", zap.String("board", "weather"))
		return nil
	}

	boardCtx, boardCancel := context.WithCancel(ctx)
	defer boardCancel()

	go w.enablerCancel(boardCtx, boardCancel)

	var scrollCanvas *rgbmatrix.ScrollCanvas
	base, ok := canvas.(*rgbmatrix.ScrollCanvas)
	if ok && w.config.ScrollMode.Load() {
		var err error
		scrollCanvas, err = rgbmatrix.NewScrollCanvas(base.Matrix, w.log)
		if err != nil {
			return fmt.Errorf("failed to get tight scroll canvas: %w", err)
		}
		scrollCanvas.SetScrollDirection(rgbmatrix.RightToLeft)
	}

	zeroed := rgbrender.ZeroedBounds(canvas.Bounds())
	forecasts := []*Forecast{}
	if w.config.CurrentForecast.Load() {
		f, err := w.api.CurrentForecast(ctx, w.config.ZipCode, w.config.Country, zeroed)
		if err != nil {
			return err
		}
		forecasts = append(forecasts, f)
	}
	if w.config.HourlyForecast.Load() {
		fs, err := w.api.HourlyForecasts(ctx, w.config.ZipCode, w.config.Country, zeroed)
		if err != nil {
			return err
		}
		// sortForecasts(fs)
		w.log.Debug("found hourly forecasts",
			zap.Int("num", len(fs)),
			zap.Int("max show", w.config.HourlyNumber),
		)
		if len(fs) > 0 {
		HOURLY:
			for i := 0; i < w.config.HourlyNumber; i++ {
				if len(fs) <= i {
					break HOURLY
				}
				forecasts = append(forecasts, fs[i])
			}
		}
	}

	if w.config.DailyForecast.Load() {
		fs, err := w.api.DailyForecasts(ctx, w.config.ZipCode, w.config.Country, zeroed)
		if err != nil {
			return err
		}
		w.log.Debug("found daily forecasts",
			zap.Int("num", len(fs)),
			zap.Int("max show", w.config.DailyNumber),
		)

		// Drop today's forecast, as it's redundant
	TODAYCHECK:
		for i := range fs {
			if fs[i].Time.YearDay() == time.Now().Local().YearDay() {
				// delete this element
				fs = append(fs[:i], fs[i+1:]...)
				break TODAYCHECK
			}
		}
		for i := 0; i < w.config.DailyNumber; i++ {
			forecasts = append(forecasts, fs[i])
		}
	}

FORECASTS:
	for _, f := range forecasts {
		if err := w.drawForecast(boardCtx, canvas, f); err != nil {
			return err
		}
		if scrollCanvas != nil {
			scrollCanvas.AddCanvas(canvas)
			draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Over)
			continue FORECASTS
		}
		if err := w.doCanvas(boardCtx, canvas, nil); err != nil {
			return err
		}
	}

	if w.config.ScrollMode.Load() && scrollCanvas != nil {
		if err := w.doCanvas(boardCtx, canvas, scrollCanvas); err != nil {
			return err
		}
	}

	return nil
}

func (w *WeatherBoard) doCanvas(ctx context.Context, canvas board.Canvas, scrollCanvas *rgbmatrix.ScrollCanvas) error {
	if canvas.Scrollable() && scrollCanvas != nil {
		scrollCanvas.Merge(w.config.TightScrollPadding)
		return scrollCanvas.Render(ctx)
	}

	if err := canvas.Render(ctx); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
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
		{
			Path: "/weather/dailyenable",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				w.log.Info("enabling board", zap.String("board", w.Name()))
				w.config.DailyForecast.Store(true)
			},
		},
		{
			Path: "/weather/dailydisable",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				w.log.Info("disabling board", zap.String("board", w.Name()))
				select {
				case w.cancelBoard <- struct{}{}:
				default:
				}
				w.config.DailyForecast.Store(false)
			},
		},
		{
			Path: "/weather/dailystatus",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				w.log.Debug("get board status", zap.String("board", w.Name()))
				wrtr.Header().Set("Content-Type", "text/plain")
				if w.config.DailyForecast.Load() {
					_, _ = wrtr.Write([]byte("true"))
					return
				}
				_, _ = wrtr.Write([]byte("false"))
			},
		},
		{
			Path: "/weather/hourlyenable",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				w.log.Info("enabling board", zap.String("board", w.Name()))
				w.config.HourlyForecast.Store(true)
			},
		},
		{
			Path: "/weather/hourlydisable",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				w.log.Info("disabling board", zap.String("board", w.Name()))
				select {
				case w.cancelBoard <- struct{}{}:
				default:
				}
				w.config.HourlyForecast.Store(false)
			},
		},
		{
			Path: "/weather/hourlystatus",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				w.log.Debug("get board status", zap.String("board", w.Name()))
				wrtr.Header().Set("Content-Type", "text/plain")
				if w.config.HourlyForecast.Load() {
					_, _ = wrtr.Write([]byte("true"))
					return
				}
				_, _ = wrtr.Write([]byte("false"))
			},
		},
	}, nil
}

// ScrollMode ...
func (w *WeatherBoard) ScrollMode() bool {
	return w.config.ScrollMode.Load()
}

/*
func sortForecasts(f []*Forecast) {
	sort.SliceStable(f, func(i, j int) bool {
		return f[i].Time.Unix() > f[j].Time.Unix()
	})
}
*/
