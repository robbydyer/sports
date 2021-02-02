package sportsmatrix

import (
	"context"
	"fmt"
	"image"
	"net/http"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	"github.com/robbydyer/sports/pkg/board"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
)

// SportsMatrix controls the RGB matrix. It rotates through a list of given board.Board
type SportsMatrix struct {
	cfg           *Config
	matrix        rgb.Matrix
	boards        []board.Board
	done          chan bool
	screenIsOn    bool
	screenOff     chan bool
	screenOn      chan bool
	log           *log.Logger
	boardCtx      context.Context
	boardCancel   context.CancelFunc
	server        http.Server
	screenLogOnce *sync.Once
	close         chan bool
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
func New(ctx context.Context, logger *log.Logger, cfg *Config, boards ...board.Board) (*SportsMatrix, error) {
	cfg.Defaults()

	s := &SportsMatrix{
		boards:     boards,
		cfg:        cfg,
		log:        logger,
		done:       make(chan bool, 1),
		screenOff:  make(chan bool, 1),
		screenOn:   make(chan bool, 1),
		close:      make(chan bool, 1),
		screenIsOn: true,
	}

	var err error

	s.log.Infof("Initializing matrix %dx%d\nBrightness:%d\nMapping:%s\n",
		s.cfg.HardwareConfig.Cols,
		s.cfg.HardwareConfig.Rows,
		s.cfg.HardwareConfig.Brightness,
		s.cfg.HardwareConfig.HardwareMapping,
	)

	for _, b := range s.boards {
		s.log.Infof("Registering board: %s", b.Name())
	}

	rt := &rgb.DefaultRuntimeOptions
	s.matrix, err = rgb.NewRGBLedMatrix(s.cfg.HardwareConfig, rt)
	if err != nil {
		return nil, err
	}

	c := cron.New()

	for _, off := range s.cfg.ScreenOffTimes {
		s.log.Infof("Screen will be scheduled to turn off at '%s'", off)
		_, err := c.AddFunc(off, func() {
			s.log.Warn("Turning screen off!")
			s.screenOff <- true
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add cron for screen off times: %w", err)
		}
	}
	for _, on := range s.cfg.ScreenOnTimes {
		s.log.Infof("Screen will be scheduled to turn on at '%s'", on)
		_, err := c.AddFunc(on, func() {
			s.log.Warn("Turning screen on!")
			s.screenOn <- true
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
				s.log.Error(err)
			case <-s.close:
				return
			}
		}
	}()

	return s, nil
}

func (s *SportsMatrix) screenWatcher(ctx context.Context, renderDone chan bool) {
	for {
		select {
		case <-ctx.Done():
			s.boardCancel()
			return
		case <-s.screenOff:
			s.Lock()
			s.log.Warn("screen turning off")
			s.screenLogOnce = &sync.Once{}
			s.screenIsOn = false
			s.boardCancel()

			// Wait for the boards to finish rendering
			// before clearing the matrix
			select {
			case <-renderDone:
			case <-time.After(30 * time.Second):
			}

			c := rgb.NewCanvas(s.matrix)
			_ = c.Clear()
			s.boardCtx, s.boardCancel = context.WithCancel(context.Background())
			s.Unlock()
		case <-s.screenOn:
			s.Lock()
			s.log.Warn("screen turning on")
			s.screenIsOn = true
			s.Unlock()
		case <-time.After(2 * time.Second):
		}
	}
}

// MatrixBounds returns an image.Rectangle of the matrix bounds
func (s *SportsMatrix) MatrixBounds() image.Rectangle {
	w, h := s.matrix.Geometry()
	return image.Rect(0, 0, w-1, h-1)
}

// Done ...
func (s *SportsMatrix) Done() chan bool {
	return s.done
}

// Serve blocks until the context is canceled
func (s *SportsMatrix) Serve(ctx context.Context) error {
	s.boardCtx, s.boardCancel = context.WithCancel(context.Background())
	defer s.boardCancel()

	renderDone := make(chan bool, 1)

	go s.screenWatcher(ctx, renderDone)

	logScreenOff := func() {
		s.log.Warn("screen is turned off")
	}

	if len(s.boards) < 1 {
		return fmt.Errorf("no boards configured")
	}

	s.log.Infof("Serving boards...")
	for {
		select {
		case <-ctx.Done():
			s.log.Info("Got context cancel")
			s.boardCancel()
			// Wait for boards to cancel
			time.Sleep(2 * time.Second)
			return nil
		default:
		}

		if !s.screenIsOn {
			time.Sleep(10 * time.Second)
			s.screenLogOnce.Do(logScreenOff)
			continue
		}
	INNER:
		for _, b := range s.boards {
			s.log.Debugf("Processing board %s", b.Name())
			if s.anyPriorities() && !b.HasPriority() {
				s.log.Warnf("skipping board %s: another has priority", b.Name())
				time.Sleep(1 * time.Second)
				continue INNER
			}
			if !b.Enabled() {
				s.log.Warnf("skipping board %s: it is disabled", b.Name())
				time.Sleep(1 * time.Second)
				continue INNER
			}

			if err := b.Render(s.boardCtx, s.matrix); err != nil {
				s.log.Error(err.Error())
			}

			renderDone <- true

			if b.HasPriority() {
				s.log.Infof("Rendering board '%s' as priority\n", b.Name())
				break INNER
			}
		}
	}
}

func (s *SportsMatrix) anyPriorities() bool {
	for _, b := range s.boards {
		if b.HasPriority() {
			return true
		}
	}

	return false
}

// Close closes the matrix
func (s *SportsMatrix) Close() {
	s.close <- true
	if s.matrix != nil {
		s.log.Warn("Sportsmatrix is shutting down- Closing matrix")
		_ = s.matrix.Close()
	}
	s.server.Close()
}
