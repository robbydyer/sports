package nhlboard

import (
	"context"
	"fmt"
	"image/color"
	"time"

	rgb "github.com/robbydyer/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/nhl"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

// scoreBoard implements board.Board
type scoreBoard struct {
	controller *nhlBoards
	liveGame   bool
}

func (b *scoreBoard) Name() string {
	return "NHL Scoreboard"
}

func (b *scoreBoard) HasPriority() bool {
	return b.liveGame
}

func (b *scoreBoard) Cleanup() {}

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

func (b *scoreBoard) isFavorite(abbrev string) bool {
	for _, t := range b.controller.favoriteTeams {
		if t == abbrev {
			return true
		}
	}

	return false
}
