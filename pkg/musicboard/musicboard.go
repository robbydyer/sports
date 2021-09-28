package musicboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"net/http"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

var (
	red        = color.RGBA{255, 0, 0, 255}
	green      = color.RGBA{0, 255, 0, 255}
	lightGreen = color.NRGBA{0, 255, 0, 50}
	lightRed   = color.NRGBA{255, 0, 0, 50}
)

// MusicBoard displays music
type MusicBoard struct {
	config      *Config
	api         API
	log         *zap.Logger
	trackWriter *rgbrender.TextWriter
	enablerLock sync.Mutex
	cancelBoard chan struct{}
	sync.Mutex
}

// Config for a MusicBoard
type Config struct {
	boardDelay         time.Duration
	updateInterval     time.Duration
	scrollDelay        time.Duration
	Enabled            *atomic.Bool `json:"enabled"`
	Symbols            []string     `json:"symbols"`
	BoardDelay         string       `json:"boardDelay"`
	ScrollMode         *atomic.Bool `json:"scrollMode"`
	TightScrollPadding int          `json:"tightScrollPadding"`
	ScrollDelay        string       `json:"scrollDelay"`
}

type Track struct {
	Artist string
	Album  string
	Song   string
	Cover  image.Image
}

// API interface for getting stock data
type API interface {
	GetPlaying(ctx context.Context) (*Track, error)
}

// SetDefaults ...
func (c *Config) SetDefaults() {
	if c.Enabled == nil {
		c.Enabled = atomic.NewBool(false)
	}
	if c.ScrollMode == nil {
		c.ScrollMode = atomic.NewBool(false)
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

	if c.ScrollDelay != "" {
		d, err := time.ParseDuration(c.ScrollDelay)
		if err != nil {
			c.scrollDelay = rgbmatrix.DefaultScrollDelay
		}
		c.scrollDelay = d
	} else {
		c.scrollDelay = rgbmatrix.DefaultScrollDelay
	}
}

// New ...
func New(api API, cfg *Config, log *zap.Logger) (*MusicBoard, error) {
	s := &MusicBoard{
		config:      cfg,
		api:         api,
		log:         log,
		cancelBoard: make(chan struct{}),
	}

	return s, nil
}

func (m *MusicBoard) cacheClear() {
}

// Enabled ...
func (m *MusicBoard) Enabled() bool {
	return m.config.Enabled.Load()
}

// Enable ...
func (m *MusicBoard) Enable() {
	m.config.Enabled.Store(true)
}

// Disable ...
func (m *MusicBoard) Disable() {
	m.config.Enabled.Store(false)
}

// Name ...
func (m *MusicBoard) Name() string {
	return "Music"
}

func (m *MusicBoard) enablerCancel(ctx context.Context, cancel context.CancelFunc) {
	m.enablerLock.Lock()
	defer m.enablerLock.Unlock()
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.cancelBoard:
			cancel()
			return
		case <-ticker.C:
			if !m.config.Enabled.Load() {
				cancel()
				return
			}
		}
	}
}

// Render ...
func (m *MusicBoard) Render(ctx context.Context, canvas board.Canvas) error {
	boardCtx, boardCancel := context.WithCancel(ctx)
	defer boardCancel()

	go m.enablerCancel(boardCtx, boardCancel)

	track, err := m.api.GetPlaying(ctx)
	if err != nil {
		return err
	}

	var scrollCanvas *rgbmatrix.ScrollCanvas
	if canvas.Scrollable() && m.config.ScrollMode.Load() {
		base, ok := canvas.(*rgbmatrix.ScrollCanvas)
		if !ok {
			return fmt.Errorf("wat")
		}

		var err error
		scrollCanvas, err = rgbmatrix.NewScrollCanvas(base.Matrix, m.log)
		if err != nil {
			return fmt.Errorf("failed to get tight scroll canvas: %w", err)
		}
		scrollCanvas.SetScrollDirection(rgbmatrix.RightToLeft)

		if err := m.render(ctx, scrollCanvas, track); err != nil {
			return err
		}

		return scrollCanvas.Render(ctx)
	}

	if err := canvas.Render(ctx); err != nil {
		return err
	}

	select {
	case <-time.After(m.config.boardDelay):
	case <-boardCtx.Done():
	}

	return nil
}

// GetHTTPHandlers ...
func (m *MusicBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return []*board.HTTPHandler{
		{
			Path: "/music/enable",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				m.log.Info("enabling board", zap.String("board", m.Name()))
				m.Enable()
			},
		},
		{
			Path: "/music/disable",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				m.log.Info("disabling board", zap.String("board", m.Name()))
				select {
				case m.cancelBoard <- struct{}{}:
				default:
				}
				m.Disable()
				m.cacheClear()
			},
		},
		{
			Path: "/music/status",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				m.log.Debug("get board status", zap.String("board", m.Name()))
				w.Header().Set("Content-Type", "text/plain")
				if m.Enabled() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
		{
			Path: "/music/scrollon",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				select {
				case m.cancelBoard <- struct{}{}:
				default:
				}
				m.config.ScrollMode.Store(true)
				m.cacheClear()
			},
		},
		{
			Path: "/music/scrolloff",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				m.config.ScrollMode.Store(false)
				select {
				case m.cancelBoard <- struct{}{}:
				default:
				}
				m.cacheClear()
			},
		},
		{
			Path: "/music/scrollstatus",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				m.log.Debug("get board scroll status", zap.String("board", m.Name()))
				w.Header().Set("Content-Type", "text/plain")
				if m.config.ScrollMode.Load() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
		{
			Path: "/music/clearcache",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				select {
				case m.cancelBoard <- struct{}{}:
				default:
				}
				m.cacheClear()
			},
		},
	}, nil
}

// ScrollMode ...
func (m *MusicBoard) ScrollMode() bool {
	return m.config.ScrollMode.Load()
}
