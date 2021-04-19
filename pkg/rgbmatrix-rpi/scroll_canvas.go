package rgbmatrix

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"time"

	"github.com/robbydyer/sports/pkg/board"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

const (
	rightToLeft = 1
	leftToRight = 2
	bottomToTop = 3
	topToBottom = 4
	defaultPad  = 32
)

type ScrollCanvas struct {
	w, h      int
	m         Matrix
	enabled   *atomic.Bool
	actual    *image.RGBA
	direction int
	interval  time.Duration
	log       *zap.Logger
	pad       int
}

type ScrollCanvasOption func(*ScrollCanvas) error

func NewScrollCanvas(m Matrix, logger *zap.Logger, opts ...ScrollCanvasOption) (*ScrollCanvas, error) {
	w, h := m.Geometry()
	c := &ScrollCanvas{
		w:        w,
		h:        h,
		m:        m,
		enabled:  atomic.NewBool(true),
		interval: 50 * time.Millisecond,
		log:      logger,
		pad:      defaultPad,
	}

	for _, f := range opts {
		if err := f(c); err != nil {
			return nil, err
		}
	}

	if c.actual == nil {
		c.actual = image.NewRGBA(image.Rect(0-c.pad, 0-c.pad, c.w+c.pad, c.h+c.pad))
	}

	c.log.Debug("creating scroll canvas",
		zap.Int("min X", c.Bounds().Min.X),
		zap.Int("min Y", c.Bounds().Min.Y),
		zap.Int("max X", c.Bounds().Max.X),
		zap.Int("max Y", c.Bounds().Max.Y),
	)

	return c, nil
}

func (c *ScrollCanvas) position(x, y int) int {
	return x + (y * c.w)
}

func (c *ScrollCanvas) Scrollable() bool {
	return true
}

func (c *ScrollCanvas) Name() string {
	return "RGB ScrollCanvas"
}

// Clear set all the leds on the matrix with color.Black
func (c *ScrollCanvas) Clear() error {
	draw.Draw(c.actual, c.actual.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)
	for x := 0; x < c.w-1; x++ {
		for y := 0; y < c.h-1; y++ {
			c.m.Set(c.position(x, y), color.Black)
		}
	}
	return c.m.Render()
}

// Close clears the matrix and close the matrix
func (c *ScrollCanvas) Close() error {
	c.Clear()
	return c.m.Close()
}

// Render update the display with the data from the LED buffer
func (c *ScrollCanvas) Render(ctx context.Context) error {
	if c.direction == rightToLeft {
		if err := c.rightToLeft(ctx); err != nil {
			return err
		}
	}

	draw.Draw(c.actual, c.actual.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)

	return nil
}

// ColorModel returns the canvas' color model, always color.RGBAModel
func (c *ScrollCanvas) ColorModel() color.Model {
	return color.RGBAModel
}

// Bounds return the topology of the Canvas
func (c *ScrollCanvas) Bounds() image.Rectangle {
	return c.actual.Bounds()
	//return image.Rect(0, 0, c.w, c.h)
}

// PaddedBounds ...
func (c *ScrollCanvas) PaddedBounds() image.Rectangle {
	return c.actual.Bounds()
}

// At returns the color of the pixel at (x, y)
func (c *ScrollCanvas) At(x, y int) color.Color {
	return c.actual.At(x, y)
}

// Set set LED at position x,y to the provided 24-bit color value
func (c *ScrollCanvas) Set(x, y int, color color.Color) {
	c.actual.Set(x, y, color)
}

// Enabled ...
func (c *ScrollCanvas) Enabled() bool {
	return c.enabled.Load()
}

// Enable ...
func (c *ScrollCanvas) Enable() {
	c.enabled.Store(true)
}

// Disable ...
func (c *ScrollCanvas) Disable() {
	c.enabled.Store(false)
}

// GetHTTPHandlers ...
func (c *ScrollCanvas) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return nil, nil
}

func (c *ScrollCanvas) rightToLeft(ctx context.Context) error {
	c.log.Debug("scrolling right to left",
		zap.Int("min X", c.actual.Bounds().Min.X),
		zap.Int("min Y", c.actual.Bounds().Min.Y),
		zap.Int("max X", c.actual.Bounds().Max.X),
		zap.Int("max Y", c.actual.Bounds().Max.Y),
	)
	thisX := c.w
	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}
		c.log.Debug("scrolling",
			zap.Int("thisX", thisX),
		)
		if thisX == c.w*-1 {
			return nil
		}

		for x := c.actual.Bounds().Min.X; x < c.actual.Bounds().Max.X; x++ {
			for y := c.actual.Bounds().Min.Y; y < c.actual.Bounds().Max.Y; y++ {
				shiftX := x + thisX
				if shiftX > 0 && shiftX < c.w && y > 0 && y < c.h {
					c.m.Set(c.position(shiftX, y), c.actual.At(x, y))
				}
			}
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(c.interval):
		}

		if err := c.m.Render(); err != nil {
			return err
		}
		thisX--
	}
}

// WithRightToLeft ...
func WithRightToLeft() ScrollCanvasOption {
	return func(c *ScrollCanvas) error {
		c.direction = rightToLeft
		return nil
	}
}
