package rgbmatrix

import (
	"context"
	"image"
	"image/color"
	"image/draw"

	"github.com/robbydyer/sports/internal/board"
	"go.uber.org/atomic"
)

// Canvas is a image.Image representation of a WS281x matrix, it implements
// image.Image interface and can be used with draw.Draw for example
type Canvas struct {
	w, h    int
	m       Matrix
	closed  bool
	enabled *atomic.Bool
}

// NewCanvas returns a new Canvas using the given width and height and creates
// a new WS281x matrix using the given config
func NewCanvas(m Matrix) *Canvas {
	w, h := m.Geometry()
	return &Canvas{
		w:       w,
		h:       h,
		m:       m,
		enabled: atomic.NewBool(true),
	}
}

func (c *Canvas) Name() string {
	return "RGB Canvas"
}

// Scrollable ...
func (c *Canvas) Scrollable() bool {
	return false
}

func (c *Canvas) AlwaysRender() bool {
	return false
}

// Render update the display with the data from the LED buffer
func (c *Canvas) Render(ctx context.Context) error {
	return c.m.Render()
}

// ColorModel returns the canvas' color model, always color.RGBAModel
func (c *Canvas) ColorModel() color.Model {
	return color.RGBAModel
}

// Bounds return the topology of the Canvas
func (c *Canvas) Bounds() image.Rectangle {
	return image.Rect(0, 0, c.w, c.h)
}

func (c *Canvas) PaddedBounds() image.Rectangle {
	return image.Rect(0, 0, c.w, c.h)
}

// At returns the color of the pixel at (x, y)
func (c *Canvas) At(x, y int) color.Color {
	return c.m.At(c.position(x, y))
}

// Set set LED at position x,y to the provided 24-bit color value
func (c *Canvas) Set(x, y int, color color.Color) {
	c.m.Set(c.position(x, y), color)
}

// SetWidth ...
func (c *Canvas) SetWidth(x int) {
}

// GetWidth ...
func (c *Canvas) GetWidth() int {
	return c.w
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
func (c *Canvas) Enable() bool {
	return c.enabled.CAS(false, true)
}

// Disable ...
func (c *Canvas) Disable() bool {
	return c.enabled.CAS(true, false)
}

func (c *Canvas) SetStateChangeCallback(s func()) {
	return
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
