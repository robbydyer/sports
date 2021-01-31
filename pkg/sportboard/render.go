package sportboard

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"time"

	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
)

func (s *SportBoard) renderLiveGame(ctx context.Context, canvas *rgb.Canvas, liveGame Game) error {
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

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		default:
		}
		if err := s.RenderHomeLogo(canvas, homeTeam.GetAbbreviation()); err != nil {
			return err
		}
		if err := s.RenderAwayLogo(canvas, awayTeam.GetAbbreviation()); err != nil {
			return err
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

		if s.config.HideFavoriteScore && isFavorite {
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

		if err := canvas.Render(); err != nil {
			return err
		}

		if !(isFavorite && s.config.FavoriteSticky) {
			return nil
		}

		var updatedGame Game
		go func() {
			updatedGame, err = liveGame.GetUpdate(ctx)
			if err != nil {
				s.log.Errorf("failed to update game for board refresh: %s", err.Error())
				return
			}
			liveGame = updatedGame
		}()

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		case <-time.After(s.config.boardDelay):
		}

		over, err := liveGame.IsComplete()
		if err != nil {
			return err
		}
		if over {
			s.log.Infof("live game %d is over", liveGame.GetID())
			return nil
		}
	}
}

func (s *SportBoard) renderUpcomingGame(ctx context.Context, canvas *rgb.Canvas, liveGame Game) error {
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

	gameTime, err := liveGame.GetStartTime(ctx)
	if err != nil {
		return err
	}
	gameTimeStr := gameTime.Local().Format("3:04PM")

	if err := s.RenderHomeLogo(canvas, homeTeam.GetAbbreviation()); err != nil {
		return err
	}
	if err := s.RenderAwayLogo(canvas, awayTeam.GetAbbreviation()); err != nil {
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

func (s *SportBoard) renderCompleteGame(canvas *rgb.Canvas, liveGame Game) error {
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

	if err := s.RenderHomeLogo(canvas, homeTeam.GetAbbreviation()); err != nil {
		return err
	}
	if err := s.RenderAwayLogo(canvas, awayTeam.GetAbbreviation()); err != nil {
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

	_ = scoreWriter.Write(
		canvas,
		scoreAlign,
		[]string{
			score,
		},
		s.config.ScoreColor,
	)

	draw.Draw(canvas, canvas.Bounds(), s.counter, image.Point{}, draw.Over)

	return canvas.Render()
}
