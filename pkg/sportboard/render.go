package sportboard

import (
	"context"
	"fmt"
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
	isFavorite := (s.isFavorite(awayTeam.GetAbbreviation()) || s.isFavorite(homeTeam.GetAbbreviation())) && s.config.FavoriteSticky

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
		if err := s.RenderHomeLogo(canvas, homeTeam.GetAbbreviation()); err != nil {
			return err
		}
		if err := s.RenderAwayLogo(canvas, awayTeam.GetAbbreviation()); err != nil {
			return err
		}
		timeWriter.Write(
			canvas,
			timeAlign,
			[]string{
				quarterStr(quarter),
				clock,
			},
			s.config.TimeColor,
		)

		scoreWriter.Write(
			canvas,
			scoreAlign,
			[]string{
				score,
			},
			s.config.ScoreColor,
		)

		if err := canvas.Render(); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		case <-time.After(s.config.boardDelay):
		}

		if !isFavorite {
			return nil
		}

		updatedGame, err := liveGame.GetUpdate(ctx)
		if err != nil {
			return err
		}

		over, err := updatedGame.IsComplete()
		if err != nil {
			return err
		}
		if over {
			return nil
		}

		liveGame = updatedGame
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

	// If this is a favorite team, we'll watch the scoreboard until the game is over
	isFavorite := (s.isFavorite(awayTeam.GetAbbreviation()) || s.isFavorite(homeTeam.GetAbbreviation())) && s.config.FavoriteSticky

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

	for {
		if err := s.RenderHomeLogo(canvas, homeTeam.GetAbbreviation()); err != nil {
			return err
		}
		if err := s.RenderAwayLogo(canvas, awayTeam.GetAbbreviation()); err != nil {
			return err
		}
		timeWriter.Write(
			canvas,
			timeAlign,
			[]string{
				gameTimeStr,
			},
			s.config.TimeColor,
		)

		scoreWriter.Write(
			canvas,
			scoreAlign,
			[]string{
				"VS",
			},
			s.config.ScoreColor,
		)

		if err := canvas.Render(); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		case <-time.After(s.config.boardDelay):
		}

		if !isFavorite {
			return nil
		}

		updatedGame, err := liveGame.GetUpdate(ctx)
		if err != nil {
			return err
		}

		over, err := updatedGame.IsComplete()
		if err != nil {
			return err
		}
		if over {
			return nil
		}

		liveGame = updatedGame
	}
}

func (s *SportBoard) renderCompleteGame(ctx context.Context, canvas *rgb.Canvas, liveGame Game) error {
	awayTeam, err := liveGame.AwayTeam()
	if err != nil {
		return err
	}
	homeTeam, err := liveGame.HomeTeam()
	if err != nil {
		return err
	}

	// If this is a favorite team, we'll watch the scoreboard until the game is over
	isFavorite := (s.isFavorite(awayTeam.GetAbbreviation()) || s.isFavorite(homeTeam.GetAbbreviation())) && s.config.FavoriteSticky

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

	for {
		if err := s.RenderHomeLogo(canvas, homeTeam.GetAbbreviation()); err != nil {
			return err
		}
		if err := s.RenderAwayLogo(canvas, awayTeam.GetAbbreviation()); err != nil {
			return err
		}
		timeWriter.Write(
			canvas,
			timeAlign,
			[]string{
				"FINAL",
			},
			s.config.TimeColor,
		)

		scoreWriter.Write(
			canvas,
			scoreAlign,
			[]string{
				score,
			},
			s.config.ScoreColor,
		)

		if err := canvas.Render(); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		case <-time.After(s.config.boardDelay):
		}

		if !isFavorite {
			return nil
		}

		updatedGame, err := liveGame.GetUpdate(ctx)
		if err != nil {
			return err
		}

		over, err := updatedGame.IsComplete()
		if err != nil {
			return err
		}
		if over {
			return nil
		}

		liveGame = updatedGame
	}
}
