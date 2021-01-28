package sportboard

import (
	"fmt"
	"image"
	"time"

	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *SportBoard) timeWriter() (*rgbrender.TextWriter, image.Rectangle, error) {
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

	return timeWriter, timeAlign, nil
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

// Today is sometimes actually yesterday
func Today() time.Time {
	if time.Now().Local().Hour() < 4 {
		return time.Now().AddDate(0, 0, 01).Local()
	}

	return time.Now().Local()
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
