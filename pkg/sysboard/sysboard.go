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
	"github.com/robfig/cron/v3"
	"github.com/twitchtv/twirp"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	pb "github.com/robbydyer/sports/internal/proto/basicboard"
	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
	"github.com/robbydyer/sports/pkg/twirphelpers"
)

const cpuTempFile = "/sys/class/thermal/thermal_zone0/temp"

// SysBoard implements board.Board. Provides System info
type SysBoard struct {
	config      *Config
	log         *zap.Logger
	textWriters map[int]*rgbrender.TextWriter
	rpcServer   pb.TwirpServer
	sync.Mutex
}

// Config ...
type Config struct {
	boardDelay time.Duration
	Enabled    *atomic.Bool `json:"enabled"`
	BoardDelay string       `json:"boardDelay"`
	OnTimes    []string     `json:"onTimes"`
	OffTimes   []string     `json:"offTimes"`
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
	s := &SysBoard{
		config:      config,
		log:         logger,
		textWriters: make(map[int]*rgbrender.TextWriter),
	}

	svr := &Server{
		board: s,
	}
	s.rpcServer = pb.NewBasicBoardServer(svr,
		twirp.WithServerPathPrefix("/sys"),
		twirp.ChainHooks(
			twirphelpers.GetDefaultHooks(s, s.log),
		),
	)

	if len(config.OffTimes) > 0 || len(config.OnTimes) > 0 {
		c := cron.New()
		for _, on := range config.OnTimes {
			s.log.Info("sysboard will be schedule to turn on",
				zap.String("turn on", on),
			)
			_, err := c.AddFunc(on, func() {
				s.log.Info("sysboard turning on")
				s.Enable()
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add cron for sysboard: %w", err)
			}
		}

		for _, off := range config.OffTimes {
			s.log.Info("sysboard will be schedule to turn off",
				zap.String("turn on", off),
			)
			_, err := c.AddFunc(off, func() {
				s.log.Info("sysboard turning off")
				s.Disable()
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add cron for sysboard: %w", err)
			}
		}

		c.Start()
	}

	return s, nil
}

// InBetween ...
func (s *SysBoard) InBetween() bool {
	return false
}

func (s *SysBoard) textWriter(canvasHeight int) (*rgbrender.TextWriter, error) {
	if w, ok := s.textWriters[canvasHeight]; ok {
		return w, nil
	}

	s.Lock()
	defer s.Unlock()
	var err error
	s.textWriters[canvasHeight], err = rgbrender.DefaultTextWriter()
	if err != nil {
		return nil, err
	}

	if canvasHeight <= 256 {
		s.textWriters[canvasHeight].FontSize = 8.0
	} else {
		s.textWriters[canvasHeight].FontSize = 0.25 * float64(canvasHeight)
	}

	return s.textWriters[canvasHeight], nil
}

// Name ...
func (s *SysBoard) Name() string {
	return "Sys"
}

// ScrollRender ...
func (s *SysBoard) ScrollRender(ctx context.Context, canvas board.Canvas, padding int) (board.Canvas, error) {
	return nil, nil
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

// ScrollMode ...
func (s *SysBoard) ScrollMode() bool {
	return false
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
