package clock

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"net/http"
	"sync"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/robfig/cron/v3"
	"github.com/twitchtv/twirp"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	pb "github.com/robbydyer/sports/internal/proto/basicboard"
	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/rgbmatrix-rpi"
	"github.com/robbydyer/sports/internal/rgbrender"
)

// Name is the default board name for this Clock
var Name = "Clock"

// Clock implements board.Board
type Clock struct {
	config              *Config
	font                *truetype.Font
	textWriters         map[int]*rgbrender.TextWriter
	log                 *zap.Logger
	rpcServer           pb.TwirpServer
	stateChangeNotifier board.StateChangeNotifier
	sync.Mutex
}

// Config is a Clock configuration
type Config struct {
	boardDelay  time.Duration
	scrollDelay time.Duration
	Enabled     *atomic.Bool `json:"enabled"`
	BoardDelay  string       `json:"boardDelay"`
	OnTimes     []string     `json:"onTimes"`
	OffTimes    []string     `json:"offTimes"`
	ShowBetween *atomic.Bool `json:"showBetween"`
	ScrollMode  *atomic.Bool `json:"scrollMode"`
	ScrollDelay string       `json:"scrollDelay"`
}

// SetDefaults ...
func (c *Config) SetDefaults() {
	if c.BoardDelay != "" {
		var err error
		c.boardDelay, err = time.ParseDuration(c.BoardDelay)
		if err != nil {
			c.boardDelay = 10 * time.Second
		}
	} else {
		c.boardDelay = 10 * time.Second
	}

	if c.Enabled == nil {
		c.Enabled = atomic.NewBool(false)
	}

	if c.ShowBetween == nil {
		c.ShowBetween = atomic.NewBool(false)
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

// New returns a new Clock board
func New(config *Config, logger *zap.Logger) (*Clock, error) {
	if config == nil {
		config = &Config{
			Enabled: atomic.NewBool(true),
		}
	}
	c := &Clock{
		config:      config,
		log:         logger,
		textWriters: make(map[int]*rgbrender.TextWriter),
	}

	svr := &Server{
		board: c,
	}
	c.rpcServer = pb.NewBasicBoardServer(svr,
		twirp.WithServerPathPrefix("/clock"),
		twirp.ChainHooks(
			&twirp.ServerHooks{
				Error: func(ctx context.Context, err twirp.Error) context.Context {
					c.log.Error("twirp API error",
						zap.Error(err),
						zap.String("board", "clock"),
					)
					return ctx
				},
			},
		),
	)

	if len(config.OffTimes) > 0 || len(config.OnTimes) > 0 {
		cr := cron.New()
		for _, on := range config.OnTimes {
			c.log.Info("clock will be schedule to turn on",
				zap.String("turn on", on),
			)
			_, err := cr.AddFunc(on, func() {
				c.log.Info("clock turning on")
				c.Enable()
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add cron for clock: %w", err)
			}
		}

		for _, off := range config.OffTimes {
			c.log.Info("clock will be schedule to turn off",
				zap.String("turn on", off),
			)
			_, err := cr.AddFunc(off, func() {
				c.log.Info("clock turning off")
				c.Disable()
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add cron for clock: %w", err)
			}
		}

		cr.Start()
	}

	return c, nil
}

// InBetween ...
func (c *Clock) InBetween() bool {
	return c.config.ShowBetween.Load()
}

// Name ...
func (c *Clock) Name() string {
	return Name
}

// Enabled ...
func (c *Clock) Enabled() bool {
	return c.config.Enabled.Load()
}

// Enable ...
func (c *Clock) Enable() bool {
	if c.config.Enabled.CAS(false, true) {
		if c.stateChangeNotifier != nil {
			c.stateChangeNotifier()
		}
		return true
	}
	return false
}

// Disable ...
func (c *Clock) Disable() bool {
	if c.config.Enabled.CAS(true, false) {
		if c.stateChangeNotifier != nil {
			c.stateChangeNotifier()
		}
		return true
	}
	return false
}

// SetStateChangeNotifier ...
func (c *Clock) SetStateChangeNotifier(st board.StateChangeNotifier) {
	c.stateChangeNotifier = st
}

// Cleanup ...
func (c *Clock) Cleanup() {}

// ScrollMode ...
func (c *Clock) ScrollMode() bool {
	return c.config.ScrollMode.Load()
}

// ScrollRender ...
func (c *Clock) ScrollRender(ctx context.Context, canvas board.Canvas, padding int) (board.Canvas, error) {
	origScrollMode := c.config.ScrollMode.Load()
	defer func() {
		c.config.ScrollMode.Store(origScrollMode)
	}()

	c.config.ScrollMode.Store(true)

	return c.render(ctx, canvas)
}

// Render ...
func (c *Clock) Render(ctx context.Context, canvas board.Canvas) error {
	canv, err := c.render(ctx, canvas)
	if err != nil {
		return err
	}
	if canv != nil {
		return canv.Render(ctx)
	}

	return nil
}

func currentTimeStr() string {
	ampm := ""
	h, m, _ := time.Now().Local().Clock()
	if h >= 12 {
		h = h - 12
		ampm = "PM"
	} else {
		ampm = "AM"
	}
	if h == 0 {
		h = 12
	}
	z := ""
	if m < 10 {
		z = "0"
	}
	return fmt.Sprintf("%d:%s%d%s", h, z, m, ampm)
}

// Render ...
func (c *Clock) render(ctx context.Context, canvas board.Canvas) (board.Canvas, error) {
	if !c.config.Enabled.Load() {
		return nil, nil
	}

	writer, err := c.getWriter(rgbrender.ZeroedBounds(canvas.Bounds()).Dy())
	if err != nil {
		return nil, err
	}

	if c.config.ScrollMode.Load() && canvas.Scrollable() {
		base, ok := canvas.(*rgbmatrix.ScrollCanvas)
		if !ok {
			return nil, fmt.Errorf("unsupported scroll canvas")
		}

		scrollCanvas, err := rgbmatrix.NewScrollCanvas(base.Matrix, c.log,
			rgbmatrix.WithScrollDirection(rgbmatrix.RightToLeft),
			rgbmatrix.WithScrollSpeed(c.config.scrollDelay),
		)
		if err != nil {
			return nil, err
		}

		if err := writer.WriteAligned(
			rgbrender.CenterCenter,
			canvas,
			rgbrender.ZeroedBounds(canvas.Bounds()),
			[]string{
				currentTimeStr(),
			},
			color.White,
		); err != nil {
			return nil, err
		}
		c.log.Debug("clock time",
			zap.String("time", currentTimeStr()),
		)
		scrollCanvas.SetPadding(0)
		scrollCanvas.AddCanvas(canvas)
		scrollCanvas.Merge(0)

		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Over)

		return scrollCanvas, nil
	}

	update := make(chan struct{})

	clockCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		prevTime := ""
		thisTime := ""
		ticker := time.NewTicker(500 * time.Millisecond)
		for {
			select {
			case <-clockCtx.Done():
				return
			case <-ticker.C:
			}
			thisTime = currentTimeStr()
			if thisTime != prevTime {
				select {
				case update <- struct{}{}:
				case <-clockCtx.Done():
					return
				}
			}
			prevTime = thisTime
		}
	}()

	go func() {
		for {
			c.log.Debug("waiting for update")
			select {
			case <-clockCtx.Done():
				return
			case <-update:
			}
			c.log.Debug("done waiting for update")

			if err := writer.WriteAligned(
				rgbrender.CenterCenter,
				canvas,
				canvas.Bounds(),
				[]string{
					currentTimeStr(),
				},
				color.White,
			); err != nil {
				c.log.Error("failed to write clock", zap.Error(err))
				return
			}

			c.log.Debug("write non scroll clock",
				zap.String("time", currentTimeStr()),
			)

			if err := canvas.Render(ctx); err != nil {
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return nil, context.Canceled
	case <-time.After(c.config.boardDelay):
	}

	return nil, nil
}

// HasPriority ...
func (c *Clock) HasPriority() bool {
	return false
}

// GetHTTPHandlers ...
func (c *Clock) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	disable := &board.HTTPHandler{
		Path: "/clock/disable",
		Handler: func(http.ResponseWriter, *http.Request) {
			c.log.Info("disabling clock board")
			c.Disable()
		},
	}
	enable := &board.HTTPHandler{
		Path: "/clock/enable",
		Handler: func(http.ResponseWriter, *http.Request) {
			c.log.Info("enabling clock board")
			c.Enable()
		},
	}
	status := &board.HTTPHandler{
		Path: "/clock/status",
		Handler: func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			if c.Enabled() {
				_, _ = w.Write([]byte("true"))
				return
			}
			_, _ = w.Write([]byte("false"))
		},
	}

	return []*board.HTTPHandler{
		disable,
		enable,
		status,
	}, nil
}

func (c *Clock) getWriter(canvasHeight int) (*rgbrender.TextWriter, error) {
	if w, ok := c.textWriters[canvasHeight]; ok {
		return w, nil
	}

	if c.font == nil {
		var err error
		c.font, err = rgbrender.GetFont("04B_03__.ttf")
		if err != nil {
			return nil, err
		}
	}

	size := 0.5 * float64(canvasHeight)

	c.Lock()
	defer c.Unlock()
	c.textWriters[canvasHeight] = rgbrender.NewTextWriter(c.font, size)
	c.textWriters[canvasHeight].YStartCorrection = -3

	return c.textWriters[canvasHeight], nil
}
