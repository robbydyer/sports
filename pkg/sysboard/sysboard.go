package sysboard

import (
	"context"
	"fmt"
	"image/color"
	"net/http"
	"time"

	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

type SysBoard struct {
	config     *Config
	log        *zap.Logger
	textWriter *rgbrender.TextWriter
}

type Config struct {
	boardDelay time.Duration
	Enabled    bool   `json:"enabled"`
	BoardDelay string `json:"boardDelay"`
}

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
}

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

func (s *SysBoard) Name() string {
	return "SysBoard"
}
func (s *SysBoard) Render(ctx context.Context, matrix rgb.Matrix) error {
	if !s.config.Enabled {
		return nil
	}

	canvas := rgb.NewCanvas(matrix)

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

	s.textWriter.WriteCentered(
		canvas,
		canvas.Bounds(),
		[]string{
			fmt.Sprintf("Mem: %d%%", memPct),
			fmt.Sprintf("CPU: %d%%", cpuPct),
		},
		color.White,
	)

	if err := canvas.Render(); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
	case <-time.After(s.config.boardDelay):
	}

	return nil
}
func (s *SysBoard) Enabled() bool {
	return s.config.Enabled
}
func (s *SysBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	disable := &board.HTTPHandler{
		Path: "/sys/disable",
		Handler: func(http.ResponseWriter, *http.Request) {
			s.log.Info("disabling sys board")
			s.config.Enabled = false
		},
	}
	enable := &board.HTTPHandler{
		Path: "/sys/enable",
		Handler: func(http.ResponseWriter, *http.Request) {
			s.log.Info("enabling sys board")
			s.config.Enabled = true
		},
	}

	return []*board.HTTPHandler{
		disable,
		enable,
	}, nil
}
