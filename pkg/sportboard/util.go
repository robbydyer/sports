package sportboard

import (
	"fmt"
	"image"

	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *SportBoard) getTimeWriter() (*rgbrender.TextWriter, image.Rectangle, error) {
	if s.timeWriter != nil {
		return s.timeWriter, s.timeAlign, nil
	}

	var timeAlign image.Rectangle
	timeWriter, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return nil, timeAlign, err
	}

	if s.config.TimeFont == nil {
		s.config.TimeFont = &FontConfig{
			Size:      8,
			LineSpace: 0,
		}
	}
	if timeWriter.FontSize == 0 {
		timeWriter.FontSize = 8
	}

	timeAlign, err = rgbrender.AlignPosition(rgbrender.CenterTop, s.matrixBounds, s.textAreaWidth(), s.matrixBounds.Dy()/2)
	if err != nil {
		return nil, timeAlign, err
	}

	s.timeWriter = timeWriter
	s.timeAlign = timeAlign
	return timeWriter, timeAlign, nil
}

func (s *SportBoard) getScoreWriter() (*rgbrender.TextWriter, image.Rectangle, error) {
	if s.scoreWriter != nil {
		return s.scoreWriter, s.scoreAlign, nil
	}

	var scoreAlign image.Rectangle
	fnt, err := rgbrender.GetFont("score.ttf")
	if err != nil {
		return nil, scoreAlign, fmt.Errorf("failed to load font for score: %w", err)
	}

	if s.config.ScoreFont == nil {
		s.config.ScoreFont = &FontConfig{
			Size:      16,
			LineSpace: 0,
		}
	}

	if s.config.ScoreFont.Size == 0 {
		s.config.ScoreFont.Size = 16
	}

	scoreWriter := rgbrender.NewTextWriter(fnt, s.config.ScoreFont.Size)

	scoreWriter.YStartCorrection = -7

	scoreAlign, err = rgbrender.AlignPosition(rgbrender.CenterBottom, s.matrixBounds, s.textAreaWidth(), s.matrixBounds.Dy()/2)
	if err != nil {
		return nil, scoreAlign, err
	}

	s.scoreWriter = scoreWriter
	s.scoreAlign = scoreAlign
	return scoreWriter, scoreAlign, nil
}

func (s *SportBoard) isFavorite(abbrev string) bool {
	for _, a := range s.config.FavoriteTeams {
		if abbrev == a {
			return true
		}
	}

	return false
}

func (s *SportBoard) textAreaWidth() int {
	return s.matrixBounds.Dx() / 4
}

func scoreStr(g Game) (string, error) {
	a, err := g.AwayTeam()
	if err != nil {
		return "", err
	}
	h, err := g.HomeTeam()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d-%d", h.Score(), a.Score()), nil
}
