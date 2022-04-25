package textboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/twitchtv/twirp"

	"github.com/robbydyer/sports/internal/board"
	cnvs "github.com/robbydyer/sports/internal/canvas"
	"github.com/robbydyer/sports/internal/enabler"
	"github.com/robbydyer/sports/internal/logo"
	pb "github.com/robbydyer/sports/internal/proto/basicboard"
	"github.com/robbydyer/sports/internal/rgbrender"
	"github.com/robbydyer/sports/internal/twirphelpers"
	"github.com/robbydyer/sports/internal/util"
)

var defaultScrollDelay = 15 * time.Millisecond

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
	enabler     board.Enabler
	sync.Mutex
}

// Config for a TextBoard
type Config struct {
	boardDelay         time.Duration
	updateInterval     time.Duration
	scrollDelay        time.Duration
	halfSizeLogo       bool
	StartEnabled       *atomic.Bool `json:"enabled"`
	BoardDelay         string       `json:"boardDelay"`
	UpdateInterval     string       `json:"updateInterval"`
	TightScrollPadding int          `json:"tightScrollPadding"`
	ScrollDelay        string       `json:"scrollDelay"`
	OnTimes            []string     `json:"onTimes"`
	OffTimes           []string     `json:"offTimes"`
	UseLogos           *atomic.Bool `json:"useLogos"`
	Max                *int         `json:"max"`
}

// OptionFunc ...
type OptionFunc func(*TextBoard) error

// API ...
type API interface {
	GetLogo(ctx context.Context) (image.Image, error)
	GetText(ctx context.Context) ([]string, error)
	HTTPPathPrefix() string
}

// SetDefaults ...
func (c *Config) SetDefaults() {
	if c.StartEnabled == nil {
		c.StartEnabled = atomic.NewBool(false)
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
			c.scrollDelay = defaultScrollDelay
		}
		c.scrollDelay = d
	} else {
		c.scrollDelay = defaultScrollDelay
	}

	if c.UseLogos == nil {
		c.UseLogos = atomic.NewBool(true)
	}
}

// New ...
func New(api API, config *Config, log *zap.Logger, opts ...OptionFunc) (*TextBoard, error) {
	s := &TextBoard{
		config:      config,
		api:         api,
		log:         log,
		cancelBoard: make(chan struct{}),
		logos:       make(map[string]*logo.Logo),
		enabler:     enabler.New(),
	}

	if config.StartEnabled.Load() {
		s.enabler.Enable()
	}

	if err := util.SetCrons(config.OnTimes, func() {
		s.log.Info("textboard turning on")
		s.Enabler().Enable()
	}); err != nil {
		return nil, err
	}
	if err := util.SetCrons(config.OffTimes, func() {
		s.log.Info("textboard turning off")
		s.Enabler().Disable()
	}); err != nil {
		return nil, err
	}

	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}

	prfx := s.api.HTTPPathPrefix()
	if !strings.HasPrefix(prfx, "/") {
		prfx = fmt.Sprintf("/%s", prfx)
	}
	prfx = fmt.Sprintf("/headlines%s", prfx)

	svr := &Server{
		board: s,
	}
	s.log.Info("registering textboard",
		zap.String("endpoint", prfx),
	)
	s.rpcServer = pb.NewBasicBoardServer(svr,
		twirp.WithServerPathPrefix(prfx),
		twirp.ChainHooks(
			twirphelpers.GetDefaultHooks(s, s.log),
		),
	)

	return s, nil
}

func (s *TextBoard) Enabler() board.Enabler {
	return s.enabler
}

// InBetween ...
func (s *TextBoard) InBetween() bool {
	return false
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
			if !s.Enabler().Enabled() {
				cancel()
				return
			}
		}
	}
}

// Render ...
func (s *TextBoard) Render(ctx context.Context, canvas board.Canvas) error {
	c, err := s.render(ctx, canvas)
	if err != nil {
		return err
	}
	if c != nil {
		return c.Render(ctx)
	}

	return nil
}

// ScrollRender ...
func (s *TextBoard) ScrollRender(ctx context.Context, canvas board.Canvas, padding int) (board.Canvas, error) {
	origPad := s.config.TightScrollPadding
	defer func() {
		s.config.TightScrollPadding = origPad
	}()

	s.config.TightScrollPadding = padding

	return s.render(ctx, canvas)
}

// Render ...
func (s *TextBoard) render(ctx context.Context, canvas board.Canvas) (board.Canvas, error) {
	if !canvas.Scrollable() || !s.Enabler().Enabled() {
		return nil, nil
	}

	boardCtx, boardCancel := context.WithCancel(ctx)
	defer boardCancel()

	go s.enablerCancel(boardCtx, boardCancel)

	texts, err := s.api.GetText(ctx)
	if err != nil {
		return nil, err
	}

	if len(texts) < 1 {
		return nil, nil
	}

	if s.writer == nil {
		var err error
		s.writer, err = rgbrender.DefaultTextWriter()
		if err != nil {
			return nil, err
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

	var scrollCanvas *cnvs.ScrollCanvas
	base, ok := canvas.(*cnvs.ScrollCanvas)
	if !ok {
		return nil, fmt.Errorf("wat")
	}

	scrollCanvas, err = cnvs.NewScrollCanvas(base.Matrix, s.log,
		cnvs.WithMergePadding(s.config.TightScrollPadding),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get tight scroll canvas: %w", err)
	}
	scrollCanvas.SetScrollDirection(cnvs.RightToLeft)
	scrollCanvas.SetScrollSpeed(s.config.scrollDelay)

	go scrollCanvas.MatchScroll(ctx, base)

	s.log.Debug("scroll config",
		zap.Duration("scroll delay", s.config.scrollDelay),
	)

	num := 0

	origWidth := canvas.GetWidth()
	defer func() {
		s.log.Debug("reset canvas width",
			zap.Int("width", origWidth),
		)
		canvas.SetWidth(origWidth)
	}()

TEXT:
	for _, text := range texts {
		select {
		case <-boardCtx.Done():
			return nil, context.Canceled
		default:
		}
		num++
		if s.config.UseLogos.Load() {
			if err := s.renderLogo(boardCtx, canvas); err != nil {
				s.log.Error("failed to render news logo",
					zap.Error(err),
				)
			}
			scrollCanvas.AddCanvas(canvas)
			draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Over)
		}

		s.log.Debug("render text",
			zap.String("text", text),
		)
		if err := s.doRender(canvas, text); err != nil {
			s.log.Error("failed to render text",
				zap.Error(err),
			)
			continue TEXT
		}

		scrollCanvas.AddCanvas(canvas)
		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Over)

		if s.config.Max != nil && num >= *s.config.Max {
			s.log.Debug("max number of headlines reached, skipping remainder",
				zap.Int("max", *s.config.Max),
				zap.Int("num shown", num),
			)
			break TEXT
		}
	}

	return scrollCanvas, nil
}

// GetHTTPHandlers ...
func (s *TextBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return []*board.HTTPHandler{}, nil
}

// ScrollMode ...
func (s *TextBoard) ScrollMode() bool {
	return true
}

// WithHalfSizeLogo option to shrink headline logo by half
func WithHalfSizeLogo() OptionFunc {
	return func(s *TextBoard) error {
		s.config.halfSizeLogo = true
		return nil
	}
}
