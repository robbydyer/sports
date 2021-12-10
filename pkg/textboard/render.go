package textboard

import (
	"context"
	"fmt"
	"image"
	"image/color"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
	"go.uber.org/zap"
)

func (s *TextBoard) render(ctx context.Context, canvas board.Canvas, text string) error {
	zeroed := rgbrender.ZeroedBounds(canvas.Bounds())
	lengths, err := s.writer.MeasureStrings(canvas, []string{text})
	if err != nil {
		return err
	}
	if len(lengths) < 1 {
		return fmt.Errorf("failed to measure text")
	}
	bounds := image.Rect(zeroed.Min.X, zeroed.Min.Y, lengths[0], zeroed.Max.Y)

	s.log.Debug("writing headline",
		zap.String("text", text),
		zap.Int("pix length", lengths[0]),
		zap.Int("X", bounds.Min.X),
		zap.Int("Y", bounds.Min.Y),
		zap.Int("X", bounds.Max.X),
		zap.Int("Y", bounds.Max.Y),
	)
	canvas.SetWidth(lengths[0])
	_ = s.writer.WriteAligned(
		rgbrender.CenterCenter,
		canvas,
		bounds,
		[]string{text},
		color.White,
	)

	return canvas.Render(ctx)

	// return nil
}
