package sportboard

import (
	"fmt"
	"image"
	"math"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *SportBoard) getTimeWriter(bounds image.Rectangle) (*rgbrender.TextWriter, image.Rectangle, error) {
	var timeAlign image.Rectangle
	timeWriter, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return nil, timeAlign, err
	}

	timeWriter.FontSize = 0.125 * float64(bounds.Dx())
	timeWriter.YStartCorrection = -1 * ((bounds.Dy() / 32) - 1)

	s.log.Debug("time writer font",
		zap.Float64("size", timeWriter.FontSize),
		zap.Int("Y correction", timeWriter.YStartCorrection),
	)

	timeAlign, err = rgbrender.AlignPosition(rgbrender.CenterTop, bounds, s.textAreaWidth(bounds), bounds.Dy()/2)
	if err != nil {
		return nil, timeAlign, err
	}

	s.timeWriter = timeWriter
	s.timeAlign = timeAlign
	return timeWriter, timeAlign, nil
}

func (s *SportBoard) getScoreWriter(bounds image.Rectangle) (*rgbrender.TextWriter, image.Rectangle, error) {
	var scoreAlign image.Rectangle
	fnt, err := rgbrender.GetFont("score.ttf")
	if err != nil {
		return nil, scoreAlign, fmt.Errorf("failed to load font for score: %w", err)
	}

	size := 0.25 * float64(bounds.Dx())

	scoreWriter := rgbrender.NewTextWriter(fnt, size)

	yCorrect := math.Ceil(float64(7.0/32.0) * float64(bounds.Dy()))
	scoreWriter.YStartCorrection = int(yCorrect * -1)

	s.log.Debug("score writer font",
		zap.Float64("size", scoreWriter.FontSize),
		zap.Int("Y correction", scoreWriter.YStartCorrection),
		zap.Float64("Y correction float", yCorrect),
	)

	scoreAlign, err = rgbrender.AlignPosition(rgbrender.CenterBottom, bounds, s.textAreaWidth(bounds), bounds.Dy()/2)
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

func (s *SportBoard) textAreaWidth(bounds image.Rectangle) int {
	return bounds.Dx() / 4
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
