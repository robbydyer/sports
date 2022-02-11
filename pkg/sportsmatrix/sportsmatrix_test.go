package sportsmatrix

import (
	"context"
	"fmt"
	"net/http"
	"sync"
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

func (b *TestBoard) InBetween() bool {
	return false
}

func (b *TestBoard) Disable() {
	b.enabled.Store(false)
}

func (b *TestBoard) Name() string {
	return "Blank Board"
}

func (b *TestBoard) ScrollRender(ctx context.Context, canvas board.Canvas, pad int) (board.Canvas, error) {
	return nil, nil
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

func (b *TestBoard) GetRPCHandler() (string, http.Handler) {
	return "", nil
}

func (b *TestBoard) HasRendered() bool {
	return b.hasRendered.Load()
}

func (b *TestBoard) ScrollMode() bool {
	return false
}

func (b *TestBoard) SetStateChangeNotifier(st board.StateChangeNotifier) {
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

func TestScreenSwitch(t *testing.T) {
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

	go func() {
		err := s.Serve(ctx)
		require.ErrorIs(t, err, context.Canceled)
	}()

	select {
	case <-s.isServing:
	case <-time.After(10 * time.Second):
		require.NotNil(t, nil, "timed out waiting for matrix to serve")
	}

	switchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	s.switchTestSleep = true

	err = s.ScreenOff(switchCtx)
	require.NoError(t, err)

	err = s.ScreenOn(switchCtx)
	require.NoError(t, err)

	err = s.ScreenOff(switchCtx)
	require.NoError(t, err)

	wg := sync.WaitGroup{}

	switchOnCtx, swOnCancel := context.WithTimeout(ctx, 5*time.Second)
	defer swOnCancel()

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.ScreenOn(switchOnCtx)
		}()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(20 * time.Second):
		require.NoError(t, fmt.Errorf("timed out waiting for ScreenOn calls"))
	}

	require.Equal(t, 2, s.switchedOn)
}
