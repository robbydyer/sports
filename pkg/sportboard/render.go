package sportboard

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
)

func (s *SportBoard) renderLiveGame(ctx context.Context, canvas board.Canvas, liveGame Game) error {
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

	timeWriter, timeAlign, err := s.getTimeWriter()
	if err != nil {
		return err
	}

	scoreWriter, scoreAlign, err := s.getScoreWriter()
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

		if err := s.RenderHomeLogo(ctx, canvas, homeTeam.GetAbbreviation()); err != nil {
			return fmt.Errorf("failed to render home logo: %w", err)
		}
		if err := s.RenderAwayLogo(ctx, canvas, awayTeam.GetAbbreviation()); err != nil {
			return fmt.Errorf("failed to render away logo: %w", err)
		}
		if err := timeWriter.Write(
			canvas,
			timeAlign,
			[]string{
				quarter,
				clock,
			},
			s.config.TimeColor,
		); err != nil {
			return fmt.Errorf("failed to write quarter and clock: %w", err)
		}

		if s.config.HideFavoriteScore.Load() && isFavorite {
			s.log.Warn("hiding score for favorite team")
		} else {
			if err := scoreWriter.Write(
				canvas,
				scoreAlign,
				[]string{
					score,
				},
				s.config.ScoreColor,
			); err != nil {
				return fmt.Errorf("failed to write score: %w", err)
			}
		}

		draw.Draw(canvas, canvas.Bounds(), s.counter, image.Point{}, draw.Over)

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		default:
		}

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

func (s *SportBoard) renderUpcomingGame(ctx context.Context, canvas board.Canvas, liveGame Game) error {
	renderCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	awayTeam, err := liveGame.AwayTeam()
	if err != nil {
		return err
	}
	homeTeam, err := liveGame.HomeTeam()
	if err != nil {
		return err
	}

	timeWriter, timeAlign, err := s.getTimeWriter()
	if err != nil {
		return err
	}

	scoreWriter, scoreAlign, err := s.getScoreWriter()
	if err != nil {
		return err
	}

	var gameTimeStr string

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

	if err := s.RenderHomeLogo(renderCtx, canvas, homeTeam.GetAbbreviation()); err != nil {
		return err
	}
	if err := s.RenderAwayLogo(renderCtx, canvas, awayTeam.GetAbbreviation()); err != nil {
		return err
	}
	_ = timeWriter.Write(
		canvas,
		timeAlign,
		[]string{
			gameTimeStr,
		},
		s.config.TimeColor,
	)

	_ = scoreWriter.Write(
		canvas,
		scoreAlign,
		[]string{
			"VS",
		},
		s.config.ScoreColor,
	)

	draw.Draw(canvas, canvas.Bounds(), s.counter, image.Point{}, draw.Over)

	return canvas.Render()
}

func (s *SportBoard) renderCompleteGame(ctx context.Context, canvas board.Canvas, liveGame Game) error {
	renderCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	awayTeam, err := liveGame.AwayTeam()
	if err != nil {
		return err
	}
	homeTeam, err := liveGame.HomeTeam()
	if err != nil {
		return err
	}

	timeWriter, timeAlign, err := s.getTimeWriter()
	if err != nil {
		return err
	}

	scoreWriter, scoreAlign, err := s.getScoreWriter()
	if err != nil {
		return err
	}

	score, err := scoreStr(liveGame)
	if err != nil {
		return err
	}

	if err := s.RenderHomeLogo(renderCtx, canvas, homeTeam.GetAbbreviation()); err != nil {
		return err
	}
	if err := s.RenderAwayLogo(renderCtx, canvas, awayTeam.GetAbbreviation()); err != nil {
		return err
	}
	_ = timeWriter.Write(
		canvas,
		timeAlign,
		[]string{
			"FINAL",
		},
		s.config.TimeColor,
	)

	isFavorite := (s.isFavorite(awayTeam.GetAbbreviation()) || s.isFavorite(homeTeam.GetAbbreviation()))

	if s.config.HideFavoriteScore.Load() && isFavorite {
		s.log.Warn("hiding score for favorite team")
	} else {
		_ = scoreWriter.Write(
			canvas,
			scoreAlign,
			[]string{
				score,
			},
			s.config.ScoreColor,
		)
	}

	draw.Draw(canvas, canvas.Bounds(), s.counter, image.Point{}, draw.Over)

	return canvas.Render()
}
