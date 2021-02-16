package sysboard

import (
	"context"
	"fmt"
	"image/color"
	"net/http"
	"time"

	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

// SysBoard implements board.Board. Provides System info
type SysBoard struct {
	config     *Config
	log        *zap.Logger
	textWriter *rgbrender.TextWriter
}

// Config ...
type Config struct {
	boardDelay time.Duration
	Enabled    *atomic.Bool `json:"enabled"`
	BoardDelay string       `json:"boardDelay"`
}

// SetDefaults ...
func (c *Config) SetDefaults() {
	if c.BoardDelay != "" {
		var err error
		c.boardDelay, err = time.ParseDuration(c.BoardDelay)
		if err != nil {
			c.boardDelay = 10 * time.Second
		}
	} else {
		c.boardDelay = 10 * time.Second
	}

	if c.Enabled == nil {
		c.Enabled = atomic.NewBool(false)
	}
}

// New ...
func New(logger *zap.Logger, config *Config) (*SysBoard, error) {
	writer, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return nil, err
	}

	return &SysBoard{
		config:     config,
		log:        logger,
		textWriter: writer,
	}, nil
}

// Name ...
func (s *SysBoard) Name() string {
	return "SysBoard"
}

// Render ...
func (s *SysBoard) Render(ctx context.Context, canvas board.Canvas) error {
	if !s.config.Enabled.Load() {
		return nil
	}

	mem, err := memory.Get()
	if err != nil {
		return err
	}

	memPct := int64(float64(mem.Used) / float64(mem.Total) * 100)

	before, err := cpu.Get()
	if err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	after, err := cpu.Get()
	if err != nil {
		return err
	}

	cpuPct := int64(float64(after.User-before.User) / float64(after.Total-before.Total) * 100)

	s.log.Debug("sys info",
		zap.Int("mem used", int(mem.Used)),
		zap.Int("mem total", int(mem.Total)),
		zap.Int64("mem Pct", memPct),
		zap.Int64("cpu pct", cpuPct),
	)

	if err := s.textWriter.WriteCentered(
		canvas,
		canvas.Bounds(),
		[]string{
			fmt.Sprintf("Mem: %d%%", memPct),
			fmt.Sprintf("CPU: %d%%", cpuPct),
		},
		color.White,
	); err != nil {
		return err
	}

	if err := canvas.Render(); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
	case <-time.After(s.config.boardDelay):
	}

	return nil
}

// Enabled ...
func (s *SysBoard) Enabled() bool {
	return s.config.Enabled.Load()
}

// GetHTTPHandlers ...
func (s *SysBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	disable := &board.HTTPHandler{
		Path: "/sys/disable",
		Handler: func(http.ResponseWriter, *http.Request) {
			s.log.Info("disabling sys board")
			s.config.Enabled.Store(false)
		},
	}
	enable := &board.HTTPHandler{
		Path: "/sys/enable",
		Handler: func(http.ResponseWriter, *http.Request) {
			s.log.Info("enabling sys board")
			s.config.Enabled.Store(true)
		},
	}

	return []*board.HTTPHandler{
		disable,
		enable,
	}, nil
}
