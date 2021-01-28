package sportboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/robbydyer/sports/pkg/logo"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
)

const maxAPITries = 3

// SportBoard implements board.Board
type SportBoard struct {
	config            *Config
	api               API
	teams             map[int]Team
	scheduledGames    map[string]Game
	logos             map[string]*logo.Logo
	log               *log.Logger
	matrixBounds      image.Rectangle
	logoDrawCache     map[string]image.Image
	logoSourceCache   map[string]image.Image
	liveGamePreloader map[int]Game
}

type Config struct {
	Enabled        bool
	BoardDelay     time.Duration
	FavoriteSticky bool
	ScoreFont      *FontConfig
	TimeFont       *FontConfig
	TimeColor      color.Color
	ScoreColor     color.Color
	LogoConfigs    []*logo.Config
	WatchTeams     []string
	FavoriteTeams  []string
}

type FontConfig struct {
	Size      float64
	LineSpace float64
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
	GetQuarter() (int, error) // Or a period, hockey fans
	GetClock() (string, error)
	GetUpdate(ctx context.Context) (Game, error)
	GetStartTime(ctx context.Context) (time.Time, error)
}

func (c *Config) SetDefaults() {
	// TODO: fix this
	c.BoardDelay = 20 * time.Second

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

func New(ctx context.Context, api API, bounds image.Rectangle, config *Config) (*SportBoard, error) {
	l := log.New()
	l.Level = log.DebugLevel
	s := &SportBoard{
		config:        config,
		api:           api,
		logos:         make(map[string]*logo.Logo),
		log:           l,
		logoDrawCache: make(map[string]image.Image),
		matrixBounds:  bounds,
	}

	if len(config.WatchTeams) == 0 {
		if len(config.FavoriteTeams) > 0 {
			config.WatchTeams = config.FavoriteTeams
		} else {
			config.WatchTeams = s.api.AllTeamAbbreviations()
		}
	}

	if _, err := s.api.GetTeams(ctx); err != nil {
		return nil, err
	}
	if _, err := s.api.GetScheduledGames(ctx, Today()); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *SportBoard) Name() string {
	return "SportBoard"
}

func (s *SportBoard) Render(ctx context.Context, matrix rgb.Matrix) error {
	canvas := rgb.NewCanvas(matrix)

	games, err := s.api.GetScheduledGames(ctx, Today())
	if err != nil {
		return err
	}

	log.Debugf("There are %d scheduled %s games today\n", len(games), s.api.League())

	if len(games) == 0 {
		log.Debug("No scheduled games for %s, not rendering", s.api.League())
		return nil
	}

	s.liveGamePreloader = make(map[int]Game)
	// preload the first live game
	s.liveGamePreloader[games[0].GetID()], err = games[0].GetUpdate(ctx)
	if err != nil {
		s.log.Errorf("failed to get live game update: %s", err.Error())
		return err
	}

	preloader := make(map[int]chan bool)
	preloader[games[0].GetID()] = make(chan bool, 1)
	preloader[games[0].GetID()] <- true

OUTER:
	for gameIndex, game := range games {
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
			return fmt.Errorf("context canceled")
		case <-preloader[game.GetID()]:
			s.log.Debugf("preloader for %d marked ready", game.GetID())
		case <-time.After(s.config.BoardDelay):
			s.log.Warnf("timed out waiting %ds for preloader for %d", s.config.BoardDelay.Seconds(), game.GetID())
		}

		liveGame, ok := s.liveGamePreloader[game.GetID()]
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

			if isLive {
				if err := s.renderLiveGame(ctx, canvas, liveGame); err != nil {
					s.log.Errorf("failed to render live game: %s", err.Error())
					continue INNER
				}
			} else if isOver {

			} else {
				// Game hasn't started yet
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("context canceled")
			case <-time.After(s.config.BoardDelay):
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
	s.log.Debugf("preloading live game data for game ID %d", game.GetID())
	tries := 0
	defer func() { preload <- true }()
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
			s.liveGamePreloader[game.GetID()] = g
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
