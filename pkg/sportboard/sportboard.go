package sportboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/logo"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
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
}

// Config ...
type Config struct {
	boardDelay        time.Duration
	TimeColor         color.Color
	ScoreColor        color.Color
	Enabled           bool           `json:"enabled"`
	BoardDelay        string         `json:"boardDelay"`
	FavoriteSticky    bool           `json:"favoriteSticky"`
	ScoreFont         *FontConfig    `json:"scoreFont"`
	TimeFont          *FontConfig    `json:"timeFont"`
	LogoConfigs       []*logo.Config `json:"logoConfigs"`
	WatchTeams        []string       `json:"watchTeams"`
	FavoriteTeams     []string       `json:"favoriteTeams"`
	HideFavoriteScore bool           `json:"hideFavoriteScore"`
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
			c.boardDelay = 20 * time.Second
		}
		c.boardDelay = d
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
	if _, err := s.api.GetScheduledGames(ctx, util.Today()); err != nil {
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
	return s.config.Enabled
}

// Render ...
func (s *SportBoard) Render(ctx context.Context, matrix rgb.Matrix) error {
	if !s.config.Enabled {
		s.log.Warn("skipping disabled board", zap.String("board", s.api.League()))
		return nil
	}
	canvas := rgb.NewCanvas(matrix)

	games, err := s.api.GetScheduledGames(ctx, util.Today())
	if err != nil {
		return err
	}

	s.log.Debug("scheduled games today",
		zap.Int("num games", len(games)),
		zap.String("today", util.Today().String()),
		zap.String("league", s.api.League()),
	)

	if len(games) == 0 {
		s.log.Debug("no scheduled games, not rendering", zap.String("league", s.api.League()))
		return nil
	}

	gameOver := false
	cached, hasCached := s.cachedLiveGames[games[0].GetID()]
	if !hasCached {
		s.log.Debug("no cached game data", zap.Int("game ID", games[0].GetID()))
	} else {
		gameOver, err = cached.IsComplete()
		if err != nil {
			s.log.Warn("failed to determine if game is over", zap.Int("game ID",
				games[0].GetID()),
				zap.Error(err),
			)
			gameOver = false
		}
	}

	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}

	if gameOver && hasCached && cached != nil {
		s.log.Debug("game is over, using cached data", zap.Int("game ID", games[0].GetID()))
	} else {
		// preload the first live game
		s.log.Debug("fetching live game data", zap.Int("game ID", games[0].GetID()))
		s.cachedLiveGames[games[0].GetID()], err = games[0].GetUpdate(ctx)
		if err != nil {
			s.log.Error("failed to get live game update", zap.Error(err), zap.Int("game ID", games[0].GetID()))
		}
	}

	preloader := make(map[int]chan struct{})
	preloader[games[0].GetID()] = make(chan struct{}, 1)
	preloader[games[0].GetID()] <- struct{}{}

	preloaderTimeout := s.config.boardDelay + (10 * time.Second)

	gameCtx, cancel := context.WithCancel(ctx)
	defer cancel()

OUTER:
	for gameIndex, game := range games {
		select {
		case <-ctx.Done():
			cancel()
			return context.Canceled
		case <-gameCtx.Done():
			return context.Canceled
		default:
		}

		if !s.config.Enabled {
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
			cancel()
			return context.Canceled
		case <-gameCtx.Done():
			return context.Canceled
		case <-preloader[game.GetID()]:
			s.log.Debug("preloader marked readt", zap.Int("game ID", game.GetID()))
		case <-time.After(preloaderTimeout):
			s.log.Warn("timed out waiting for preload",
				zap.Duration("timeout", preloaderTimeout),
				zap.Int("game ID", game.GetID()),
			)
		}

		liveGame, ok := s.cachedLiveGames[game.GetID()]
		if !ok {
			s.log.Warn("live game data no ready in time, UNDEFINED", zap.Int("game ID", game.GetID()))
			continue OUTER
		}
		if liveGame == nil {
			s.log.Warn("live game data no ready in time, NIL", zap.Int("game ID", game.GetID()))
			continue OUTER
		}

		awayTeam, err := liveGame.AwayTeam()
		if err != nil {
			return err
		}
		homeTeam, err := liveGame.HomeTeam()
		if err != nil {
			return err
		}

	INNER:
		for _, watchTeam := range s.config.WatchTeams {
			select {
			case <-ctx.Done():
				cancel()
				return context.Canceled
			case <-gameCtx.Done():
				return context.Canceled
			default:
			}

			team, err := s.api.TeamFromAbbreviation(ctx, watchTeam)
			if err != nil {
				return err
			}

			if awayTeam.GetID() != team.GetID() && homeTeam.GetID() != team.GetID() {
				continue INNER
			}

			isLive, err := liveGame.IsLive()
			if err != nil {
				s.log.Error("failed to determine if game is live", zap.Error(err))
			}
			isOver, err := liveGame.IsComplete()
			if err != nil {
				s.log.Error("failed to determine if game is complete", zap.Error(err))
			}

			_, err = s.RenderGameCounter(canvas, len(games), gameIndex, 1)
			if err != nil {
				return err
			}

			if isLive {
				if err := s.renderLiveGame(gameCtx, canvas, liveGame); err != nil {
					s.log.Error("failed to render live game", zap.Error(err))
					continue INNER
				}
			} else if isOver {
				if err := s.renderCompleteGame(gameCtx, canvas, liveGame); err != nil {
					s.log.Error("failed to render complete game", zap.Error(err))
					continue INNER
				}
			} else {
				if err := s.renderUpcomingGame(gameCtx, canvas, liveGame); err != nil {
					s.log.Error("failed to render upcomingh game", zap.Error(err))
					continue INNER
				}
			}

			select {
			case <-ctx.Done():
				cancel()
				return context.Canceled
			case <-gameCtx.Done():
				return context.Canceled
			case <-time.After(s.config.boardDelay):
			}

			continue OUTER
		}
	}

	return nil
}

// HasPriority ...
func (s *SportBoard) HasPriority() bool {
	return false
}

func (s *SportBoard) preloadLiveGame(ctx context.Context, game Game, preload chan struct{}) error {
	defer func() {
		select {
		case preload <- struct{}{}:
		default:
		}
	}()

	gameOver := false
	cached, hasCached := s.cachedLiveGames[game.GetID()]
	if hasCached {
		var err error
		gameOver, err = cached.IsComplete()
		if err != nil {
			gameOver = false
		}
	}

	if gameOver && hasCached && cached != nil {
		s.log.Debug("game is complete, not fetching any more data", zap.Int("game ID", game.GetID()))

		return nil
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
		} else {
			s.cachedLiveGames[game.GetID()] = g
			s.log.Debug("successfully set preloeader data", zap.Int("game ID", game.GetID()))
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		case <-time.After(10 * time.Second):
		}
	}
}
