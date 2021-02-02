package clock

import (
	"context"
	"fmt"
	"image/color"
	"time"

	"github.com/golang/freetype/truetype"

	"github.com/robbydyer/sports/pkg/board"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

// Board implements board.Board
type Clock struct {
	config     *Config
	font       *truetype.Font
	textWriter *rgbrender.TextWriter
}

type Config struct {
	boardDelay time.Duration
	Enabled    bool   `json:"enabled"`
	BoardDelay string `json:"boardDelay"`
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
}

func New(config *Config) (*Clock, error) {
	if config == nil {
		config = &Config{
			Enabled: true,
		}
	}
	c := &Clock{
		config: config,
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
	return true
}

func (c *Clock) Cleanup() {}

func (c *Clock) Render(ctx context.Context, matrix rgb.Matrix) error {
	canvas := rgb.NewCanvas(matrix)

	if !c.config.Enabled {
		return nil
	}

	update := make(chan bool, 1)
	done := make(chan bool, 1)
	defer func() { done <- true }()

	var h int
	var m int
	ampm := "AM"

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
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
				update <- true
			}
			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
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
	case <-time.After(10 * time.Second):
	}

	return nil
}

func (c *Clock) HasPriority() bool {
	return false
}

func (c *Clock) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return nil, nil
}
