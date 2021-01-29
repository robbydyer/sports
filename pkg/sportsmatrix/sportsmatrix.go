package sportsmatrix

import (
	"context"
	"fmt"
	"image"
	_ "image/png"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	"github.com/robbydyer/sports/pkg/board"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
)

type SportsMatrix struct {
	cfg         *Config
	matrix      rgb.Matrix
	boards      []board.Board
	done        chan bool
	screenIsOn  bool
	screenOff   chan bool
	screenOn    chan bool
	log         *log.Logger
	boardCtx    context.Context
	boardCancel context.CancelFunc
	sync.Mutex
}

type Config struct {
	HardwareConfig *rgb.HardwareConfig `json:"hardwareConfig"`
	ScreenOffTimes []string            `json:"screenOffTimes"`
	ScreenOnTimes  []string            `json:"screenOnTimes"`
	EnableNHL      bool                `json:"enableNHL,omitempty"`
}

func (c *Config) Defaults() {
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

func New(ctx context.Context, logger *log.Logger, cfg *Config, boards ...board.Board) (*SportsMatrix, error) {
	cfg.Defaults()

	s := &SportsMatrix{
		boards:     boards,
		cfg:        cfg,
		log:        logger,
		done:       make(chan bool, 1),
		screenOff:  make(chan bool, 1),
		screenOn:   make(chan bool, 1),
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
		c.AddFunc(off, func() {
			s.log.Warn("Turning screen off!")
			s.screenOff <- true
		})
	}
	for _, on := range s.cfg.ScreenOnTimes {
		s.log.Infof("Screen will be scheduled to turn on at '%s'", on)
		c.AddFunc(on, func() {
			s.log.Warn("Turning screen on!")
			s.screenOn <- true
		})
	}
	c.Start()

	return s, nil
}
func (s *SportsMatrix) screenWatcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.boardCancel()
			return
		case <-s.screenOff:
			s.Lock()
			s.log.Warn("screen turning off")
			s.screenIsOn = false
			s.boardCancel()
			c := rgb.NewCanvas(s.matrix)
			c.Clear()
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

func (s *SportsMatrix) Done() chan bool {
	return s.done
}

func (s *SportsMatrix) Serve(ctx context.Context) error {
	s.boardCtx, s.boardCancel = context.WithCancel(context.Background())
	defer s.boardCancel()

	go s.screenWatcher(ctx)

	var once sync.Once
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
			time.Sleep(5)
			return nil
		default:
		}

		if !s.screenIsOn {
			time.Sleep(10 * time.Second)
			once.Do(logScreenOff)
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

			if b.HasPriority() {
				s.log.Infof("Rendering board '%s' as priority\n", b.Name())
				break INNER
			}
			b.Cleanup()
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

func (s *SportsMatrix) Close() {
	if s.matrix != nil {
		s.log.Warn("Sportsmatrix is shutting down- Closing matrix")
		_ = s.matrix.Close()
	}
}
