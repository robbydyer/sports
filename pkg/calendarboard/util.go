package calendarboard

import (
	"image"

	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *CalendarBoard) getScheduleWriter(bounds image.Rectangle) (*rgbrender.TextWriter, error) {
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
