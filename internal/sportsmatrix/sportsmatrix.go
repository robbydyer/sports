package sportsmatrix

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/imgcanvas"
	rgb "github.com/robbydyer/sports/internal/rgbmatrix-rpi"
)

var version = "noversion"

// SportsMatrix controls the RGB matrix. It rotates through a list of given board.Board
type SportsMatrix struct {
	cfg                *Config
	isServing          chan struct{}
	canvases           []board.Canvas
	boards             []board.Board
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
	liveOnly           *atomic.Bool
	scrollStatus       chan float64
	scrollInProgress   *atomic.Bool
	sync.Mutex
}

// Config ...
type Config struct {
	combinedScrollDelay   time.Duration
	ServeWebUI            bool                `json:"serveWebUI"`
	HTTPListenPort        int                 `json:"httpListenPort"`
	HardwareConfig        *rgb.HardwareConfig `json:"hardwareConfig"`
	RuntimeOptions        *rgb.RuntimeOptions `json:"runtimeOptions"`
	ScreenOffTimes        []string            `json:"screenOffTimes"`
	ScreenOnTimes         []string            `json:"screenOnTimes"`
	WebBoardWidth         int                 `json:"webBoardWidth"`
	WebBoardHeight        int                 `json:"webBoardHeight"`
	LaunchWebBoard        bool                `json:"launchWebBoard"`
	WebBoardUser          string              `json:"webBoardUser"`
	CombinedScroll        *atomic.Bool        `json:"combinedScroll"`
	CombinedScrollDelay   string              `json:"combinedScrollDelay"`
	CombinedScrollPadding int                 `json:"combinedScrollPadding"`
}

type orderedBoard struct {
	order        int
	board        board.Board
	scrollCanvas *rgb.ScrollCanvas
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
	if c.CombinedScroll == nil {
		c.CombinedScroll = atomic.NewBool(false)
	}
	if c.CombinedScrollDelay != "" {
		d, err := time.ParseDuration(c.CombinedScrollDelay)
		if err != nil {
			c.combinedScrollDelay = rgb.DefaultScrollDelay
		} else {
			c.combinedScrollDelay = d
		}
	} else {
		c.combinedScrollDelay = rgb.DefaultScrollDelay
	}
}

// New ...
func New(ctx context.Context, logger *zap.Logger, cfg *Config, canvases []board.Canvas, boards ...board.Board) (*SportsMatrix, error) {
	cfg.Defaults()

	s := &SportsMatrix{
		boards:           boards,
		cfg:              cfg,
		log:              logger,
		serveBlock:       make(chan struct{}),
		close:            make(chan struct{}),
		screenIsOn:       atomic.NewBool(true),
		webBoardIsOn:     atomic.NewBool(false),
		webBoardOn:       make(chan struct{}),
		webBoardOff:      make(chan struct{}),
		isServing:        make(chan struct{}, 1),
		jumpTo:           make(chan string, 1),
		canvases:         canvases,
		jumping:          atomic.NewBool(false),
		screenSwitch:     make(chan struct{}, 1),
		webBoardWasOn:    atomic.NewBool(false),
		liveOnly:         atomic.NewBool(false),
		scrollStatus:     make(chan float64),
		scrollInProgress: atomic.NewBool(false),
	}

	s.boardCtx, s.boardCancel = context.WithCancel(context.Background())

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
			if err := s.ScreenOff(context.Background()); err != nil {
				s.log.Error("failed to turn screen off during ScreenOfftimes",
					zap.Error(err),
				)
			}
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add cron for screen off times: %w", err)
		}
	}
	for _, on := range s.cfg.ScreenOnTimes {
		s.log.Info("Screen will be scheduled to turn on", zap.String("turn on", on))
		_, err := c.AddFunc(on, func() {
			s.log.Warn("Turning screen on!")
			if err := s.ScreenOn(context.Background()); err != nil {
				s.log.Error("failed to turn screen on during ScreenOnTimes",
					zap.Error(err),
				)
			}
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
	// The screenSwitch channel is used just like a sync.Mutex, but with
	// a timeout. It has a buffer size of 1. This is so we don't try to turn the screen
	// off or on at the same time and that we don't pool up a bunch of on/off requests

	select {
	case s.screenSwitch <- struct{}{}:
	case <-time.After(2 * time.Second):
		return fmt.Errorf("timed out waiting for switch lock")
	case <-ctx.Done():
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
	case <-time.After(10 * time.Second):
		s.log.Error("timed out while trying to unblock serveBlock")
	}

	if s.webBoardWasOn.Load() {
		s.startWebBoard(s.serveContext)
	}

	return nil
}

// ScreenOff turns the matrix off
func (s *SportsMatrix) ScreenOff(ctx context.Context) error {
	select {
	case s.screenSwitch <- struct{}{}:
	case <-time.After(2 * time.Second):
		return fmt.Errorf("timed out waiting for switch lock")
	case <-ctx.Done():
		return context.Canceled
	}

	defer func() {
		<-s.screenSwitch
	}()

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
	setServing := func() {
		select {
		case s.isServing <- struct{}{}:
		default:
		}
	}

	setServingOnce := sync.Once{}

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
				s.log.Warn("context canceled while waiting for screen to come back on")
				return context.Canceled
			case <-s.serveBlock:
				s.log.Warn("screen is back on")
				continue
			}
		}

		setServingOnce.Do(setServing)

		if s.cfg.CombinedScroll.Load() {
			if err := s.doCombinedScroll(s.boardCtx); err != nil {
				s.log.Error("combined scroll error",
					zap.Error(err),
				)
			}
		} else {
			s.serveLoop(s.boardCtx)
		}
	}
}

func (s *SportsMatrix) serveLoop(ctx context.Context) {
BOARDS:
	for _, b := range s.boards {
		select {
		case <-ctx.Done():
			return
		default:
		}

		s.currentBoardCtx, s.currentBoardCancel = context.WithCancel(ctx)
		if err := s.doBoard(s.currentBoardCtx, b); err != nil {
			s.currentBoardCancel()
			continue BOARDS
		}

		if b.Enabler().Enabled() {
		BETWEEN_BOARDS:
			for _, between := range s.betweenBoards {
				select {
				case <-ctx.Done():
					return
				case <-s.currentBoardCtx.Done():
					s.log.Debug("current board context canceled while rendering in-between boards",
						zap.String("board", b.Name()),
						zap.String("in-between", between.Name()),
					)
					continue BOARDS
				default:
				}
				s.log.Debug("rendering in-between board",
					zap.String("board", between.Name()),
					zap.String("prior board", b.Name()),
				)
				if err := s.doBoard(s.currentBoardCtx, between); err != nil {
					continue BETWEEN_BOARDS
				}
			}
		}

		s.currentBoardCancel()
	}
}

// doCombinedScroll gets a scrollCanvas version of each board, then combines them
// into one large ScrollCanvas. It maintains ordering
func (s *SportsMatrix) doCombinedScroll(ctx context.Context) error {
	// nolint: govet
	scrollCtx, cancel := context.WithCancel(ctx)

	boards := []board.Board{}

	canceler := func() {
		cancel()
	}
	for _, board := range s.boards {
		board.Enabler().SetStateChangeCallback(canceler)
		if board.Enabler().Enabled() {
			boards = append(boards, board)
		}
	}
	for _, board := range s.betweenBoards {
		if board.Enabler().Enabled() {
			board.Enabler().SetStateChangeCallback(canceler)
		}
	}

CANVASES:
	for _, canvas := range s.canvases {
		if !canvas.Enabled() || !canvas.Scrollable() {
			continue CANVASES
		}

		base, ok := canvas.(*rgb.ScrollCanvas)
		if !ok {
			continue CANVASES
		}

		scrollCanvas, err := rgb.NewScrollCanvas(base.Matrix, s.log,
			rgb.WithScrollSpeed(s.cfg.combinedScrollDelay),
			rgb.WithScrollDirection(rgb.RightToLeft),
			rgb.WithMergePadding(s.cfg.CombinedScrollPadding),
		)
		if err != nil {
			cancel()
			return err
		}

		ch := make(chan *orderedBoard, len(boards))
		s.prepOrderedBoards(ctx, s.boards, base.Matrix, ch)

		betweenCh := make(chan *orderedBoard, len(boards))
		s.prepOrderedBoards(ctx, s.betweenBoards, base.Matrix, betweenCh)

		allOrderedBoards := []*orderedBoard{}
	SCR:
		for scrCanvas := range ch {
			if scrCanvas.scrollCanvas == nil {
				continue SCR
			}
			allOrderedBoards = append(allOrderedBoards, scrCanvas)
		}

		betweenBoards := []*orderedBoard{}

		for c := range betweenCh {
			if c != nil {
				betweenBoards = append(betweenBoards, c)
			}
		}

		sort.SliceStable(allOrderedBoards, func(i, j int) bool {
			return allOrderedBoards[i].order < allOrderedBoards[j].order
		})
		sort.SliceStable(betweenBoards, func(i, j int) bool {
			return betweenBoards[i].order < betweenBoards[j].order
		})

		for _, ordered := range allOrderedBoards {
			if ordered.scrollCanvas.Len() > 0 {
				scrollCanvas.AddCanvas(ordered.scrollCanvas)
				for _, c := range betweenBoards {
					scrollCanvas.AddCanvas(c.scrollCanvas)
				}
			} else {
				s.log.Debug("board had less than 1 canvas rendered",
					zap.String("board", ordered.board.Name()),
					zap.Int("number", ordered.scrollCanvas.Len()),
				)
			}
		}

		scrollCanvas.PrepareSubCanvases()

		s.log.Debug("prepared combined canvas, waiting for previous to finish")

		ticker := time.NewTicker(500 * time.Millisecond)
	WAIT:
		for {
			if !s.scrollInProgress.Load() {
				break WAIT
			}
			select {
			case <-scrollCtx.Done():
				cancel()
				return context.Canceled
			case <-ticker.C:
			}
		}

		s.scrollInProgress.Store(true)
		go func() {
			defer func() {
				s.scrollInProgress.Store(false)
				cancel()
			}()

			s.log.Debug("performing combined scroll",
				zap.Int("pad", s.cfg.CombinedScrollPadding),
				zap.Duration("scroll delay", s.cfg.combinedScrollDelay),
			)

			if err := scrollCanvas.RenderNoMerge(scrollCtx, s.scrollStatus); err != nil {
				s.log.Error("combined scroll failed",
					zap.Error(err),
				)
			}
		}()

		s.waitForScroll(scrollCtx, 0.7, 5*time.Minute)
		s.log.Debug("done waiting for combined scroll")
	}

	// nolint: govet
	return nil
}

func (s *SportsMatrix) waitForScroll(ctx context.Context, waitFor float64, timeout time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(timeout):
			s.log.Error("timed out waiting for scroll",
				zap.Duration("timeout", timeout),
			)
			return
		case status := <-s.scrollStatus:
			s.log.Debug("scroll progress",
				zap.Float64("percentage", status*100),
			)
			if status >= waitFor {
				return
			}
		}
	}
}

func (s *SportsMatrix) doBoard(ctx context.Context, b board.Board) error {
	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}

	s.boardLock.Lock()
	defer s.boardLock.Unlock()

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

	if !b.Enabler().Enabled() {
		// s.log.Debug("skipping disabled board", zap.String("board", b.Name()))
		return nil
	}

	var wg sync.WaitGroup

	var boardErr error

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
				boardErr = err
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

	return boardErr
}

// Close closes the matrix
func (s *SportsMatrix) Close() {
	s.close <- struct{}{}
	s.server.Close()
}

func (s *SportsMatrix) allDisabled() bool {
	for _, b := range s.boards {
		if b.Enabler().Enabled() {
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

	boards := append(s.boards, s.betweenBoards...)

	for _, b := range boards {
		if strings.EqualFold(b.Name(), boardName) {
			b.Enabler().Enable()

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

			if err := s.ScreenOn(context.Background()); err != nil {
				s.log.Error("failed to turn screen back on after jump",
					zap.String("board", b.Name()),
					zap.Error(err),
				)
			}

			return nil
		}
	}

	return fmt.Errorf("could not find board %s to jump to", boardName)
}

func (s *SportsMatrix) prepOrderedBoards(ctx context.Context, boards []board.Board, matrix rgb.Matrix, canvases chan *orderedBoard) {
	wg := sync.WaitGroup{}

	index := 0
	for _, b := range boards {
		if !b.Enabler().Enabled() {
			continue
		}

		wg.Add(1)
		go func(thisBoard board.Board, i int) {
			defer wg.Done()
			myBase, err := rgb.NewScrollCanvas(matrix, s.log,
				rgb.WithScrollDirection(rgb.RightToLeft),
				rgb.WithScrollSpeed(s.cfg.combinedScrollDelay),
			)
			if err != nil {
				s.log.Error("failed to create scroll canvas",
					zap.Error(err),
					zap.String("board", thisBoard.Name()),
				)
				return
			}
			boardCanvas, err := thisBoard.ScrollRender(ctx, myBase, s.cfg.CombinedScrollPadding)
			if err != nil {
				s.log.Error("failed to render between board scroll canvas",
					zap.Error(err),
				)
				return
			}
			myCanvas, ok := boardCanvas.(*rgb.ScrollCanvas)
			if !ok {
				s.log.Error("unexpected board type in combined scroll",
					zap.String("board", thisBoard.Name()),
				)
				return
			}
			s.log.Debug("ordered board prepped",
				zap.String("board", thisBoard.Name()),
				zap.Int("index", i),
			)
			canvases <- &orderedBoard{
				order:        i,
				board:        thisBoard,
				scrollCanvas: myCanvas,
			}
		}(b, index)

		index++
	}

	wg.Wait()
	close(canvases)
}
