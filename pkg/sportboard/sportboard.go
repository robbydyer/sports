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
	config         *Config
	api            API
	teams          map[int]Team
	scheduledGames map[string]Game
	logos          map[string]*logo.Logo
	log            *log.Logger
	matrixBounds   image.Rectangle
	logoDrawCache  map[string]image.Image
}

type Config struct {
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
	if len(c.WatchTeams) == 0 {
		if len(c.FavoriteTeams) > 0 {
			c.WatchTeams = c.FavoriteTeams
		} else {
			// TODO:fix this
			c.WatchTeams = []string{"ANA",
				"ARI",
				"BOS",
				"BUF",
				"CAR",
				"CBJ",
				"CGY",
				"CHI",
				"COL",
				"DAL",
				"DET",
				"EDM",
				"FLA",
				"LAK",
				"MIN",
				"MTL",
				"NJD",
				"NSH",
				"NYI",
				"NYR",
				"OTT",
				"PHI",
				"PIT",
				"SJS",
				"STL",
				"TBL",
				"TOR",
				"VAN",
				"VGK",
				"WPG",
				"WSH"}
		}
	}
}

func New(ctx context.Context, api API, bounds image.Rectangle, config *Config) (*SportBoard, error) {
	s := &SportBoard{
		config:        config,
		api:           api,
		logos:         make(map[string]*logo.Logo),
		log:           log.New(),
		logoDrawCache: make(map[string]image.Image),
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

	log.Infof("There are %d scheduled %s games today\n", len(games), s.api.League())

	if len(games) == 0 {
		return nil
	}

	seenTeams := make(map[string]bool)

	liveGames := make(map[int]Game)
	// preload the first live game
	liveGames[games[0].GetID()], err = games[0].GetUpdate(ctx)
	if err != nil {
		return err
	}

	preloader := make(map[int]chan bool)
	preloader[games[0].GetID()] = make(chan bool, 1)
	preloader[games[0].GetID()] <- true

OUTER:
	for gameIndex, game := range games {
		// preload data for the next game
		if gameIndex+1 <= len(games)-1 {
			nextID := games[gameIndex+1].GetID()
			preloader[nextID] = make(chan bool, 1)
			go func() {
				liveGames[gameIndex+1], err = s.preloadLiveGame(ctx, games[gameIndex+1], preloader[nextID])
			}()
		}

		// Wait for the preloader to finish getting data, but with a timeout.
		// Stale data beats no data
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		case <-preloader[game.GetID()]:
		case <-time.After(s.config.BoardDelay):
		}

		liveGame, ok := liveGames[game.GetID()]
		if !ok || liveGame == nil {
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
			seen, ok := seenTeams[watchTeam]
			if ok && seen {
				continue OUTER
			}
			seenTeams[watchTeam] = true

			team, err := s.api.TeamFromAbbreviation(ctx, watchTeam)
			if err != nil {
				return err
			}

			if awayTeam.GetID() != team.GetID() || homeTeam.GetID() != team.GetID() {
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

		}

	}

	return nil
}
func (s *SportBoard) HasPriority() bool {
	return false
}
func (s *SportBoard) Cleanup() {}

func (s *SportBoard) preloadLiveGame(ctx context.Context, game Game, preload chan bool) (Game, error) {
	tries := 0
	defer func() { preload <- true }()
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled")
		default:
		}

		if tries > maxAPITries {
			return nil, fmt.Errorf("failed API call %d times", maxAPITries)
		}
		tries++

		g, err := game.GetUpdate(ctx)
		if err != nil {
			s.log.Errorf("api call to get live game failed: %s", err.Error())
		} else {
			return g, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled")
		case <-time.After(10 * time.Second):
		}
	}
}
