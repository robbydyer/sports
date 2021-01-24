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
	api           *nhl.Nhl
	watchTeams    []string
	favoriteTeams []string
	scheduler     *gocron.Scheduler
	logos         map[string]*logoInfo
	logoCache     map[string]image.Image
	matrixBounds  image.Rectangle
	config        *Config
	cancel        chan bool
}

type Config struct {
	Delay time.Duration
}

func New(ctx context.Context, matrixBounds image.Rectangle, config *Config) ([]board.Board, error) {
	var err error

	controller := &nhlBoards{
		watchTeams:    ALL,
		favoriteTeams: []string{"NYI"},
		logos:         make(map[string]*logoInfo),
		logoCache:     make(map[string]image.Image),
		matrixBounds:  matrixBounds,
		config:        config,
		cancel:        make(chan bool, 1),
	}

	controller.api, err = nhl.New(ctx)
	if err != nil {
		return nil, err
	}

	// Schedule game updates
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return nil, err
	}
	controller.scheduler = gocron.NewScheduler(loc)
	controller.scheduler.Every(1).Day().At("05:00").Do(controller.updateGames)
	controller.scheduler.StartAsync()

	controller.logos, err = getLogos()

	// Initialize logo cache
	for _, t := range ALL {
		for _, h := range []string{"_HOME", "_AWAY"} {
			_, err := controller.getLogo(t + h)
			if err != nil {
				return nil, fmt.Errorf("failed to get logo thumbnail for %s: %w", t, err)
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

	fmt.Printf("getting thumbnail logo for %s\n", logoKey)
	var err error
	b.logoCache[logoKey], err = b.logos[logoKey].logo.GetThumbnail(b.matrixBounds)
	if err != nil {
		return nil, err
	}

	return b.logoCache[logoKey], nil
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
