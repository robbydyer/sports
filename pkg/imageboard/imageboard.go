package imageboard

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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

// ImageBoard is a board for displaying image files
type ImageBoard struct {
	config     *Config
	log        *zap.Logger
	fs         afero.Fs
	imageCache map[string]image.Image
	gifCache   map[string]*gif.GIF
	lockers    map[string]*sync.Mutex
	sync.Mutex
}

// Config ...
type Config struct {
	boardDelay   time.Duration
	BoardDelay   string       `json:"boardDelay"`
	Enabled      *atomic.Bool `json:"enabled"`
	Directories  []string     `json:"directories"`
	UseDiskCache *atomic.Bool `json:"useDiskCache"`
	UseMemCache  *atomic.Bool `json:"useMemCache"`
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
	if c.UseDiskCache == nil {
		c.UseDiskCache = atomic.NewBool(true)
	}
	if c.UseMemCache == nil {
		c.UseMemCache = atomic.NewBool(true)
	}
}

// New ...
func New(fs afero.Fs, cfg *Config, logger *zap.Logger) (*ImageBoard, error) {
	if fs == nil {
		fs = afero.NewOsFs()
	}
	i := &ImageBoard{
		config:     cfg,
		log:        logger,
		fs:         fs,
		imageCache: make(map[string]image.Image),
		gifCache:   make(map[string]*gif.GIF),
		lockers:    make(map[string]*sync.Mutex),
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

// Enable ...
func (i *ImageBoard) Enable() {
	i.config.Enabled.Store(true)
}

// Disable ...
func (i *ImageBoard) Disable() {
	i.config.Enabled.Store(false)
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

	if i.config.UseDiskCache.Load() {
		if _, err := os.Stat(diskCacheDir); err != nil {
			if os.IsNotExist(err) {
				if err := os.MkdirAll(diskCacheDir, 0755); err != nil {
					return err
				}
			}
		}
	}

	images := make(map[string]struct{})
	gifs := make(map[string]struct{})

	dirWalker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// recurse?
			return nil
		}

		if strings.HasSuffix(strings.ToLower(info.Name()), "gif") {
			gifs[path] = struct{}{}
			return nil
		}

		images[path] = struct{}{}

		return nil
	}

	for _, dir := range i.config.Directories {
		if !i.config.Enabled.Load() {
			i.log.Warn("ImageBoard is disabled, not rendering")
			return nil
		}
		i.log.Debug("walking directory", zap.String("directory", dir))

		err := afero.Walk(i.fs, dir, dirWalker)
		if err != nil {
			i.log.Error("failed to prepare image for board", zap.Error(err))
		}
	}

	imageList := []string{}
	for i := range images {
		imageList = append(imageList, i)
	}
	gifList := []string{}
	for i := range gifs {
		gifList = append(gifList, i)
	}

	sort.Strings(imageList)
	sort.Strings(gifList)

	if err := i.renderImages(ctx, canvas, imageList); err != nil {
		i.log.Error("error rendering images", zap.Error(err))
	}

	if err := i.renderGIFs(ctx, canvas, gifList); err != nil {
		i.log.Error("error rendering GIFs", zap.Error(err))
	}

	return nil
}

func (i *ImageBoard) renderGIFs(ctx context.Context, canvas board.Canvas, images []string) error {
	preloader := make(map[string]chan struct{})
	preloadImages := make(map[string]*gif.GIF, len(images))

	preload := func(path string) {
		preloader[path] = make(chan struct{}, 1)
		img, err := i.getSizedGIF(path, canvas.Bounds(), preloader[path])
		if err != nil {
			i.log.Error("failed to prepare image", zap.Error(err), zap.String("path", path))
			preloadImages[path] = nil
			return
		}
		preloadImages[path] = img
	}

	if len(images) > 0 {
		preload(images[0])
	}

	preloaderTimeout := i.config.boardDelay + (1 * time.Minute)
	for index, p := range images {
		if !i.config.Enabled.Load() {
			i.log.Warn("ImageBoard is disabled, not rendering")
			return nil
		}

		nextIndex := index + 1

		if nextIndex < len(images) {
			go preload(images[nextIndex])
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-preloader[p]:
			i.log.Debug("preloader finished", zap.String("path", p))
		case <-time.After(preloaderTimeout):
			i.log.Error("timed out waiting for image preloader", zap.String("path", p))
			continue
		}

		g, ok := preloadImages[p]
		if !ok {
			i.log.Error("preloaded GIF was not ready", zap.String("path", p))
			continue
		}

		i.log.Debug("playing GIF", zap.String("path", p))
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

func (i *ImageBoard) renderImages(ctx context.Context, canvas board.Canvas, images []string) error {
	preloader := make(map[string]chan struct{})
	preloadImages := make(map[string]image.Image, len(images))

	preload := func(path string) {
		preloader[path] = make(chan struct{}, 1)
		img, err := i.getSizedImage(path, canvas.Bounds(), preloader[path])
		if err != nil {
			i.log.Error("failed to prepare image", zap.Error(err), zap.String("path", path))
			preloadImages[path] = nil
			return
		}
		preloadImages[path] = img
	}

	if len(images) > 0 {
		preload(images[0])
	}

	preloaderTimeout := i.config.boardDelay + (1 * time.Minute)

	for index, p := range images {
		if !i.config.Enabled.Load() {
			i.log.Warn("ImageBoard is disabled, not rendering")
			return nil
		}

		nextIndex := index + 1

		if nextIndex < len(images) {
			go preload(images[nextIndex])
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-preloader[p]:
			i.log.Debug("preloader finished", zap.String("path", p))
		case <-time.After(preloaderTimeout):
			i.log.Error("timed out waiting for image preloader", zap.String("path", p))
			continue
		}

		img, ok := preloadImages[p]
		if !ok || img == nil {
			i.log.Error("preloaded image was not ready", zap.String("path", p))
			continue
		}

		i.log.Debug("playing image")

		align, err := rgbrender.AlignPosition(rgbrender.CenterCenter, canvas.Bounds(), img.Bounds().Dx(), img.Bounds().Dy())
		if err != nil {
			return err
		}

		draw.Draw(canvas, align, img, image.Point{}, draw.Over)

		if err := canvas.Render(ctx); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(i.config.boardDelay):
		}
	}

	return nil
}

func cacheKey(path string, bounds image.Rectangle) string {
	return fmt.Sprintf("%s_%dx%d", path, bounds.Dx(), bounds.Dy())
}

func (i *ImageBoard) cachedFile(baseName string, bounds image.Rectangle) string {
	parts := strings.Split(baseName, ".")
	n := fmt.Sprintf("%s_%dx%d.%s", strings.Join(parts[0:len(parts)-1], "."), bounds.Dx(), bounds.Dy(), parts[len(parts)-1])
	return filepath.Join(diskCacheDir, n)
}

func (i *ImageBoard) getSizedImage(path string, bounds image.Rectangle, preloader chan<- struct{}) (image.Image, error) {
	defer func() {
		select {
		case preloader <- struct{}{}:
		default:
		}
	}()

	key := cacheKey(path, bounds)

	// Make sure we don't process the same image simultaneously
	locker, ok := i.lockers[key]
	if !ok {
		i.Lock()
		locker = &sync.Mutex{}
		i.lockers[key] = locker
		i.Unlock()
	}

	locker.Lock()
	defer locker.Unlock()

	var err error

	if i.config.UseMemCache.Load() {
		if p, ok := i.imageCache[key]; ok {
			i.log.Debug("loading image from memory cache",
				zap.String("path", path),
				zap.Int("X", bounds.Dx()),
				zap.Int("Y", bounds.Dy()),
			)
			return p, nil
		}
	}

	cachedFile := i.cachedFile(filepath.Base(path), bounds)

	if i.config.UseDiskCache.Load() {
		i.log.Debug("checking for cached file", zap.String("file", cachedFile))
		if exists, err := afero.Exists(i.fs, cachedFile); err == nil && exists {
			i.log.Debug("cached file exists", zap.String("file", cachedFile))
			img, err := i.getSizedImageDiskCache(cachedFile)
			if err != nil {
				return nil, err
			}
			i.log.Debug("got file from disk cache",
				zap.String("file", cachedFile),
				zap.String("size", fmt.Sprintf("%d_%d", img.Bounds().Dx(), img.Bounds().Dy())),
			)
			if i.config.UseMemCache.Load() {
				i.imageCache[key] = img
			}
			return img, nil
		}
	}

	f, err := i.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	// Resize to matrix bounds
	i.log.Debug("resizing image",
		zap.String("name", path),
		zap.Int("size X", bounds.Dx()),
		zap.Int("size Y", bounds.Dy()),
	)
	sizedImg := rgbrender.ResizeImage(img, bounds, 1)

	if i.config.UseDiskCache.Load() {
		if err := rgbrender.SavePngAfero(i.fs, sizedImg, cachedFile); err != nil {
			i.log.Error("failed to save resized PNG to disk", zap.Error(err))
		}
	}

	if i.config.UseMemCache.Load() {
		i.imageCache[key] = sizedImg
	}

	return sizedImg, nil
}

func (i *ImageBoard) getSizedGIF(path string, bounds image.Rectangle, preloader chan<- struct{}) (*gif.GIF, error) {
	defer func() {
		select {
		case preloader <- struct{}{}:
		default:
		}
	}()

	key := cacheKey(path, bounds)

	// Make sure we don't process the same image simultaneously
	locker, ok := i.lockers[key]
	if !ok {
		i.Lock()
		locker = &sync.Mutex{}
		i.lockers[key] = locker
		i.Unlock()
	}

	locker.Lock()
	defer locker.Unlock()

	var err error

	if i.config.UseMemCache.Load() {
		if p, ok := i.gifCache[key]; ok {
			i.log.Debug("loading GIF from memory cache",
				zap.String("path", path),
				zap.Int("X", bounds.Dx()),
				zap.Int("Y", bounds.Dy()),
			)
			return p, nil
		}
	}

	cachedFile := i.cachedFile(filepath.Base(path), bounds)

	if i.config.UseDiskCache.Load() {
		i.log.Debug("checking for cached file", zap.String("file", cachedFile))
		if exists, err := afero.Exists(i.fs, cachedFile); err == nil && exists {
			g, err := i.getSizedGIFDiskCache(cachedFile)
			if err != nil {
				return nil, err
			}
			if i.config.UseMemCache.Load() {
				i.gifCache[key] = g
			}

			return g, nil
		}
	}

	f, err := i.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil {
		return nil, err
	}
	i.log.Debug("resizing GIF",
		zap.String("name", path),
		zap.Int("X Size", bounds.Dx()),
		zap.Int("Y Size", bounds.Dy()),
		zap.Int("num images", len(g.Image)),
		zap.Int("delays", len(g.Delay)),
	)
	if err := rgbrender.ResizeGIF(g, bounds, 1); err != nil {
		return nil, err
	}
	i.log.Debug("after GIF resize",
		zap.Int("num images", len(g.Image)),
		zap.Int("delays", len(g.Delay)),
	)

	if i.config.UseDiskCache.Load() {
		i.log.Debug("saving resized GIF", zap.String("filename", cachedFile))
		if err := rgbrender.SaveGifAfero(i.fs, g, cachedFile); err != nil {
			i.log.Error("failed to save resized GIF to disk", zap.Error(err))
		}
	}

	if i.config.UseMemCache.Load() {
		i.gifCache[key] = g
	}

	return g, nil
}

func (i *ImageBoard) getSizedGIFDiskCache(path string) (*gif.GIF, error) {
	f, err := i.fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open cached GIF image: %w", err)
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode GIF: %w", err)
	}

	return g, nil
}

func (i *ImageBoard) getSizedImageDiskCache(path string) (image.Image, error) {
	f, err := i.fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open cached image: %w", err)
	}
	defer f.Close()

	if strings.HasSuffix(strings.ToLower(path), ".png") {
		img, err := png.Decode(f)
		if err != nil {
			return nil, err
		}

		i.log.Debug("got PNG from disk cache",
			zap.String("path", path),
			zap.String("size", fmt.Sprintf("%dx%d", img.Bounds().Dx(), img.Bounds().Dy())),
		)

		return img, nil
	}

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	i.log.Debug("got non-png from disk cache",
		zap.String("path", path),
		zap.String("size", fmt.Sprintf("%dx%d", img.Bounds().Dx(), img.Bounds().Dy())),
	)

	return img, nil
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
