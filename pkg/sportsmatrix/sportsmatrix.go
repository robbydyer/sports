package sportsmatrix

import (
	"context"
	"fmt"
	"net/http"
	"strings"
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
	cfg                *Config
	isServing          chan struct{}
	canvases           []board.Canvas
	boards             []board.Board
	registeredBoards   []string
	screenIsOn         *atomic.Bool
	webBoardIsOn       *atomic.Bool
	webBoardOn         chan struct{}
	webBoardOff        chan struct{}
	serveBlock         chan struct{}
	log                *zap.Logger
	boardCtx           context.Context
	boardCancel        context.CancelFunc
	currentBoardCtx    context.Context
	currentBoardCancel context.CancelFunc
	server             http.Server
	close              chan struct{}
	httpEndpoints      []string
	jumpLock           sync.Mutex
	boardLock          sync.Mutex
	webBoardLock       sync.Mutex
	screenSwitch       chan struct{}
	jumpTo             chan string
	betweenBoards      []board.Board
	currentJump        string
	jumping            *atomic.Bool
	switchedOn         int
	switchedOff        int
	switchTestSleep    bool
	webBoardWasOn      *atomic.Bool
	serveContext       context.Context
	webBoardCtx        context.Context
	webBoardCancel     context.CancelFunc
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
	WebBoardHeight int                 `json:"webBoardHeight"`
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
		boards:        boards,
		cfg:           cfg,
		log:           logger,
		serveBlock:    make(chan struct{}),
		close:         make(chan struct{}),
		screenIsOn:    atomic.NewBool(true),
		webBoardIsOn:  atomic.NewBool(false),
		webBoardOn:    make(chan struct{}),
		webBoardOff:   make(chan struct{}),
		isServing:     make(chan struct{}, 1),
		jumpTo:        make(chan string, 1),
		canvases:      canvases,
		jumping:       atomic.NewBool(false),
		screenSwitch:  make(chan struct{}, 1),
		webBoardWasOn: atomic.NewBool(false),
	}

	// Add an ImgCanvas
	if s.cfg.WebBoardWidth == 0 {
		if s.cfg.WebBoardHeight != 0 {
			s.cfg.WebBoardWidth = s.cfg.WebBoardHeight * 2
		} else {
			s.cfg.WebBoardWidth = 800
		}
	}
	if s.cfg.WebBoardHeight == 0 {
		s.cfg.WebBoardHeight = s.cfg.WebBoardWidth / 2
	}
	s.log.Info("init web baord",
		zap.Int("X", s.cfg.WebBoardWidth),
		zap.Int("Y", s.cfg.WebBoardHeight),
	)
	s.canvases = append(s.canvases, imgcanvas.New(s.cfg.WebBoardWidth, s.cfg.WebBoardHeight, s.log))

	for _, b := range s.boards {
		s.log.Info("Registering board", zap.String("board", b.Name()))
	}

	c := cron.New()

	for _, off := range s.cfg.ScreenOffTimes {
		s.log.Info("Screen will be scheduled to turn off", zap.String("turn off", off))
		_, err := c.AddFunc(off, func() {
			s.log.Warn("Turning screen off!")
			s.ScreenOff(context.Background())
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add cron for screen off times: %w", err)
		}
	}
	for _, on := range s.cfg.ScreenOnTimes {
		s.log.Info("Screen will be scheduled to turn on", zap.String("turn on", on))
		_, err := c.AddFunc(on, func() {
			s.log.Warn("Turning screen on!")
			s.ScreenOn(context.Background())
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add cron for screen on times: %w", err)
		}
	}
	c.Start()

	return s, nil
}

// AddBetweenBoard adds a board to be run between each enabled board
func (s *SportsMatrix) AddBetweenBoard(board board.Board) {
	s.betweenBoards = append(s.betweenBoards, board)
}

// ScreenOn turns the matrix on
func (s *SportsMatrix) ScreenOn(ctx context.Context) error {
	switchCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	select {
	case s.screenSwitch <- struct{}{}:
	case <-switchCtx.Done():
		return context.Canceled
	}

	defer func() {
		<-s.screenSwitch
	}()

	if changed := s.screenIsOn.CAS(false, true); !changed {
		s.log.Warn("screen is already on")
		return nil
	}

	if s.switchTestSleep {
		s.switchedOn++
	}
	s.log.Warn("screen turning on")
	select {
	case s.serveBlock <- struct{}{}:
	default:
	}

	if s.webBoardWasOn.Load() {
		s.startWebBoard(s.serveContext)
	}

	return nil
}

// ScreenOff turns the matrix off
func (s *SportsMatrix) ScreenOff(ctx context.Context) error {
	switchCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	select {
	case s.screenSwitch <- struct{}{}:
	case <-switchCtx.Done():
		return context.Canceled
	}

	defer func() {
		<-s.screenSwitch
	}()

	s.Lock()
	defer s.Unlock()

	if changed := s.screenIsOn.CAS(true, false); !changed {
		s.log.Warn("screen is already off")
		return nil
	}

	s.log.Warn("screen turning off")

	if s.switchTestSleep {
		s.switchedOff++
	}

	s.boardCancel()
	for _, canvas := range s.canvases {
		_ = canvas.Clear()
	}

	s.boardCtx, s.boardCancel = context.WithCancel(s.serveContext)

	s.webBoardWasOn.Store(s.webBoardIsOn.Load())
	s.stopWebBoard()

	return nil
}

// startServices starts the HTTP/RPC services for the boards and the matrix itself
func (s *SportsMatrix) startServices(ctx context.Context) error {
	errChan := s.startHTTP()

	// check for startup error
	s.log.Debug("checking http server for startup error")
	select {
	case <-ctx.Done():
		return context.Canceled
	case err := <-errChan:
		if err != nil {
			return err
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

	return nil
}

func (s *SportsMatrix) startWebBoard(ctx context.Context) {
	s.webBoardLock.Lock()
	defer s.webBoardLock.Unlock()

	s.webBoardCtx, s.webBoardCancel = context.WithCancel(ctx)

	tries := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := s.launchWebBoard(ctx); err != nil {
			if err == context.Canceled {
				s.log.Warn("web board context canceled, closing", zap.Error(err))
				return
			}
			s.log.Error("failed to launch web board", zap.Error(err))
		} else {
			s.webBoardIsOn.Store(true)
			return
		}
		tries++
		if tries > 10 {
			s.log.Error("failed too many times to launch web board")
			return
		}
		time.Sleep(5 * time.Second)
	}
}

func (s *SportsMatrix) stopWebBoard() {
	s.webBoardLock.Lock()
	defer s.webBoardLock.Unlock()
	if !s.webBoardIsOn.Load() {
		return
	}
	s.webBoardCancel()
	s.webBoardIsOn.Store(false)
}

// Serve blocks until the context is canceled
func (s *SportsMatrix) Serve(ctx context.Context) error {
	if err := s.startServices(ctx); err != nil {
		return err
	}

	defer func() {
		for _, canvas := range s.canvases {
			_ = canvas.Close()
		}
	}()

	s.serveContext = ctx

	s.boardCtx, s.boardCancel = context.WithCancel(ctx)
	defer s.boardCancel()

	if s.cfg.LaunchWebBoard {
		s.startWebBoard(ctx)
	}

	if len(s.boards) < 1 {
		return fmt.Errorf("no boards configured")
	}

	clearer := sync.Once{}

	// This is really only for testing.
	select {
	case s.isServing <- struct{}{}:
	default:
	}

	boardOrder := []string{}
	for _, b := range s.boards {
		boardOrder = append(boardOrder, b.Name())

		for _, inb := range s.betweenBoards {
			boardOrder = append(boardOrder, inb.Name())
		}
	}

	s.log.Info("Board Render order",
		zap.Strings("order", boardOrder),
	)

	for {
		select {
		case <-ctx.Done():
			s.log.Warn("context canceled during matrix loop")
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
			case <-s.boardCtx.Done():
				continue
			}
		}

		s.serveLoop(s.boardCtx)
	}
}

func (s *SportsMatrix) serveLoop(ctx context.Context) {
BOARDS:
	for _, b := range s.boards {
		s.currentBoardCtx, s.currentBoardCancel = context.WithCancel(ctx)
		if err := s.doBoard(s.currentBoardCtx, b); err != nil {
			continue BOARDS
		}

		if b.Enabled() {
			for _, between := range s.betweenBoards {
				s.log.Debug("rendering in-between board",
					zap.String("board", between.Name()),
					zap.String("prior board", b.Name()),
				)
				if err := s.doBoard(s.currentBoardCtx, between); err != nil {
					continue BOARDS
				}
			}
		}
	}
}

func (s *SportsMatrix) doBoard(ctx context.Context, b board.Board) error {
	s.boardLock.Lock()
	defer s.boardLock.Unlock()

	renderDone := make(chan struct{})
	defer func() {
		select {
		case renderDone <- struct{}{}:
		default:
		}
	}()

	select {
	case <-ctx.Done():
		s.log.Error("serve loop context was canceled",
			zap.String("board", b.Name()),
		)
		return context.Canceled
	case j := <-s.jumpTo:
		s.currentJump = j
	default:
	}

	if s.currentJump != "" {
		if !strings.EqualFold(b.Name(), s.currentJump) {
			return nil
		}
		s.log.Info("jumping to board",
			zap.String("board", b.Name()),
		)
	}

	s.currentJump = ""

	s.log.Debug("Processing board", zap.String("board", b.Name()))

	if !b.Enabled() {
		// s.log.Debug("skipping disabled board", zap.String("board", b.Name()))
		return nil
	}

	go func() {
		select {
		case <-ctx.Done():
			s.log.Error("serve loop context was canceled",
				zap.String("board", b.Name()),
			)
		case <-renderDone:
		case <-time.After(5 * time.Minute):
			s.log.Error("board rendered longer than normal", zap.String("board", b.Name()))
		}
	}()

	var wg sync.WaitGroup

CANVASES:
	for _, canvas := range s.canvases {
		if !canvas.Enabled() {
			// s.log.Warn("canvas is disabled, skipping", zap.String("canvas", canvas.Name()))
			continue CANVASES
		}

		if (b.ScrollMode() && !canvas.Scrollable()) || (!b.ScrollMode() && canvas.Scrollable()) {
			if !canvas.AlwaysRender() {
				continue CANVASES
			}
		}

		wg.Add(1)
		go func(canvas board.Canvas) {
			defer wg.Done()
			s.log.Debug("rendering board", zap.String("board", b.Name()))
			if err := b.Render(s.currentBoardCtx, canvas); err != nil {
				s.log.Error("board render returned error",
					zap.Error(err),
				)
			}
		}(canvas)
	}
	done := make(chan struct{})

	go func() {
		defer close(done)
		wg.Wait()
	}()

	s.log.Debug("waiting for canvases to be rendered to")
	select {
	case <-ctx.Done():
		s.log.Error("context canceled waiting for canvases to render")
		return context.Canceled
	case <-done:
	}
	s.log.Debug("done waiting for canvases")

	return nil
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

// JumpTo jumps to a board with a given name
func (s *SportsMatrix) JumpTo(ctx context.Context, boardName string) error {
	s.jumpLock.Lock()
	defer s.jumpLock.Unlock()

	s.jumping.Store(true)
	defer s.jumping.Store(false)

	for _, b := range s.boards {
		if strings.EqualFold(b.Name(), boardName) {
			b.Enable()

			defer func() {
				if err := s.ScreenOn(context.Background()); err != nil {
					s.log.Error("failed to turn screen back on after jump",
						zap.String("board", b.Name()),
						zap.Error(err),
					)
				}
			}()

			if err := s.ScreenOff(ctx); err != nil {
				s.log.Error("error while jumping board trying to turn screen off",
					zap.String("board", b.Name()),
					zap.Error(err),
				)
			}

			select {
			case s.jumpTo <- b.Name():
			case <-ctx.Done():
				s.log.Error("context canceled while setting jump board",
					zap.String("board", b.Name()),
				)
				return context.Canceled
			}

			return nil
		}
	}

	return fmt.Errorf("could not find board %s to jump to", boardName)
}
