package nhlboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
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
			if game.Teams.Away.Team.Abbreviation != team.Abbreviation && game.Teams.Home.Team.Abbreviation != team.Abbreviation {
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
		}
	}

	return nil
}

func (b *scoreBoard) textAreaWidth() int {
	return b.controller.matrixBounds.Dx() / 4
}

func (b *scoreBoard) renderHomeLogo(canvas *rgb.Canvas, logoKey string) error {
	logo, err := b.controller.getLogo(logoKey)
	if err != nil {
		return err
	}
	textWdith := b.textAreaWidth()
	logoWidth := (b.controller.matrixBounds.Dx() - textWdith) / 2

	startX := logoWidth - logo.Bounds().Dx()

	shiftX, shiftY := b.controller.logoShiftPt(logoKey)

	fmt.Printf("Adding shift to %s: %d, %d\n", logoKey, shiftX, shiftY)
	startX = startX + shiftX
	startY := 0 + shiftY

	bounds := image.Rect(startX, startY, canvas.Bounds().Dx()-1, canvas.Bounds().Dy()-1)

	i := image.NewRGBA(bounds)
	draw.Draw(i, bounds, logo, image.ZP, draw.Over)

	fmt.Printf("%s size is %dx%d\n", logoKey, logo.Bounds().Dx(), logo.Bounds().Dy())
	fmt.Printf("Starting pt for %s is %d, 0 within bounds: %d, %d to %d, %d\n",
		logoKey,
		startX,
		bounds.Min.X,
		bounds.Min.Y,
		bounds.Max.X,
		bounds.Max.Y,
	)

	draw.Draw(canvas, canvas.Bounds(), i, image.ZP, draw.Over)

	return nil
}
func (b *scoreBoard) renderAwayLogo(canvas *rgb.Canvas, logoKey string) error {
	logo, err := b.controller.getLogo(logoKey)
	if err != nil {
		return err
	}
	textWdith := b.textAreaWidth()
	logoWidth := (b.controller.matrixBounds.Dx() - textWdith) / 2

	startX := logoWidth + textWdith

	shiftX, shiftY := b.controller.logoShiftPt(logoKey)
	fmt.Printf("Adding shift to %s: %d, %d\n", logoKey, shiftX, shiftY)

	startX = startX + shiftX
	startY := 0 + shiftY

	bounds := image.Rect(startX, startY, canvas.Bounds().Dx()-1, canvas.Bounds().Dy()-1)

	i := image.NewRGBA(bounds)
	draw.Draw(i, bounds, logo, image.ZP, draw.Over)

	draw.Draw(canvas, canvas.Bounds(), i, image.ZP, draw.Over)

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

	for {
		if err := b.renderHomeLogo(canvas, hKey); err != nil {
			return fmt.Errorf("failed to render home logo: %w", err)
		}
		if err := b.renderAwayLogo(canvas, aKey); err != nil {
			return fmt.Errorf("failed to render away logo: %w", err)
		}

		wrter, err := rgbrender.DefaultTextWriter()
		if err != nil {
			return err
		}

		center, err := rgbrender.AlignPosition(rgbrender.CenterTop, canvas.Bounds(), b.textAreaWidth(), 32)
		if err != nil {
			return err
		}

		fmt.Printf("text centertop align: %dx%d to %dx%d\n",
			center.Min.X,
			center.Min.Y,
			center.Max.X,
			center.Max.Y,
		)
		scoreAlign, err := rgbrender.AlignPosition(rgbrender.CenterBottom, canvas.Bounds(), b.textAreaWidth(), 16)
		if err != nil {
			return err
		}

		fmt.Printf("text centerbottom align: %dx%d to %dx%d\n",
			scoreAlign.Min.X,
			scoreAlign.Min.Y,
			scoreAlign.Max.X,
			scoreAlign.Max.Y-1,
		)

		wrter.Write(
			canvas,
			center,
			[]string{
				periodStr(liveGame.LiveData.Linescore.CurrentPeriod),
				liveGame.LiveData.Linescore.CurrentPeriodTimeRemaining,
				"",
				"",
				scoreStr(liveGame),
			},
			color.White,
		)
		/*
			wrter.Write(
				canvas,
				scoreAlign,
				[]string{
					scoreStr(liveGame),
				},
				color.White,
			)
		*/

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
	/*
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
	*/
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
	/*
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
	*/
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
