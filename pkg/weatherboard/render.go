package weatherboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (w *WeatherBoard) drawForecast(ctx context.Context, canvas board.Canvas) error {
	canvasBounds := rgbrender.ZeroedBounds(canvas.Bounds())

	f, err := w.api.CurrentForecast(ctx, w.config.City, w.config.State, w.config.Country, canvasBounds)
	if err != nil {
		return err
	}

	tempBounds := image.Rect(canvasBounds.Max.X/2, canvasBounds.Min.Y, canvasBounds.Max.X, canvasBounds.Max.Y)
	smallWriter, err := w.getSmallWriter(canvasBounds)
	if err != nil {
		return err
	}

	if err := smallWriter.WriteAligned(
		rgbrender.CenterCenter,
		canvas,
		tempBounds,
		[]string{
			fmt.Sprintf("%.0f%s", f.Temperature, f.TempUnit),
		},
		color.White,
	); err != nil {
		return err
	}

	draw.Draw(canvas, canvasBounds, f.Icon, image.Point{}, draw.Over)

	return nil
}
