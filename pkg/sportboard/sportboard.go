package sportboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/twitchtv/twirp"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	pb "github.com/robbydyer/sports/internal/proto/sportboard"
	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
	"github.com/robbydyer/sports/pkg/statboard"
	"github.com/robbydyer/sports/pkg/twirphelpers"
	"github.com/robbydyer/sports/pkg/util"
)

type side int

const (
	maxAPITries      = 3
	left        side = iota
	right
)

// SportBoard implements board.Board
type SportBoard struct {
	config          *Config
	api             API
	cachedLiveGames map[int]Game
	logos           map[string]*logo.Logo
	log             *zap.Logger
	logoDrawCache   map[string]image.Image
	scoreWriters    map[string]*rgbrender.TextWriter
	timeWriters     map[string]*rgbrender.TextWriter
	teamInfoWidths  map[string]map[string]int
	watchTeams      []string
	teamInfoLock    sync.RWMutex
	drawLock        sync.RWMutex
	logoLock        sync.RWMutex
	enablerLock     sync.Mutex
	cancelBoard     chan struct{}
	previousScores  []*previousScore
	prevScoreLock   sync.Mutex
	rpcServer       pb.TwirpServer
	sync.Mutex
}

// Todayer is a func that returns a string representing a date
// that will be used for determining "Today's" games.
// This is useful in testing what past days looked like
type Todayer func() []time.Time

// Config ...
type Config struct {
	TodayFunc            Todayer
	boardDelay           time.Duration
	scrollDelay          time.Duration
	TimeColor            color.Color
	ScoreColor           color.Color
	Enabled              *atomic.Bool      `json:"enabled"`
	BoardDelay           string            `json:"boardDelay"`
	FavoriteSticky       *atomic.Bool      `json:"favoriteSticky"`
	ScoreFont            *FontConfig       `json:"scoreFont"`
	TimeFont             *FontConfig       `json:"timeFont"`
	LogoConfigs          []*logo.Config    `json:"logoConfigs"`
	WatchTeams           []string          `json:"watchTeams"`
	FavoriteTeams        []string          `json:"favoriteTeams"`
	HideFavoriteScore    *atomic.Bool      `json:"hideFavoriteScore"`
	ShowRecord           *atomic.Bool      `json:"showRecord"`
	GridCols             int               `json:"gridCols"`
	GridRows             int               `json:"gridRows"`
	GridPadRatio         float64           `json:"gridPadRatio"`
	MinimumGridWidth     int               `json:"minimumGridWidth"`
	MinimumGridHeight    int               `json:"minimumGridHeight"`
	Stats                *statboard.Config `json:"stats"`
	ScrollMode           *atomic.Bool      `json:"scrollMode"`
	TightScroll          *atomic.Bool      `json:"tightScroll"`
	TightScrollPadding   int               `json:"tightScrollPadding"`
	ScrollDelay          string            `json:"scrollDelay"`
	GamblingSpread       *atomic.Bool      `json:"showOdds"`
	ShowNoScheduledLogo  *atomic.Bool      `json:"showNotScheduled"`
	ScoreHighlightRepeat *int              `json:"scoreHighlightRepeat"`
	OnTimes              []string          `json:"onTimes"`
	OffTimes             []string          `json:"offTimes"`
}

// FontConfig ...
type FontConfig struct {
	Size      float64 `json:"size"`
	LineSpace float64 `json:"lineSpace"`
}

// API ...
type API interface {
	GetTeams(ctx context.Context) ([]Team, error)
	TeamFromID(ctx context.Context, abbreviation string) (Team, error)
	GetScheduledGames(ctx context.Context, date []time.Time) ([]Game, error)
	DateStr(d time.Time) string
	League() string
	HTTPPathPrefix() string
	GetLogo(ctx context.Context, logoKey string, logoConf *logo.Config, bounds image.Rectangle) (*logo.Logo, error)
	// AllTeamAbbreviations() []string
	GetWatchTeams(teams []string, season string) []string
	TeamRecord(ctx context.Context, team Team, season string) string
	TeamRank(ctx context.Context, team Team, season string) string
	CacheClear(ctx context.Context)
	// LeagueLogo(ctx context.Context) (*logo.Logo, error)
}

// Team ...
type Team interface {
	GetID() string
	GetName() string
	GetAbbreviation() string
	GetDisplayName() string
	Score() int
	ConferenceName() string
}

// Game ...
type Game interface {
	GetID() int
	GetLink() (string, error)
	IsLive() (bool, error)
	IsComplete() (bool, error)
	IsPostponed() (bool, error)
	HomeTeam() (Team, error)
	AwayTeam() (Team, error)
	GetQuarter() (string, error) // Or a period, inning
	GetClock() (string, error)
	GetUpdate(ctx context.Context) (Game, error)
	GetStartTime(ctx context.Context) (time.Time, error)
	GetOdds() (string, string, error)
}

// SetDefaults sets config defaults
func (c *Config) SetDefaults() {
	if c.BoardDelay != "" {
		d, err := time.ParseDuration(c.BoardDelay)
		if err != nil {
			c.boardDelay = 10 * time.Second
		}
		c.boardDelay = d
	} else {
		c.boardDelay = 10 * time.Second
	}

	if c.TimeColor == nil {
		c.TimeColor = color.White
	}
	if c.ScoreColor == nil {
		c.ScoreColor = color.White
	}
	if c.HideFavoriteScore == nil {
		c.HideFavoriteScore = atomic.NewBool(false)
	}
	if c.FavoriteSticky == nil {
		c.FavoriteSticky = atomic.NewBool(false)
	}
	if c.Enabled == nil {
		c.Enabled = atomic.NewBool(false)
	}
	if c.ShowRecord == nil {
		c.ShowRecord = atomic.NewBool(false)
	}
	if c.ScrollMode == nil {
		c.ScrollMode = atomic.NewBool(false)
	}
	if c.TightScroll == nil {
		c.TightScroll = atomic.NewBool(false)
	}
	if c.GamblingSpread == nil {
		c.GamblingSpread = atomic.NewBool(false)
	}
	if c.ShowNoScheduledLogo == nil {
		c.ShowNoScheduledLogo = atomic.NewBool(false)
	}
	if c.MinimumGridWidth == 0 {
		c.MinimumGridWidth = 64
	}
	if c.MinimumGridHeight == 0 {
		c.MinimumGridHeight = 64
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

	if c.ScoreHighlightRepeat == nil {
		p := 3
		c.ScoreHighlightRepeat = &p
	}
}

// New ...
func New(ctx context.Context, api API, bounds image.Rectangle, logger *zap.Logger, config *Config) (*SportBoard, error) {
	s := &SportBoard{
		config:          config,
		api:             api,
		logos:           make(map[string]*logo.Logo),
		log:             logger,
		logoDrawCache:   make(map[string]image.Image),
		cachedLiveGames: make(map[int]Game),
		timeWriters:     make(map[string]*rgbrender.TextWriter),
		scoreWriters:    make(map[string]*rgbrender.TextWriter),
		cancelBoard:     make(chan struct{}),
		teamInfoWidths:  make(map[string]map[string]int),
	}

	if s.config.boardDelay < 10*time.Second {
		s.log.Warn("cannot set sportboard delay below 10 sec")
		s.config.boardDelay = 10 * time.Second
	}

	if s.config.TodayFunc == nil {
		s.config.TodayFunc = util.Today
		if strings.ToLower(s.api.League()) == "ncaaf" {
			f := func() []time.Time {
				return util.NCAAFToday(util.Today()[0])
			}
			s.config.TodayFunc = f
		}
		if strings.ToLower(s.api.League()) == "nfl" {
			f := func() []time.Time {
				return util.NFLToday(util.Today()[0])
			}
			s.config.TodayFunc = f
		}
	}

	if len(config.WatchTeams) == 0 {
		config.WatchTeams = []string{"ALL"}
	}

	c := cron.New()

	if _, err := c.AddFunc("0 4 * * *", s.cacheClear); err != nil {
		return nil, fmt.Errorf("failed to set cron for cacheClear: %w", err)
	}

	svr := &Server{
		board: s,
	}
	prfx := s.api.HTTPPathPrefix()
	if !strings.HasPrefix(prfx, "/") {
		prfx = fmt.Sprintf("/%s", prfx)
	}
	s.rpcServer = pb.NewSportServer(svr,
		twirp.WithServerPathPrefix(prfx),
		twirp.ChainHooks(
			twirphelpers.GetDefaultHooks(s, s.log),
		),
	)

	for _, on := range config.OnTimes {
		s.log.Info("sportboard will be schedule to turn on",
			zap.String("league", s.api.League()),
			zap.String("turn on", on),
		)
		_, err := c.AddFunc(on, func() {
			s.log.Info("sportboard turning on",
				zap.String("league", s.api.League()),
			)
			s.Enable()
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add cron for sportboard: %w", err)
		}
	}

	for _, off := range config.OffTimes {
		s.log.Info("sportboard will be schedule to turn off",
			zap.String("league", s.api.League()),
			zap.String("turn on", off),
		)
		_, err := c.AddFunc(off, func() {
			s.log.Info("sportboard turning off",
				zap.String("league", s.api.League()),
			)
			s.Disable()
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add cron for sportboard: %w", err)
		}
	}

	c.Start()

	return s, nil
}

func (s *SportBoard) cacheClear() {
	s.Lock()
	defer s.Unlock()
	s.drawLock.Lock()
	defer s.drawLock.Unlock()
	s.logoLock.Lock()
	defer s.logoLock.Unlock()
	s.teamInfoLock.Lock()
	defer s.teamInfoLock.Unlock()
	s.prevScoreLock.Lock()
	defer s.prevScoreLock.Unlock()

	s.log.Warn("Clearing cache")
	for k := range s.cachedLiveGames {
		delete(s.cachedLiveGames, k)
	}
	for k := range s.logoDrawCache {
		delete(s.logoDrawCache, k)
	}
	for k := range s.logos {
		delete(s.logos, k)
	}
	for k := range s.teamInfoWidths {
		delete(s.teamInfoWidths, k)
	}
	s.previousScores = []*previousScore{}
}

// Name ...
func (s *SportBoard) Name() string {
	if l := s.api.League(); l != "" {
		return l
	}
	return "SportBoard"
}

// Enabled ...
func (s *SportBoard) Enabled() bool {
	return s.config.Enabled.Load()
}

// Enable ...
func (s *SportBoard) Enable() {
	s.config.Enabled.Store(true)
}

// Disable ...
func (s *SportBoard) Disable() {
	s.config.Enabled.Store(false)
}

// ScrollMode ...
func (s *SportBoard) ScrollMode() bool {
	return s.config.ScrollMode.Load()
}

// GridSize returns the column width and row height for a grid layout. 0 is returned for
// both if the canvas is too small for a grid.
func (s *SportBoard) GridSize(bounds image.Rectangle) (int, int) {
	width := 0
	height := 0
	if s.config.GridCols > 0 {
		pixW := bounds.Dx() / s.config.GridCols
		if pixW > s.config.MinimumGridWidth {
			width = s.config.GridCols
		} else {
			width = bounds.Dx() / s.config.MinimumGridWidth
		}
	}
	if s.config.GridRows > 0 {
		pixH := bounds.Dy() / s.config.GridRows
		if pixH > s.config.MinimumGridHeight {
			height = s.config.GridRows
		} else {
			height = bounds.Dy() / s.config.MinimumGridHeight
		}
	}

	if width > 0 && height < 1 {
		height = 1
	}
	if height > 0 && width < 1 {
		width = 1
	}

	return width, height
}

func (s *SportBoard) enablerCancel(ctx context.Context, cancel context.CancelFunc) {
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
func (s *SportBoard) Render(ctx context.Context, canvas board.Canvas) error {
	if !s.config.Enabled.Load() {
		s.log.Warn("skipping disabled board", zap.String("board", s.api.League()))
		return nil
	}

	s.logCanvas(canvas, "sportboard Render() called canvas")

	boardCtx, boardCancel := context.WithCancel(ctx)
	defer boardCancel()

	go s.enablerCancel(boardCtx, boardCancel)

	loadCtx, loadCancel := context.WithTimeout(boardCtx, 10*time.Minute)
	defer loadCancel()
	go s.renderLoading(loadCtx, canvas)

	allGames, err := s.api.GetScheduledGames(boardCtx, s.config.TodayFunc())
	if err != nil {
		s.log.Error("failed to get scheduled games",
			zap.String("league", s.api.League()),
			zap.Error(err),
		)
		return err
	}

	if len(allGames) < 1 {
		s.log.Debug("no games scheduled",
			zap.String("league", s.api.League()),
		)
		return nil
	}

	if _, err := s.api.GetTeams(ctx); err != nil {
		return err
	}

	// Determine which games are watched so that the game counter is accurate
	if len(s.watchTeams) < 1 {
		s.log.Debug("fetching watch teams",
			zap.String("league", s.api.League()),
		)
		s.watchTeams = s.api.GetWatchTeams(s.config.WatchTeams, s.season())
		s.log.Debug("watch teams",
			zap.String("league", s.api.League()),
			zap.Strings("teams", s.watchTeams),
		)
	}

	var games []Game
OUTER:
	for _, game := range allGames {
		home, err := game.HomeTeam()
		if err != nil {
			s.log.Error("failed to get home team", zap.Error(err))
			continue OUTER
		}
		away, err := game.AwayTeam()
		if err != nil {
			s.log.Error("failed to get away team", zap.Error(err))
			continue OUTER
		}
		for _, watchTeamID := range s.watchTeams {
			if home.GetID() == watchTeamID || away.GetID() == watchTeamID {
				games = append(games, game)

				// Ensures the daily data for this team has been fetched
				_ = s.api.TeamRecord(boardCtx, home, s.season())
				_ = s.api.TeamRecord(boardCtx, away, s.season())
				continue OUTER
			}
		}
	}

	var todays []string
	for _, t := range s.config.TodayFunc() {
		todays = append(todays, t.String())
	}
	s.log.Debug("scheduled games today",
		zap.Int("watched games", len(games)),
		zap.Int("num games", len(allGames)),
		zap.Strings("todays", todays),
		zap.String("league", s.api.League()),
	)

	select {
	case <-boardCtx.Done():
		return context.Canceled
	default:
	}

	if (!s.config.ScrollMode.Load()) || (!canvas.Scrollable()) {
		bounds := rgbrender.ZeroedBounds(canvas.Bounds())
		w, h := s.GridSize(bounds)
		s.log.Debug("calculated grid size",
			zap.Int("cols", w),
			zap.Int("rows", h),
			zap.Int("canvas width", canvas.Bounds().Dx()),
			zap.Int("canvas height", canvas.Bounds().Dy()),
		)
		if w > 1 || h > 1 {
			width := bounds.Dx() / w
			height := bounds.Dy() / h
			s.log.Debug("rendering board as grid",
				zap.Int("cols", w),
				zap.Int("rows", h),
				zap.Int("cell width", width),
				zap.Int("cell height", height),
			)
			loadCancel()
			return s.renderGrid(boardCtx, canvas, games, w, h)
		}
	}

	s.logCanvas(canvas, "sportboard Render() called canvas after grid")
	if len(games) < 1 {
		s.log.Debug("no scheduled games, not rendering", zap.String("league", s.api.League()))
		if !s.config.ShowNoScheduledLogo.Load() {
			loadCancel()
			return nil
		}

		loadCancel()
		return s.renderNoScheduled(boardCtx, canvas)
	}

	preloader := make(map[int]chan struct{})
	preloader[games[0].GetID()] = make(chan struct{}, 1)

	if err := s.preloadLiveGame(ctx, games[0], preloader[games[0].GetID()]); err != nil {
		s.log.Error("error while loading live game data for first game", zap.Error(err))
	}

	preloaderTimeout := s.config.boardDelay + (10 * time.Second)

	defer func() { _ = canvas.Clear() }()

	var tightCanvas *rgbmatrix.ScrollCanvas
	if canvas.Scrollable() && s.config.TightScroll.Load() {
		base, ok := canvas.(*rgbmatrix.ScrollCanvas)
		if !ok {
			return fmt.Errorf("wat")
		}

		var err error
		tightCanvas, err = rgbmatrix.NewScrollCanvas(base.Matrix, s.log)
		if err != nil {
			return fmt.Errorf("failed to get tight scroll canvas: %w", err)
		}

		tightCanvas.SetScrollDirection(rgbmatrix.RightToLeft)
		tightCanvas.SetScrollSpeed(s.config.scrollDelay)
	} else if canvas.Scrollable() && s.config.ScrollMode.Load() {
		scroll, ok := canvas.(*rgbmatrix.ScrollCanvas)
		if ok {
			orig := scroll.GetScrollSpeed()
			defer func() { scroll.SetScrollSpeed(orig) }()
			s.log.Debug("setting scroll delay",
				zap.String("league", s.api.League()),
				zap.String("delay", s.config.scrollDelay.String()),
			)
			scroll.SetScrollSpeed(s.config.scrollDelay)
		}
	}

GAMES:
	for gameIndex, game := range games {
		select {
		case <-boardCtx.Done():
			return context.Canceled
		default:
		}

		if !s.config.Enabled.Load() {
			s.log.Warn("skipping disabled board", zap.String("board", s.api.League()))
			return nil
		}

		nextGameIndex := gameIndex + 1
		s.log.Debug("current game", zap.Int("index", gameIndex), zap.Int("game ID", game.GetID()))
		// preload data for the next game
		if nextGameIndex < len(games) {
			nextID := games[nextGameIndex].GetID()
			preloader[nextID] = make(chan struct{}, 1)
			go func() {
				if err := s.preloadLiveGame(boardCtx, games[nextGameIndex], preloader[nextID]); err != nil {
					s.log.Error("error while preloading next game", zap.Error(err))
				}
			}()
		}

		// Wait for the preloader to finish getting data, but with a timeout.
		select {
		case <-boardCtx.Done():
			return context.Canceled
		case <-preloader[game.GetID()]:
			s.log.Debug("preloader marked ready", zap.Int("game ID", game.GetID()))
		case <-time.After(preloaderTimeout):
			s.log.Warn("timed out waiting for preload",
				zap.Duration("timeout", preloaderTimeout),
				zap.Int("game ID", game.GetID()),
				zap.String("League", s.api.League()),
			)
		}

		cachedGame, ok := s.cachedLiveGames[game.GetID()]
		if !ok {
			s.log.Warn("live game data not ready in time, UNDEFINED", zap.Int("game ID", game.GetID()))
			continue GAMES
		}

		if cachedGame == nil {
			s.log.Warn("live game data not ready in time, NIL", zap.Int("game ID", game.GetID()))
			continue GAMES
		}

		counter, err := s.RenderGameCounter(canvas, len(games), gameIndex)
		if err != nil {
			s.log.Error("failed to render game counter", zap.Error(err))
		}

		loadCancel()

		if err := s.renderGame(boardCtx, canvas, cachedGame, counter); err != nil {
			s.log.Error("failed to render sportboard game", zap.Error(err))
			continue GAMES
		}

		if canvas.Scrollable() && s.config.TightScroll.Load() && tightCanvas != nil {
			s.log.Debug("adding to tight scroll canvas",
				zap.Int("total width", tightCanvas.Width()),
			)
			tightCanvas.AddCanvas(canvas)

			draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Over)
			continue GAMES
		}

		if err := canvas.Render(boardCtx); err != nil {
			s.log.Error("failed to render", zap.Error(err))
			continue GAMES
		}

		if !s.config.ScrollMode.Load() {
			select {
			case <-boardCtx.Done():
				return context.Canceled
			case <-time.After(s.config.boardDelay):
			}
		}
	}

	if canvas.Scrollable() && tightCanvas != nil {
		tightCanvas.Merge(s.config.TightScrollPadding)
		s.log.Debug("rendering tight scroll canvas")
		return tightCanvas.Render(boardCtx)
	}

	return nil
}

func (s *SportBoard) renderGrid(ctx context.Context, canvas board.Canvas, games []Game, cols int, rows int) error {
	if len(games) < 1 {
		return nil
	}
	var opts []rgbrender.GridOption
	if s.config.GridPadRatio > 0 {
		opts = append(opts, rgbrender.WithPadding(s.config.GridPadRatio))
	}
	opts = append(opts, rgbrender.WithUniformCells())
	grid, err := rgbrender.NewGrid(
		canvas,
		cols,
		rows,
		s.log,
		opts...,
	)
	if err != nil {
		return err
	}

	numCells := len(grid.Cells())
	numGrids := int(math.Ceil(float64(len(games)) / float64(numCells)))
	totalDelay := int(s.config.boardDelay.Seconds()) * len(games)

	if numGrids == 0 {
		numGrids = 1
	}
	gridDelay := time.Duration(totalDelay/numGrids) * time.Second

	s.log.Debug("setting grid delay", zap.Float64("seconds", gridDelay.Seconds()))

	gridIndex := 0
	for i := 0; i < len(games); i++ {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}
		endIndex := i + numCells
		if endIndex > len(games)-1 {
			endIndex = len(games)
		}
		s.log.Debug("grid layout",
			zap.Int("game start index", i),
			zap.Int("game end index", endIndex),
		)
		counter, err := s.RenderGameCounter(canvas, numGrids, gridIndex)
		if err != nil {
			return err
		}
		if err := s.doGrid(ctx, grid, canvas, games[i:endIndex], counter); err != nil {
			return err
		}
		i += numCells - 1
		if err := grid.Clear(); err != nil {
			return err
		}
		gridIndex++

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(gridDelay):
		}
	}

	return nil
}

func (s *SportBoard) doGrid(ctx context.Context, grid *rgbrender.Grid, canvas board.Canvas, games []Game, counter image.Image) error {
	// Fetch all the scores
	wg := sync.WaitGroup{}

	for _, game := range games {
		wg.Add(1)
		go func(game Game) {
			defer wg.Done()
			p := make(chan struct{}, 1)
			if err := s.preloadLiveGame(ctx, game, p); err != nil {
				s.log.Error("error while loading live game", zap.Error(err), zap.Int("id", game.GetID()))
			}
		}(game)
	}

	preload := make(chan struct{})

	go func() {
		defer close(preload)
		wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return context.Canceled
	case <-preload:
	}

	gameWg := sync.WaitGroup{}
	index := -1
	for _, game := range games {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}
		index++

		gameWg.Add(1)
		go func(game Game, index int) {
			defer gameWg.Done()
			liveGame, err := s.getCachedGame(game.GetID())
			if err != nil {
				s.log.Error("failed to get cached game", zap.Error(err))
				return
			}
			cell, err := grid.Cell(index)
			if err != nil {
				s.log.Error("invalid cell index", zap.Int("index", index))
				return
			}

			if err := s.renderGame(ctx, cell.Canvas, liveGame, nil); err != nil {
				s.log.Error("failed to render game in grid", zap.Error(err))
				return
			}
		}(game, index)
	}

	preload = make(chan struct{})

	go func() {
		defer close(preload)
		gameWg.Wait()
	}()

	select {
	case <-ctx.Done():
		return context.Canceled
	case <-preload:
	}

	if err := grid.DrawToBase(canvas); err != nil {
		return err
	}

	grid.FillPadded(canvas, color.White)

	draw.Draw(canvas, canvas.Bounds(), counter, image.Point{}, draw.Over)

	return canvas.Render(ctx)
}

func (s *SportBoard) renderGame(ctx context.Context, canvas board.Canvas, liveGame Game, counter image.Image) error {
	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}

	isLive, err := liveGame.IsLive()
	if err != nil {
		return fmt.Errorf("failed to determine if game is live: %w", err)
	}
	isOver, err := liveGame.IsComplete()
	if err != nil {
		return fmt.Errorf("failed to determine if game is complete: %w", err)
	}

	s.logCanvas(canvas, "sportboard renderGame canvas")

	if isLive {
		if err := s.renderLiveGame(ctx, canvas, liveGame, counter); err != nil {
			return fmt.Errorf("failed to render live game: %w", err)
		}
	} else if isOver {
		if err := s.renderCompleteGame(ctx, canvas, liveGame, counter); err != nil {
			return fmt.Errorf("failed to render complete game: %w", err)
		}
	} else {
		if err := s.renderUpcomingGame(ctx, canvas, liveGame, counter); err != nil {
			return fmt.Errorf("failed to render upcoming game: %w", err)
		}
	}

	return nil
}

// HasPriority ...
func (s *SportBoard) HasPriority() bool {
	return false
}

func (s *SportBoard) setCachedGame(key int, game Game) {
	s.Lock()
	defer s.Unlock()
	s.cachedLiveGames[key] = game
}

func (s *SportBoard) getCachedGame(key int) (Game, error) {
	s.Lock()
	defer s.Unlock()
	g, ok := s.cachedLiveGames[key]
	if ok {
		return g, nil
	}

	return nil, fmt.Errorf("no cache for game %d", key)
}

func (s *SportBoard) preloadLiveGame(ctx context.Context, game Game, preload chan struct{}) error {
	defer func() {
		select {
		case preload <- struct{}{}:
		default:
		}
	}()

	gameOver := false
	cached, err := s.getCachedGame(game.GetID())

	// If a game is over or is more than 30min away from scheduled start,
	// let's not load live game data.
	if err == nil && cached != nil {
		var err error

		gameOver, err = cached.IsComplete()
		if err != nil {
			gameOver = false
		}

		if gameOver {
			s.log.Debug("game is complete, not fetching any more data", zap.Int("game ID", game.GetID()))

			return nil
		}

		startTime, err := cached.GetStartTime(ctx)
		if err != nil {
			return fmt.Errorf("failed to determine start time of game: %w", err)
		}

		if time.Until(startTime).Minutes() > 30 {
			s.log.Warn("game has not started, not fetching live data yet",
				zap.Int("game ID", cached.GetID()),
				zap.Float64("min until start", time.Until(startTime).Minutes()),
			)

			return nil
		}
	}

	s.log.Debug("preloading live game data", zap.Int("game ID", game.GetID()))
	tries := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		default:
		}

		if tries > maxAPITries {
			return fmt.Errorf("failed API call %d times", maxAPITries)
		}
		tries++

		g, err := game.GetUpdate(ctx)
		if err != nil {
			s.log.Error("api call to get live game failed", zap.Int("attempt", tries), zap.Error(err))
			select {
			case <-ctx.Done():
				return fmt.Errorf("context canceled")
			case <-time.After(10 * time.Second):
			}
			continue
		}

		s.setCachedGame(game.GetID(), g)

		s.log.Debug("successfully set preloader data", zap.Int("game ID", game.GetID()))
		return nil
	}
}
