package nhlboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	rgb "github.com/robbydyer/rgbmatrix-rpi"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/nhl"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

var scorePollRate = 30 * time.Second

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

type scoreBoard struct {
	controller *nhlBoards
	liveGame   bool
}

type Config struct {
	Delay time.Duration
}

func New(ctx context.Context, matrixBounds image.Rectangle, config *Config) ([]board.Board, error) {
	var err error

	controller := &nhlBoards{
		watchTeams:    []string{"NYI", "MTL", "COL"},
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

	// Intialize logo cache
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

func (b *scoreBoard) Name() string {
	return "NHL Scoreboard"
}

func (b *scoreBoard) HasPriority() bool {
	return b.liveGame
}

func (b *scoreBoard) Cleanup() {
	for _, l := range b.controller.logos {
		_ = l.logo.Close()
	}
}

func (b *scoreBoard) Render(ctx context.Context, matrix rgb.Matrix) error {
	canvas := rgb.NewCanvas(matrix)
	canvas.Clear()

	seenTeams := make(map[string]bool)

OUTER:
	for _, abbrev := range b.controller.watchTeams {
		seen, ok := seenTeams[abbrev]
		if ok && seen {
			continue OUTER
		}
		seenTeams[abbrev] = true

		team, err := b.controller.api.TeamFromAbbreviation(abbrev)
		if err != nil {
			return err
		}

		for _, game := range b.controller.api.Games[nhl.Today()] {
			if game.Teams.Away.Team.ID == team.ID || game.Teams.Home.Team.ID == team.ID {
				liveGame, err := nhl.GetLiveGame(ctx, game.Link)
				if err != nil {
					return fmt.Errorf("failed to get live game status of game: %w", err)
				}
				if gameNotStarted(liveGame) {
					if err := b.RenderUpcomingGame(ctx, canvas, liveGame); err != nil {
						return err
					}
				}
				if gameIsOver(liveGame) {
					fmt.Printf("%s game is over\n", team.Name)
					continue OUTER
				}

				if err := b.RenderGameUntilOver(ctx, canvas, liveGame); err != nil {
					return err
				}

			}
		}
		select {
		case <-ctx.Done():
		case <-time.After(b.controller.config.Delay):
		}
	}

	return nil
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

func (b *scoreBoard) isFavorite(abbrev string) bool {
	for _, t := range b.controller.favoriteTeams {
		if t == abbrev {
			return true
		}
	}

	return false
}

// getLogo checks cache first
func (b *nhlBoards) getLogo(logoKey string) (image.Image, error) {
	// Check cache first
	logo, ok := b.logoCache[logoKey]
	if ok {
		return logo, nil
	}

	var err error
	b.logoCache[logoKey], err = b.logos[logoKey].logo.GetThumbnail(b.matrixBounds)
	if err != nil {
		return nil, err
	}

	return b.logoCache[logoKey], nil
}

func (b *scoreBoard) RenderGameUntilOver(ctx context.Context, canvas *rgb.Canvas, liveGame *nhl.LiveGame) error {
	isFavorite := b.isFavorite(liveGame.LiveData.Linescore.Teams.Home.Team.Abbreviation) ||
		b.isFavorite(liveGame.LiveData.Linescore.Teams.Away.Team.Abbreviation)

	if isFavorite {
		// TODO: Make atomic?
		b.liveGame = true
		defer func() { b.liveGame = false }()
	}

	for {
		hKey := fmt.Sprintf("%s_HOME", liveGame.LiveData.Linescore.Teams.Home.Team.Abbreviation)
		aKey := fmt.Sprintf("%s_AWAY", liveGame.LiveData.Linescore.Teams.Away.Team.Abbreviation)

		homeLogo, err := b.controller.getLogo(hKey)
		if err != nil {
			return err
		}
		awayLogo, err := b.controller.getLogo(aKey)
		if err != nil {
			return err
		}

		homeShift, err := b.controller.logoShift(hKey)
		if err != nil {
			return err
		}
		awayShift, err := b.controller.logoShift(aKey)
		if err != nil {
			return err
		}

		if err := rgbrender.DrawImage(canvas, homeShift, homeLogo); err != nil {
			return err
		}
		if err := rgbrender.DrawImage(canvas, awayShift, awayLogo); err != nil {
			return err
		}

		// This game is live, scoreboard time
		fmt.Printf("Live Game: %s vs. %s\nScore: %d - %d\n%s period %s\n",
			liveGame.LiveData.Linescore.Teams.Home.Team.Name,
			liveGame.LiveData.Linescore.Teams.Away.Team.Name,
			liveGame.LiveData.Linescore.Teams.Home.Goals,
			liveGame.LiveData.Linescore.Teams.Away.Goals,
			periodStr(liveGame.LiveData.Linescore.CurrentPeriod),
			liveGame.LiveData.Linescore.CurrentPeriodTimeRemaining,
		)

		if !isFavorite {
			return nil
		}

		updated, err := nhl.GetLiveGame(ctx, liveGame.Link)
		if err != nil {
			fmt.Printf("failed to update live game: %s", err.Error())
		}

		if gameIsOver(updated) {
			return nil
		}

		if updated != nil {
			liveGame = updated
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(scorePollRate):
		}
	}
}

func (b *scoreBoard) RenderUpcomingGame(ctx context.Context, canvas *rgb.Canvas, liveGame *nhl.LiveGame) error {
	hKey := fmt.Sprintf("%s_HOME", liveGame.LiveData.Linescore.Teams.Home.Team.Abbreviation)
	aKey := fmt.Sprintf("%s_AWAY", liveGame.LiveData.Linescore.Teams.Away.Team.Abbreviation)

	homeLogo, err := b.controller.getLogo(hKey)
	if err != nil {
		return err
	}
	awayLogo, err := b.controller.getLogo(aKey)
	if err != nil {
		return err
	}

	homeShift, err := b.controller.logoShift(hKey)
	if err != nil {
		return err
	}
	awayShift, err := b.controller.logoShift(aKey)
	if err != nil {
		return err
	}

	if err := rgbrender.DrawImage(canvas, homeShift, homeLogo); err != nil {
		return err
	}
	if err := rgbrender.DrawImage(canvas, awayShift, awayLogo); err != nil {
		return err
	}
	wrter, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return err
	}

	center, err := rgbrender.AlignPosition(rgbrender.CenterCenter, canvas.Bounds(), 5, 5)
	if err != nil {
		return err
	}

	wrter.Write(canvas, center, []string{"vs."}, color.Black)

	return nil
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
