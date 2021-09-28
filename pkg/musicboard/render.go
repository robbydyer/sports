package musicboard

import (
	"context"
	"image"
	"image/color"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (m *MusicBoard) render(ctx context.Context, canvas board.Canvas, track *Track) error {
	zBounds := rgbrender.ZeroedBounds(canvas.Bounds())
	writer, err := m.getTrackWriter(zBounds)
	if err != nil {
		return err
	}

	return writer.WriteAligned(rgbrender.CenterCenter,
		canvas,
		zBounds,
		[]string{
			track.Artist,
		},
		color.White,
	)
}

func (m *MusicBoard) getTrackWriter(canvasBounds image.Rectangle) (*rgbrender.TextWriter, error) {
	m.Lock()
	defer m.Unlock()

	if m.trackWriter != nil {
		return m.trackWriter, nil
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

	m.trackWriter = writer
	return writer, nil
}
