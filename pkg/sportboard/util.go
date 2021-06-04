package sportboard

import (
	"fmt"
	"image"
	"math"
	"strings"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *SportBoard) logCanvas(canvas board.Canvas, msg string) {
	s.log.Debug(msg,
		zap.Int("minX", canvas.Bounds().Min.X),
		zap.Int("minY", canvas.Bounds().Min.Y),
		zap.Int("maxX", canvas.Bounds().Max.X),
		zap.Int("maxY", canvas.Bounds().Max.Y),
	)
}

func (s *SportBoard) getTimeWriter(canvasBounds image.Rectangle) (*rgbrender.TextWriter, error) {
	bounds := rgbrender.ZeroedBounds(canvasBounds)

	s.log.Debug("time writer bounds",
		zap.Int("minX", bounds.Min.X),
		zap.Int("minY", bounds.Min.Y),
		zap.Int("maxX", bounds.Max.X),
		zap.Int("maxY", bounds.Max.Y),
		zap.Int("starting minX", canvasBounds.Min.X),
		zap.Int("starting minY", canvasBounds.Min.Y),
		zap.Int("starting maxX", canvasBounds.Max.X),
		zap.Int("starting maxY", canvasBounds.Max.Y),
	)

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

	if bounds.Dy() <= 128 {
		timeWriter.FontSize = 8.0
		timeWriter.YStartCorrection = -2
	} else {
		timeWriter.FontSize = 0.25 * float64(bounds.Dy())
		timeWriter.YStartCorrection = -1 * ((bounds.Dy() / 32) + 1)
	}

	s.log.Debug("time writer font",
		zap.Float64("size", timeWriter.FontSize),
		zap.Int("Y correction", timeWriter.YStartCorrection),
	)

	s.Lock()
	defer s.Unlock()
	s.timeWriters[k] = timeWriter

	return timeWriter, nil
}

func (s *SportBoard) getScoreWriter(canvasBounds image.Rectangle) (*rgbrender.TextWriter, error) {
	bounds := rgbrender.ZeroedBounds(canvasBounds)

	k := fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy())
	w, ok := s.scoreWriters[k]
	if ok {
		s.log.Debug("using cached score writer")
		return w, nil
	}

	var scoreWriter *rgbrender.TextWriter

	if (bounds.Dx() == bounds.Dy()) && bounds.Dx() <= 32 {
		var err error
		scoreWriter, err = rgbrender.DefaultTextWriter()
		if err != nil {
			return nil, err
		}
		scoreWriter.FontSize = 0.25 * float64(bounds.Dy())
		scoreWriter.YStartCorrection = -1 * ((bounds.Dy() / 32) + 1)
	} else {
		fnt, err := rgbrender.GetFont("score.ttf")
		if err != nil {
			return nil, fmt.Errorf("failed to load font for score: %w", err)
		}
		size := 0.5 * float64(bounds.Dy())
		scoreWriter = rgbrender.NewTextWriter(fnt, size)
		yCorrect := math.Ceil(float64(3.0/32.0) * float64(bounds.Dy()))
		scoreWriter.YStartCorrection = int(yCorrect * -1)
	}

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
	bounds = rgbrender.ZeroedBounds(bounds)
	if bounds.Dx() == bounds.Dy() {
		return bounds.Dx() / 8
	}

	if bounds.Dx() >= 64 && bounds.Dy() <= 64 {
		return 16
	}
	return bounds.Dx() / 4
}

func scoreStr(g Game, homeSide string) (string, error) {
	a, err := g.AwayTeam()
	if err != nil {
		return "", err
	}
	h, err := g.HomeTeam()
	if err != nil {
		return "", err
	}

	if strings.ToLower(homeSide) == "left" {
		return fmt.Sprintf("%d-%d", h.Score(), a.Score()), nil
	}
	return fmt.Sprintf("%d-%d", a.Score(), h.Score()), nil
}
