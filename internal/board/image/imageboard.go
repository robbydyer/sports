package imageboard

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	"github.com/twitchtv/twirp"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/enabler"
	pb "github.com/robbydyer/sports/internal/proto/imageboard"
	"github.com/robbydyer/sports/internal/rgbrender"
	"github.com/robbydyer/sports/internal/twirphelpers"
	"github.com/robbydyer/sports/internal/util"
)

const (
	diskCacheDir = "/tmp/sportsmatrix_logos/imageboard"

	// Name is the board name
	Name = "Img"
)

var preloaderTimeout = (20 * time.Second)

// Jumper is a function that jumps to a board
type Jumper func(ctx context.Context, boardName string) error

// ImageBoard is a board for displaying image files
type ImageBoard struct {
	config         *Config
	log            *zap.Logger
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
	enabler        board.Enabler
	sync.Mutex
}

// ImageDirectory is a configurable directory of images
type ImageDirectory struct {
	Directory string `json:"directory"`
	JumpOnly  bool   `json:"jumpOnly"`
}

// Config ...
type Config struct {
	boardDelay    time.Duration
	BoardDelay    string            `json:"boardDelay"`
	StartEnabled  *atomic.Bool      `json:"enabled"`
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

	if c.StartEnabled == nil {
		c.StartEnabled = atomic.NewBool(false)
	}
	if c.UseDiskCache == nil {
		c.UseDiskCache = atomic.NewBool(true)
	}
	if c.UseMemCache == nil {
		c.UseMemCache = atomic.NewBool(true)
	}
}

// New ...
func New(config *Config, logger *zap.Logger) (*ImageBoard, error) {
	i := &ImageBoard{
		config:         config,
		log:            logger,
		imageCache:     make(map[string]image.Image),
		gifCache:       make(map[string]*gif.GIF),
		lockers:        make(map[string]*sync.Mutex),
		jumpTo:         make(chan string, 1),
		preloaders:     make(map[string]chan struct{}),
		priorJumpState: atomic.NewBool(config.StartEnabled.Load()),
		enabler:        enabler.New(),
	}
	if config.StartEnabled.Load() {
		i.enabler.Enable()
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

	if err := util.SetCrons(config.OnTimes, func() {
		i.log.Info("imageboard turning on")
		i.Enabler().Enable()
	}); err != nil {
		return nil, err
	}
	if err := util.SetCrons(config.OffTimes, func() {
		i.log.Info("imageboard turning off")
		i.Enabler().Disable()
	}); err != nil {
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
	return Name
}

func (i *ImageBoard) Enabler() board.Enabler {
	return i.enabler
}

// InBetween ...
func (i *ImageBoard) InBetween() bool {
	return false
}

// ScrollMode ...
func (i *ImageBoard) ScrollMode() bool {
	return false
}

// ScrollRender ...
func (i *ImageBoard) ScrollRender(ctx context.Context, canvas board.Canvas, padding int) (board.Canvas, error) {
	return nil, nil
}

// Render ...
func (i *ImageBoard) Render(ctx context.Context, canvas board.Canvas) error {
	if !i.Enabler().Enabled() {
		i.log.Warn("ImageBoard is disabled, not rendering")
		return nil
	}

	if len(i.config.Directories) < 1 && len(i.config.DirectoryList) < 1 {
		return fmt.Errorf("image board has no directories configured")
	}

	if i.config.UseDiskCache.Load() {
		if _, err := os.Stat(diskCacheDir); err != nil {
			if os.IsNotExist(err) {
				if err := os.MkdirAll(diskCacheDir, 0o755); err != nil {
					return err
				}
			}
		}
	}

	imageList := []img{}
	gifList := []img{}

	dirWalker := func(dir string, jumpOnly bool) func(string, fs.DirEntry, error) error {
		return func(path string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if dirEntry.IsDir() {
				return nil
			}

			fullPath := filepath.Join(dir, path)

			if strings.HasSuffix(strings.ToLower(dirEntry.Name()), "gif") {
				i.log.Debug("found gif during walk",
					zap.String("path", path),
				)
				gifList = append(gifList, img{
					path:     fullPath,
					jumpOnly: jumpOnly,
				})

				return nil
			}

			i.log.Debug("found image during walk",
				zap.String("path", path),
			)

			imageList = append(imageList, img{
				path:     path,
				jumpOnly: jumpOnly,
			})

			return nil
		}
	}

	for _, dir := range i.config.Directories {
		i.log.Debug("walking directory", zap.String("directory", dir))

		fileSystem := os.DirFS(dir)

		if err := fs.WalkDir(fileSystem, ".", dirWalker(dir, false)); err != nil {
			i.log.Error("failed to prepare image for board", zap.Error(err))
		}
	}

	for _, dir := range i.config.DirectoryList {
		i.log.Debug("walking directory list",
			zap.String("directory", dir.Directory),
		)

		fileSystem := os.DirFS(dir.Directory)

		if err := fs.WalkDir(fileSystem, ".", dirWalker(dir.Directory, dir.JumpOnly)); err != nil {
			i.log.Error("failed to prepare image walking directory list",
				zap.Error(err),
			)
		}
	}

	sort.SliceStable(imageList, func(i, j int) bool {
		return imageList[i].path < imageList[j].path
	})
	sort.SliceStable(gifList, func(i, j int) bool {
		return gifList[i].path < gifList[j].path
	})

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
		i.Enabler().Store(i.priorJumpState.Load())
	}

	return nil
}

func (i *ImageBoard) renderGIFs(ctx context.Context, canvas board.Canvas, images []img, jump string) error {
	preloadImages := make(map[string]*gif.GIF, len(images))

	jump = strings.ToLower(jump)

	wg := sync.WaitGroup{}
	defer wg.Wait()

	preload := func(preloadCtx context.Context, path string) {
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
		preload(pCtx, images[0].path)
	}

IMAGES:
	for index, thisImg := range images {
		p := thisImg.path
		if !i.enabler.Enabled() {
			i.log.Warn("ImageBoard is disabled, not rendering")
			return nil
		}

		nextIndex := index + 1

		pCtx, pCancel := context.WithTimeout(ctx, preloaderTimeout)
		defer pCancel()

		if nextIndex < len(images) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				preload(pCtx, images[nextIndex].path)
			}()
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
		if !i.enabler.Enabled() {
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
	suffix := "tiff"
	if strings.EqualFold(parts[len(parts)-1], "gif") {
		suffix = "gif"
	}
	n := fmt.Sprintf("%s_%dx%d.%s", strings.Join(parts[0:len(parts)-1], "."), bounds.Dx(), bounds.Dy(), suffix)
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
		if exists, err := util.FileExists(cachedFile); err == nil && exists {
			i.log.Debug("cached file exists", zap.String("file", cachedFile))
			img, err := imaging.Open(cachedFile)
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

	img, err := imaging.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}

	// Resize to matrix bounds
	i.log.Debug("resizing image",
		zap.String("name", path),
		zap.Int("size X", bounds.Dx()),
		zap.Int("size Y", bounds.Dy()),
	)
	sizedImg := rgbrender.ResizeImage(img, bounds, 1)

	if i.config.UseDiskCache.Load() {
		if err := imaging.Save(sizedImg, cachedFile); err != nil {
			i.log.Error("failed to save resized image to disk",
				zap.Error(err),
			)
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
		if exists, err := util.FileExists(cachedFile); err == nil && exists {
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

	f, err := os.Open(path)
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
		if err := rgbrender.SaveGif(g, cachedFile); err != nil {
			i.log.Error("failed to save resized GIF to disk", zap.Error(err))
		}
	}

	if i.config.UseMemCache.Load() {
		i.setGIFCache(key, g)
	}

	return g, nil
}

func (i *ImageBoard) getSizedGIFDiskCache(path string) (*gif.GIF, error) {
	f, err := os.Open(path)
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

// HasPriority ...
func (i *ImageBoard) HasPriority() bool {
	return false
}

func (i *ImageBoard) validateDirectories() error {
	for _, dir := range i.config.Directories {
		exists, err := util.FileExists(dir)
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
