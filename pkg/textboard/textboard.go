package textboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robfig/cron/v3"
	"github.com/twitchtv/twirp"

	pb "github.com/robbydyer/sports/internal/proto/basicboard"
	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
	"github.com/robbydyer/sports/pkg/twirphelpers"
)

var (
	red        = color.RGBA{255, 0, 0, 255}
	green      = color.RGBA{0, 255, 0, 255}
	lightGreen = color.NRGBA{0, 255, 0, 50}
	lightRed   = color.NRGBA{255, 0, 0, 50}
)

// TextBoard displays stocks
type TextBoard struct {
	config      *Config
	api         API
	log         *zap.Logger
	writer      *rgbrender.TextWriter
	enablerLock sync.Mutex
	cancelBoard chan struct{}
	rpcServer   pb.TwirpServer
	logos       map[string]*logo.Logo
	logoLock    sync.Mutex
	sync.Mutex
}

// Config for a TextBoard
type Config struct {
	boardDelay         time.Duration
	updateInterval     time.Duration
	scrollDelay        time.Duration
	adjustedResolution int
	Enabled            *atomic.Bool `json:"enabled"`
	Symbols            []string     `json:"symbols"`
	ChartResolution    int          `json:"chartResolution"`
	BoardDelay         string       `json:"boardDelay"`
	UpdateInterval     string       `json:"updateInterval"`
	TightScrollPadding int          `json:"tightScrollPadding"`
	ScrollDelay        string       `json:"scrollDelay"`
	OnTimes            []string     `json:"onTimes"`
	OffTimes           []string     `json:"offTimes"`
	UseLogos           *atomic.Bool `json:"useLogos"`
	Leagues            []string     `json:"leagues"`
}

// API ...
type API interface {
	GetLogo(ctx context.Context) (image.Image, error)
	GetText(ctx context.Context) ([]string, error)
}

// SetDefaults ...
func (c *Config) SetDefaults() {
	if c.Enabled == nil {
		c.Enabled = atomic.NewBool(false)
	}
	if c.ChartResolution <= 0 {
		c.ChartResolution = 4
	}
	if c.BoardDelay != "" {
		d, err := time.ParseDuration(c.BoardDelay)
		if err != nil {
			c.boardDelay = 10 * time.Second
		} else {
			c.boardDelay = d
		}
	} else {
		c.boardDelay = 10 * time.Second
	}

	if c.UpdateInterval != "" {
		d, err := time.ParseDuration(c.UpdateInterval)
		if err != nil {
			c.updateInterval = 5 * time.Minute
		} else {
			c.updateInterval = d
		}
	} else {
		c.updateInterval = 5 * time.Minute
	}

	if c.ScrollDelay != "" {
		d, err := time.ParseDuration(c.ScrollDelay)
		if err != nil {
			c.scrollDelay = rgbmatrix.DefaultScrollDelay
		}
		c.scrollDelay = d
	} else {
		c.scrollDelay = rgbmatrix.DefaultScrollDelay
	}

	if c.UseLogos == nil {
		c.UseLogos = atomic.NewBool(false)
	}
}

// New ...
func New(api API, config *Config, log *zap.Logger) (*TextBoard, error) {
	s := &TextBoard{
		config:      config,
		api:         api,
		log:         log,
		cancelBoard: make(chan struct{}),
		logos:       make(map[string]*logo.Logo),
	}

	svr := &Server{
		board: s,
	}
	s.rpcServer = pb.NewBasicBoardServer(svr,
		twirp.WithServerPathPrefix("/stocks"),
		twirp.ChainHooks(
			twirphelpers.GetDefaultHooks(s, s.log),
		),
	)

	if len(config.OffTimes) > 0 || len(config.OnTimes) > 0 {
		c := cron.New()
		for _, on := range config.OnTimes {
			s.log.Info("textboard will be schedule to turn on",
				zap.String("turn on", on),
			)
			_, err := c.AddFunc(on, func() {
				s.log.Info("textboard turning on")
				s.Enable()
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add cron for textboard: %w", err)
			}
		}

		for _, off := range config.OffTimes {
			s.log.Info("textboard will be schedule to turn off",
				zap.String("turn on", off),
			)
			_, err := c.AddFunc(off, func() {
				s.log.Info("textboard turning off")
				s.Disable()
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add cron for textboard: %w", err)
			}
		}

		c.Start()
	}

	return s, nil
}

// Enabled ...
func (s *TextBoard) Enabled() bool {
	return s.config.Enabled.Load()
}

// Enable ...
func (s *TextBoard) Enable() {
	s.config.Enabled.Store(true)
}

// Disable ...
func (s *TextBoard) Disable() {
	s.config.Enabled.Store(false)
}

// Name ...
func (s *TextBoard) Name() string {
	return "Texts"
}

func (s *TextBoard) enablerCancel(ctx context.Context, cancel context.CancelFunc) {
	s.enablerLock.Lock()
	defer s.enablerLock.Unlock()
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.cancelBoard:
			cancel()
			return
		case <-ticker.C:
			if !s.config.Enabled.Load() {
				cancel()
				return
			}
		}
	}
}

// Render ...
func (s *TextBoard) Render(ctx context.Context, canvas board.Canvas) error {
	if !canvas.Scrollable() {
		return nil
	}

	boardCtx, boardCancel := context.WithCancel(ctx)
	defer boardCancel()

	go s.enablerCancel(boardCtx, boardCancel)

	texts, err := s.api.GetText(ctx)
	if err != nil {
		return err
	}

	if s.writer == nil {
		var err error
		s.writer, err = rgbrender.DefaultTextWriter()
		if err != nil {
			return err
		}

		bounds := rgbrender.ZeroedBounds(canvas.Bounds())
		if bounds.Dy() <= 256 {
			s.writer.FontSize = 8.0
			s.writer.YStartCorrection = -2
		} else {
			s.writer.FontSize = 0.25 * float64(bounds.Dy())
			s.writer.YStartCorrection = -1 * ((bounds.Dy() / 32) + 1)
		}

		s.log.Debug("created writer for textboard",
			zap.Float64("font size", s.writer.FontSize),
			zap.Int("Y Start correction", s.writer.YStartCorrection),
		)
	}

	var scrollCanvas *rgbmatrix.ScrollCanvas
	base, ok := canvas.(*rgbmatrix.ScrollCanvas)
	if !ok {
		return fmt.Errorf("wat")
	}

	scrollCanvas, err = rgbmatrix.NewScrollCanvas(base.Matrix, s.log)
	if err != nil {
		return fmt.Errorf("failed to get tight scroll canvas: %w", err)
	}
	scrollCanvas.SetScrollDirection(rgbmatrix.RightToLeft)
	scrollCanvas.SetScrollSpeed(s.config.scrollDelay)

TEXT:
	for _, text := range texts {
		s.log.Debug("render text",
			zap.String("text", text),
		)
		if err := s.render(boardCtx, canvas, text); err != nil {
			s.log.Error("failed to render text",
				zap.Error(err),
			)
			continue TEXT
		}

		scrollCanvas.AddCanvas(canvas)
		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Over)
	}

	scrollCanvas.Merge(s.config.TightScrollPadding)
	return scrollCanvas.Render(boardCtx)
}

// GetHTTPHandlers ...
func (s *TextBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return []*board.HTTPHandler{}, nil
}

// ScrollMode ...
func (s *TextBoard) ScrollMode() bool {
	return true
}
