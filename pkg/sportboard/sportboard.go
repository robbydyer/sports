package sportboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	"github.com/robbydyer/sports/pkg/logo"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
	"github.com/robbydyer/sports/pkg/util"
)

const maxAPITries = 3

// SportBoard implements board.Board
type SportBoard struct {
	config            *Config
	api               API
	teams             map[int]Team
	cachedLiveGames   map[int]Game
	logos             map[string]*logo.Logo
	log               *log.Logger
	matrixBounds      image.Rectangle
	logoDrawCache     map[string]image.Image
	logoSourceCache   map[string]image.Image
	liveGamePreloader map[int]Game
	scoreWriter       *rgbrender.TextWriter
	scoreAlign        image.Rectangle
	timeWriter        *rgbrender.TextWriter
	timeAlign         image.Rectangle
	counter           image.Image
}

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

type FontConfig struct {
	Size      float64 `json:"size"`
	LineSpace float64 `json:"lineSpace"`
}

type API interface {
	GetTeams(ctx context.Context) ([]Team, error)
	TeamFromAbbreviation(ctx context.Context, abbreviation string) (Team, error)
	GetScheduledGames(ctx context.Context, date time.Time) ([]Game, error)
	DateStr(d time.Time) string
	League() string
	GetLogo(logoKey string, logoConf *logo.Config, bounds image.Rectangle) (*logo.Logo, error)
	AllTeamAbbreviations() []string
}

type Team interface {
	GetID() int
	GetName() string
	GetAbbreviation() string
	Score() int
}

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

func New(ctx context.Context, api API, bounds image.Rectangle, logger *log.Logger, config *Config) (*SportBoard, error) {
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

	c.AddFunc("0 4 * * *", s.CacheClear)
	c.Start()

	return s, nil
}

func (s *SportBoard) CacheClear() {
	s.log.Warn("Clearing cached live games")
	for k, _ := range s.cachedLiveGames {
		delete(s.cachedLiveGames, k)
	}
}

func (s *SportBoard) Name() string {
	if l := s.api.League(); l != "" {
		return l
	}
	return "SportBoard"
}

func (s *SportBoard) Enabled() bool {
	return s.config.Enabled
}

func (s *SportBoard) Render(ctx context.Context, matrix rgb.Matrix) error {
	if !s.config.Enabled {
		s.log.Warnf("%s board is not enabled, skipping", s.api.League())
		return nil
	}
	canvas := rgb.NewCanvas(matrix)

	games, err := s.api.GetScheduledGames(ctx, util.Today())
	if err != nil {
		return err
	}

	s.log.Debugf("There are %d scheduled %s games today", len(games), s.api.League())

	if len(games) == 0 {
		log.Debug("No scheduled games for %s, not rendering", s.api.League())
		return nil
	}

	gameOver := false
	cached, hasCached := s.cachedLiveGames[games[0].GetID()]
	if !hasCached {
		s.log.Debugf("no cached game data for %d", games[0].GetID())
	} else {
		gameOver, err = cached.IsComplete()
		if err != nil {
			s.log.Warnf("Failed to determine if game %d is over: %s", games[0].GetID(), err.Error())
			gameOver = false
		}
	}

	if gameOver && hasCached && cached != nil {
		s.log.Debugf("Game %d is over, using cached data", games[0].GetID())
	} else {
		// preload the first live game
		s.log.Debugf("fetching live data for game %d", games[0].GetID())
		s.cachedLiveGames[games[0].GetID()], err = games[0].GetUpdate(ctx)
		if err != nil {
			s.log.Errorf("failed to get live game update: %s", err.Error())
		}
	}

	preloader := make(map[int]chan bool)
	preloader[games[0].GetID()] = make(chan bool, 1)
	preloader[games[0].GetID()] <- true

OUTER:
	for gameIndex, game := range games {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}

		nextGameIndex := gameIndex + 1
		s.log.Debugf("Current game index is %d, current ID is %d", gameIndex, game.GetID())
		// preload data for the next game
		if nextGameIndex < len(games) {
			nextID := games[nextGameIndex].GetID()
			preloader[nextID] = make(chan bool, 1)
			go func() {
				if err := s.preloadLiveGame(ctx, games[nextGameIndex], preloader[nextID]); err != nil {
					s.log.Errorf("error while preloading next game: %s", err.Error())
				}
			}()
		}

		// Wait for the preloader to finish getting data, but with a timeout.
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-preloader[game.GetID()]:
			s.log.Debugf("preloader for %d marked ready", game.GetID())
		case <-time.After(s.config.boardDelay):
			s.log.Warnf("timed out waiting %ds for preloader for %d", s.config.boardDelay.Seconds(), game.GetID())
		}

		liveGame, ok := s.cachedLiveGames[game.GetID()]
		if !ok {
			s.log.Warnf("live game data for ID %d was not ready in time: UNDEFINED", game.GetID())
			continue OUTER
		}
		if liveGame == nil {
			s.log.Warnf("live game data for ID %d was not ready in time: NIL", game.GetID())
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
				return context.Canceled
			default:
			}
			s.log.Debugf("checking if %s is involved in game between %s vs %s", watchTeam, homeTeam.GetAbbreviation(), awayTeam.GetAbbreviation())

			team, err := s.api.TeamFromAbbreviation(ctx, watchTeam)
			if err != nil {
				return err
			}

			if awayTeam.GetID() != team.GetID() && homeTeam.GetID() != team.GetID() {
				s.log.Debugf("team %s with ID %d is not in %s (%d) or %s (%d)",
					watchTeam,
					team.GetID(),
					homeTeam.GetAbbreviation(),
					homeTeam.GetID(),
					awayTeam.GetAbbreviation(),
					awayTeam.GetID(),
				)
				continue INNER
			}

			isLive, err := liveGame.IsLive()
			if err != nil {
				s.log.Errorf("failed to determine if game is live: %s", err.Error())
			}
			isOver, err := liveGame.IsComplete()
			if err != nil {
				s.log.Errorf("failed to determine if game is complete: %s", err.Error())
			}

			_, err = s.RenderGameCounter(canvas, len(games), gameIndex, 1)
			if err != nil {
				return err
			}

			if isLive {
				if err := s.renderLiveGame(ctx, canvas, liveGame); err != nil {
					s.log.Errorf("failed to render live game: %s", err.Error())
					continue INNER
				}
			} else if isOver {
				if err := s.renderCompleteGame(ctx, canvas, liveGame); err != nil {
					s.log.Errorf("failed to render complete game: %s", err.Error())
					continue INNER
				}
			} else {
				if err := s.renderUpcomingGame(ctx, canvas, liveGame); err != nil {
					s.log.Errorf("failed to render upcoming game: %s", err.Error())
					continue INNER
				}
			}

			select {
			case <-ctx.Done():
				return context.Canceled
			case <-time.After(s.config.boardDelay):
			}

			continue OUTER
		}

	}

	return nil
}
func (s *SportBoard) HasPriority() bool {
	return false
}
func (s *SportBoard) Cleanup() {}

func (s *SportBoard) preloadLiveGame(ctx context.Context, game Game, preload chan bool) error {
	defer func() { preload <- true }()

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
		s.log.Debugf("Game %d is complete, not fetching any more data", game.GetID())

		return nil
	}

	s.log.Debugf("preloading live game data for game ID %d", game.GetID())
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
			s.log.Errorf("api call to get live game failed on attempt %d: %s", tries, err.Error())
		} else {
			s.cachedLiveGames[game.GetID()] = g
			s.log.Debugf("successfully set preloader data for game ID %d", game.GetID())
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		case <-time.After(10 * time.Second):
		}
	}
}
