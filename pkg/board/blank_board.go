package board

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type BlankBoard struct {
	log         *zap.Logger
	enabled     *atomic.Bool
	hasRendered *atomic.Bool
	tester      *testing.T
}

type Option func(b *BlankBoard) error

func NewBlankBoard(logger *zap.Logger, opts ...Option) (*BlankBoard, error) {
	b := &BlankBoard{
		log:         logger,
		enabled:     atomic.NewBool(false),
		hasRendered: atomic.NewBool(false),
	}
	for _, f := range opts {
		if err := f(b); err != nil {
			return nil, err
		}
	}

	return b, nil
}

func WithTester(t *testing.T) Option {
	return func(b *BlankBoard) error {
		b.tester = t
		return nil
	}
}

func (b *BlankBoard) Enabled() bool {
	return b.enabled.Load()
}

func (b *BlankBoard) Enable() {
	b.enabled.Store(true)
}

func (b *BlankBoard) Disable() {
	b.enabled.Store(false)
}

func (b *BlankBoard) Name() string {
	return "Blank Board"
}
func (b *BlankBoard) Render(ctx context.Context, canvases Canvas) error {
	if b.tester != nil {
		b.log.Info("rendering blank board for test")
		require.Nil(b.tester, nil, "Blank Board render in test")
		b.hasRendered.Store(true)
	}
	return nil
}
func (b *BlankBoard) GetHTTPHandlers() ([]*HTTPHandler, error) {
	return nil, nil
}

func (b *BlankBoard) HasRendered() bool {
	return b.hasRendered.Load()
}
