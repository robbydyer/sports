package sportsmatrix

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"

	"github.com/robbydyer/sports/pkg/board"
)

type TestBoard struct {
	log         *zap.Logger
	enabled     *atomic.Bool
	hasRendered *atomic.Bool
	tester      *testing.T
}

func (b *TestBoard) Enabled() bool {
	return b.enabled.Load()
}

func (b *TestBoard) Enable() {
	b.enabled.Store(true)
}

func (b *TestBoard) Disable() {
	b.enabled.Store(false)
}

func (b *TestBoard) Name() string {
	return "Blank Board"
}

func (b *TestBoard) Render(ctx context.Context, canvases board.Canvas) error {
	defer b.log.Info("TestBoard done rendering")
	if b.tester != nil {
		b.log.Info("rendering blank board for test")
		require.Nil(b.tester, nil, "Blank Board render in test")
		b.hasRendered.Store(true)
	}
	return nil
}

func (b *TestBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return nil, nil
}

func (b *TestBoard) HasRendered() bool {
	return b.hasRendered.Load()
}

func TestSportsMatrix(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := zaptest.NewLogger(t, zaptest.Level(zapcore.ErrorLevel))
	cfg := &Config{
		ServeWebUI:     false,
		HTTPListenPort: 8080,
		WebBoardWidth:  1,
	}
	cfg.Defaults()

	canvas := board.NewBlankCanvas(1, 1, logger)
	canvas.Enable()

	require.True(t, canvas.Enabled())

	b := &TestBoard{
		log:         logger,
		enabled:     atomic.NewBool(true),
		hasRendered: atomic.NewBool(false),
		tester:      t,
	}

	require.True(t, b.Enabled())

	s, err := New(ctx, logger, cfg, []board.Canvas{canvas}, b)
	require.NoError(t, err)
	defer s.Close()

	serveDone := make(chan struct{})

	go func(ctx context.Context, t *testing.T) {
		defer close(serveDone)
		logger.Debug("starting sportsmatrix test")
		err := s.Serve(ctx)
		logger.Debug("serve returned", zap.Error(err))
		require.ErrorIs(t, err, context.Canceled)
	}(ctx, t)

	select {
	case <-s.isServing:
	case <-time.After(10 * time.Second):
		require.NotNil(t, nil, "timed out waiting for matrix to serve")
	}

	rendered := make(chan struct{})

	go func() {
		defer close(rendered)
		ticker := time.NewTicker(250 * time.Millisecond)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if b.HasRendered() {
					cancel()
					return
				}
			}
		}
	}()

	select {
	case <-rendered:
	case <-time.After(10 * time.Second):
		require.NotNil(t, nil, "timed out waiting for TestBoard to render")
	}

	select {
	case <-ctx.Done():
	default:
		require.NotNil(t, nil, "context was not canceled as expected")
	}

	select {
	case <-serveDone:
	case <-time.After(10 * time.Second):
		require.NotNil(t, nil, "timed out waiting for context to cancel")
	}
}
