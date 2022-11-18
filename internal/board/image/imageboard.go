package imageboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
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
	scrcnvs "github.com/robbydyer/sports/internal/scrollcanvas"
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
	preloadLock    sync.RWMutex
	jumpLock       sync.Mutex
	jumper         Jumper
	jumpTo         chan string
	rpcServer      pb.TwirpServer
	priorJumpState *atomic.Bool
	enabler        board.Enabler
	preloaded      map[string]*img
	sync.Mutex
}

// ImageDirectory is a configurable directory of images
type ImageDirectory struct {
	Directory string `json:"directory"`
	JumpOnly  bool   `json:"jumpOnly"`
}

// Config ...
type Config struct {
	boardDelay         time.Duration
	scrollDelay        time.Duration
	BoardDelay         string            `json:"boardDelay"`
	StartEnabled       *atomic.Bool      `json:"enabled"`
	Directories        []string          `json:"directories"`
	DirectoryList      []*ImageDirectory `json:"directoryList"`
	UseDiskCache       *atomic.Bool      `json:"useDiskCache"`
	UseMemCache        *atomic.Bool      `json:"useMemCache"`
	OnTimes            []string          `json:"onTimes"`
	OffTimes           []string          `json:"offTimes"`
	ScrollMode         *atomic.Bool      `json:"scrollMode"`
	TightScrollPadding int               `json:"tightScrollPadding"`
	ScrollDelay        string            `json:"scrollDelay"`
}

type img struct {
	path     string
	isGif    bool
	jumpOnly bool
	img      image.Image
	gif      *gif.GIF
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
	if c.ScrollMode == nil {
		c.ScrollMode = atomic.NewBool(false)
	}
	if c.ScrollDelay != "" {
		d, err := time.ParseDuration(c.ScrollDelay)
		if err != nil {
			c.scrollDelay = scrcnvs.DefaultScrollDelay
		}
		c.scrollDelay = d
	} else {
		c.scrollDelay = scrcnvs.DefaultScrollDelay
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
		preloaded:      make(map[string]*img),
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
	return i.config.ScrollMode.Load()
}

// ScrollRender ...
func (i *ImageBoard) ScrollRender(ctx context.Context, canvas board.Canvas, padding int) (board.Canvas, error) {
	origScrollMode := i.config.ScrollMode.Load()
	defer func() {
		i.config.ScrollMode.Store(origScrollMode)
	}()

	i.config.ScrollMode.Store(true)

	return i.render(ctx, canvas)
}

// Render ...
func (i *ImageBoard) Render(ctx context.Context, canvas board.Canvas) error {
	c, err := i.render(ctx, canvas)
	if err != nil {
		return err
	}
	if c != nil {
		return c.Render(ctx)
	}

	return nil
}

func (i *ImageBoard) render(ctx context.Context, canvas board.Canvas) (board.Canvas, error) {
	if !i.Enabler().Enabled() {
		i.log.Warn("ImageBoard is disabled, not rendering")
		return nil, nil
	}

	if len(i.config.Directories) < 1 && len(i.config.DirectoryList) < 1 {
		return nil, fmt.Errorf("image board has no directories configured")
	}

	if i.config.UseDiskCache.Load() {
		if _, err := os.Stat(diskCacheDir); err != nil {
			if os.IsNotExist(err) {
				if err := os.MkdirAll(diskCacheDir, 0o755); err != nil {
					return nil, err
				}
			}
		}
	}

	imageList := []*img{}
	gifList := []*img{}

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
					zap.String("path", fullPath),
					zap.Bool("jump only", jumpOnly),
				)
				gifList = append(gifList, &img{
					path:     fullPath,
					jumpOnly: jumpOnly,
					isGif:    true,
				})

				return nil
			}

			i.log.Debug("found image during walk",
				zap.String("path", fullPath),
				zap.Bool("jump only", jumpOnly),
			)

			imageList = append(imageList, &img{
				path:     fullPath,
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
			zap.Bool("jump only", dir.JumpOnly),
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
		return nil, context.Canceled
	case j := <-i.jumpTo:
		jump = j
		isJumping = true
	default:
	}

	imgNames := []string{}
	for _, thisImg := range imageList {
		imgNames = append(imgNames, thisImg.path)
	}
	gifNames := []string{}
	for _, thisGif := range gifList {
		gifNames = append(gifNames, thisGif.path)
	}

	i.log.Debug("imageboard rendering images",
		zap.Int("number of images", len(imageList)),
		zap.Strings("images", imgNames),
	)

	tightCanvas, err := i.renderImages(ctx, canvas, imageList, jump)
	if err != nil {
		i.log.Error("error rendering images", zap.Error(err))
	}
	if i.config.ScrollMode.Load() && tightCanvas != nil {
		if len(gifList) > 1 {
			i.log.Warn("ignoring GIFs in imageboard while scroll mode is enabled")
		}
		return tightCanvas, nil
	}

	i.log.Debug("imageboard rendering gifs",
		zap.Int("number of gifs", len(gifList)),
		zap.Strings("images", gifNames),
	)
	if _, err := i.renderImages(ctx, canvas, gifList, jump); err != nil {
		i.log.Error("error rendering gifs", zap.Error(err))
	}

	if isJumping {
		i.Enabler().Store(i.priorJumpState.Load())
	}

	return nil, nil
}

func (i *ImageBoard) renderImages(ctx context.Context, canvas board.Canvas, images []*img, jump string) (board.Canvas, error) {
	preloader := make(map[string]chan struct{})

	jump = strings.ToLower(jump)

	preloadCh := make(chan *img)

	go func() {
		for p := range preloadCh {
			i.preloadLock.Lock()
			i.preloaded[p.path] = p
			i.preloadLock.Unlock()
		}
	}()

	wg := sync.WaitGroup{}
	defer func() {
		wg.Wait()
		close(preloadCh)
	}()

	zereodCanvas := rgbrender.ZeroedBounds(canvas.Bounds())

	preload := func(im *img) {
		defer wg.Done()
		img, err := i.getSizedImage(im.path, zereodCanvas, preloader[im.path])
		if err != nil {
			i.log.Error("failed to prepare image", zap.Error(err), zap.String("path", im.path))
			return
		}
		im.img = img
		select {
		case preloadCh <- im:
		default:
		}
	}

	preloadGif := func(preloadCtx context.Context, im *img) {
		defer wg.Done()
		img, err := i.getSizedGIF(preloadCtx, im.path, canvas.Bounds())
		if err != nil {
			i.log.Error("failed to prepare image", zap.Error(err), zap.String("path", im.path))
			return
		}
		im.gif = img
		select {
		case preloadCh <- im:
		default:
		}
	}

	if len(images) > 0 {
		wg.Add(1)
		preload(images[0])
	}

	var tightCanvas *scrcnvs.ScrollCanvas
	base, ok := canvas.(*scrcnvs.ScrollCanvas)
	if canvas.Scrollable() && i.config.ScrollMode.Load() && ok {
		var err error
		tightCanvas, err = scrcnvs.NewScrollCanvas(base.Matrix, i.log,
			scrcnvs.WithMergePadding(i.config.TightScrollPadding),
			scrcnvs.WithName("img"),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get tight scroll canvas for imageboard: %w", err)
		}
		tightCanvas.SetScrollDirection(scrcnvs.RightToLeft)
		base.SetScrollSpeed(i.config.scrollDelay)
		tightCanvas.SetScrollSpeed(i.config.scrollDelay)

		go tightCanvas.MatchScroll(ctx, base)
	}

IMAGES:
	for index, thisImg := range images {
		p := thisImg.path
		if !i.enabler.Enabled() {
			i.log.Warn("ImageBoard is disabled, not rendering")
			return nil, nil
		}

		nextIndex := index + 1

		if nextIndex < len(images) {
			if images[nextIndex].isGif {
				pCtx, pCancel := context.WithTimeout(ctx, preloaderTimeout)
				defer pCancel()
				wg.Add(1)
				go preloadGif(pCtx, images[nextIndex])
			} else {
				wg.Add(1)
				go preload(images[nextIndex])
			}
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

		pCtx, pCancel := context.WithTimeout(ctx, preloaderTimeout)
		defer pCancel()

		img, err := i.getPreloaded(pCtx, thisImg.path)
		if err != nil {
			return nil, err
		}

		if img.isGif {
			i.log.Debug("playing GIF", zap.String("path", p))
			gifCtx, gifCancel := context.WithTimeout(ctx, i.config.boardDelay)
			defer gifCancel()

			if err := rgbrender.PlayGIF(gifCtx, canvas, img.gif); err != nil {
				i.log.Error("GIF player failed", zap.Error(err))
			}

			if jump != "" {
				return nil, nil
			}

			continue IMAGES
		}

		i.log.Debug("playing image",
			zap.String("image", img.path),
		)

		align, err := rgbrender.AlignPosition(
			rgbrender.CenterCenter,
			zereodCanvas,
			img.img.Bounds().Dx(),
			img.img.Bounds().Dy(),
		)
		if err != nil {
			return nil, err
		}

		draw.Draw(canvas, align, img.img, image.Point{}, draw.Over)

		if i.config.ScrollMode.Load() && tightCanvas != nil {
			tightCanvas.AddCanvas(canvas)
			draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Over)
			if jump != "" {
				return tightCanvas, nil
			}
			continue IMAGES
		}

		if err := canvas.Render(ctx); err != nil {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, context.Canceled
		case <-time.After(i.config.boardDelay):
		}

		if jump != "" {
			return nil, nil
		}
	}

	if i.config.ScrollMode.Load() && tightCanvas != nil {
		return tightCanvas, nil
	}

	return nil, nil
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

func (i *ImageBoard) getSizedGIF(ctx context.Context, path string, bounds image.Rectangle) (*gif.GIF, error) {
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
