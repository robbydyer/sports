package sportboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/rgbrender"
	"github.com/robbydyer/sports/pkg/util"
)

const maxAPITries = 3

// SportBoard implements board.Board
type SportBoard struct {
	config          *Config
	api             API
	cachedLiveGames map[int]Game
	logos           map[string]*logo.Logo
	log             *zap.Logger
	matrixBounds    image.Rectangle
	logoDrawCache   map[string]image.Image
	scoreWriter     *rgbrender.TextWriter
	scoreAlign      image.Rectangle
	timeWriter      *rgbrender.TextWriter
	timeAlign       image.Rectangle
	counter         image.Image
	sync.Mutex
}

// Todayer is a func that returns a string representing a date
// that will be used for determining "Today's" games.
// This is useful in testing what past days looked like
type Todayer func() time.Time

// Config ...
type Config struct {
	TodayFunc         Todayer
	boardDelay        time.Duration
	TimeColor         color.Color
	ScoreColor        color.Color
	Enabled           *atomic.Bool   `json:"enabled"`
	BoardDelay        string         `json:"boardDelay"`
	FavoriteSticky    *atomic.Bool   `json:"favoriteSticky"`
	ScoreFont         *FontConfig    `json:"scoreFont"`
	TimeFont          *FontConfig    `json:"timeFont"`
	LogoConfigs       []*logo.Config `json:"logoConfigs"`
	WatchTeams        []string       `json:"watchTeams"`
	FavoriteTeams     []string       `json:"favoriteTeams"`
	HideFavoriteScore *atomic.Bool   `json:"hideFavoriteScore"`
}

// FontConfig ...
type FontConfig struct {
	Size      float64 `json:"size"`
	LineSpace float64 `json:"lineSpace"`
}

// API ...
type API interface {
	GetTeams(ctx context.Context) ([]Team, error)
	TeamFromAbbreviation(ctx context.Context, abbreviation string) (Team, error)
	GetScheduledGames(ctx context.Context, date time.Time) ([]Game, error)
	DateStr(d time.Time) string
	League() string
	HTTPPathPrefix() string
	GetLogo(ctx context.Context, logoKey string, logoConf *logo.Config, bounds image.Rectangle) (*logo.Logo, error)
	AllTeamAbbreviations() []string
}

// Team ...
type Team interface {
	GetID() int
	GetName() string
	GetAbbreviation() string
	Score() int
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

	if c.ScoreFont == nil {
		c.ScoreFont = &FontConfig{
			Size:      16,
			LineSpace: 0,
		}
	}
	if c.TimeFont == nil {
		c.TimeFont = &FontConfig{
			Size:      8,
			LineSpace: 0,
		}
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
}

// New ...
func New(ctx context.Context, api API, bounds image.Rectangle, logger *zap.Logger, config *Config) (*SportBoard, error) {
	s := &SportBoard{
		config:          config,
		api:             api,
		logos:           make(map[string]*logo.Logo),
		log:             logger,
		logoDrawCache:   make(map[string]image.Image),
		matrixBounds:    bounds,
		cachedLiveGames: make(map[int]Game),
	}

	if s.config.boardDelay < 10*time.Second {
		s.log.Warn("cannot set sportboard delay below 10 sec")
		s.config.boardDelay = 10 * time.Second
	}

	if s.config.TodayFunc == nil {
		s.config.TodayFunc = util.Today
	}

	if len(config.WatchTeams) == 0 {
		if len(config.FavoriteTeams) > 0 {
			config.WatchTeams = config.FavoriteTeams
		} else {
			config.WatchTeams = s.api.AllTeamAbbreviations()
		}
	}

	for _, i := range config.WatchTeams {
		if strings.ToUpper(i) == "ALL" {
			config.WatchTeams = s.api.AllTeamAbbreviations()
		}
	}

	if _, err := s.api.GetTeams(ctx); err != nil {
		return nil, err
	}
	if _, err := s.api.GetScheduledGames(ctx, s.config.TodayFunc()); err != nil {
		return nil, err
	}

	c := cron.New()

	if _, err := c.AddFunc("0 4 * * *", s.cacheClear); err != nil {
		return nil, fmt.Errorf("failed to set cron for cacheClear: %w", err)
	}
	c.Start()

	return s, nil
}

func (s *SportBoard) cacheClear() {
	s.log.Warn("Clearing cached live games")
	for k := range s.cachedLiveGames {
		delete(s.cachedLiveGames, k)
	}
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

// Render ...
func (s *SportBoard) Render(ctx context.Context, canvas board.Canvas) error {
	if !s.config.Enabled.Load() {
		s.log.Warn("skipping disabled board", zap.String("board", s.api.League()))
		return nil
	}

	allGames, err := s.api.GetScheduledGames(ctx, s.config.TodayFunc())
	if err != nil {
		return err
	}

	// Determine which games are watched so that the game counter is accurate

	var games []Game
OUTER:
	for _, game := range allGames {
		home, err := game.HomeTeam()
		if err != nil {
			return err
		}
		away, err := game.AwayTeam()
		if err != nil {
			return err
		}
		for _, watchTeam := range s.config.WatchTeams {
			team, err := s.api.TeamFromAbbreviation(ctx, watchTeam)
			if err != nil {
				return err
			}

			if home.GetID() == team.GetID() || away.GetID() == team.GetID() {
				games = append(games, game)
				continue OUTER
			}
		}
	}

	s.log.Debug("scheduled games today",
		zap.Int("watched games", len(games)),
		zap.Int("num games", len(allGames)),
		zap.String("today", s.config.TodayFunc().String()),
		zap.String("league", s.api.League()),
	)

	if len(games) == 0 {
		s.log.Debug("no scheduled games, not rendering", zap.String("league", s.api.League()))
		return nil
	}

	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}

	preloader := make(map[int]chan struct{})
	preloader[games[0].GetID()] = make(chan struct{}, 1)

	if err := s.preloadLiveGame(ctx, games[0], preloader[games[0].GetID()]); err != nil {
		s.log.Error("error while loading live game data for first game", zap.Error(err))
	}

	preloaderTimeout := s.config.boardDelay + (10 * time.Second)

	for gameIndex, game := range games {
		select {
		case <-ctx.Done():
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
				if err := s.preloadLiveGame(ctx, games[nextGameIndex], preloader[nextID]); err != nil {
					s.log.Error("error while preloading next game", zap.Error(err))
				}
			}()
		}

		// Wait for the preloader to finish getting data, but with a timeout.
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-preloader[game.GetID()]:
			s.log.Debug("preloader marked ready", zap.Int("game ID", game.GetID()))
		case <-time.After(preloaderTimeout):
			s.log.Warn("timed out waiting for preload",
				zap.Duration("timeout", preloaderTimeout),
				zap.Int("game ID", game.GetID()),
			)
		}

		cachedGame, ok := s.cachedLiveGames[game.GetID()]
		if !ok {
			s.log.Warn("live game data not ready in time, UNDEFINED", zap.Int("game ID", game.GetID()))
			continue
		}

		if cachedGame == nil {
			s.log.Warn("live game data not ready in time, NIL", zap.Int("game ID", game.GetID()))
			continue
		}

		_, err = s.RenderGameCounter(canvas, len(games), gameIndex, 1)
		if err != nil {
			return err
		}

		if err := s.renderGame(ctx, canvas, cachedGame); err != nil {
			return err
		}
	}

	return nil
}

func (s *SportBoard) renderGame(ctx context.Context, canvas board.Canvas, liveGame Game) error {
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

	if isLive {
		if err := s.renderLiveGame(ctx, canvas, liveGame); err != nil {
			return fmt.Errorf("failed to render live game: %w", err)
		}
	} else if isOver {
		if err := s.renderCompleteGame(ctx, canvas, liveGame); err != nil {
			return fmt.Errorf("failed to render complete game: %w", err)
		}
	} else {
		if err := s.renderUpcomingGame(ctx, canvas, liveGame); err != nil {
			return fmt.Errorf("failed to render upcoming game: %w", err)
		}
	}

	select {
	case <-ctx.Done():
		return context.Canceled
	case <-time.After(s.config.boardDelay):
	}

	return nil
}

// HasPriority ...
func (s *SportBoard) HasPriority() bool {
	return false
}

func (s *SportBoard) preloadLiveGame(ctx context.Context, game Game, preload chan struct{}) error {
	s.Lock()
	defer s.Unlock()

	defer func() {
		select {
		case preload <- struct{}{}:
		default:
		}
	}()

	gameOver := false
	cached, hasCached := s.cachedLiveGames[game.GetID()]

	// If a game is over or is more than 30min away from scheduled start,
	// let's not load live game data.
	if hasCached && cached != nil {
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

		s.cachedLiveGames[game.GetID()] = g

		s.log.Debug("successfully set preloader data", zap.Int("game ID", game.GetID()))
		return nil
	}
}
