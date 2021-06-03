package rgbmatrix

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"time"

	"github.com/robbydyer/sports/pkg/board"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

const tightPad = 10

var black = color.RGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x0}

// ScrollDirection represents the direction the canvas scrolls
type ScrollDirection int

const (
	// RightToLeft ...
	RightToLeft ScrollDirection = iota
	// LeftToRight ...
	LeftToRight
	// BottomToTop ...
	BottomToTop
	// TopToBottom ...
	TopToBottom
)

type ScrollCanvas struct {
	w, h          int
	Matrix        Matrix
	enabled       *atomic.Bool
	actual        *image.RGBA
	direction     ScrollDirection
	interval      time.Duration
	log           *zap.Logger
	pad           int
	addedCanvases int
	maxAdded      int
	actuals       []*image.RGBA
}

type ScrollCanvasOption func(*ScrollCanvas) error

func NewScrollCanvas(m Matrix, logger *zap.Logger, opts ...ScrollCanvasOption) (*ScrollCanvas, error) {
	w, h := m.Geometry()
	c := &ScrollCanvas{
		w:         w,
		h:         h,
		Matrix:    m,
		enabled:   atomic.NewBool(true),
		interval:  50 * time.Millisecond,
		log:       logger,
		direction: RightToLeft,
		maxAdded:  1,
	}

	for _, f := range opts {
		if err := f(c); err != nil {
			return nil, err
		}
	}

	if c.actual == nil {
		c.SetPadding(w + int(float64(w)*0.25))
	}

	return c, nil
}

func (s *ScrollCanvas) Width() int {
	return s.w
}

func (s *ScrollCanvas) SetWidth(w int) {
	s.w = w
	s.SetPadding(s.pad)
}

func (s *ScrollCanvas) SetAddMax(max int) {
	s.maxAdded = max
}

func (s *ScrollCanvas) AddCanvas(add draw.Image) {
	if s.direction != RightToLeft && s.direction != LeftToRight {
		return
	}

	img := image.NewRGBA(add.Bounds())
	draw.Draw(img, add.Bounds(), add, add.Bounds().Min, draw.Over)

	s.actuals = append(s.actuals, img)
}

func (s *ScrollCanvas) Merge() {
	maxX := 0
	maxY := 0
	for _, img := range s.actuals {
		maxX += img.Bounds().Dx()
		maxY = img.Bounds().Dy()
	}

	merged := image.NewRGBA(image.Rect(0, 0, maxX, maxY))

	s.log.Debug("merging tight scroll canvas",
		zap.Int("width", maxX),
		zap.Int("height", maxY),
	)

	lastX := 0
	for _, img := range s.actuals {
		startX := firstNonBlankX(img)
		endX := lastNonBlankX(img)
		negStart := 0
		if startX < 0 {
			negStart = startX * -1
		}

		buffered := false
		x := 0
		for x = startX; x < endX; x++ {
			if !buffered {
				if (x+lastX+negStart)-lastX < tightPad {
					lastX += tightPad
				}
			}
			buffered = true
			for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
				merged.Set(x+lastX+negStart, y, img.At(x, y))
			}
		}
		lastX += x + negStart
	}

	s.actual = merged
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

func (c *ScrollCanvas) AlwaysRender() bool {
	return false
}

// SetScrollSpeed ...
func (c *ScrollCanvas) SetScrollSpeed(d time.Duration) {
	c.interval = d
}

// GetScrollSpeed ...
func (c *ScrollCanvas) GetScrollSpeed() time.Duration {
	return c.interval
}

// SetScrollDirection ...
func (c *ScrollCanvas) SetScrollDirection(d ScrollDirection) {
	c.direction = d
}

// GetScrollDirection ...
func (c *ScrollCanvas) GetScrollDirection() ScrollDirection {
	return c.direction
}

// SetPadding ...
func (c *ScrollCanvas) SetPadding(pad int) {
	c.pad = pad

	c.actual = image.NewRGBA(image.Rect(0-c.pad, 0-c.pad, c.w+c.pad, c.h+c.pad))
	draw.Draw(c.actual, c.actual.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Over)

	c.log.Debug("creating scroll canvas",
		zap.Int("padding", c.pad),
		zap.Int("width", c.w),
		zap.Int("height", c.h),
		zap.Int("min X", c.Bounds().Min.X),
		zap.Int("min Y", c.Bounds().Min.Y),
		zap.Int("max X", c.Bounds().Max.X),
		zap.Int("max Y", c.Bounds().Max.Y),
	)
}

// GetPadding
func (c *ScrollCanvas) GetPadding() int {
	return c.pad
}

// Clear set all the leds on the matrix with color.Black
func (c *ScrollCanvas) Clear() error {
	draw.Draw(c.actual, c.actual.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Over)
	for x := 0; x < c.w-1; x++ {
		for y := 0; y < c.h-1; y++ {
			c.Matrix.Set(c.position(x, y), color.Black)
		}
	}
	return c.Matrix.Render()
}

// Close clears the matrix and close the matrix
func (c *ScrollCanvas) Close() error {
	c.Clear()
	return c.Matrix.Close()
}

// Render update the display with the data from the LED buffer
func (c *ScrollCanvas) Render(ctx context.Context) error {
	switch c.direction {
	case RightToLeft:
		c.log.Debug("scrolling right to left")
		if err := c.rightToLeft(ctx); err != nil {
			return err
		}
	case BottomToTop:
		c.log.Debug("scrolling bottom to top")
		if err := c.bottomToTop(ctx); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported scroll direction")
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
	//thisX := c.actual.Bounds().Min.X * -1
	thisX := firstNonBlankX(c.actual)
	thisX -= c.w
	if thisX < 0 {
		thisX = thisX * -1
	}
	finish := (lastNonBlankX(c.actual) + 1) * -1
	c.log.Debug("scrolling right to left",
		zap.Int("min X", c.actual.Bounds().Min.X),
		zap.Int("min Y", c.actual.Bounds().Min.Y),
		zap.Int("max X", c.actual.Bounds().Max.X),
		zap.Int("max Y", c.actual.Bounds().Max.Y),
		zap.Int("startX", thisX),
		zap.Int("finish", finish),
	)
	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(c.interval):
		}
		c.log.Debug("scrolling",
			zap.Int("thisX", thisX),
		)
		if thisX == finish {
			return nil
		}

		for x := c.actual.Bounds().Min.X; x <= c.actual.Bounds().Max.X; x++ {
			for y := c.actual.Bounds().Min.Y; y <= c.actual.Bounds().Max.Y; y++ {
				shiftX := x + thisX
				if shiftX > 0 && shiftX < c.w && y > 0 && y < c.h {
					c.Matrix.Set(c.position(shiftX, y), c.actual.At(x, y))
				}
			}
		}

		if err := c.Matrix.Render(); err != nil {
			return err
		}
		thisX--
	}
}

func (c *ScrollCanvas) topToBottom(ctx context.Context) error {
	thisY := c.actual.Bounds().Min.Y
	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(c.interval):
		}
		c.log.Debug("scrolling",
			zap.Int("thisY", thisY),
		)
		if thisY == c.actual.Bounds().Max.Y {
			return nil
		}

		for x := c.actual.Bounds().Min.X; x <= c.actual.Bounds().Max.X; x++ {
			for y := c.actual.Bounds().Min.Y; y <= c.actual.Bounds().Max.Y; y++ {
				shiftY := y + thisY
				if shiftY > 0 && shiftY < c.h && x > 0 && x < c.w {
					c.Matrix.Set(c.position(x, shiftY), c.actual.At(x, y))
				}
			}
		}

		if err := c.Matrix.Render(); err != nil {
			return err
		}
		thisY++
	}
}

func (c *ScrollCanvas) bottomToTop(ctx context.Context) error {
	thisY := firstNonBlankY(c.actual) + c.h
	finish := (lastNonBlankY(c.actual) + 1) * -1
	c.log.Debug("scrolling until line",
		zap.Int("finish line", finish),
		zap.Int("last Y index", c.actual.Bounds().Max.Y),
	)
	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(c.interval):
		}
		c.log.Debug("scrolling",
			zap.Int("thisY", thisY),
		)
		if thisY == finish {
			return nil
		}

		for x := c.actual.Bounds().Min.X; x <= c.actual.Bounds().Max.X; x++ {
			for y := c.actual.Bounds().Min.Y; y <= c.actual.Bounds().Max.Y; y++ {
				shiftY := y + thisY
				if shiftY > 0 && shiftY < c.h && x > 0 && x < c.w {
					c.Matrix.Set(c.position(x, shiftY), c.actual.At(x, y))
				}
			}
		}

		if err := c.Matrix.Render(); err != nil {
			return err
		}
		thisY--
	}
}

func firstNonBlankY(img image.Image) int {
	for y := img.Bounds().Min.Y; y <= img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x <= img.Bounds().Max.X; x++ {
			if !isBlack(img.At(x, y)) {
				return y
			}
		}
	}

	return img.Bounds().Min.Y
}
func firstNonBlankX(img image.Image) int {
	for x := img.Bounds().Min.X; x <= img.Bounds().Max.X; x++ {
		for y := img.Bounds().Min.Y; y <= img.Bounds().Max.Y; y++ {
			if !isBlack(img.At(x, y)) {
				return x
			}
		}
	}

	return img.Bounds().Min.X
}
func lastNonBlankY(img image.Image) int {
	for y := img.Bounds().Max.Y; y >= img.Bounds().Min.Y; y-- {
		for x := img.Bounds().Min.X; x <= img.Bounds().Max.X; x++ {
			if !isBlack(img.At(x, y)) {
				return y
			}
		}
	}

	return img.Bounds().Max.Y
}
func lastNonBlankX(img image.Image) int {
	for x := img.Bounds().Max.X; x >= img.Bounds().Min.X; x-- {
		for y := img.Bounds().Max.Y; y >= img.Bounds().Min.Y; y-- {
			if !isBlack(img.At(x, y)) {
				return x
			}
		}
	}

	return img.Bounds().Max.X
}

func isBlack(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r == 0 && b == 0 && g == 0
}

// WithScrollSpeed ...
func WithScrollSpeed(d time.Duration) ScrollCanvasOption {
	return func(c *ScrollCanvas) error {
		c.interval = d
		return nil
	}
}
