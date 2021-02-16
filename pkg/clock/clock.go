package clock

import (
	"context"
	"fmt"
	"image/color"
	"net/http"
	"time"

	"github.com/golang/freetype/truetype"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

// Board implements board.Board
type Clock struct {
	config     *Config
	font       *truetype.Font
	textWriter *rgbrender.TextWriter
	log        *zap.Logger
}

type Config struct {
	boardDelay time.Duration
	Enabled    *atomic.Bool `json:"enabled"`
	BoardDelay string       `json:"boardDelay"`
}

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

func New(config *Config, logger *zap.Logger) (*Clock, error) {
	if config == nil {
		config = &Config{
			Enabled: atomic.NewBool(true),
		}
	}
	c := &Clock{
		config: config,
		log:    logger,
	}

	var err error
	c.font, err = rgbrender.GetFont("04B_03__.ttf")
	if err != nil {
		return nil, err
	}

	c.textWriter = rgbrender.NewTextWriter(c.font, 16)

	return c, nil
}

func (c *Clock) Name() string {
	return "Clock"
}

func (c *Clock) Enabled() bool {
	return c.config.Enabled.Load()
}

func (c *Clock) Cleanup() {}

func (c *Clock) Render(ctx context.Context, canvas board.Canvas) error {
	if !c.config.Enabled.Load() {
		return nil
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

			c.textWriter.WriteCentered(
				canvas,
				canvas.Bounds(),
				[]string{
					fmt.Sprintf("%d:%s%d%s", h, z, m, ampm),
				},
				color.White,
			)

			if err := canvas.Render(); err != nil {
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

func (c *Clock) HasPriority() bool {
	return false
}

func (c *Clock) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	disable := &board.HTTPHandler{
		Path: "/clock/disable",
		Handler: func(http.ResponseWriter, *http.Request) {
			c.log.Info("disabling clock board")
			c.config.Enabled.Store(false)
		},
	}
	enable := &board.HTTPHandler{
		Path: "/clock/enable",
		Handler: func(http.ResponseWriter, *http.Request) {
			c.log.Info("enabling clock board")
			c.config.Enabled.Store(true)
		},
	}

	return []*board.HTTPHandler{
		disable,
		enable,
	}, nil
}
