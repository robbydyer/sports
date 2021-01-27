package nhlboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"time"

	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/nhl"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

const maxAPITries = 3

// scoreBoard implements board.Board
type scoreBoard struct {
	controller    *nhlBoards
	liveGame      bool
	logoDrawCache map[string]image.Image // Contains cached image.Image of home/away logos after being positioned
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
						select {
						case <-ctx.Done():
							return
						default:
						}
						if tries > maxAPITries {
							// I give up
							preloader[nextGame.ID] <- true
							return
						}
						live, err := b.controller.liveGameGetter(ctx, nextGame.Link)
						if err != nil {
							tries++
							select {
							case <-ctx.Done():
								return
							case <-time.After(5 * time.Second):
							}
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
			case <-ctx.Done():
				return nil
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
	i, ok := b.logoDrawCache[logoKey]
	if ok {
		draw.Draw(canvas, canvas.Bounds(), i, image.ZP, draw.Over)
		return nil
	}

	l, err := b.controller.getLogo(logoKey)
	if err != nil {
		return err
	}
	textWdith := b.textAreaWidth()
	logoWidth := (b.controller.matrixBounds.Dx() - textWdith) / 2
	xShift, yShift := b.controller.logoShiftPt(logoKey)

	if xShift != 0 || yShift != 0 {
		fmt.Printf("Shifting %s %d, %d\n", logoKey, xShift, yShift)
	}

	i, err = logo.RenderLeftAligned(canvas, l, logoWidth, xShift, yShift)
	if err != nil {
		return err
	}

	b.logoDrawCache[logoKey] = i

	draw.Draw(canvas, canvas.Bounds(), i, image.ZP, draw.Over)

	return nil
}
func (b *scoreBoard) renderAwayLogo(canvas *rgb.Canvas, logoKey string) error {
	i, ok := b.logoDrawCache[logoKey]
	if ok {
		draw.Draw(canvas, canvas.Bounds(), i, image.ZP, draw.Over)
		return nil
	}

	l, err := b.controller.getLogo(logoKey)
	if err != nil {
		return err
	}
	textWdith := b.textAreaWidth()
	logoWidth := (b.controller.matrixBounds.Dx() - textWdith) / 2
	xShift, yShift := b.controller.logoShiftPt(logoKey)

	if xShift != 0 || yShift != 0 {
		fmt.Printf("Shifting %s %d, %d\n", logoKey, xShift, yShift)
	}
	xShift += textWdith

	i, err = logo.RenderRightAligned(canvas, l, logoWidth, xShift, yShift)
	if err != nil {
		return err
	}

	b.logoDrawCache[logoKey] = i

	draw.Draw(canvas, canvas.Bounds(), i, image.ZP, draw.Over)

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

	hKey, err := key("HOME", liveGame)
	if err != nil {
		return err
	}
	aKey, err := key("AWAY", liveGame)
	if err != nil {
		return err
	}

	wrter, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return err
	}

	wrter.LineSpace = b.controller.config.FontSizes.LineSpace

	center, err := rgbrender.AlignPosition(rgbrender.CenterTop, canvas.Bounds(), b.textAreaWidth(), 16)
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
	wrter.FontSize = b.controller.config.FontSizes.Period

	scoreWrtr, err := scoreWriter(b.controller.config.FontSizes.Score)
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
		wrter.Write2(
			canvas,
			center,
			[]string{
				periodStr(liveGame.LiveData.Linescore.CurrentPeriod),
			},
			color.White,
		)
		wrter.FontSize = b.controller.config.FontSizes.PeriodTime
		wrter.Write2(
			canvas,
			center,
			[]string{
				"",
				liveGame.LiveData.Linescore.CurrentPeriodTimeRemaining,
			},
			color.White,
		)

		scoreWrtr.Write2(
			canvas,
			scoreAlign,
			[]string{
				scoreStr(liveGame),
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
	if err := b.renderHomeLogo(canvas, hKey); err != nil {
		return fmt.Errorf("failed to render home logo: %w", err)
	}
	if err := b.renderAwayLogo(canvas, aKey); err != nil {
		return fmt.Errorf("failed to render away logo: %w", err)
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

func scoreWriter(size float64) (*rgbrender.TextWriter, error) {
	fnt, err := rgbrender.FontFromAsset("github.com/robbydyer/sports:/assets/fonts/score.ttf")
	if err != nil {
		return nil, fmt.Errorf("failed to load font for score: %w", err)
	}

	wrtr := rgbrender.NewTextWriter(fnt, size)

	wrtr.YStartCorrection = -7

	return wrtr, nil
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
