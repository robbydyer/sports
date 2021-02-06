package sportsmatrix

import (
	"context"
	"fmt"
	"image"
	"net/http"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
)

// SportsMatrix controls the RGB matrix. It rotates through a list of given board.Board
type SportsMatrix struct {
	cfg           *Config
	matrix        rgb.Matrix
	boards        []board.Board
	screenIsOn    bool
	screenOff     chan struct{}
	screenOn      chan struct{}
	log           *zap.Logger
	boardCtx      context.Context
	boardCancel   context.CancelFunc
	server        http.Server
	screenLogOnce *sync.Once
	close         chan struct{}
	sync.Mutex
}

// Config ...
type Config struct {
	HTTPListenPort int                 `json:"httpListenPort"`
	HardwareConfig *rgb.HardwareConfig `json:"hardwareConfig"`
	ScreenOffTimes []string            `json:"screenOffTimes"`
	ScreenOnTimes  []string            `json:"screenOnTimes"`
}

// Defaults sets some sane config defaults
func (c *Config) Defaults() {
	if c.HTTPListenPort == 0 {
		c.HTTPListenPort = 8080
	}
	if c.HardwareConfig == nil {
		c.HardwareConfig = &rgb.DefaultConfig
	}

	if c.HardwareConfig.Rows == 0 {
		c.HardwareConfig.Rows = 32
	}
	if c.HardwareConfig.Cols == 32 || c.HardwareConfig.Cols == 0 {
		// We don't support 32x32 matrix
		c.HardwareConfig.Cols = 64
	}
	// The defaults do 100, but that's too much
	if c.HardwareConfig.Brightness == 0 || c.HardwareConfig.Brightness == 100 {
		c.HardwareConfig.Brightness = 60
	}
	if c.HardwareConfig.HardwareMapping == "" {
		c.HardwareConfig.HardwareMapping = "adafruit-hat-pwm"
	}
	if c.HardwareConfig.ChainLength == 0 {
		c.HardwareConfig.ChainLength = 1
	}
	if c.HardwareConfig.Parallel == 0 {
		c.HardwareConfig.Parallel = 1
	}
	if c.HardwareConfig.PWMBits == 0 {
		c.HardwareConfig.PWMBits = 11
	}
	if c.HardwareConfig.PWMLSBNanoseconds == 0 {
		c.HardwareConfig.PWMLSBNanoseconds = 130
	}
}

// New ...
func New(ctx context.Context, logger *zap.Logger, cfg *Config, boards ...board.Board) (*SportsMatrix, error) {
	cfg.Defaults()

	s := &SportsMatrix{
		boards:     boards,
		cfg:        cfg,
		log:        logger,
		screenOff:  make(chan struct{}),
		screenOn:   make(chan struct{}),
		close:      make(chan struct{}),
		screenIsOn: true,
	}

	var err error

	s.log.Info("initializing matrix",
		zap.Int("Cols", s.cfg.HardwareConfig.Cols),
		zap.Int("Rows", s.cfg.HardwareConfig.Rows),
		zap.Int("Brightness", s.cfg.HardwareConfig.Brightness),
		zap.String("Mapping", s.cfg.HardwareConfig.HardwareMapping),
	)

	for _, b := range s.boards {
		s.log.Info("Registering board", zap.String("board", b.Name()))
	}

	rt := &rgb.DefaultRuntimeOptions
	s.matrix, err = rgb.NewRGBLedMatrix(s.cfg.HardwareConfig, rt)
	if err != nil {
		return nil, err
	}

	c := cron.New()

	for _, off := range s.cfg.ScreenOffTimes {
		s.log.Info("Screen will be scheduled to turn off", zap.String("turn off", off))
		_, err := c.AddFunc(off, func() {
			s.log.Warn("Turning screen off!")
			s.Lock()
			s.screenOff <- struct{}{}
			s.Unlock()
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add cron for screen off times: %w", err)
		}
	}
	for _, on := range s.cfg.ScreenOnTimes {
		s.log.Info("Screen will be scheduled to turn on", zap.String("turn on", on))
		_, err := c.AddFunc(on, func() {
			s.log.Warn("Turning screen on!")
			s.Lock()
			s.screenOn <- struct{}{}
			s.Unlock()
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add cron for screen on times: %w", err)
		}
	}
	c.Start()

	errChan := s.startHTTP()

	// check for startup error
	s.log.Debug("checking http server for startup error")
	select {
	case <-ctx.Done():
		return nil, context.Canceled
	case err := <-errChan:
		if err != nil {
			return nil, err
		}
	default:
	}

	go func() {
		for {
			select {
			case err := <-errChan:
				s.log.Error("http server failed", zap.Error(err))
			case <-s.close:
				return
			}
		}
	}()

	return s, nil
}

// MatrixBounds returns an image.Rectangle of the matrix bounds
func (s *SportsMatrix) MatrixBounds() image.Rectangle {
	w, h := s.matrix.Geometry()
	return image.Rect(0, 0, w-1, h-1)
}

func (s *SportsMatrix) screenWatcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.screenOff:
			s.Lock()
			if !s.screenIsOn {
				s.Unlock()
				s.log.Warn("Screen is already off")
				continue
			}
			s.log.Warn("screen turning off")
			s.screenLogOnce = &sync.Once{}
			s.screenIsOn = false
			s.boardCancel()

			_ = rgb.NewCanvas(s.matrix).Clear()
			s.boardCtx, s.boardCancel = context.WithCancel(ctx)
			s.Unlock()
		case <-s.screenOn:
			s.Lock()
			if !s.screenIsOn {
				s.log.Warn("screen turning on")
			} else {
				s.log.Warn("screen is already on")
			}
			s.screenIsOn = true
			s.Unlock()
		}
	}
}

// Serve blocks until the context is canceled
func (s *SportsMatrix) Serve(ctx context.Context) error {
	s.boardCtx, s.boardCancel = context.WithCancel(ctx)
	defer s.boardCancel()

	go s.screenWatcher(ctx)

	if len(s.boards) < 1 {
		return fmt.Errorf("no boards configured")
	}

	clearer := sync.Once{}

	for {
		select {
		case <-ctx.Done():
			s.boardCancel()
			return context.Canceled
		default:
		}

		if s.allDisabled() {
			clearer.Do(func() {
				if err := rgb.NewCanvas(s.matrix).Clear(); err != nil {
					s.log.Error("failed to clear matrix when all board were disabled", zap.Error(err))
				}
			})

			continue
		}

		clearer = sync.Once{}

		if !s.screenIsOn {
			time.Sleep(1 * time.Second)
			s.screenLogOnce.Do(func() {
				s.log.Warn("screen is turned off")
			})
			continue
		}

		s.serveLoop(s.boardCtx)
	}
}

func (s *SportsMatrix) serveLoop(ctx context.Context) {
	renderDone := make(chan struct{})

	for _, b := range s.boards {
		select {
		case <-ctx.Done():
			return
		default:
		}

		s.log.Debug("Processing board", zap.String("board", b.Name()))

		if !b.Enabled() {
			s.log.Warn("skipping disabled board", zap.String("board", b.Name()))
			continue
		}

		go func() {
			select {
			case <-ctx.Done():
				return
			case <-renderDone:
			case <-time.After(5 * time.Minute):
				s.log.Error("board rendered longer than normal", zap.String("board", b.Name()))
			}
		}()

		if err := b.Render(ctx, s.matrix); err != nil {
			s.log.Error(err.Error())
		}
		select {
		case renderDone <- struct{}{}:
		default:
		}
	}
}

// Close closes the matrix
func (s *SportsMatrix) Close() {
	s.close <- struct{}{}
	if s.matrix != nil {
		s.log.Warn("Sportsmatrix is shutting down- Closing matrix")
		_ = s.matrix.Close()
	}
	s.server.Close()
}

func (s *SportsMatrix) allDisabled() bool {
	for _, b := range s.boards {
		if b.Enabled() {
			return false
		}
	}

	return true
}
