package weatherboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"time"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

var orange = color.RGBA{R: 255, G: 165, B: 0}
var blue = color.RGBA{R: 30, G: 144, B: 255}

func (w *WeatherBoard) drawForecast(ctx context.Context, canvas board.Canvas, f *Forecast) error {
	canvasBounds := rgbrender.ZeroedBounds(canvas.Bounds())

	iconBounds := image.Rect(canvasBounds.Min.X, canvasBounds.Min.Y, canvasBounds.Max.X/2, canvasBounds.Max.Y)
	tempBounds := image.Rect(canvasBounds.Max.X/2, canvasBounds.Min.Y, canvasBounds.Max.X, canvasBounds.Max.Y)
	smallWriter, err := w.getSmallWriter(canvasBounds)
	if err != nil {
		return err
	}

	bigWriter, err := w.getBigWriter(canvasBounds)
	if err != nil {
		return err
	}

	if f.Temperature != nil {
		if err := bigWriter.WriteAligned(
			rgbrender.RightTop,
			canvas,
			tempBounds,
			[]string{
				fmt.Sprintf("%.0f%s", *f.Temperature, f.TempUnit),
			},
			color.White,
		); err != nil {
			return err
		}
	}

	clrCodes := &rgbrender.ColorChar{
		BoxClr: color.Black,
		Lines:  []*rgbrender.ColorCharLine{},
	}

	if f.HighTemp != nil && f.LowTemp != nil {
		line := &rgbrender.ColorCharLine{}
		high := fmt.Sprintf("%0.f", *f.HighTemp)
		for i := 0; i < len(high); i++ {
			line.Chars = append(line.Chars, string(high[i]))
			line.Clrs = append(line.Clrs, orange)
		}
		line.Chars = append(line.Chars, "/")
		line.Clrs = append(line.Clrs, color.White)

		low := fmt.Sprintf("%0.f", *f.LowTemp)
		for i := 0; i < len(low); i++ {
			line.Chars = append(line.Chars, string(low[i]))
			line.Clrs = append(line.Clrs, blue)
		}
		clrCodes.Lines = append(clrCodes.Lines, line)
	}

	humidity := strings.Split(fmt.Sprintf("%d", f.Humidity), "")
	line := &rgbrender.ColorCharLine{
		Chars: humidity,
	}
	for i := 0; i < len(humidity); i++ {
		line.Clrs = append(line.Clrs, color.White)
	}

	clrCodes.Lines = append(clrCodes.Lines, line)

	if err := smallWriter.WriteAlignedColorCodes(
		rgbrender.RightBottom,
		canvas,
		tempBounds,
		clrCodes,
	); err != nil {
		return err
	}

	draw.Draw(canvas, iconBounds, f.Icon, image.Point{}, draw.Over)

	timeStr := "Now"

	if f.IsHourly {
		timeStr = f.Time.Format("3:04PM")
	} else {
		_, mo, day := f.Time.Date()
		wkd := f.Time.Weekday()
		timeStr = fmt.Sprintf("%d/%d %s", mo, day, shortWeekday(wkd))
	}

	if err := smallWriter.WriteAlignedBoxed(
		rgbrender.CenterBottom,
		canvas,
		iconBounds,
		[]string{
			timeStr,
		},
		color.White,
		color.Black,
	); err != nil {
		return err
	}

	return nil
}

func shortWeekday(d time.Weekday) string {
	switch d {
	case time.Sunday:
		return "Sun"
	case time.Monday:
		return "Mon"
	case time.Tuesday:
		return "Tue"
	case time.Wednesday:
		return "Wed"
	case time.Thursday:
		return "Thu"
	case time.Friday:
		return "Fri"
	case time.Saturday:
		return "Sat"
	default:
		return ""
	}
}
