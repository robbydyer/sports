package clock

import (
	"context"
	"fmt"
	"image/color"
	"net/http"
	"sync"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/twitchtv/twirp"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	pb "github.com/robbydyer/sports/internal/proto/basicboard"
	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

// Clock implements board.Board
type Clock struct {
	config      *Config
	font        *truetype.Font
	textWriters map[int]*rgbrender.TextWriter
	log         *zap.Logger
	rpcServer   pb.TwirpServer
	sync.Mutex
}

// Config is a Clock configuration
type Config struct {
	boardDelay time.Duration
	Enabled    *atomic.Bool `json:"enabled"`
	BoardDelay string       `json:"boardDelay"`
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

	return c, nil
}

// Name ...
func (c *Clock) Name() string {
	return "Clock"
}

// Enabled ...
func (c *Clock) Enabled() bool {
	return c.config.Enabled.Load()
}

// Enable ...
func (c *Clock) Enable() {
	c.config.Enabled.Store(true)
}

// Disable ...
func (c *Clock) Disable() {
	c.config.Enabled.Store(false)
}

// Cleanup ...
func (c *Clock) Cleanup() {}

// ScrollMode ...
func (c *Clock) ScrollMode() bool {
	return false
}

// Render ...
func (c *Clock) Render(ctx context.Context, canvas board.Canvas) error {
	if !c.config.Enabled.Load() {
		return nil
	}

	writer, err := c.getWriter(rgbrender.ZeroedBounds(canvas.Bounds()).Dy())
	if err != nil {
		return err
	}

	update := make(chan struct{})
	clockCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var h int
	var m int
	ampm := "AM"

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-clockCtx.Done():
				return
			default:
			}
			prevH := h
			prevM := m
			h, m, _ = time.Now().Local().Clock()
			if h > 12 {
				h = h - 12
				ampm = "PM"
			} else {
				ampm = "AM"
			}
			if h == 0 {
				h = 12
			}
			if h != prevH || m != prevM {
				select {
				case update <- struct{}{}:
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-clockCtx.Done():
				return
			case <-update:
			}

			z := ""
			if m < 10 {
				z = "0"
			}

			if err := writer.WriteAligned(
				rgbrender.CenterCenter,
				canvas,
				canvas.Bounds(),
				[]string{
					fmt.Sprintf("%d:%s%d%s", h, z, m, ampm),
				},
				color.White,
			); err != nil {
				c.log.Error("failed to write clock", zap.Error(err))
				return
			}

			if err := canvas.Render(ctx); err != nil {
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return context.Canceled
	case <-time.After(c.config.boardDelay):
	}

	return nil
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

	return c.textWriters[canvasHeight], nil
}
