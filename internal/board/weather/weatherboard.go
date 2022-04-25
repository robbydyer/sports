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

	"github.com/twitchtv/twirp"

	"github.com/robbydyer/sports/internal/board"
	cnvs "github.com/robbydyer/sports/internal/canvas"
	"github.com/robbydyer/sports/internal/enabler"
	"github.com/robbydyer/sports/internal/logo"
	pb "github.com/robbydyer/sports/internal/proto/weatherboard"
	"github.com/robbydyer/sports/internal/rgbrender"
	"github.com/robbydyer/sports/internal/twirphelpers"
	"github.com/robbydyer/sports/internal/util"
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
	rpcServer   pb.TwirpServer
	enabler     board.Enabler
	sync.Mutex
}

// Config for a WeatherBoard
type Config struct {
	boardDelay         time.Duration
	scrollDelay        time.Duration
	StartEnabled       *atomic.Bool `json:"enabled"`
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
	OnTimes            []string     `json:"onTimes"`
	OffTimes           []string     `json:"offTimes"`
	MetricUnits        *atomic.Bool `json:"metricUnits"`
	ShowBetween        *atomic.Bool `json:"showBetween"`
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

// API interface for getting weather data
type API interface {
	CurrentForecast(ctx context.Context, zipCode string, country string, bounds image.Rectangle, metricUnits bool) (*Forecast, error)
	DailyForecasts(ctx context.Context, zipCode string, country string, bounds image.Rectangle, metricUnits bool) ([]*Forecast, error)
	HourlyForecasts(ctx context.Context, zipCode string, country string, bounds image.Rectangle, metricUnits bool) ([]*Forecast, error)
	CacheClear()
}

// SetDefaults ...
func (c *Config) SetDefaults() {
	if c.StartEnabled == nil {
		c.StartEnabled = atomic.NewBool(false)
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
	if c.MetricUnits == nil {
		c.MetricUnits = atomic.NewBool(false)
	}
	if c.ShowBetween == nil {
		c.ShowBetween = atomic.NewBool(false)
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
			c.scrollDelay = cnvs.DefaultScrollDelay
		}
		c.scrollDelay = d
	} else {
		c.scrollDelay = cnvs.DefaultScrollDelay
	}

	if c.DailyNumber == 0 {
		c.DailyNumber = 3
	}

	if c.HourlyNumber == 0 {
		c.HourlyNumber = 3
	}
}

// New ...
func New(api API, config *Config, log *zap.Logger) (*WeatherBoard, error) {
	s := &WeatherBoard{
		config:      config,
		api:         api,
		log:         log,
		cancelBoard: make(chan struct{}),
		iconCache:   make(map[string]*logo.Logo),
		enabler:     enabler.New(),
	}

	if config.StartEnabled.Load() {
		s.enabler.Enable()
	}

	svr := &Server{
		board: s,
	}
	s.rpcServer = pb.NewWeatherBoardServer(svr,
		twirp.WithServerPathPrefix(""),
		twirp.ChainHooks(
			twirphelpers.GetDefaultHooks(s, s.log),
		),
	)

	if err := util.SetCrons(config.OnTimes, func() {
		s.log.Info("weatherboard turning on")
		s.Enabler().Enable()
	}); err != nil {
		return nil, err
	}
	if err := util.SetCrons(config.OffTimes, func() {
		s.log.Info("weatherboard turning off")
		s.Enabler().Disable()
	}); err != nil {
		return nil, err
	}

	return s, nil
}

func (w *WeatherBoard) cacheClear() {
	w.api.CacheClear()
}

func (w *WeatherBoard) Enabler() board.Enabler {
	return w.enabler
}

// InBetween ...
func (w *WeatherBoard) InBetween() bool {
	return w.config.ShowBetween.Load()
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
			if !w.Enabler().Enabled() {
				cancel()
				return
			}
		}
	}
}

// Render ...
func (w *WeatherBoard) Render(ctx context.Context, canvas board.Canvas) error {
	c, err := w.render(ctx, canvas)
	if err != nil {
		return err
	}
	if c != nil {
		return c.Render(ctx)
	}

	return nil
}

// ScrollRender ...
func (w *WeatherBoard) ScrollRender(ctx context.Context, canvas board.Canvas, padding int) (board.Canvas, error) {
	origScrollMode := w.config.ScrollMode.Load()
	defer func() {
		w.config.ScrollMode.Store(origScrollMode)
	}()

	w.config.ScrollMode.Store(true)

	return w.render(ctx, canvas)
}

// Render ...
func (w *WeatherBoard) render(ctx context.Context, canvas board.Canvas) (board.Canvas, error) {
	if !w.Enabler().Enabled() {
		w.log.Warn("skipping disabled board", zap.String("board", "weather"))
		return nil, nil
	}

	boardCtx, boardCancel := context.WithCancel(ctx)
	defer boardCancel()

	go w.enablerCancel(boardCtx, boardCancel)

	var scrollCanvas *cnvs.ScrollCanvas
	base, ok := canvas.(*cnvs.ScrollCanvas)
	if ok && w.config.ScrollMode.Load() {
		var err error
		scrollCanvas, err = cnvs.NewScrollCanvas(base.Matrix, w.log,
			cnvs.WithMergePadding(w.config.TightScrollPadding),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get tight scroll canvas: %w", err)
		}
		scrollCanvas.SetScrollDirection(cnvs.RightToLeft)

		go scrollCanvas.MatchScroll(ctx, base)
	}

	zeroed := rgbrender.ZeroedBounds(canvas.Bounds())
	forecasts := []*Forecast{}
	if w.config.CurrentForecast.Load() {
		f, err := w.api.CurrentForecast(ctx, w.config.ZipCode, w.config.Country, zeroed, w.config.MetricUnits.Load())
		if err != nil {
			return nil, err
		}
		forecasts = append(forecasts, f)
	}
	if w.config.HourlyForecast.Load() {
		fs, err := w.api.HourlyForecasts(ctx, w.config.ZipCode, w.config.Country, zeroed, w.config.MetricUnits.Load())
		if err != nil {
			return nil, err
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
		fs, err := w.api.DailyForecasts(ctx, w.config.ZipCode, w.config.Country, zeroed, w.config.MetricUnits.Load())
		if err != nil {
			return nil, err
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
		if len(fs) > 0 {
		DAILY:
			for i := 0; i < w.config.DailyNumber; i++ {
				if len(fs) <= i {
					break DAILY
				}
				forecasts = append(forecasts, fs[i])
			}
		}
	}

FORECASTS:
	for _, f := range forecasts {
		if err := w.drawForecast(boardCtx, canvas, f); err != nil {
			return nil, err
		}
		if scrollCanvas != nil {
			scrollCanvas.AddCanvas(canvas)
			draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Over)
			continue FORECASTS
		}
		if err := canvas.Render(ctx); err != nil {
			return nil, err
		}
		select {
		case <-boardCtx.Done():
			return nil, context.Canceled
		case <-time.After(w.config.boardDelay):
		}
	}

	if w.config.ScrollMode.Load() && scrollCanvas != nil {
		return scrollCanvas, nil
	}

	return nil, nil
}

// GetHTTPHandlers ...
func (w *WeatherBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return []*board.HTTPHandler{
		{
			Path: "/weather/enable",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				w.log.Info("enabling board", zap.String("board", w.Name()))
				w.Enabler().Enable()
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
				w.Enabler().Disable()
				w.cacheClear()
			},
		},
		{
			Path: "/weather/status",
			Handler: func(wrtr http.ResponseWriter, req *http.Request) {
				w.log.Debug("get board status", zap.String("board", w.Name()))
				wrtr.Header().Set("Content-Type", "text/plain")
				if w.Enabler().Enabled() {
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
