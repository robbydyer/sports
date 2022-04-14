package weatherboard

import (
	"fmt"
	"image"
	"math"

	"github.com/robbydyer/sports/internal/rgbrender"
)

func (s *WeatherBoard) getSmallWriter(canvasBounds image.Rectangle) (*rgbrender.TextWriter, error) {
	s.Lock()
	defer s.Unlock()

	if s.smallWriter != nil {
		return s.smallWriter, nil
	}
	bounds := rgbrender.ZeroedBounds(canvasBounds)

	writer, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return nil, err
	}

	if bounds.Dy() <= 256 {
		writer.FontSize = 8.0
		writer.YStartCorrection = -2
	} else {
		writer.FontSize = 0.25 * float64(bounds.Dy())
		writer.YStartCorrection = -1 * ((bounds.Dy() / 32) + 1)
	}

	s.smallWriter = writer
	return writer, nil
}

func (s *WeatherBoard) getBigWriter(canvasBounds image.Rectangle) (*rgbrender.TextWriter, error) {
	s.Lock()
	defer s.Unlock()

	if s.bigWriter != nil {
		return s.bigWriter, nil
	}
	bounds := rgbrender.ZeroedBounds(canvasBounds)

	var writer *rgbrender.TextWriter

	if (bounds.Dx() == bounds.Dy()) && bounds.Dx() <= 32 {
		var err error
		writer, err = rgbrender.DefaultTextWriter()
		if err != nil {
			return nil, err
		}
		writer.FontSize = 0.25 * float64(bounds.Dy())
		writer.YStartCorrection = -1 * ((bounds.Dy() / 32) + 1)
	} else {
		fnt, err := rgbrender.GetFont("score.ttf")
		if err != nil {
			return nil, fmt.Errorf("failed to load font for symbol: %w", err)
		}
		size := 0.5 * float64(bounds.Dy())
		writer = rgbrender.NewTextWriter(fnt, size)
		yCorrect := math.Ceil(float64(3.0/32.0) * float64(bounds.Dy()))
		writer.YStartCorrection = int(yCorrect * -1)
	}

	s.bigWriter = writer
	return writer, nil
}
