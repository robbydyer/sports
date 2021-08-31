package weatherboard

import (
	"context"
	"fmt"
	"image"
	"image/color"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (w *WeatherBoard) drawForecast(ctx context.Context, canvas board.Canvas) error {
	f, err := w.api.CurrentForecast(ctx, w.config.CityID)
	if err != nil {
		return err
	}

	canvasBounds := rgbrender.ZeroedBounds(canvas.Bounds())
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

	return nil
}
