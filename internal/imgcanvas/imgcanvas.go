package imgcanvas

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// ImgCanvas is a board.Canvas type that just stores the state
// as an image.Image
type ImgCanvas struct {
	width   int
	height  int
	pixels  []uint32
	lastPng []byte
	enabled *atomic.Bool
	log     *zap.Logger
	done    chan struct{}
	sync.Mutex
}

// New ...
func New(width int, height int, logger *zap.Logger) *ImgCanvas {
	i := &ImgCanvas{
		width:   width,
		height:  height,
		pixels:  make([]uint32, (width * height)),
		enabled: atomic.NewBool(false),
		log:     logger,
		done:    make(chan struct{}),
	}

	_ = i.Clear()

	_ = i.Render(context.Background())

	go i.disableWatcher()

	return i
}

// Name ...
func (i *ImgCanvas) Name() string {
	return "ImgCanvas"
}

// Scrollable ...
func (i *ImgCanvas) Scrollable() bool {
	return false
}

// AlwaysRender ...
func (i *ImgCanvas) AlwaysRender() bool {
	return true
}

// Close ...
func (i *ImgCanvas) Close() error {
	i.done <- struct{}{}

	return nil
}

// SetWidth ...
func (i *ImgCanvas) SetWidth(x int) {}

// GetWidth ...
func (i *ImgCanvas) GetWidth() int {
	return i.width
}

// disableWatcher checks if the canvas is disabled for 20 sec consecutively in 500ms increments.
// If so, it clears the lastPng cache to save memory.
func (i *ImgCanvas) disableWatcher() {
	ticker := time.NewTicker(500 * time.Millisecond)
	ticks := 0
	for {
		select {
		case <-i.done:
			return
		case <-ticker.C:
			ticks++
			if !i.Enabled() {
				if ticks >= 20 {
					ticks = 0
					if len(i.lastPng) > 0 {
						i.log.Warn("imgcanvas has been disabled for 20sec, clearing cache")
						i.lastPng = []byte{}
					}
				}
			} else {
				ticks = 0
			}
		}
	}
}

// Clear sets the canvas to all black
func (i *ImgCanvas) Clear() error {
	i.blackOut()
	return i.Render(context.Background())
}

func (i *ImgCanvas) blackOut() {
	for x := range i.pixels {
		i.pixels[x] = colorToUint32(color.Black)
	}
}

// Render stores the state of the image as a PNG
func (i *ImgCanvas) Render(ctx context.Context) error {
	defer i.blackOut()

	if !i.Enabled() {
		return nil
	}

	buf := &bytes.Buffer{}

	if err := png.Encode(buf, i); err != nil {
		return err
	}

	i.Lock()
	defer i.Unlock()
	i.lastPng = buf.Bytes()

	return nil
}

// ColorModel returns the canvas' color model, always color.RGBAModel
func (i *ImgCanvas) ColorModel() color.Model {
	return color.RGBAModel
}

// Bounds return the topology of the Canvas
func (i *ImgCanvas) Bounds() image.Rectangle {
	return image.Rect(0, 0, i.width, i.height)
}

// At returns the color of the pixel at (x, y)
func (i *ImgCanvas) At(x, y int) color.Color {
	pos := i.position(x, y)
	if pos > len(i.pixels)-1 || pos < 0 {
		return color.Black
	}
	return uint32ToColor(i.pixels[pos])
}

// Set set LED at position x,y to the provided 24-bit color value
func (i *ImgCanvas) Set(x, y int, clr color.Color) {
	pos := i.position(x, y)
	if pos > len(i.pixels)-1 || pos < 0 {
		return
	}
	i.pixels[pos] = colorToUint32(clr)
}

// Enabled ...
func (i *ImgCanvas) Enabled() bool {
	return i.enabled.Load()
}

// Enable ...
func (i *ImgCanvas) Enable() bool {
	return i.enabled.CAS(false, true)
}

// Disable ...
func (i *ImgCanvas) Disable() bool {
	return i.enabled.CAS(true, false)
}

func (i *ImgCanvas) position(x, y int) int {
	return x + (y * i.width)
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
