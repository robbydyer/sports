package nhlboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	rgb "github.com/robbydyer/rgbmatrix-rpi"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/pkg/nhl"
)

var scorePollRate = 30 * time.Second

type nhlBoards struct {
	api        *nhl.Nhl
	watchTeams []string
	scheduler  *gocron.Scheduler
}

type scoreBoard struct {
	controller *nhlBoards
	liveGame   bool
}

func New(ctx context.Context) ([]board.Board, error) {
	var err error

	controller := &nhlBoards{
		watchTeams: []string{"NYI", "MTL", "COL"},
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

func (b *scoreBoard) Cleanup() {}

func (b *scoreBoard) Render(ctx context.Context, matrix rgb.Matrix, rotationDelay time.Duration) error {
	select {
	case <-ctx.Done():
	case <-time.After(rotationDelay):
	}

OUTER:
	for _, abbrev := range b.controller.watchTeams {
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
				if gameIsOver(liveGame) {
					fmt.Printf("%s game is not live\n", team.Name)
					continue OUTER
				}

				b.renderGameUntilOver(ctx, liveGame)

			}
		}
	}

	return nil
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
	if game.LiveData.Linescore.CurrentPeriod < 1 {
		return true
	}

	return false
}

func (b *scoreBoard) renderGameUntilOver(ctx context.Context, liveGame *nhl.LiveGame) error {
	// TODO: Make this atomic?
	b.liveGame = true
	defer func() { b.liveGame = false }()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(scorePollRate):
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
	}
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
