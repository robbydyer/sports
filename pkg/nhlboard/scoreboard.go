package nhlboard

import (
	"context"
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/robbydyer/sports/pkg/nhl"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
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

	games, err := b.controller.api.Games(nhl.Today())
	if err != nil {
		fmt.Printf("no games for today: %s", err.Error())
		return nil
	}

	liveGames := make(map[int]*nhl.LiveGame)
	for _, game := range games {
		live, err := b.controller.liveGameGetter(ctx, game.Link)
		if err != nil {
			return err
		}
		liveGames[game.ID] = live
	}

	preloader := make(map[int]chan bool)
	seenTeams := make(map[string]bool)

OUTER:
	for _, abbrev := range b.controller.config.WatchTeams {
		seen, ok := seenTeams[abbrev]
		if ok && seen {
			continue OUTER
		}
		seenTeams[abbrev] = true

		team, err := b.controller.api.TeamFromAbbreviation(abbrev)
		if err != nil {
			return err
		}

		fmt.Printf("Checking for games for %s\n", team.Abbreviation)

	INNER:
		for gameIndex, game := range games {
			fmt.Printf("Checking if %s in %s vs. %s for Game ID %d\n",
				team.Abbreviation,
				game.Teams.Home.Team.Abbreviation,
				game.Teams.Away.Team.Abbreviation,
				game.ID,
			)
			if game.Teams.Away.Team.Abbreviation != team.Abbreviation && game.Teams.Home.Team.Abbreviation != team.Abbreviation {
				fmt.Println("it is not")
				continue INNER
			}
			// Preload the next game's data
			if gameIndex+1 <= len(games)-1 {
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
						live, err := b.controller.liveGameGetter(ctx, nextGame.Link)
						if err != nil {
							tries++
							time.Sleep(5 * time.Second)
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
			case <-time.After(b.controller.config.boardDelay() / 2):
			}

			liveGame, ok := liveGames[game.ID]
			if !ok {
				// This means we failed to get data from the API for this game
				fmt.Printf("failed to fetch live game data for ID %d\n", game.ID)
				continue OUTER
			}

			if gameNotStarted(liveGame) {
				if err := b.RenderUpcomingGame(ctx, canvas, liveGame); err != nil {
					return fmt.Errorf("failed to render upcoming game: %w", err)
				}
			} else if gameIsOver(liveGame) {
				fmt.Printf("%s game is over\n", team.Name)
				if err := b.RenderUpcomingGame(ctx, canvas, liveGame); err != nil {
					return fmt.Errorf("failed to render upcoming game: %w", err)
				}
			} else {
				if err := b.RenderGameUntilOver(ctx, canvas, liveGame); err != nil {
					return fmt.Errorf("failed to render live game: %w", err)
				}
			}

			select {
			case <-ctx.Done():
				return nil
			case <-time.After(b.controller.config.boardDelay()):
			}
			continue OUTER
			//break INNER
		}
	}

	return nil
}

func (b *scoreBoard) RenderGameUntilOver(ctx context.Context, canvas *rgb.Canvas, liveGame *nhl.LiveGame) error {
	isFavorite := b.isFavorite(liveGame.LiveData.Linescore.Teams.Home.Team.Abbreviation) ||
		b.isFavorite(liveGame.LiveData.Linescore.Teams.Away.Team.Abbreviation)

	isFavorite = false

	if isFavorite {
		// TODO: Make atomic?
		b.liveGame = true
		defer func() { b.liveGame = false }()
	}

	hKey, err := key("HOME", liveGame)
	if err != nil {
		return err
	}
	aKey, err := key("AWAY", liveGame)
	if err != nil {
		return err
	}

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

	hLogo, err := rgbrender.SetImageAlign(rgbrender.LeftCenter, homeLogo)
	if err != nil {
		return err
	}
	aLogo, err := rgbrender.SetImageAlign(rgbrender.RightCenter, awayLogo)
	if err != nil {
		return err
	}
	count := 0
	for {
		count++
		fmt.Printf("NUM in loop %d\n", count)

		/*
			if err := rgbrender.DrawImage(canvas, homeShift, homeLogo); err != nil {
				return err
			}
			if err := rgbrender.DrawImage(canvas, awayShift, awayLogo); err != nil {
				return err
			}
		*/
		if err := rgbrender.DrawImage(canvas, homeShift, hLogo); err != nil {
			return err
		}
		if err := rgbrender.DrawImage(canvas, awayShift, aLogo); err != nil {
			return err
		}

		wrter, err := rgbrender.DefaultTextWriter()
		if err != nil {
			return err
		}

		center, err := rgbrender.AlignPosition(rgbrender.CenterTop, canvas.Bounds(), 16, 16)
		if err != nil {
			return err
		}

		fmt.Printf("text centertop align: %dx%d to %dx%d\n",
			center.Min.X,
			center.Min.Y,
			center.Max.X,
			center.Max.Y,
		)
		scoreAlign, err := rgbrender.AlignPosition(rgbrender.CenterBottom, canvas.Bounds(), 16, 16)
		if err != nil {
			return err
		}

		fmt.Printf("text centerbottom align: %dx%d to %dx%d\n",
			scoreAlign.Min.X,
			scoreAlign.Min.Y,
			scoreAlign.Max.X,
			scoreAlign.Max.Y,
		)

		wrter.Write(
			canvas,
			scoreAlign,
			[]string{
				scoreStr(liveGame),
			},
			color.White,
		)
		wrter.Write(
			canvas,
			center,
			[]string{
				periodStr(liveGame.LiveData.Linescore.CurrentPeriod),
				liveGame.LiveData.Linescore.CurrentPeriodTimeRemaining,
			},
			color.White,
		)

		if err := canvas.Render(); err != nil {
			return fmt.Errorf("failed to render live scoreboard: %w", err)
		}

		// This game is live, scoreboard time
		fmt.Printf("Live Game: %s vs. %s\nScore: %s\n%s period %s\n",
			liveGame.LiveData.Linescore.Teams.Home.Team.Name,
			liveGame.LiveData.Linescore.Teams.Away.Team.Name,
			scoreStr(liveGame),
			periodStr(liveGame.LiveData.Linescore.CurrentPeriod),
			liveGame.LiveData.Linescore.CurrentPeriodTimeRemaining,
		)

		if !isFavorite {
			fmt.Printf("Game %s v. %s is not a favorite, continuing rotation\n",
				liveGame.LiveData.Linescore.Teams.Home.Team.Name,
				liveGame.LiveData.Linescore.Teams.Away.Team.Name,
			)
			return nil
		}

		updated, err := b.controller.liveGameGetter(ctx, liveGame.Link)
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
	hKey, err := key("HOME", liveGame)
	if err != nil {
		return err
	}
	aKey, err := key("AWAY", liveGame)
	if err != nil {
		return err
	}

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
		return fmt.Errorf("failed to draw home logo: %w", err)
	}
	if err := rgbrender.DrawImage(canvas, awayShift, awayLogo); err != nil {
		return fmt.Errorf("failed to draw away logo: %w", err)
	}
	wrter, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return fmt.Errorf("failed to get text writer: %w", err)
	}

	center, err := rgbrender.AlignPosition(rgbrender.CenterCenter, canvas.Bounds(), 16, 16)
	if err != nil {
		return fmt.Errorf("failed to get text align position: %w", err)
	}

	wrter.Write(canvas, center, []string{"vs."}, color.White)

	if err := canvas.Render(); err != nil {
		return fmt.Errorf("failed to render upcoming game: %w", err)
	}

	fmt.Printf("Upcoming: %s vs. %s %s\n",
		liveGame.LiveData.Linescore.Teams.Home.Team.Abbreviation,
		liveGame.LiveData.Linescore.Teams.Away.Team.Abbreviation,
		liveGame.GameTime.String(),
	)

	return nil
}

func (b *scoreBoard) RenderFinal(ctx context.Context, canvas *rgb.Canvas, liveGame *nhl.LiveGame) error {
	hKey, err := key("HOME", liveGame)
	if err != nil {
		return err
	}
	aKey, err := key("AWAY", liveGame)
	if err != nil {
		return err
	}

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

	center, err := rgbrender.AlignPosition(rgbrender.CenterCenter, canvas.Bounds(), 16, 16)
	if err != nil {
		return err
	}

	wrter.Write(canvas, center, []string{"FINAL", scoreStr(liveGame)}, color.White)

	if err := canvas.Render(); err != nil {
		return fmt.Errorf("failed to render game final: %w", err)
	}

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
	for _, t := range b.controller.config.FavoriteTeams {
		if t == abbrev {
			return true
		}
	}

	return false
}

func key(homeAway string, liveGame *nhl.LiveGame) (string, error) {
	if liveGame == nil ||
		liveGame.LiveData == nil ||
		liveGame.LiveData.Linescore == nil ||
		liveGame.LiveData.Linescore.Teams == nil {
		return "", fmt.Errorf("invalid LiveGame: missing LiveData %v", liveGame)
	}

	homeAway = strings.ToUpper(homeAway)
	if homeAway == "AWAY" {
		if liveGame.LiveData.Linescore.Teams.Away == nil ||
			liveGame.LiveData.Linescore.Teams.Away.Team == nil ||
			liveGame.LiveData.Linescore.Teams.Away.Team.Abbreviation == "" {
			return "", fmt.Errorf("invalid LiveGame: missing Away team %v", liveGame)
		}

		return fmt.Sprintf("%s_AWAY", liveGame.LiveData.Linescore.Teams.Away.Team.Abbreviation), nil
	}

	if liveGame.LiveData.Linescore.Teams.Home == nil ||
		liveGame.LiveData.Linescore.Teams.Home.Team == nil ||
		liveGame.LiveData.Linescore.Teams.Home.Team.Abbreviation == "" {
		return "", fmt.Errorf("invalid LiveGame: missing Home team %v", liveGame)
	}

	return fmt.Sprintf("%s_HOME", liveGame.LiveData.Linescore.Teams.Home.Team.Abbreviation), nil
}
