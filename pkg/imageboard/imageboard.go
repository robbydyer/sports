package imageboard

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

const (
	diskCacheDir = "/tmp/sportsmatrix/imageboard"
)

// var supportedImageTypes = []string{"png", "gif"}

// ImageBoard is a board for displaying image files
type ImageBoard struct {
	config       *Config
	log          *zap.Logger
	fs           afero.Fs
	imageCache   map[string]image.Image
	gifCache     map[string]*gif.GIF
	matrixBounds image.Rectangle
}

// Config ...
type Config struct {
	boardDelay   time.Duration
	BoardDelay   string       `json:"boardDelay"`
	Enabled      *atomic.Bool `json:"enabled"`
	Directories  []string     `json:"directories"`
	UseDiskCache bool         `json:"useDiskCache"`
}

// SetDefaults sets some Config defaults
func (c *Config) SetDefaults() {
	if c.BoardDelay != "" {
		d, err := time.ParseDuration(c.BoardDelay)
		if err == nil {
			c.boardDelay = d
		} else {
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
func New(fs afero.Fs, bounds image.Rectangle, cfg *Config, logger *zap.Logger) (*ImageBoard, error) {
	if fs == nil {
		fs = afero.NewOsFs()
	}
	i := &ImageBoard{
		matrixBounds: bounds,
		config:       cfg,
		log:          logger,
		fs:           fs,
		imageCache:   make(map[string]image.Image),
		gifCache:     make(map[string]*gif.GIF),
	}

	if err := i.validateDirectories(); err != nil {
		return nil, err
	}

	return i, nil
}

func (i *ImageBoard) cacheClear() {
	for k := range i.gifCache {
		delete(i.gifCache, k)
	}
	for k := range i.imageCache {
		delete(i.imageCache, k)
	}
}

// Name ...
func (i *ImageBoard) Name() string {
	return "Image Board"
}

// Enabled ...
func (i *ImageBoard) Enabled() bool {
	return i.config.Enabled.Load()
}

// Render ...
func (i *ImageBoard) Render(ctx context.Context, canvas board.Canvas) error {
	if !i.config.Enabled.Load() {
		i.log.Warn("ImageBoard is disabled, not rendering")
		return nil
	}

	if len(i.config.Directories) < 1 {
		return fmt.Errorf("image board has no directories configured")
	}

	if i.config.UseDiskCache {
		if _, err := os.Stat(diskCacheDir); err != nil {
			if os.IsNotExist(err) {
				if err := os.MkdirAll(diskCacheDir, 0755); err != nil {
					return err
				}
			}
		}
	}

	for _, dir := range i.config.Directories {
		if !i.config.Enabled.Load() {
			i.log.Warn("ImageBoard is disabled, not rendering")
			return nil
		}
		i.log.Debug("walking directory", zap.String("directory", dir))

		err := afero.Walk(i.fs, dir, i.dirWalk)
		if err != nil {
			i.log.Error("failed to prepare image for board", zap.Error(err))
		}
	}

	for _, img := range i.imageCache {
		if !i.config.Enabled.Load() {
			i.log.Warn("ImageBoard is disabled, not rendering")
			return nil
		}
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}

		i.log.Debug("playing image")

		align, err := rgbrender.AlignPosition(rgbrender.CenterCenter, canvas.Bounds(), img.Bounds().Dx(), img.Bounds().Dy())
		if err != nil {
			return err
		}

		draw.Draw(canvas, align, img, image.Point{}, draw.Over)

		if err := canvas.Render(); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(i.config.boardDelay):
		}
	}

	for _, g := range i.gifCache {
		if !i.config.Enabled.Load() {
			i.log.Warn("ImageBoard is disabled, not rendering")
			return nil
		}
		i.log.Debug("playing GIF")
		gifCtx, gifCancel := context.WithCancel(ctx)
		defer gifCancel()

		select {
		case <-ctx.Done():
			gifCancel()
			return context.Canceled
		default:
		}

		go func() {
			time.Sleep(i.config.boardDelay)
			gifCancel()
		}()

		if err := rgbrender.PlayGIF(gifCtx, canvas, g); err != nil {
			i.log.Error("GIF player failed", zap.Error(err))
		}
	}

	return nil
}

func (i *ImageBoard) cachedFile(baseName string) string {
	parts := strings.Split(baseName, ".")
	n := fmt.Sprintf("%s_%dx%d.%s", strings.Join(parts[0:len(parts)-1], "."), i.matrixBounds.Dx(), i.matrixBounds.Dy(), parts[len(parts)-1])
	return filepath.Join(diskCacheDir, n)
}

func (i *ImageBoard) dirWalk(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		// recurse?
		return nil
	}

	_, imgOk := i.imageCache[path]
	_, gifOk := i.gifCache[path]
	if imgOk || gifOk {
		i.log.Debug("using cached image", zap.String("path", path))
		return nil
	}

	// Check file cache
	cacheFileName := i.cachedFile(info.Name())

	i.log.Debug("processing image file", zap.String("path", path))

	var f io.ReadCloser

	noCache := false
	if i.config.UseDiskCache {
		i.log.Debug("checking for cached file", zap.String("file", cacheFileName))
		if exists, err := afero.Exists(i.fs, cacheFileName); err == nil && exists {
			f, err = i.fs.Open(cacheFileName)
			if err != nil {
				return err
			}
			defer f.Close()
		} else {
			noCache = true
		}
	}

	if noCache {
		f, err = i.fs.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
	}

	if strings.HasSuffix(strings.ToLower(info.Name()), "gif") {
		g, err := gif.DecodeAll(f)
		if err != nil {
			return err
		}
		i.log.Debug("resizing GIF",
			zap.String("name", info.Name()),
			zap.Int("X Size", i.matrixBounds.Dx()),
			zap.Int("Y Size", i.matrixBounds.Dy()),
			zap.Int("num images", len(g.Image)),
			zap.Int("delays", len(g.Delay)),
		)
		if err := rgbrender.ResizeGIF(g, i.matrixBounds, 1); err != nil {
			return err
		}
		i.log.Debug("after GIF resize",
			zap.Int("num images", len(g.Image)),
			zap.Int("delays", len(g.Delay)),
		)

		if i.config.UseDiskCache {
			i.log.Debug("saving resized GIF", zap.String("filename", cacheFileName))
			if err := rgbrender.SaveGifAfero(i.fs, g, cacheFileName); err != nil {
				i.log.Error("failed to save resized GIF to disk", zap.Error(err))
			}
		}

		i.gifCache[path] = g

		return nil
	}

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}
	// Resize to matrix bounds
	i.log.Debug("resizing image",
		zap.String("name", info.Name()),
		zap.Int("size X", i.matrixBounds.Dx()),
		zap.Int("size Y", i.matrixBounds.Dy()),
	)
	i.imageCache[path] = rgbrender.ResizeImage(img, i.matrixBounds, 1)

	if i.config.UseDiskCache {
		if err := rgbrender.SavePngAfero(i.fs, img, cacheFileName); err != nil {
			i.log.Error("failed to save resized PNG to disk", zap.Error(err))
		}
	}

	return nil
}

// HasPriority ...
func (i *ImageBoard) HasPriority() bool {
	return false
}

func (i *ImageBoard) validateDirectories() error {
	for _, dir := range i.config.Directories {
		exists, err := afero.DirExists(i.fs, dir)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("configured directory '%s' does not exist", dir)
		}
	}

	return nil
}
