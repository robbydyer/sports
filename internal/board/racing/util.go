package racingboard

import (
	"image"

	"github.com/robbydyer/sports/internal/rgbrender"
)

func (s *RacingBoard) getScheduleWriter(bounds image.Rectangle) (*rgbrender.TextWriter, error) {
	if s.scheduleWriter != nil {
		return s.scheduleWriter, nil
	}

	var err error
	s.scheduleWriter, err = rgbrender.DefaultTextWriter()
	if err != nil {
		return nil, err
	}

	if bounds.Dy() <= 256 {
		s.scheduleWriter.FontSize = 8.0
	} else {
		s.scheduleWriter.FontSize = 0.25 * float64(bounds.Dy())
	}

	if bounds.Dy() <= 256 {
		s.scheduleWriter.YStartCorrection = -2
	} else {
		s.scheduleWriter.YStartCorrection = -1 * ((bounds.Dy() / 32) + 1)
	}

	return s.scheduleWriter, nil
}

/*
func (s *RacingBoard) getScheduleWriter(bounds image.Rectangle) (*rgbrender.TextWriter, error) {
	if s.scheduleWriter != nil {
		return s.scheduleWriter, nil
	}

	var scheduleWriter *rgbrender.TextWriter

	if (bounds.Dx() == bounds.Dy()) && bounds.Dx() <= 32 {
		var err error
		scheduleWriter, err = rgbrender.DefaultTextWriter()
		if err != nil {
			return nil, err
		}
		scheduleWriter.FontSize = 0.25 * float64(bounds.Dy())
		scheduleWriter.YStartCorrection = -1 * ((bounds.Dy() / 32) + 1)
	} else {
		fnt, err := rgbrender.GetFont("score.ttf")
		if err != nil {
			return nil, fmt.Errorf("failed to load font for score: %w", err)
		}
		size := 0.5 * float64(bounds.Dy())
		scheduleWriter = rgbrender.NewTextWriter(fnt, size)
		yCorrect := math.Ceil(float64(3.0/32.0) * float64(bounds.Dy()))
		scheduleWriter.YStartCorrection = int(yCorrect * -1)
	}

	s.scheduleWriter = scheduleWriter

	return s.scheduleWriter, nil
}
*/
