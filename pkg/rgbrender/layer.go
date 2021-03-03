package rgbrender

import (
	"context"
	"fmt"
	"image"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
)

const (
	// BackgroundPriority sets a layer's priority to the rearmost layer
	BackgroundPriority = 0

	// ForegroundPriority sets a layer's priority to the frontmost layer
	ForegroundPriority = -1
)

type (
	// Prepare is a func type for preparing a Layer for drawing
	Prepare func(ctx context.Context) (image.Image, error)

	// TextPrepare is a func type for preparing a TextLayer for drawing
	TextPrepare func(ctx context.Context) (*TextWriter, []string, error)

	// Draw is a func type that draws a Layer
	Draw func(canvas board.Canvas, img image.Image) error

	// TextDraw is a func type that draws a TextLayer
	Write func(canvas board.Canvas, writer *TextWriter, text []string) error
)

// LayerDrawer draws layers on a board.Canvas. It prepares layers simultaneously, then
// draws each priority simultaneously.
type LayerDrawer struct {
	drawTimeout     time.Duration
	layerPriorities map[int]struct{}
	layers          []*Layer
	textLayers      []*TextLayer
	maxLayer        int
	log             *zap.Logger
	prepared        bool
}

// Layer is a layer that draws an image.Image onto a board.Canvas
type Layer struct {
	priority int
	prepare  Prepare
	draw     Draw
	prepared image.Image
}

// TextLayer writes text onto a board.Canvas
type TextLayer struct {
	priority     int
	prepare      TextPrepare
	write        Write
	prepared     *TextWriter
	preparedText []string
}

// NewLayerDrawer ...
func NewLayerDrawer(timeout time.Duration, log *zap.Logger) (*LayerDrawer, error) {
	if log == nil {
		var err error
		log, err = zap.NewProduction()
		if err != nil {
			return nil, err
		}
	}
	return &LayerDrawer{
		drawTimeout:     timeout,
		layerPriorities: make(map[int]struct{}),
		log:             log,
	}, nil
}

// NewLayer ...
func NewLayer(prepare Prepare, draw Draw) *Layer {
	return &Layer{
		draw:    draw,
		prepare: prepare,
	}
}

// NewTextLayer ...
func NewTextLayer(prepare TextPrepare, write Write) *TextLayer {
	return &TextLayer{
		write:   write,
		prepare: prepare,
	}
}

// AddLayer ...
func (l *LayerDrawer) AddLayer(priority int, layer *Layer) {
	layer.priority = priority
	l.layerPriorities[layer.priority] = struct{}{}
	l.layers = append(l.layers, layer)
}

// AddTextLayer ...
func (l *LayerDrawer) AddTextLayer(priority int, layer *TextLayer) {
	layer.priority = priority
	l.layerPriorities[layer.priority] = struct{}{}
	l.textLayers = append(l.textLayers, layer)
}

// ClearLayers ...
func (l *LayerDrawer) ClearLayers() {
	l.layers = []*Layer{}
	l.textLayers = []*TextLayer{}
	l.layerPriorities = make(map[int]struct{})
	l.prepared = false
}

func (l *LayerDrawer) setForegroundPriority() {
	hasForeground := false
	max := BackgroundPriority
	for i := range l.layerPriorities {
		if i > max {
			max = i
		}
		if i == ForegroundPriority {
			hasForeground = true
			delete(l.layerPriorities, i)
		}
	}

	l.maxLayer = max

	if !hasForeground {
		return
	}

	l.maxLayer = max + 1

	l.layerPriorities[l.maxLayer] = struct{}{}

	for _, layer := range l.layers {
		if layer.priority == ForegroundPriority {
			layer.priority = l.maxLayer
		}
	}
	for _, layer := range l.textLayers {
		if layer.priority == ForegroundPriority {
			layer.priority = l.maxLayer
		}
	}
}

// priorities returns a sorted list of priorities, with the foreground
// priority calculated
func (l *LayerDrawer) priorities() []int {
	l.setForegroundPriority()
	p := []int{}
	for i := range l.layerPriorities {
		p = append(p, i)
	}

	sort.Ints(p)

	return p
}

// Prepare runs the prepare func of each layer concurrently
func (l *LayerDrawer) Prepare(ctx context.Context) error {
	prepareWg := sync.WaitGroup{}
	prepErrs := make(chan error, len(l.layers)+len(l.textLayers))

	for _, layer := range l.layers {
		if layer.prepare == nil {
			continue
		}
		prepareWg.Add(1)
		go func(layer *Layer) {
			defer prepareWg.Done()
			var err error
			layer.prepared, err = layer.prepare(ctx)
			if err != nil {
				prepErrs <- err
			}
		}(layer)
	}
	for _, layer := range l.textLayers {
		if layer.prepare == nil {
			continue
		}
		prepareWg.Add(1)
		go func(layer *TextLayer) {
			defer prepareWg.Done()
			var err error
			layer.prepared, layer.preparedText, err = layer.prepare(ctx)
			if err != nil {
				prepErrs <- err
			}
		}(layer)
	}

	prepDone := make(chan struct{})

	go func() {
		defer close(prepDone)
		prepareWg.Wait()
	}()

	select {
	case <-ctx.Done():
		return context.Canceled
	case <-prepDone:
	case <-time.After(l.drawTimeout):
		return fmt.Errorf("timed out LayerDrawer")
	}

ERR:
	for {
		select {
		case err := <-prepErrs:
			if err != nil {
				return err
			}
			continue ERR
		default:
			break ERR
		}
	}

	l.prepared = true

	return nil
}

// Draw draws each layer. It does each priority level concurrently
func (l *LayerDrawer) Draw(ctx context.Context, canvas board.Canvas) error {
	if !l.prepared {
		if err := l.Prepare(ctx); err != nil {
			return fmt.Errorf("failed to prepare layers before drawing: %w", err)
		}
	}

	errs := make(chan error, len(l.layers)+len(l.textLayers))

	for priority := range l.priorities() {
		wg := &sync.WaitGroup{}
		l.log.Debug("drawing priority", zap.Int("priority", priority))
	LAYER:
		for _, layer := range l.layers {
			if layer.priority != priority {
				continue LAYER
			}
			if layer.draw == nil {
				return fmt.Errorf("draw func not defined for layer")
			}
			l.log.Debug("drawing layer",
				zap.Int("priority", priority),
			)
			wg.Add(1)
			go func(layer *Layer) {
				defer wg.Done()
				if err := layer.draw(canvas, layer.prepared); err != nil {
					errs <- err
				}
			}(layer)
		}
	TEXT:
		for _, layer := range l.textLayers {
			if layer.priority != priority {
				continue TEXT
			}
			if layer.write == nil {
				return fmt.Errorf("draw func not defined for layer")
			}
			l.log.Debug("drawing text layer", zap.Int("priority", priority))
			wg.Add(1)
			go func(layer *TextLayer) {
				defer wg.Done()
				if err := layer.write(canvas, layer.prepared, layer.preparedText); err != nil {
					errs <- err
				}
			}(layer)
		}

		drawDone := make(chan struct{})

		go func() {
			defer close(drawDone)
			wg.Wait()
		}()

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-drawDone:
		case <-time.After(l.drawTimeout):
			return fmt.Errorf("timed out LayerDrawer")
		}

	ERR:
		for {
			select {
			case err := <-errs:
				if err != nil {
					return err
				}
				continue ERR
			default:
				break ERR
			}
		}
	}

	return nil
}
