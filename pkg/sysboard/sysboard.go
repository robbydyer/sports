package sysboard

import (
	"context"
	"fmt"
	"image/color"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

const cpuTempFile = "/sys/class/thermal/thermal_zone0/temp"

// SysBoard implements board.Board. Provides System info
type SysBoard struct {
	config      *Config
	log         *zap.Logger
	textWriters map[int]*rgbrender.TextWriter
	sync.Mutex
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
	return &SysBoard{
		config:      config,
		log:         logger,
		textWriters: make(map[int]*rgbrender.TextWriter),
	}, nil
}

func (s *SysBoard) textWriter(canvasWidth int) (*rgbrender.TextWriter, error) {
	if w, ok := s.textWriters[canvasWidth]; ok {
		return w, nil
	}

	s.Lock()
	defer s.Unlock()
	var err error
	s.textWriters[canvasWidth], err = rgbrender.DefaultTextWriter()
	if err != nil {
		return nil, err
	}

	s.textWriters[canvasWidth].FontSize = 0.125 * float64(canvasWidth)

	return s.textWriters[canvasWidth], nil
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

	writer, err := s.textWriter(canvas.Bounds().Dx())
	if err != nil {
		return err
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

	cpuTemp, err := getCPUTemp()
	if err != nil {
		s.log.Error("failed to get CPU temp", zap.Error(err))
	}

	s.log.Debug("sys info",
		zap.Int("mem used", int(mem.Used)),
		zap.Int("mem total", int(mem.Total)),
		zap.Int64("mem Pct", memPct),
		zap.Int64("cpu pct", cpuPct),
		zap.Int("cpu temp", cpuTemp),
	)

	things := []string{
		fmt.Sprintf("Mem: %d%%", memPct),
		fmt.Sprintf("CPU: %d%%", cpuPct),
	}

	if cpuTemp != 0 {
		things = append(things, fmt.Sprintf("CPU Temp: %d", cpuTemp))
	}

	if err := writer.WriteAligned(
		rgbrender.CenterCenter,
		canvas,
		canvas.Bounds(),
		things,
		color.White,
	); err != nil {
		return err
	}

	if err := canvas.Render(ctx); err != nil {
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

// Enable ...
func (s *SysBoard) Enable() {
	s.config.Enabled.Store(true)
}

// Disable ...
func (s *SysBoard) Disable() {
	s.config.Enabled.Store(false)
}

// GetHTTPHandlers ...
func (s *SysBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	disable := &board.HTTPHandler{
		Path: "/sys/disable",
		Handler: func(http.ResponseWriter, *http.Request) {
			s.log.Info("disabling sys board")
			s.Disable()
		},
	}
	enable := &board.HTTPHandler{
		Path: "/sys/enable",
		Handler: func(http.ResponseWriter, *http.Request) {
			s.log.Info("enabling sys board")
			s.Enable()
		},
	}
	status := &board.HTTPHandler{
		Path: "/sys/status",
		Handler: func(w http.ResponseWriter, req *http.Request) {
			s.log.Debug("get board status", zap.String("board", s.Name()))
			w.Header().Set("Content-Type", "text/plain")
			if s.Enabled() {
				_, _ = w.Write([]byte("true"))
				return
			}
			_, _ = w.Write([]byte("false"))
		},
	}

	return []*board.HTTPHandler{
		disable,
		enable,
		status,
	}, nil
}

func getCPUTemp() (int, error) {
	d := path.Dir(cpuTempFile)
	n := path.Base(cpuTempFile)
	dat, err := fs.ReadFile(os.DirFS(d), n)
	if err != nil {
		return 0, err
	}

	t, err := strconv.Atoi(strings.TrimSpace(string(dat)))
	if err != nil {
		return 0, err
	}

	return t / 1000, nil
}
