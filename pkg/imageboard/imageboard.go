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
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/spf13/afero"
	"github.com/twitchtv/twirp"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	pb "github.com/robbydyer/sports/internal/proto/imageboard"
	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
	"github.com/robbydyer/sports/pkg/twirphelpers"
)

const (
	diskCacheDir = "/tmp/sportsmatrix/imageboard"
)

var preloaderTimeout = (20 * time.Second)

// Jumper is a function that jumps to a board
type Jumper func(ctx context.Context, boardName string) error

// ImageBoard is a board for displaying image files
type ImageBoard struct {
	config         *Config
	log            *zap.Logger
	fs             afero.Fs
	imageCache     map[string]image.Image
	gifCacheLock   sync.Mutex
	gifCache       map[string]*gif.GIF
	lockers        map[string]*sync.Mutex
	preloaders     map[string]chan struct{}
	preloadLock    sync.Mutex
	jumpLock       sync.Mutex
	jumper         Jumper
	jumpTo         chan string
	rpcServer      pb.TwirpServer
	priorJumpState *atomic.Bool
	sync.Mutex
}

type ImageDirectory struct {
	Directory string `json:"directory"`
	JumpOnly  bool   `json:"jumpOnly"`
}

// Config ...
type Config struct {
	boardDelay    time.Duration
	BoardDelay    string            `json:"boardDelay"`
	Enabled       *atomic.Bool      `json:"enabled"`
	Directories   []string          `json:"directories"`
	DirectoryList []*ImageDirectory `json:"directoryList"`
	UseDiskCache  *atomic.Bool      `json:"useDiskCache"`
	UseMemCache   *atomic.Bool      `json:"useMemCache"`
	OnTimes       []string          `json:"onTimes"`
	OffTimes      []string          `json:"offTimes"`
}

type img struct {
	path     string
	jumpOnly bool
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
func New(fs afero.Fs, config *Config, logger *zap.Logger) (*ImageBoard, error) {
	if fs == nil {
		fs = afero.NewOsFs()
	}
	i := &ImageBoard{
		config:         config,
		log:            logger,
		fs:             fs,
		imageCache:     make(map[string]image.Image),
		gifCache:       make(map[string]*gif.GIF),
		lockers:        make(map[string]*sync.Mutex),
		jumpTo:         make(chan string, 1),
		preloaders:     make(map[string]chan struct{}),
		priorJumpState: atomic.NewBool(config.Enabled.Load()),
	}

	if err := i.validateDirectories(); err != nil {
		return nil, err
	}

	svr := &Server{
		board: i,
	}
	i.rpcServer = pb.NewImageBoardServer(svr,
		twirp.WithServerPathPrefix(""),
		twirp.ChainHooks(
			twirphelpers.GetDefaultHooks(i, i.log),
		),
	)

	if len(config.OffTimes) > 0 || len(config.OnTimes) > 0 {
		c := cron.New()
		for _, on := range config.OnTimes {
			i.log.Info("imageboard will be schedule to turn on",
				zap.String("turn on", on),
			)
			_, err := c.AddFunc(on, func() {
				i.log.Info("imageboard turning on")
				i.Enable()
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add cron for imageboard: %w", err)
			}
		}

		for _, off := range config.OffTimes {
			i.log.Info("imageboard will be schedule to turn off",
				zap.String("turn on", off),
			)
			_, err := c.AddFunc(off, func() {
				i.log.Info("imageboard turning off")
				i.Disable()
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add cron for imageboard: %w", err)
			}
		}

		c.Start()
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
	return "Img"
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

// ScrollMode ...
func (i *ImageBoard) ScrollMode() bool {
	return false
}

// Render ...
func (i *ImageBoard) Render(ctx context.Context, canvas board.Canvas) error {
	if !i.config.Enabled.Load() {
		i.log.Warn("ImageBoard is disabled, not rendering")
		return nil
	}

	if len(i.config.Directories) < 1 && len(i.config.DirectoryList) < 1 {
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

	gifs := make(map[string]img)

	images := make(map[string]img)

	dirWalker := func(jumpOnly bool) func(string, os.FileInfo, error) error {
		return func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				// recurse?
				return nil
			}

			if strings.HasSuffix(strings.ToLower(info.Name()), "gif") {
				i.log.Debug("found gif during walk",
					zap.String("path", path),
				)
				gifs[path] = img{
					path:     path,
					jumpOnly: jumpOnly,
				}
				return nil
			}

			i.log.Debug("found image during walk",
				zap.String("path", path),
			)

			images[path] = img{
				path:     path,
				jumpOnly: jumpOnly,
			}

			return nil
		}
	}

	for _, dir := range i.config.Directories {
		i.log.Debug("walking directory", zap.String("directory", dir))

		err := afero.Walk(i.fs, dir, dirWalker(false))
		if err != nil {
			i.log.Error("failed to prepare image for board", zap.Error(err))
		}
	}

	for _, dir := range i.config.DirectoryList {
		i.log.Debug("walking directory list",
			zap.String("directory", dir.Directory),
		)

		if err := afero.Walk(i.fs, dir.Directory, dirWalker(dir.JumpOnly)); err != nil {
			i.log.Error("failed to prepare image walking directory list",
				zap.Error(err),
			)
		}
	}

	imageList := []img{}
	for _, thisImg := range images {
		imageList = append(imageList, thisImg)
	}
	gifList := []img{}
	for _, thisImg := range gifs {
		gifList = append(gifList, thisImg)
	}

	jump := ""
	isJumping := false
	select {
	case <-ctx.Done():
		return context.Canceled
	case j := <-i.jumpTo:
		jump = j
		isJumping = true
	default:
	}

	if err := i.renderImages(ctx, canvas, imageList, jump); err != nil {
		i.log.Error("error rendering images", zap.Error(err))
	}

	if err := i.renderGIFs(ctx, canvas, gifList, jump); err != nil {
		i.log.Error("error rendering GIFs", zap.Error(err))
	}

	if isJumping {
		i.config.Enabled.Store(i.priorJumpState.Load())
	}

	return nil
}

func (i *ImageBoard) renderGIFs(ctx context.Context, canvas board.Canvas, images []img, jump string) error {
	preloadImages := make(map[string]*gif.GIF, len(images))

	jump = strings.ToLower(jump)

	wg := sync.WaitGroup{}
	defer wg.Wait()

	preload := func(preloadCtx context.Context, path string) {
		defer wg.Done()

		ch := i.getPreloader(path)

		img, err := i.getSizedGIF(preloadCtx, path, canvas.Bounds(), ch)
		if err != nil {
			i.log.Error("failed to prepare image", zap.Error(err), zap.String("path", path))
			i.preloadLock.Lock()
			preloadImages[path] = nil
			i.preloadLock.Unlock()
			return
		}
		i.preloadLock.Lock()
		preloadImages[path] = img
		i.preloadLock.Unlock()
	}

	pCtx, pCancel := context.WithTimeout(ctx, preloaderTimeout)
	defer pCancel()

	if len(images) > 0 {
		wg.Add(1)
		preload(pCtx, images[0].path)
	}

IMAGES:
	for index, thisImg := range images {
		p := thisImg.path
		if !i.config.Enabled.Load() {
			i.log.Warn("ImageBoard is disabled, not rendering")
			return nil
		}

		nextIndex := index + 1

		pCtx, pCancel := context.WithTimeout(ctx, preloaderTimeout)
		defer pCancel()

		if nextIndex < len(images) {
			wg.Add(1)
			go preload(pCtx, images[nextIndex].path)
		}

		if jump != "" && !filenameCompare(p, jump) {
			i.log.Debug("skipping image",
				zap.String("this", p),
				zap.String("jump", jump),
			)
			continue IMAGES
		} else if jump != "" {
			i.log.Info("jumping to image",
				zap.String("this", p),
				zap.String("jump", jump),
			)
		}

		if thisImg.jumpOnly && jump == "" {
			continue IMAGES
		}

		ch := i.getPreloader(p)

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-ch:
			i.log.Debug("preloader finished", zap.String("path", p))
		case <-time.After(preloaderTimeout):
			i.log.Error("timed out waiting for image preloader",
				zap.String("path", p),
				zap.Duration("preloader timeout", preloaderTimeout),
			)
			continue IMAGES
		}

		g, ok := preloadImages[p]
		if !ok {
			i.log.Error("preloaded GIF was not ready", zap.String("path", p))
			continue IMAGES
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

		if jump != "" {
			return nil
		}
	}

	return nil
}

func (i *ImageBoard) renderImages(ctx context.Context, canvas board.Canvas, images []img, jump string) error {
	preloader := make(map[string]chan struct{})
	preloadImages := make(map[string]image.Image, len(images))

	jump = strings.ToLower(jump)

	preload := func(path string) {
		i.preloadLock.Lock()
		defer i.preloadLock.Unlock()
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
		preload(images[0].path)
	}

IMAGES:
	for index, thisImg := range images {
		p := thisImg.path
		if !i.config.Enabled.Load() {
			i.log.Warn("ImageBoard is disabled, not rendering")
			return nil
		}

		nextIndex := index + 1

		if nextIndex < len(images) {
			go preload(images[nextIndex].path)
		}

		if jump != "" && !filenameCompare(p, jump) {
			i.log.Debug("skipping image",
				zap.String("this", p),
				zap.String("jump", jump),
			)
			continue IMAGES
		} else if jump != "" {
			i.log.Info("jumping to image",
				zap.String("this", p),
				zap.String("jump", jump),
			)
		}

		if thisImg.jumpOnly && jump == "" {
			continue IMAGES
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-preloader[p]:
			i.log.Debug("preloader finished", zap.String("path", p))
		case <-time.After(preloaderTimeout):
			i.log.Error("timed out waiting for image preloader",
				zap.String("path", p),
				zap.Duration("preloader timeout", preloaderTimeout),
			)
			continue IMAGES
		}

		img, ok := preloadImages[p]
		if !ok || img == nil {
			i.log.Error("preloaded image was not ready", zap.String("path", p))
			continue IMAGES
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

		if jump != "" {
			return nil
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

func (i *ImageBoard) getSizedGIF(ctx context.Context, path string, bounds image.Rectangle, preloader chan<- struct{}) (*gif.GIF, error) {
	defer func() {
		select {
		case preloader <- struct{}{}:
		default:
		}
	}()

	key := cacheKey(path, bounds)

	// Make sure we don't process the same image simultaneously
	i.Lock()
	locker, ok := i.lockers[key]
	if !ok {
		locker = &sync.Mutex{}
		i.lockers[key] = locker
	}
	i.Unlock()

	locker.Lock()
	defer locker.Unlock()

	var err error

	if i.config.UseMemCache.Load() {
		p := i.getGIFCache(key)
		if p != nil {
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
				i.setGIFCache(key, g)
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
	if err := rgbrender.ResizeGIF(ctx, g, bounds, 1); err != nil {
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
		i.setGIFCache(key, g)
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

// SetJumper sets the jumper function
func (i *ImageBoard) SetJumper(j Jumper) {
	i.jumper = j
}
