package sportsmatrix

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/imgcanvas"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
)

var version = "noversion"

// SportsMatrix controls the RGB matrix. It rotates through a list of given board.Board
type SportsMatrix struct {
	cfg           *Config
	canvases      []board.Canvas
	boards        []board.Board
	screenIsOn    *atomic.Bool
	screenOff     chan struct{}
	screenOn      chan struct{}
	webBoardIsOn  *atomic.Bool
	webBoardOn    chan struct{}
	webBoardOff   chan struct{}
	serveBlock    chan struct{}
	log           *zap.Logger
	boardCtx      context.Context
	boardCancel   context.CancelFunc
	server        http.Server
	close         chan struct{}
	httpEndpoints []string
	sync.Mutex
}

// Config ...
type Config struct {
	ServeWebUI     bool                `json:"serveWebUI"`
	HTTPListenPort int                 `json:"httpListenPort"`
	HardwareConfig *rgb.HardwareConfig `json:"hardwareConfig"`
	RuntimeOptions *rgb.RuntimeOptions `json:"runtimeOptions"`
	ScreenOffTimes []string            `json:"screenOffTimes"`
	ScreenOnTimes  []string            `json:"screenOnTimes"`
	WebBoardWidth  int                 `json:"webBoardWidth"`
	LaunchWebBoard bool                `json:"launchWebBoard"`
	WebBoardUser   string              `json:"webBoardUser"`
}

// Defaults sets some sane config defaults
func (c *Config) Defaults() {
	if c.RuntimeOptions == nil {
		c.RuntimeOptions = &rgb.DefaultRuntimeOptions
	}
	c.RuntimeOptions.Daemon = 0
	c.RuntimeOptions.DoGPIOInit = true

	if c.HTTPListenPort == 0 {
		c.HTTPListenPort = 8080
	}
	if c.WebBoardUser == "" {
		c.WebBoardUser = "pi"
	}

	if c.HardwareConfig == nil {
		c.HardwareConfig = &rgb.DefaultConfig
		c.HardwareConfig.Cols = 64
		c.HardwareConfig.Rows = 32
	}

	if c.HardwareConfig.Rows == 0 {
		c.HardwareConfig.Rows = 32
	}
	if c.HardwareConfig.Cols == 0 {
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
func New(ctx context.Context, logger *zap.Logger, cfg *Config, canvases []board.Canvas, boards ...board.Board) (*SportsMatrix, error) {
	cfg.Defaults()

	s := &SportsMatrix{
		boards:       boards,
		cfg:          cfg,
		log:          logger,
		screenOff:    make(chan struct{}),
		screenOn:     make(chan struct{}),
		serveBlock:   make(chan struct{}),
		close:        make(chan struct{}),
		screenIsOn:   atomic.NewBool(true),
		webBoardIsOn: atomic.NewBool(false),
		webBoardOn:   make(chan struct{}),
		webBoardOff:  make(chan struct{}),
		canvases:     canvases,
	}

	// Add an ImgCanvas
	if s.cfg.WebBoardWidth == 0 {
		s.cfg.WebBoardWidth = 800
	}
	height := s.cfg.WebBoardWidth / 2
	s.log.Info("init web baord",
		zap.Int("X", s.cfg.WebBoardWidth),
		zap.Int("Y", height),
	)
	s.canvases = append(s.canvases, imgcanvas.New(s.cfg.WebBoardWidth, height, s.log))

	for _, b := range s.boards {
		s.log.Info("Registering board", zap.String("board", b.Name()))
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

// GetImgCanvas checks the matrix for board.Canvas of type imgcanvas.ImgCanvas
func (s *SportsMatrix) GetImgCanvas() (*imgcanvas.ImgCanvas, error) {
	for _, canvas := range s.canvases {
		switch c := canvas.(type) {
		case *imgcanvas.ImgCanvas:
			return c, nil
		default:
		}
	}

	return nil, fmt.Errorf("ImgCanvas is not set")
}

func (s *SportsMatrix) screenWatcher(ctx context.Context) {
	webBoardWasOn := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.screenOff:
			changed := s.screenIsOn.CAS(true, false)
			if !changed {
				s.log.Warn("Screen is already off")
				continue
			}
			s.log.Warn("screen turning off")

			s.Lock()
			s.boardCancel()
			for _, canvas := range s.canvases {
				_ = canvas.Clear()
			}
			s.boardCtx, s.boardCancel = context.WithCancel(ctx)
			webBoardWasOn = s.webBoardIsOn.Load()
			s.webBoardOff <- struct{}{}
			s.Unlock()
		case <-s.screenOn:
			changed := s.screenIsOn.CAS(false, true)
			if changed {
				s.log.Warn("screen turning on")
				s.serveBlock <- struct{}{}
				if webBoardWasOn {
					s.webBoardOn <- struct{}{}
				}
			} else {
				s.log.Warn("screen is already on")
			}
		}
	}
}

func (s *SportsMatrix) webBoardWatcher(ctx context.Context) {
	var webCtx context.Context
	var webCancel context.CancelFunc
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.webBoardOff:
			if !s.webBoardIsOn.Load() {
				s.log.Warn("web board is already off")
				continue
			}
			webCancel()
			s.webBoardIsOn.Store(false)
		case <-s.webBoardOn:
			if s.webBoardIsOn.Load() {
				s.log.Warn("web board is already on")
				continue
			}
			webCtx, webCancel = context.WithCancel(ctx)
			defer webCancel()
			go func() {
				tries := 0
				for {
					select {
					case <-webCtx.Done():
						return
					default:
					}
					if err := s.launchWebBoard(webCtx); err != nil {
						if err == context.Canceled {
							s.log.Warn("web board context canceled, closing", zap.Error(err))
							return
						}
						s.log.Error("failed to launch web board", zap.Error(err))
					}
					tries++
					if tries > 10 {
						s.log.Error("failed too many times to launch web board")
						return
					}
					time.Sleep(5 * time.Second)
				}
			}()
			s.webBoardIsOn.Store(true)
		}
	}
}

// Serve blocks until the context is canceled
func (s *SportsMatrix) Serve(ctx context.Context) error {
	defer func() {
		for _, canvas := range s.canvases {
			_ = canvas.Close()
		}
	}()

	s.boardCtx, s.boardCancel = context.WithCancel(ctx)
	defer s.boardCancel()

	go s.webBoardWatcher(ctx)
	if s.cfg.LaunchWebBoard {
		s.webBoardOn <- struct{}{}
	}

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
				for _, canvas := range s.canvases {
					if err := canvas.Clear(); err != nil {
						s.log.Error("failed to clear matrix when all boards were disabled", zap.Error(err))
					}
				}
			})

			time.Sleep(2 * time.Second)
			continue
		}

		clearer = sync.Once{}

		if !s.screenIsOn.Load() {
			s.log.Warn("screen is turned off")

			// Block until the screen is turned back on
			select {
			case <-ctx.Done():
				return context.Canceled
			case <-s.serveBlock:
				continue
			}
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

		renderStart := time.Now()

		var wg sync.WaitGroup

		for _, canvas := range s.canvases {
			wg.Add(1)
			go func(canvas board.Canvas) {
				defer wg.Done()
				if !canvas.Enabled() {
					s.log.Warn("canvas is disabled, skipping", zap.String("canvas", canvas.Name()))
					return
				}
				if err := b.Render(ctx, canvas); err != nil {
					s.log.Error(err.Error())
				}
			}(canvas)
		}
		wg.Wait()

		select {
		case renderDone <- struct{}{}:
		default:
		}

		// If for some reason the render returns really quickly, like
		// the board not implementing a delay, let's sleep here for a bit
		if time.Since(renderStart) < 2*time.Second {
			time.Sleep(5 * time.Second)
		}
	}
}

// Close closes the matrix
func (s *SportsMatrix) Close() {
	s.close <- struct{}{}
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
