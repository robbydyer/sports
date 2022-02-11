package rgbmatrix

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sort"
	"time"

	"github.com/robbydyer/sports/pkg/board"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

var (
	black              = color.RGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x0}
	DefaultScrollDelay = 50 * time.Millisecond
)

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
	w, h         int
	Matrix       Matrix
	enabled      *atomic.Bool
	actual       *image.RGBA
	direction    ScrollDirection
	interval     time.Duration
	log          *zap.Logger
	pad          int
	actuals      []*image.RGBA
	merged       *atomic.Bool
	subCanvases  []*subCanvasHorizontal
	mergePad     int
	scrollStatus chan float64
}

type subCanvasHorizontal struct {
	actualStartX  int
	actualEndX    int
	virtualStartX int
	virtualEndX   int
	img           *image.RGBA
	index         int
}

type ScrollCanvasOption func(*ScrollCanvas) error

func NewScrollCanvas(m Matrix, logger *zap.Logger, opts ...ScrollCanvasOption) (*ScrollCanvas, error) {
	w, h := m.Geometry()
	c := &ScrollCanvas{
		w:         w,
		h:         h,
		Matrix:    m,
		enabled:   atomic.NewBool(true),
		interval:  DefaultScrollDelay,
		log:       logger,
		direction: RightToLeft,
		merged:    atomic.NewBool(false),
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

func (c *ScrollCanvas) Width() int {
	return c.w
}

func (c *ScrollCanvas) SetWidth(w int) {
	c.w = w
	c.SetPadding(w + int(float64(w)*0.25))
}

func (c *ScrollCanvas) GetWidth() int {
	return c.w
}

func (c *ScrollCanvas) GetActual() *image.RGBA {
	return c.actual
}

func (c *ScrollCanvas) AddCanvas(add draw.Image) {
	if c.direction != RightToLeft && c.direction != LeftToRight {
		return
	}

	img := image.NewRGBA(add.Bounds())
	draw.Draw(img, add.Bounds(), add, add.Bounds().Min, draw.Over)

	c.actuals = append(c.actuals, img)
}

func (c *ScrollCanvas) Merge(padding int) {
	if c.merged.CAS(true, true) {
		return
	}

	maxX := 0
	maxY := 0
	for _, img := range c.actuals {
		maxX += img.Bounds().Dx()
		if img.Bounds().Dy() > maxY {
			maxY = img.Bounds().Dy()
		}
	}

	merged := image.NewRGBA(image.Rect(0, 0, maxX, maxY))

	c.log.Debug("merging tight scroll canvas",
		zap.Int("width", maxX),
		zap.Int("height", maxY),
	)

	lastX := 0
	for _, img := range c.actuals {
		startX := firstNonBlankX(img)
		endX := lastNonBlankX(img) + 1
		negStart := 0
		if startX < 0 {
			negStart = startX * -1
		}

		buffered := false
		x := 0
		for x = startX; x < endX; x++ {
			if !buffered {
				lastX += padding
			}
			buffered = true
			for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
				merged.Set(x+lastX+negStart, y, img.At(x, y))
			}
		}
		lastX += x + negStart
	}

	c.actual = merged
}

// Append the actual canvases of another ScrollCanvas to this one
func (c *ScrollCanvas) Append(other *ScrollCanvas) {
	for _, actual := range other.actuals {
		c.actuals = append(c.actuals, actual)
	}
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

// RenderNoMerge update the display with the data from the LED buffer
func (c *ScrollCanvas) RenderNoMerge(ctx context.Context, pad int, status chan float64) error {
	c.scrollStatus = status
	switch c.direction {
	case RightToLeft:
		c.log.Debug("scrolling right to left")
		c.mergePad = pad
		if err := c.rightToLeftNoMerge(ctx); err != nil {
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
	// thisX := c.actual.Bounds().Min.X * -1
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
		zap.String("delay", c.interval.String()),
	)
	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(c.interval):
		}
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

func (c *ScrollCanvas) rightToLeftNoMerge(ctx context.Context) error {
	c.PrepareSubCanvases()

	finish := c.subCanvases[len(c.subCanvases)-1].virtualEndX

	virtualX := c.subCanvases[0].virtualStartX

	c.log.Debug("performing right to left scroll without canvas merge",
		zap.Int("virtualX start", virtualX),
		zap.Int("finish", finish),
	)

	pctDone := float64(0)

	defer func() {
		select {
		case c.scrollStatus <- 1.0:
		default:
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(c.interval):
		}
		if virtualX == finish {
			return nil
		}

		for x := 0; x < c.w; x++ {
			for y := 0; y < c.h; y++ {
				thisVirtualX := x + virtualX

				c.Matrix.Set(c.position(x, y), c.getActualPixel(thisVirtualX, y))
			}
		}
		virtualX++

		if err := c.Matrix.Render(); err != nil {
			return err
		}

		pctDone = float64(virtualX) / float64(finish)

		if math.Floor(pctDone/0.1) == pctDone/0.1 {
			select {
			case c.scrollStatus <- pctDone:
			default:
			}
		}
	}
	return nil
}

func (c *ScrollCanvas) getActualPixel(virtualX int, virtualY int) color.Color {
	if len(c.subCanvases) < 1 {
		c.PrepareSubCanvases()
	}

	for _, sub := range c.subCanvases {
		if virtualX >= sub.virtualStartX && virtualX <= sub.virtualEndX {
			actualX := (virtualX - sub.virtualStartX) + sub.actualStartX
			return sub.img.At(actualX, virtualY)
		}
	}

	return color.Black
}

// PrepareSubCanvases
func (c *ScrollCanvas) PrepareSubCanvases() {
	if len(c.actuals) < 1 {
		return
	}
	c.subCanvases = []*subCanvasHorizontal{
		{
			index:         0,
			actualStartX:  0,
			actualEndX:    c.w,
			virtualStartX: 0,
			virtualEndX:   c.w,
			img:           image.NewRGBA(image.Rect(0, 0, c.w, c.h)),
		},
	}

	index := 1
	for _, actual := range c.actuals {
		c.subCanvases = append(c.subCanvases,
			&subCanvasHorizontal{
				actualStartX: firstNonBlankX(actual),
				actualEndX:   lastNonBlankX(actual),
				img:          actual,
				index:        index,
			},
		)
		index++
		c.subCanvases = append(c.subCanvases,
			&subCanvasHorizontal{
				actualStartX: 0,
				actualEndX:   c.mergePad,
				img:          image.NewRGBA(image.Rect(0, 0, c.mergePad, c.h)),
				index:        index,
			},
		)
		index++
	}

	c.subCanvases = append(c.subCanvases,
		&subCanvasHorizontal{
			index:        index,
			actualStartX: 0,
			actualEndX:   c.w,
			img:          image.NewRGBA(image.Rect(0, 0, c.w, c.h)),
		},
	)

	sort.SliceStable(c.subCanvases, func(i int, j int) bool {
		return c.subCanvases[i].index < c.subCanvases[j].index
	})

	c.log.Debug("done initializing sub canvases",
		zap.Int("num", len(c.subCanvases)),
	)

SUBS:
	for _, sub := range c.subCanvases {
		if sub.index == 0 {
			continue SUBS
		}

		prev := c.prevSub(sub)

		if prev == nil {
			continue SUBS
		}

		sub.virtualStartX = prev.virtualEndX + 1
		diff := sub.actualEndX - sub.actualStartX
		if sub.actualStartX < 1 {
			diff = sub.actualEndX - (sub.actualStartX * -1)
		}
		sub.virtualEndX = sub.virtualStartX + diff

		c.log.Debug("define sub canvas",
			zap.Int("index", sub.index),
			zap.Int("actualstartX", sub.actualStartX),
			zap.Int("actualendX", sub.actualEndX),
			zap.Int("virtualstartX", sub.virtualStartX),
			zap.Int("virtualendx", sub.virtualEndX),
			zap.Int("actual canvas Width", c.w),
		)
	}
	c.log.Debug("done defining sub canvases")
}

func (c *ScrollCanvas) prevSub(me *subCanvasHorizontal) *subCanvasHorizontal {
	if me.index == 0 {
		return nil
	}
	for _, sub := range c.subCanvases {
		if sub.index == me.index-1 {
			return sub
		}
	}
	return nil
}

func (c *ScrollCanvas) topToBottom(ctx context.Context) error {
	thisY := c.actual.Bounds().Min.Y
	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(c.interval):
		}
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

// WithScrollDirection ...
func WithScrollDirection(direct ScrollDirection) ScrollCanvasOption {
	return func(c *ScrollCanvas) error {
		c.SetScrollDirection(direct)
		return nil
	}
}
