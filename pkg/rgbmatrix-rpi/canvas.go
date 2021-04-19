package rgbmatrix

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/robbydyer/sports/pkg/board"
	"go.uber.org/atomic"
)

// Canvas is a image.Image representation of a WS281x matrix, it implements
// image.Image interface and can be used with draw.Draw for example
type Canvas struct {
	w, h    int
	extra   draw.Image
	m       Matrix
	closed  bool
	enabled *atomic.Bool
}

type CanvasOption func(*Canvas) error

// NewCanvas returns a new Canvas using the given width and height and creates
// a new WS281x matrix using the given config
func NewCanvas(m Matrix, opts ...CanvasOption) (*Canvas, error) {
	w, h := m.Geometry()
	c := &Canvas{
		w:       w,
		h:       h,
		m:       m,
		enabled: atomic.NewBool(true),
	}

	for _, f := range opts {
		if err := f(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func WithXPadding(pad int) CanvasOption {
	return func(c *Canvas) error {
		c.extra = image.NewRGBA(image.Rect(0, 0, pad*2, c.h))
		return nil
	}
}

func (c *Canvas) Name() string {
	return "RGB Canvas"
}

// Render update the display with the data from the LED buffer
func (c *Canvas) Render() error {
	return c.m.Render()
}

// ColorModel returns the canvas' color model, always color.RGBAModel
func (c *Canvas) ColorModel() color.Model {
	return color.RGBAModel
}

// Bounds return the topology of the Canvas
func (c *Canvas) Bounds() image.Rectangle {
	pad := 0
	if c.extra != nil {
		pad = c.extra.Bounds().Dx() / 2
	}
	return image.Rect(0+pad, 0, c.w+pad, c.h)
}

// At returns the color of the pixel at (x, y)
func (c *Canvas) At(x, y int) color.Color {
	if c.extra == nil {
		return c.m.At(c.position(x, y))
	}

	if x < 0 {
		return c.extra.At(x*-1, y)
	}
	if x > c.w-1 {
		return c.extra.At(x-c.w-1, y)
	}

	return c.m.At(c.position(x, y))
}

// Set set LED at position x,y to the provided 24-bit color value
func (c *Canvas) Set(x, y int, color color.Color) {
	if c.extra == nil {
		c.m.Set(c.position(x, y), color)
		return
	}

	if x < 0 {
		c.extra.Set(x*-1, y, color)
		return
	}
	if x > c.w-1 {
		c.extra.Set(x-c.w-1, y, color)
	}

	c.m.Set(c.position(x, y), color)
}

func (c *Canvas) position(x, y int) int {
	return x + (y * c.w)
}

// Clear set all the leds on the matrix with color.Black
func (c *Canvas) Clear() error {
	draw.Draw(c, c.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)
	return c.m.Render()
}

// Close clears the matrix and close the matrix
func (c *Canvas) Close() error {
	c.Clear()
	return c.m.Close()
}

// Enabled ...
func (c *Canvas) Enabled() bool {
	return c.enabled.Load()
}

// Enable ...
func (c *Canvas) Enable() {
	c.enabled.Store(true)
}

// Disable ...
func (c *Canvas) Disable() {
	c.enabled.Store(false)
}

// GetHTTPHandlers ...
func (c *Canvas) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return nil, nil
}

// Matrix is an interface that represent any RGB matrix, very useful for testing
type Matrix interface {
	Geometry() (width, height int)
	At(position int) color.Color
	Set(position int, c color.Color)
	Apply([]color.Color) error
	Render() error
	Close() error
	SetBrightness(brightness int)
}
