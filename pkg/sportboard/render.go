package sportboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *SportBoard) renderLiveGame(ctx context.Context, canvas board.Canvas, liveGame Game, counter image.Image) error {
	awayTeam, err := liveGame.AwayTeam()
	if err != nil {
		return err
	}
	homeTeam, err := liveGame.HomeTeam()
	if err != nil {
		return err
	}

	// If this is a favorite team, we'll watch the scoreboard until the game is over
	isFavorite := (s.isFavorite(awayTeam.GetAbbreviation()) || s.isFavorite(homeTeam.GetAbbreviation()))

	timeWriter, err := s.getTimeWriter(canvas.Bounds())
	if err != nil {
		return err
	}

	scoreWriter, err := s.getScoreWriter(canvas.Bounds())
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		default:
		}

		quarter, err := liveGame.GetQuarter()
		if err != nil {
			return err
		}

		clock, err := liveGame.GetClock()
		if err != nil {
			return err
		}

		score, err := scoreStr(liveGame)
		if err != nil {
			return err
		}

		s.log.Debug("Live game",
			zap.String("score", score),
			zap.String("clock", clock),
		)

		var wg sync.WaitGroup
		errs := make(chan error, 1)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.RenderHomeLogo(ctx, canvas, homeTeam.GetAbbreviation()); err != nil {
				select {
				case errs <- fmt.Errorf("failed to render home logo: %w", err):
				default:
				}
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.RenderAwayLogo(ctx, canvas, awayTeam.GetAbbreviation()); err != nil {
				select {
				case errs <- fmt.Errorf("failed to render away logo: %w", err):
				default:
				}
			}
		}()

		// Wait, but with a timeout
		doneWait := make(chan struct{})
		go func() {
			defer close(doneWait)
			wg.Wait()
		}()

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		case <-doneWait:
			s.log.Debug("logo renderers completed",
				zap.String("home", homeTeam.GetAbbreviation()),
				zap.String("away", awayTeam.GetAbbreviation()),
			)
		case <-time.After(60 * time.Second):
			return fmt.Errorf("timed out waiting for board to draw logos")
		}

		select {
		case err := <-errs:
			if err != nil {
				return err
			}
		default:
		}

		if s.config.TimeColor == nil {
			s.config.TimeColor = color.White
		}

		_ = timeWriter.WriteAligned(
			rgbrender.CenterTop,
			canvas,
			canvas.Bounds(),
			[]string{
				quarter,
				clock,
			},
			s.config.TimeColor,
		)

		if s.config.ScoreColor == nil {
			s.config.ScoreColor = color.White
		}

		if s.config.HideFavoriteScore.Load() && isFavorite {
			s.log.Warn("hiding score for favorite team")
		} else {
			_ = scoreWriter.WriteAligned(
				rgbrender.CenterBottom,
				canvas,
				canvas.Bounds(),
				[]string{
					score,
				},
				s.config.ScoreColor,
			)
		}

		draw.Draw(canvas, canvas.Bounds(), counter, image.Point{}, draw.Over)

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		default:
		}

		s.log.Debug("rendering sportboard",
			zap.String("home", homeTeam.GetAbbreviation()),
			zap.String("away", awayTeam.GetAbbreviation()),
		)
		if err := canvas.Render(); err != nil {
			return err
		}

		if !(isFavorite && s.config.FavoriteSticky.Load()) {
			return nil
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(s.config.boardDelay - 3*time.Second):
		}

		liveGame, err = liveGame.GetUpdate(ctx)
		if err != nil {
			return fmt.Errorf("failed to update stickey game: %w", err)
		}
		s.log.Debug("updated live sticky game")

		over, err := liveGame.IsComplete()
		if err != nil {
			return err
		}
		if over {
			s.log.Info("live game is over", zap.Int("game ID", liveGame.GetID()))
			return nil
		}
	}
}

func (s *SportBoard) renderUpcomingGame(ctx context.Context, canvas board.Canvas, liveGame Game, counter image.Image) error {
	awayTeam, err := liveGame.AwayTeam()
	if err != nil {
		return err
	}
	homeTeam, err := liveGame.HomeTeam()
	if err != nil {
		return err
	}

	timeWriter, err := s.getTimeWriter(canvas.Bounds())
	if err != nil {
		return err
	}

	scoreWriter, err := s.getScoreWriter(canvas.Bounds())
	if err != nil {
		return err
	}

	gameTimeStr := ""

	if is, err := liveGame.IsPostponed(); err == nil && is {
		s.log.Debug("game was postponed", zap.Int("game ID", liveGame.GetID()))
		gameTimeStr = "PPD"
	} else {
		gameTime, err := liveGame.GetStartTime(ctx)
		if err != nil {
			return err
		}
		gameTimeStr = gameTime.Local().Format("3:04PM")
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.RenderHomeLogo(ctx, canvas, homeTeam.GetAbbreviation()); err != nil {
			select {
			case errs <- fmt.Errorf("failed to render home logo: %w", err):
			default:
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.RenderAwayLogo(ctx, canvas, awayTeam.GetAbbreviation()); err != nil {
			select {
			case errs <- fmt.Errorf("failed to render away logo: %w", err):
			default:
			}
		}
	}()

	// Wait, but with a timeout
	doneWait := make(chan struct{})
	go func() {
		defer close(doneWait)
		wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
	case <-doneWait:
		s.log.Debug("logo renderers completed",
			zap.String("home", homeTeam.GetAbbreviation()),
			zap.String("away", awayTeam.GetAbbreviation()),
		)
	case <-time.After(60 * time.Second):
		return fmt.Errorf("timed out waiting for board to draw logos")
	}

	select {
	case err := <-errs:
		if err != nil {
			return err
		}
	default:
	}

	_ = timeWriter.WriteAligned(
		rgbrender.CenterTop,
		canvas,
		canvas.Bounds(),
		[]string{
			gameTimeStr,
		},
		s.config.TimeColor,
	)

	_ = scoreWriter.WriteAligned(
		rgbrender.CenterCenter,
		canvas,
		canvas.Bounds(),
		[]string{
			"VS",
		},
		s.config.ScoreColor,
	)

	draw.Draw(canvas, canvas.Bounds(), counter, image.Point{}, draw.Over)

	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
	default:
	}

	return canvas.Render()
}

func (s *SportBoard) renderCompleteGame(ctx context.Context, canvas board.Canvas, liveGame Game, counter image.Image) error {
	awayTeam, err := liveGame.AwayTeam()
	if err != nil {
		return err
	}
	homeTeam, err := liveGame.HomeTeam()
	if err != nil {
		return err
	}

	timeWriter, err := s.getTimeWriter(canvas.Bounds())
	if err != nil {
		return err
	}

	scoreWriter, err := s.getScoreWriter(canvas.Bounds())
	if err != nil {
		return err
	}

	score, err := scoreStr(liveGame)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.RenderHomeLogo(ctx, canvas, homeTeam.GetAbbreviation()); err != nil {
			select {
			case errs <- fmt.Errorf("failed to render home logo: %w", err):
			default:
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.RenderAwayLogo(ctx, canvas, awayTeam.GetAbbreviation()); err != nil {
			select {
			case errs <- fmt.Errorf("failed to render away logo: %w", err):
			default:
			}
		}
	}()

	// Wait, but with a timeout
	doneWait := make(chan struct{})
	go func() {
		defer close(doneWait)
		wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
	case <-doneWait:
		s.log.Debug("logo renderers completed",
			zap.String("home", homeTeam.GetAbbreviation()),
			zap.String("away", awayTeam.GetAbbreviation()),
		)
	case <-time.After(60 * time.Second):
		_ = canvas.Clear()
		return fmt.Errorf("timed out waiting for board to draw logos")
	}

	select {
	case err := <-errs:
		if err != nil {
			_ = canvas.Clear()
			return err
		}
	default:
	}

	_ = timeWriter.WriteAligned(
		rgbrender.CenterTop,
		canvas,
		canvas.Bounds(),
		[]string{
			"FINAL",
		},
		s.config.TimeColor,
	)

	isFavorite := (s.isFavorite(awayTeam.GetAbbreviation()) || s.isFavorite(homeTeam.GetAbbreviation()))

	if s.config.HideFavoriteScore.Load() && isFavorite {
		s.log.Warn("hiding score for favorite team")
	} else {
		_ = scoreWriter.WriteAligned(
			rgbrender.CenterBottom,
			canvas,
			canvas.Bounds(),
			[]string{
				score,
			},
			s.config.ScoreColor,
		)
	}

	draw.Draw(canvas, canvas.Bounds(), counter, image.Point{}, draw.Over)

	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
	default:
	}

	return canvas.Render()
}
