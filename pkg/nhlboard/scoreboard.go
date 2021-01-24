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

const maxAPITries = 3

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

	games, ok := b.controller.api.Games[nhl.Today()]
	if !ok {
		fmt.Printf("No NHL games scheduled today %s\n", nhl.Today())
		return nil
	}

	liveGames := make(map[int]*nhl.LiveGame)
	for _, game := range games {
		live, err := nhl.GetLiveGame(ctx, game.Link)
		if err != nil {
			return err
		}
		liveGames[game.ID] = live
	}

	preloader := make(map[int]chan bool)

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

	INNER:
		for gameIndex, game := range games {
			if game.Teams.Away.Team.ID != team.ID && game.Teams.Home.Team.ID != team.ID {
				continue INNER
			}
			// Preload the next game's data
			if gameIndex+1 < len(games)-1 {
				nextGame := games[gameIndex+1]
				preloader[nextGame.ID] = make(chan bool, 1)
				go func() {
					tries := 0
					for {
						if tries > maxAPITries {
							// I give up
							preloader[nextGame.ID] <- true
							return
						}
						live, err := nhl.GetLiveGame(ctx, nextGame.Link)
						if err != nil {
							tries++
							continue
						}
						liveGames[nextGame.ID] = live
						preloader[nextGame.ID] <- true
						return
					}
				}()
			}

			// Wait for the preloader to finish getting data, but with a timeout
			select {
			case <-preloader[game.ID]:
			case <-time.After(b.controller.config.Delay / 2):
			}

			liveGame, ok := liveGames[game.ID]
			if !ok {
				// This means we failed to get data from the API for this game
				break INNER
			}

			if gameNotStarted(liveGame) {
				if err := b.RenderUpcomingGame(ctx, canvas, liveGame); err != nil {
					return err
				}
			} else if gameIsOver(liveGame) {
				fmt.Printf("%s game is over\n", team.Name)
			} else {
				if err := b.RenderGameUntilOver(ctx, canvas, liveGame); err != nil {
					return err
				}
			}

			select {
			case <-ctx.Done():
				return nil
			case <-time.After(b.controller.config.Delay):
			}
			break INNER
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
		canvas.Clear()
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

		center, err := rgbrender.AlignPosition(rgbrender.CenterCenter, canvas.Bounds(), 10, 10)
		if err != nil {
			return err
		}

		wrter.Write(canvas, center, []string{scoreStr(liveGame)}, color.White)

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

	canvas.Clear()

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

	center, err := rgbrender.AlignPosition(rgbrender.CenterCenter, canvas.Bounds(), 10, 10)
	if err != nil {
		return err
	}

	wrter.Write(canvas, center, []string{"vs."}, color.White)

	fmt.Printf("Upcoming: %s vs. %s %s\n",
		liveGame.LiveData.Linescore.Teams.Home.Team.Abbreviation,
		liveGame.LiveData.Linescore.Teams.Away.Team.Abbreviation,
		liveGame.GameTime.String(),
	)

	return nil
}

func (b *scoreBoard) RenderFinal(ctx context.Context, canvas *rgb.Canvas, liveGame *nhl.LiveGame) error {
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

	canvas.Clear()

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

	center, err := rgbrender.AlignPosition(rgbrender.CenterCenter, canvas.Bounds(), 10, 10)
	if err != nil {
		return err
	}

	wrter.Write(canvas, center, []string{"FINAL", scoreStr(liveGame)}, color.White)

	fmt.Printf("FINAL: %s vs. %s %s\n",
		liveGame.LiveData.Linescore.Teams.Home.Team.Abbreviation,
		liveGame.LiveData.Linescore.Teams.Away.Team.Abbreviation,
		scoreStr(liveGame),
	)

	return nil
}

func scoreStr(liveGame *nhl.LiveGame) string {
	return fmt.Sprintf("%d-%d",
		liveGame.LiveData.Linescore.Teams.Home.Goals,
		liveGame.LiveData.Linescore.Teams.Away.Goals,
	)
}

func (b *scoreBoard) isFavorite(abbrev string) bool {
	for _, t := range b.controller.favoriteTeams {
		if t == abbrev {
			return true
		}
	}

	return false
}
