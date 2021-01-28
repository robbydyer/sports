package sportboard

import (
	"context"
	"fmt"
	"time"

	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

const scorePollRate = 30 * time.Second

func (s *SportBoard) isFavorite(abbrev string) bool {
	for _, a := range s.config.FavoriteTeams {
		if abbrev == a {
			return true
		}
	}

	return false
}

func (b *SportBoard) textAreaWidth() int {
	return b.matrixBounds.Dx() / 4
}

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

	timeWriter, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return err
	}
	timeWriter.LineSpace = s.config.TimeFont.LineSpace
	timeWriter.FontSize = s.config.TimeFont.Size

	timeAlign, err := rgbrender.AlignPosition(rgbrender.CenterTop, canvas.Bounds(), s.textAreaWidth(), s.matrixBounds.Dy()/2)
	if err != nil {
		return err
	}

	scoreWriter, err := scoreWriter(s.config.ScoreFont.Size)
	if err != nil {
		return err
	}

	scoreAlign, err := rgbrender.AlignPosition(rgbrender.CenterBottom, canvas.Bounds(), s.textAreaWidth(), s.matrixBounds.Dy()/2)
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

		if !isFavorite {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		case <-time.After(scorePollRate):
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
