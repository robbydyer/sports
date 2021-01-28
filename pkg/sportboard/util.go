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
	timeWriter.LineSpace = s.config.TimeFont.LineSpace
	timeWriter.FontSize = s.config.TimeFont.Size

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
	fnt, err := rgbrender.FontFromAsset("github.com/robbydyer/sports:/assets/fonts/score.ttf")
	if err != nil {
		return nil, scoreAlign, fmt.Errorf("failed to load font for score: %w", err)
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

func (b *SportBoard) textAreaWidth() int {
	return b.matrixBounds.Dx() / 4
}

func quarterStr(period int) string {
	switch period {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	case 4:
		return "4th"
	default:
		return ""
	}
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
