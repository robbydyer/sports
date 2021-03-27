package sportsmatrix

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/robbydyer/sports/pkg/board"
)

func TestSportsMatrix(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := zaptest.NewLogger(t)
	cfg := &Config{
		ServeWebUI:     false,
		HTTPListenPort: 8080,
	}
	cfg.Defaults()

	canvas := board.NewBlankCanvas(1, 1, logger)
	canvas.Enable()

	require.True(t, canvas.Enabled())

	b, err := board.NewBlankBoard(logger, board.WithTester(t))
	require.NoError(t, err)

	b.Enable()

	require.True(t, b.Enabled())

	s, err := New(ctx, logger, cfg, []board.Canvas{canvas}, b)
	require.NoError(t, err)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = s.Serve(ctx)
		require.ErrorIs(t, err, context.Canceled)
	}()

	select {
	case <-s.isServing:
	case <-time.After(10 * time.Second):
		require.NotNil(t, nil, "timed out waiting for matrix to serve")
	}

	cancel()

	done := make(chan struct{})

	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-time.After(10 * time.Second):
		require.NoError(t, fmt.Errorf("failed waiting for context to cancel"))
	case <-done:
	}

	require.True(t, b.HasRendered(), "Blank board did not render")
}
