package canvas

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/matrix"
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
	name                string
	w, h                int
	Matrix              matrix.Matrix
	enabled             *atomic.Bool
	actual              *image.RGBA
	direction           ScrollDirection
	interval            *atomic.Duration
	log                 *zap.Logger
	pad                 int
	actuals             []*image.RGBA
	merged              *atomic.Bool
	subCanvases         []*subCanvasHorizontal
	mergePad            int
	scrollStatus        chan float64
	stateChangeCallback func()
	sendScrollSpeedChan chan time.Duration
	speedLock           sync.Mutex
	matchScrollCtx      context.Context
	matchScrollCancel   context.CancelFunc
}

type subCanvasHorizontal struct {
	actualStartX  int
	actualEndX    int
	virtualStartX int
	virtualEndX   int
	img           *image.RGBA
	previous      *subCanvasHorizontal
}

type ScrollCanvasOption func(*ScrollCanvas) error

func NewScrollCanvas(m matrix.Matrix, logger *zap.Logger, opts ...ScrollCanvasOption) (*ScrollCanvas, error) {
	w, h := m.Geometry()
	c := &ScrollCanvas{
		w:         w,
		h:         h,
		Matrix:    m,
		enabled:   atomic.NewBool(true),
		interval:  atomic.NewDuration(DefaultScrollDelay),
		log:       logger,
		direction: RightToLeft,
		merged:    atomic.NewBool(false),
		pad:       w + int(float64(w)*0.25),
	}

	for _, f := range opts {
		if err := f(c); err != nil {
			return nil, err
		}
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

// GC clears out the underlying struct fields that hold image data.
// This should be called whenever a ScrollCanvas is used that is not Rendered
// at some point or after Rendering
func (c *ScrollCanvas) GC() {
	for i := range c.subCanvases {
		c.subCanvases[i] = nil
	}
	for i := range c.actuals {
		c.actuals[i] = nil
	}
	c.subCanvases = nil
	c.actuals = nil
}

func (c *ScrollCanvas) AddCanvas(add draw.Image) {
	if c.direction != RightToLeft && c.direction != LeftToRight {
		return
	}

	img := image.NewRGBA(add.Bounds())
	draw.Draw(img, add.Bounds(), add, add.Bounds().Min, draw.Over)

	c.actuals = append(c.actuals, img)
}

// Append the actual canvases of another ScrollCanvas to this one
func (c *ScrollCanvas) Append(other *ScrollCanvas) {
	c.actuals = append(c.actuals, other.actuals...)
}

// Append the actual canvases of another ScrollCanvas to this one
func (c *ScrollCanvas) AppendAndGC(other *ScrollCanvas) {
	c.actuals = append(c.actuals, other.actuals...)
	other.GC()
}

// Len returns the number of canvases
func (c *ScrollCanvas) Len() int {
	return len(c.actuals)
}

func (c *ScrollCanvas) merge(padding int) {
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
	// Even though updating interval is atomic, we still need the lock
	// to notify the channel
	c.speedLock.Lock()
	defer c.speedLock.Unlock()

	if !c.interval.CAS(c.interval.Load(), d) {
		// no change
		return
	}

	if c.sendScrollSpeedChan == nil {
		return
	}

	max := 2
	try := 0
	for {
		if try >= max {
			break
		}
		try++
		select {
		case c.sendScrollSpeedChan <- c.interval.Load():
			c.log.Info("scroll canvas sending new speed to channel",
				zap.String("name", c.name),
				zap.Duration("speed", c.interval.Load()),
			)
			return
		default:
			c.log.Info("failed to send scroll canvas sending new speed to channel",
				zap.String("name", c.name),
				zap.Duration("speed", c.interval.Load()),
			)
			// Clear the buffer
			for i := 0; i < cap(c.sendScrollSpeedChan); i++ {
				select {
				case <-c.sendScrollSpeedChan:
					c.log.Info("cleared canvas speed channel buffer",
						zap.String("name", c.name),
						zap.Int("index", i),
					)
				default:
				}
			}
		}
	}
}

// GetScrollSpeed ...
func (c *ScrollCanvas) GetScrollSpeed() time.Duration {
	return c.interval.Load()
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

	c.actual = image.NewRGBA(c.getBounds())
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

func (c *ScrollCanvas) getBounds() image.Rectangle {
	return image.Rect(0-c.pad, 0-c.pad, c.w+c.pad, c.h+c.pad)
}

// GetPadding
func (c *ScrollCanvas) GetPadding() int {
	return c.pad
}

// Clear set all the leds on the matrix with color.Black
func (c *ScrollCanvas) Clear() error {
	if c.actual == nil {
		c.SetPadding(c.pad)
	}
	draw.Draw(c.actual, c.actual.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Over)
	for x := 0; x < c.w-1; x++ {
		for y := 0; y < c.h-1; y++ {
			c.Matrix.Set(x, y, color.Black)
		}
	}
	return c.Matrix.Render()
}

// Close clears the matrix and close the matrix
func (c *ScrollCanvas) Close() error {
	_ = c.Clear()
	return c.Matrix.Close()
}

// Render update the display with the data from the LED buffer
func (c *ScrollCanvas) Render(ctx context.Context) error {
	defer func() {
		if c.matchScrollCancel != nil {
			c.matchScrollCancel()
			c.log.Info("scroll canvas cancel MatchScroll")
		}

		// Make sure to nil out these to ensure we don't leak memory
		c.GC()
	}()
	switch c.direction {
	case RightToLeft:
		c.log.Debug("scrolling right to left")
		if err := c.rightToLeft(ctx); err != nil {
			return err
		}
	case BottomToTop:
		c.merge(c.mergePad)
		c.log.Debug("scrolling bottom to top")
		if err := c.bottomToTop(ctx); err != nil {
			return err
		}
		draw.Draw(c.actual, c.actual.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)
	case TopToBottom:
		c.merge(c.mergePad)
		if err := c.topToBottom(ctx); err != nil {
			return err
		}
		draw.Draw(c.actual, c.actual.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)
	default:
		return fmt.Errorf("unsupported scroll direction")
	}

	return nil
}

// RenderWithStatus update the display with the data from the LED buffer
func (c *ScrollCanvas) RenderWithStatus(ctx context.Context, status chan float64) error {
	c.scrollStatus = status

	return c.Render(ctx)
}

// ColorModel returns the canvas' color model, always color.RGBAModel
func (c *ScrollCanvas) ColorModel() color.Model {
	return color.RGBAModel
}

// Bounds return the topology of the Canvas
func (c *ScrollCanvas) Bounds() image.Rectangle {
	if c.actual == nil {
		return c.getBounds()
	}
	return c.actual.Bounds()
}

// At returns the color of the pixel at (x, y)
func (c *ScrollCanvas) At(x, y int) color.Color {
	if c.actual == nil {
		c.SetPadding(c.pad)
	}
	return c.actual.At(x, y)
}

// Set set LED at position x,y to the provided 24-bit color value
func (c *ScrollCanvas) Set(x, y int, color color.Color) {
	if c.actual == nil {
		c.SetPadding(c.pad)
	}
	c.actual.Set(x, y, color)
}

// Enabled ...
func (c *ScrollCanvas) Enabled() bool {
	return c.enabled.Load()
}

// Enable ...
func (c *ScrollCanvas) Enable() bool {
	return c.enabled.CAS(false, true)
}

// Disable ...
func (c *ScrollCanvas) Disable() bool {
	return c.enabled.CAS(true, false)
}

func (c *ScrollCanvas) SetStateChangeCallback(s func()) {
	c.stateChangeCallback = s
}

func (c *ScrollCanvas) Store(s bool) bool {
	return c.enabled.CAS(!s, s)
}

// GetHTTPHandlers ...
func (c *ScrollCanvas) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return nil, nil
}

func (c *ScrollCanvas) topToBottom(ctx context.Context) error {
	thisY := c.actual.Bounds().Min.Y
	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(c.interval.Load()):
		}
		if thisY == c.actual.Bounds().Max.Y {
			return nil
		}

		for x := c.actual.Bounds().Min.X; x <= c.actual.Bounds().Max.X; x++ {
			for y := c.actual.Bounds().Min.Y; y <= c.actual.Bounds().Max.Y; y++ {
				shiftY := y + thisY
				if shiftY > 0 && shiftY < c.h && x > 0 && x < c.w {
					c.Matrix.Set(x, shiftY, c.actual.At(x, y))
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
		case <-time.After(c.interval.Load()):
		}
		if thisY == finish {
			return nil
		}

		for x := c.actual.Bounds().Min.X; x <= c.actual.Bounds().Max.X; x++ {
			for y := c.actual.Bounds().Min.Y; y <= c.actual.Bounds().Max.Y; y++ {
				shiftY := y + thisY
				if shiftY > 0 && shiftY < c.h && x > 0 && x < c.w {
					c.Matrix.Set(x, shiftY, c.actual.At(x, y))
				}
			}
		}

		if err := c.Matrix.Render(); err != nil {
			return err
		}
		thisY--
	}
}

// getActualPixel returns the pixel color at virtual coordinates in unmerged canvas list
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
		c.actuals = append(c.actuals, c.actual)
	}

	c.log.Debug("preparing sub canvases",
		zap.Int("num actuals", len(c.actuals)),
	)

	c.subCanvases = make([]*subCanvasHorizontal, (len(c.actuals)*2)+1)
	subIndex := 0

	// Add a matrix-width empty subcanvas so that we start
	// scrolling with a totally blank screen
	c.subCanvases[subIndex] = &subCanvasHorizontal{
		actualStartX:  0,
		actualEndX:    c.w,
		virtualStartX: 0,
		virtualEndX:   c.w,
		img:           image.NewRGBA(image.Rect(0, 0, c.w, c.h)),
		previous:      nil,
	}
	subIndex++

	for i, actual := range c.actuals {
		// Add the actual subcanvas
		c.subCanvases[subIndex] = &subCanvasHorizontal{
			actualStartX: firstNonBlankX(actual),
			actualEndX:   lastNonBlankX(actual),
			img:          actual,
			previous:     c.subCanvases[subIndex-1],
		}
		subIndex++

		if i != len(c.actuals)-1 {
			// Add a subcanvas for padding between
			c.subCanvases[subIndex] = &subCanvasHorizontal{
				actualStartX: 0,
				actualEndX:   c.mergePad,
				img:          image.NewRGBA(image.Rect(0, 0, c.mergePad, c.h)),
				previous:     c.subCanvases[subIndex-1],
			}
			subIndex++
		}
	}

	// Add another matrix-width empty subcanvas
	c.subCanvases[subIndex] = &subCanvasHorizontal{
		actualStartX: 0,
		actualEndX:   c.w,
		img:          image.NewRGBA(image.Rect(0, 0, c.w, c.h)),
		previous:     c.subCanvases[subIndex-1],
	}

	c.log.Debug("done initializing sub canvases",
		zap.Int("num", len(c.subCanvases)),
	)

SUBS:
	for _, sub := range c.subCanvases {
		prev := sub.previous

		if prev == nil {
			continue SUBS
		}

		sub.virtualStartX = prev.virtualEndX + 1
		diff := sub.actualEndX - sub.actualStartX
		sub.virtualEndX = sub.virtualStartX + diff

		c.log.Debug("define sub canvas",
			zap.Int("actualstartX", sub.actualStartX),
			zap.Int("min X", sub.img.Bounds().Min.X),
			zap.Int("actualendX", sub.actualEndX),
			zap.Int("max X", sub.img.Bounds().Max.X),
			zap.Int("virtualstartX", sub.virtualStartX),
			zap.Int("virtualendx", sub.virtualEndX),
			zap.Int("actual canvas Width", c.w),
			zap.Int("pad", c.pad),
		)
	}
	c.log.Debug("done defining sub canvases")
}

func (c *ScrollCanvas) rightToLeft(ctx context.Context) error {
	if len(c.subCanvases) < 1 {
		c.PrepareSubCanvases()
	}
	if len(c.subCanvases) < 1 {
		return fmt.Errorf("not enough subcanvases to merge")
	}

	finish := c.subCanvases[len(c.subCanvases)-1].virtualEndX

	virtualX := c.subCanvases[0].virtualStartX

	c.log.Debug("performing right to left scroll without canvas merge",
		zap.Int("virtualX start", virtualX),
		zap.Int("finish", finish),
		zap.Duration("interval", c.GetScrollSpeed()),
	)

	sceneIndex := 0
	wg := sync.WaitGroup{}
	for {
		if virtualX == finish {
			break
		}

		wg.Add(1)
		go func(sceneIndex int, virtualX int) {
			defer wg.Done()
			loader := make([]matrix.MatrixPoint, c.w*c.h)

			index := 0
			for x := 0; x < c.w; x++ {
				for y := 0; y < c.h; y++ {
					thisVirtualX := x + virtualX

					loader[index] = matrix.MatrixPoint{
						X:     x,
						Y:     y,
						Color: c.getActualPixel(thisVirtualX, y),
					}
					index++
				}
			}
			c.Matrix.PreLoad(&matrix.MatrixScene{
				Index:  sceneIndex,
				Points: loader,
			})
		}(sceneIndex, virtualX)
		sceneIndex++
		virtualX++
	}

	wg.Wait()

	c.log.Debug("loaded matrix scenes",
		zap.Int("num scenes", sceneIndex+1),
	)

	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}

	c.sendScrollSpeedChan = make(chan time.Duration, 1)

	return c.Matrix.Play(ctx, c.GetScrollSpeed(), c.sendScrollSpeedChan)
}

// MatchScroll will match the scroll speed of this canvas from the given one.
// It will block until the context is canceled
func (c *ScrollCanvas) MatchScroll(ctx context.Context, match *ScrollCanvas) {
	c.matchScrollCtx, c.matchScrollCancel = context.WithCancel(ctx)
	defer c.matchScrollCancel()
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-c.matchScrollCtx.Done():
			return
		case <-ticker.C:
		}
		if curr := match.GetScrollSpeed(); curr != c.GetScrollSpeed() {
			c.SetScrollSpeed(curr)
		}
	}
}

// WithScrollSpeed ...
func WithScrollSpeed(d time.Duration) ScrollCanvasOption {
	return func(c *ScrollCanvas) error {
		c.interval.Store(d)
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

// WithMergePadding ...
func WithMergePadding(pad int) ScrollCanvasOption {
	return func(c *ScrollCanvas) error {
		c.mergePad = pad
		return nil
	}
}

func WithName(name string) ScrollCanvasOption {
	return func(c *ScrollCanvas) error {
		c.name = name
		return nil
	}
}
