package board

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// BlankBoard is useful for testing
type BlankBoard struct {
	log         *zap.Logger
	enabled     *atomic.Bool
	hasRendered *atomic.Bool
	tester      *testing.T
}

// BlankBoardOption ...
type BlankBoardOption func(b *BlankBoard) error

// NewBlankBoard ...
func NewBlankBoard(logger *zap.Logger, opts ...BlankBoardOption) (*BlankBoard, error) {
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

// WithTester sets a tester for the board
func WithTester(t *testing.T) BlankBoardOption {
	return func(b *BlankBoard) error {
		b.tester = t
		return nil
	}
}

// Enabled ...
func (b *BlankBoard) Enabled() bool {
	return b.enabled.Load()
}

// Enable ...
func (b *BlankBoard) Enable() {
	b.enabled.Store(true)
}

// Disable ...
func (b *BlankBoard) Disable() {
	b.enabled.Store(false)
}

// Name ...
func (b *BlankBoard) Name() string {
	return "Blank Board"
}

// Render ...
func (b *BlankBoard) Render(ctx context.Context, canvases Canvas) error {
	if b.tester != nil {
		b.log.Info("rendering blank board for test")
		require.Nil(b.tester, nil, "Blank Board render in test")
		b.hasRendered.Store(true)
	}
	return nil
}

// GetHTTPHandlers ...
func (b *BlankBoard) GetHTTPHandlers() ([]*HTTPHandler, error) {
	return nil, nil
}

// HasRendered ...
func (b *BlankBoard) HasRendered() bool {
	return b.hasRendered.Load()
}
