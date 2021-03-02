package sportboard

import (
	"fmt"
	"image"
	"math"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *SportBoard) getTimeWriter(bounds image.Rectangle) (*rgbrender.TextWriter, error) {
	k := fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy())
	w, ok := s.timeWriters[k]
	if ok {
		s.log.Debug("using cached time writer")
		return w, nil
	}

	timeWriter, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return nil, err
	}

	timeWriter.FontSize = 0.125 * float64(bounds.Dx())
	timeWriter.YStartCorrection = -1 * ((bounds.Dy() / 32) + 1)

	s.log.Debug("time writer font",
		zap.Float64("size", timeWriter.FontSize),
		zap.Int("Y correction", timeWriter.YStartCorrection),
	)

	s.Lock()
	defer s.Unlock()
	s.timeWriters[k] = timeWriter

	return timeWriter, nil
}

func (s *SportBoard) getScoreWriter(bounds image.Rectangle) (*rgbrender.TextWriter, error) {
	k := fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy())
	w, ok := s.scoreWriters[k]
	if ok {
		s.log.Debug("using cached score writer")
		return w, nil
	}

	fnt, err := rgbrender.GetFont("score.ttf")
	if err != nil {
		return nil, fmt.Errorf("failed to load font for score: %w", err)
	}

	size := 0.25 * float64(bounds.Dx())

	scoreWriter := rgbrender.NewTextWriter(fnt, size)

	yCorrect := math.Ceil(float64(3.0/32.0) * float64(bounds.Dy()))
	scoreWriter.YStartCorrection = int(yCorrect * -1)

	s.log.Debug("score writer font",
		zap.Float64("size", scoreWriter.FontSize),
		zap.Int("Y correction", scoreWriter.YStartCorrection),
	)

	s.Lock()
	defer s.Unlock()
	s.scoreWriters[k] = scoreWriter
	return scoreWriter, nil
}

func (s *SportBoard) isFavoriteGame(game Game) (bool, error) {
	homeTeam, err := game.HomeTeam()
	if err != nil {
		return false, err
	}
	awayTeam, err := game.AwayTeam()
	if err != nil {
		return false, err
	}

	return (s.isFavorite(awayTeam.GetAbbreviation()) || s.isFavorite(homeTeam.GetAbbreviation())), nil
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
