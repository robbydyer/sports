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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

const (
	diskCacheDir = "/tmp/sportsmatrix/imageboard"
)

var supportedImageTypes = []string{"png", "gif"}

type ImageBoard struct {
	config       *Config
	log          *log.Logger
	fs           afero.Fs
	imageCache   map[string]image.Image
	gifCache     map[string]*gif.GIF
	matrixBounds image.Rectangle
}

type Config struct {
	boardDelay   time.Duration
	BoardDelay   string   `json:"boardDelay"`
	Enabled      bool     `json:"enabled"`
	Directories  []string `json:"directories"`
	UseDiskCache bool     `json:"useDiskCache"`
}

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
}

func New(fs afero.Fs, bounds image.Rectangle, cfg *Config, logger *log.Logger) (*ImageBoard, error) {
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

func (i *ImageBoard) Name() string {
	return "Image Board"
}
func (i *ImageBoard) Enabled() bool {
	return i.config.Enabled
}

func (i *ImageBoard) Render(ctx context.Context, matrix rgb.Matrix) error {
	if !i.config.Enabled {
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
		i.log.Debugf("walking directory %s", dir)

		err := afero.Walk(i.fs, dir, i.dirWalk)
		if err != nil {
			i.log.Errorf("failed to prepare image for board: %s", err.Error())
		}
	}

	canvas := rgb.NewCanvas(matrix)
	for _, img := range i.imageCache {
		i.log.Debug("playing image")
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}

		draw.Draw(canvas, canvas.Bounds(), img, image.ZP, draw.Over)

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
		i.log.Debug("playing GIF")
		gifCtx, gifCancel := context.WithCancel(context.Background())
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
			i.log.Errorf("GIF player failed: %s", err.Error())
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
		i.log.Debugf("using cached image for %s", path)
		return nil
	}

	// Check file cache
	cacheFileName := i.cachedFile(info.Name())

	i.log.Debugf("processing image file %s", path)

	var f io.ReadCloser

	noCache := false
	if i.config.UseDiskCache {
		i.log.Debugf("checking for cached file %s", cacheFileName)
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
		i.log.Debugf("resizing %s to %dx%d. GIF contains %d images and %d delays",
			info.Name(),
			i.matrixBounds.Dx(),
			i.matrixBounds.Dy(),
			len(g.Image),
			len(g.Delay),
		)
		if err := rgbrender.ResizeGIF(g, i.matrixBounds, 1); err != nil {
			return err
		}
		i.log.Debugf("after resizing GIF contains %d images and %d delays",
			len(g.Image),
			len(g.Delay),
		)

		if i.config.UseDiskCache {
			i.log.Debugf("saving resized GIF to disk cache %s", cacheFileName)
			if err := rgbrender.SaveGifAfero(i.fs, g, cacheFileName); err != nil {
				i.log.Errorf("failed to save resized GIF to disk: %s", err.Error())
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
	i.log.Debugf("resizing %s to %dx%d", info.Name(), i.matrixBounds.Dx(), i.matrixBounds.Dy())
	i.imageCache[path] = rgbrender.ResizeImage(img, i.matrixBounds, 1)

	if i.config.UseDiskCache {
		if err := rgbrender.SavePngAfero(i.fs, img, cacheFileName); err != nil {
			i.log.Errorf("failed to cache resized PNG to disk: %s", err.Error())
		}
	}

	return nil
}

func (i *ImageBoard) HasPriority() bool {
	return false
}
func (i *ImageBoard) Cleanup() {}

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
