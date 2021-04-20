package board

import (
	"context"
	"image"
	"image/color"
	"sync"

	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// BlankCanvas is a board.Canvas type that does not render
type BlankCanvas struct {
	width   int
	height  int
	pixels  []uint32
	enabled *atomic.Bool
	log     *zap.Logger
	sync.Mutex
}

// NewBlankCanvas ...
func NewBlankCanvas(width int, height int, logger *zap.Logger) *BlankCanvas {
	i := &BlankCanvas{
		width:   width,
		height:  height,
		pixels:  make([]uint32, (width * height)),
		enabled: atomic.NewBool(false),
		log:     logger,
	}

	return i
}

// Name ...
func (i *BlankCanvas) Name() string {
	return "BlankCanvas"
}

func (i *BlankCanvas) Scrollable() bool {
	return false
}

// Close ...
func (i *BlankCanvas) Close() error {
	return nil
}

// Clear sets the canvas to all black
func (i *BlankCanvas) Clear() error {
	i.blackOut()
	return i.Render(context.Background())
}

func (i *BlankCanvas) AlwaysRender() bool {
	return true
}

func (i *BlankCanvas) blackOut() {
	for x := range i.pixels {
		i.pixels[x] = colorToUint32(color.Black)
	}
}

// Render stores the state of the image as a PNG
func (i *BlankCanvas) Render(ctx context.Context) error {
	return nil
}

// ColorModel returns the canvas' color model, always color.RGBAModel
func (i *BlankCanvas) ColorModel() color.Model {
	return color.RGBAModel
}

// Bounds return the topology of the Canvas
func (i *BlankCanvas) Bounds() image.Rectangle {
	return image.Rect(0, 0, i.width, i.height)
}

func (i *BlankCanvas) PaddedBounds() image.Rectangle {
	return image.Rect(0, 0, i.width, i.height)
}

// At returns the color of the pixel at (x, y)
func (i *BlankCanvas) At(x, y int) color.Color {
	pos := i.position(x, y)
	if pos > len(i.pixels)-1 || pos < 0 {
		i.log.Debug("imgcanvas no pixel", zap.Int("x", x), zap.Int("y", y))
		return color.Black
	}
	return uint32ToColor(i.pixels[pos])
}

// Set set LED at position x,y to the provided 24-bit color value
func (i *BlankCanvas) Set(x, y int, clr color.Color) {
	pos := i.position(x, y)
	if pos > len(i.pixels)-1 || pos < 0 {
		return
	}
	i.pixels[pos] = colorToUint32(clr)
}

// Enabled ...
func (i *BlankCanvas) Enabled() bool {
	return i.enabled.Load()
}

// Enable ...
func (i *BlankCanvas) Enable() {
	i.enabled.Store(true)
}

// Disable ...
func (i *BlankCanvas) Disable() {
	i.enabled.Store(false)
}

func (i *BlankCanvas) position(x, y int) int {
	return x + (y * i.width)
}

// GetHTTPHandlers ...
func (i *BlankCanvas) GetHTTPHandlers() ([]*HTTPHandler, error) {
	return nil, nil
}

func uint32ToColor(u uint32) color.Color {
	return color.RGBA{
		uint8(u>>16) & 255,
		uint8(u>>8) & 255,
		uint8(u>>0) & 255,
		255,
	}
}

func colorToUint32(c color.Color) uint32 {
	if c == nil {
		return 0
	}

	// A color's RGBA method returns values in the range [0, 65535]
	red, green, blue, _ := c.RGBA()
	return (red>>8)<<16 | (green>>8)<<8 | blue>>8
}
