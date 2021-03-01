package rgbrender

import (
	"context"
	"fmt"
	"image"
	"sort"
	"sync"
	"time"

	"github.com/robbydyer/sports/pkg/board"
	"go.uber.org/zap"
)

const BackgroundPriority = 0
const ForegroundPriority = -1

type Prepare func(ctx context.Context) (image.Image, error)
type TextPrepare func(ctx context.Context) (*TextWriter, []string, error)
type Render func(canvas board.Canvas, img image.Image) error
type TextRender func(canvas board.Canvas, writer *TextWriter, text []string) error

type LayerRenderer struct {
	renderTimeout   time.Duration
	layerPriorities map[int]struct{}
	layers          []*Layer
	textLayers      []*TextLayer
	maxLayer        int
	log             *zap.Logger
	prepared        bool
}

type Layer struct {
	priority int
	prepare  Prepare
	render   Render
	prepared image.Image
}
type TextLayer struct {
	priority     int
	prepare      TextPrepare
	render       TextRender
	prepared     *TextWriter
	preparedText []string
}

func NewLayerRenderer(timeout time.Duration, log *zap.Logger) *LayerRenderer {
	return &LayerRenderer{
		renderTimeout:   timeout,
		layerPriorities: make(map[int]struct{}),
		log:             log,
	}
}

func NewLayer(prepare Prepare, render Render) *Layer {
	return &Layer{
		render:  render,
		prepare: prepare,
	}
}

func NewTextLayer(prepare TextPrepare, render TextRender) *TextLayer {
	return &TextLayer{
		render:  render,
		prepare: prepare,
	}
}

func (l *LayerRenderer) AddLayer(priority int, layer *Layer) {
	layer.priority = priority
	l.layerPriorities[layer.priority] = struct{}{}
	l.layers = append(l.layers, layer)
}

func (l *LayerRenderer) AddTextLayer(priority int, layer *TextLayer) {
	layer.priority = priority
	l.layerPriorities[layer.priority] = struct{}{}
	l.textLayers = append(l.textLayers, layer)
}

func (l *LayerRenderer) ClearLayers() {
	l.layers = []*Layer{}
	l.layerPriorities = make(map[int]struct{})
	l.prepared = false
}

func (l *LayerRenderer) setForegroundPriority() {
	hasForeground := false
	max := 0
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

	l.layerPriorities[max+1] = struct{}{}

	for _, layer := range l.layers {
		if layer.priority == -1 {
			layer.priority = l.maxLayer
		}
	}
	for _, layer := range l.textLayers {
		if layer.priority == -1 {
			layer.priority = l.maxLayer
		}
	}
}

// priorities returns a sorted list of priorities, with the foreground
// priority calculated
func (l *LayerRenderer) priorities() []int {
	l.setForegroundPriority()
	p := []int{}
	for i := range l.layerPriorities {
		p = append(p, i)
	}

	sort.Ints(p)

	return p
}

func (l *LayerRenderer) Prepare(ctx context.Context) error {
	prepareWg := sync.WaitGroup{}
	prepErrs := make(chan error, (len(l.layers)*2)+(len(l.textLayers)*2))

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
	case <-time.After(l.renderTimeout):
		return fmt.Errorf("timed out LayerRenderer")
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

func (l *LayerRenderer) Render(ctx context.Context, canvas board.Canvas) error {
	if !l.prepared {
		if err := l.Prepare(ctx); err != nil {
			return fmt.Errorf("failed to prepare layers before rendering: %w", err)
		}
	}

	errs := make(chan error, len(l.layers)+len(l.textLayers))

	for priority := range l.priorities() {
		wg := &sync.WaitGroup{}
		l.log.Debug("rendering priority", zap.Int("priority", priority))
	LAYER:
		for _, layer := range l.layers {
			if layer.priority != priority {
				continue LAYER
			}
			if layer.render == nil {
				return fmt.Errorf("render func not defined for layer")
			}
			l.log.Debug("rendering layer",
				zap.Int("priority", priority),
			)
			wg.Add(1)
			go func(layer *Layer) {
				defer wg.Done()
				if err := layer.render(canvas, layer.prepared); err != nil {
					errs <- err
				}
			}(layer)
		}
	TEXT:
		for _, layer := range l.textLayers {
			if layer.priority != priority {
				continue TEXT
			}
			if layer.render == nil {
				return fmt.Errorf("render func not defined for layer")
			}
			l.log.Debug("rendering text layer", zap.Int("priority", priority))
			wg.Add(1)
			go func(layer *TextLayer) {
				defer wg.Done()
				if err := layer.render(canvas, layer.prepared, layer.preparedText); err != nil {
					errs <- err
				}
			}(layer)
		}

		renderDone := make(chan struct{})

		go func() {
			defer close(renderDone)
			wg.Wait()
		}()

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-renderDone:
		case <-time.After(l.renderTimeout):
			return fmt.Errorf("timed out LayerRenderer")
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

	return canvas.Render()
}
