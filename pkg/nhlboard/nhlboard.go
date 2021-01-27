package nhlboard

import (
	"context"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/go-co-op/gocron"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/nhl"
)

var scorePollRate = 20 * time.Second

type nhlBoards struct {
	api            DataAPI
	liveGameGetter LiveGameGetter
	scheduler      *gocron.Scheduler
	logos          map[string]*logoInfo
	logoCache      map[string]image.Image
	matrixBounds   image.Rectangle
	config         *Config
	cancel         chan bool
}

type Config struct {
	BoardDelay    string        `json:"boardDelay"`
	FavoriteTeams []string      `json:"favoriteTeams"`
	WatchTeams    []string      `json:"watchTeams"`
	LogoPosition  []*logoConfig `json:"logoPosition"`
}

type DataAPI interface {
	UpdateTeams(ctx context.Context) error
	UpdateGames(ctx context.Context, dateStr string) error
	TeamFromAbbreviation(abbrev string) (*nhl.Team, error)
	Games(dateStr string) ([]*nhl.Game, error)
}

type LiveGameGetter func(ctx context.Context, link string) (*nhl.LiveGame, error)

func New(ctx context.Context, matrixBounds image.Rectangle, dataAPI DataAPI, liveGameGetter LiveGameGetter, config *Config) ([]board.Board, error) {
	var err error

	if len(config.WatchTeams) == 0 && len(config.FavoriteTeams) == 0 {
		config.WatchTeams = ALL
	}

	fmt.Printf("Initializing NHL Board %dx%d\nWatch Teams: %s\nFavorites:%s\n",
		matrixBounds.Dx(),
		matrixBounds.Dy(),
		strings.Join(config.WatchTeams, ", "),
		strings.Join(config.FavoriteTeams, ", "),
	)

	controller := &nhlBoards{
		api:            dataAPI,
		liveGameGetter: liveGameGetter,
		logoCache:      make(map[string]image.Image),
		matrixBounds:   matrixBounds,
		config:         config,
		cancel:         make(chan bool, 1),
	}

	// Schedule game updates
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return nil, err
	}

	controller.scheduler = gocron.NewScheduler(loc)
	controller.scheduler.Every(1).Day().At("05:00").Do(controller.updateGames)
	controller.scheduler.StartAsync()

	if err := controller.setLogoInfo(); err != nil {
		return nil, fmt.Errorf("failed to get logos: %w", err)
	}

	// Initialize logo cache
	for _, t := range ALL {
		for _, h := range []string{"HOME", "AWAY"} {
			key := fmt.Sprintf("%s_%s", t, h)
			_, err := controller.getLogo(key)
			if err != nil {
				return nil, fmt.Errorf("failed to get logo thumbnail for %s: %w", key, err)
			}
		}
	}

	var boards []board.Board

	b := &scoreBoard{
		controller: controller,
	}

	boards = append(boards, b)

	return boards, nil
}

func (c *Config) Defaults() {
	if c.BoardDelay == "" {
		c.BoardDelay = "20s"
	}
	if len(c.FavoriteTeams) == 0 && len(c.WatchTeams) == 0 {
		c.WatchTeams = ALL
	}
}

func (c *Config) boardDelay() time.Duration {
	d, err := time.ParseDuration(c.BoardDelay)
	if err != nil {
		fmt.Printf("could not parse board delay '%s', defaulting to 20 sec", c.BoardDelay)
		return 20 * time.Second
	}

	return d
}

func (n *nhlBoards) updateGames() {
	_ = n.api.UpdateGames(context.Background(), nhl.Today())
}

func gameNotStarted(game *nhl.LiveGame) bool {
	if game == nil || game.LiveData == nil {
		return true
	}
	if game.LiveData.Linescore.CurrentPeriod < 1 {
		return true
	}

	return false
}

func gameIsOver(game *nhl.LiveGame) bool {
	if game == nil || game.LiveData == nil {
		return true
	}
	if game.GameData.Status != nil && strings.Contains(strings.ToLower(game.GameData.Status.AbstractGameState), "final") {
		return true
	}
	if strings.Contains(strings.ToLower(game.LiveData.Linescore.CurrentPeriodTimeRemaining), "final") {
		return true
	}

	return false
}

// getLogo checks cache first
func (b *nhlBoards) getLogo(logoKey string) (image.Image, error) {
	// Check cache first
	logo, ok := b.logoCache[logoKey]
	if ok && logo.Bounds().Dx() > 0 && logo.Bounds().Dy() > 0 {
		fmt.Printf("found logo cache for %s\n", logoKey)
		return logo, nil
	}

	if _, ok := b.logos[logoKey]; !ok {
		return nil, fmt.Errorf("no logo defined for %s", logoKey)
	}

	var err error
	b.logoCache[logoKey], err = b.logos[logoKey].logo.GetThumbnail(b.matrixBounds)
	if err != nil {
		return nil, err
	}

	return b.logoCache[logoKey], nil
}

func (b *nhlBoards) logoShiftPt(logoKey string) (int, int) {
	l, ok := b.logos[logoKey]
	if !ok {
		fmt.Printf("WARNING! No logo position found for %s\n", logoKey)
		return 0, 0
	}

	return l.xPosition, l.yPosition
}

func periodStr(period int) string {
	switch period {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	default:
		return ""
	}
}
