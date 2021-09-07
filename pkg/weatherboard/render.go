package weatherboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"time"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

const sectionBufferRatio = 0.0625

var (
	orange = color.RGBA{R: 255, G: 165, B: 0}
	blue   = color.RGBA{R: 30, G: 144, B: 255}
)

func (w *WeatherBoard) drawForecast(ctx context.Context, canvas board.Canvas, f *Forecast) error {
	canvasBounds := rgbrender.ZeroedBounds(canvas.Bounds())

	spacing := int(math.Ceil(sectionBufferRatio*float64(canvasBounds.Dx()))) / 2

	iconBounds := image.Rect(canvasBounds.Min.X, canvasBounds.Min.Y, (canvasBounds.Max.X/2)-spacing, canvasBounds.Max.Y)
	tempBounds := image.Rect((canvasBounds.Max.X/2)+spacing, canvasBounds.Min.Y, canvasBounds.Max.X, canvasBounds.Max.Y)
	smallWriter, err := w.getSmallWriter(canvasBounds)
	if err != nil {
		return err
	}

	bigWriter, err := w.getBigWriter(canvasBounds)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}

	if f.Icon != nil {
		i, err := f.Icon.RenderRightAlignedWithEnd(ctx, iconBounds, iconBounds.Max.X)
		if err != nil {
			return fmt.Errorf("failed to render weather icon: %w", err)
		}

		draw.Draw(canvas, iconBounds, i, image.Point{}, draw.Over)
	}

	if f.Temperature != nil {
		if err := bigWriter.WriteAligned(
			rgbrender.LeftTop,
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

	if l := w.tempHighLowLine(f); l != nil {
		clrCodes.Lines = append(clrCodes.Lines, l)
	}

	if l := w.rainLine(f); l != nil {
		clrCodes.Lines = append(clrCodes.Lines, l)
	}

	if l := w.humidityLine(f); l != nil {
		clrCodes.Lines = append(clrCodes.Lines, l)
	}

	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}

	if err := smallWriter.WriteAlignedColorCodes(
		rgbrender.LeftBottom,
		canvas,
		tempBounds,
		clrCodes,
	); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}

	timeStr := "Now"

	if f.IsHourly {
		timeStr = f.Time.Format("3:04PM")
	} else if f.Time.YearDay() != time.Now().Local().YearDay() {
		_, mo, day := f.Time.Date()
		wkd := f.Time.Weekday()
		timeStr = fmt.Sprintf("%d/%d %s", mo, day, shortWeekday(wkd))
	}

	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}

	err = smallWriter.WriteAlignedBoxed(
		rgbrender.RightBottom,
		canvas,
		iconBounds,
		[]string{
			timeStr,
		},
		color.White,
		color.Black,
	)

	return err
}

func (w *WeatherBoard) rainLine(f *Forecast) *rgbrender.ColorCharLine {
	if f.PrecipChance == nil {
		return nil
	}
	c := strings.Split(fmt.Sprintf("Prcp: %d%%", *f.PrecipChance), "")
	l := &rgbrender.ColorCharLine{
		Chars: c,
		Clrs:  colors(color.White, c),
	}

	return l
}

func (w *WeatherBoard) tempHighLowLine(f *Forecast) *rgbrender.ColorCharLine {
	if f == nil || f.HighTemp == nil && f.LowTemp == nil {
		return nil
	}

	line := &rgbrender.ColorCharLine{}
	high := fmt.Sprintf("%0.fF", *f.HighTemp)
	for i := 0; i < len(high); i++ {
		line.Chars = append(line.Chars, string(high[i]))
		line.Clrs = append(line.Clrs, orange)
	}
	line.Chars = append(line.Chars, "/")
	line.Clrs = append(line.Clrs, color.White)

	low := fmt.Sprintf("%0.fF", *f.LowTemp)
	for i := 0; i < len(low); i++ {
		line.Chars = append(line.Chars, string(low[i]))
		line.Clrs = append(line.Clrs, blue)
	}

	return line
}

func (w *WeatherBoard) humidityLine(f *Forecast) *rgbrender.ColorCharLine {
	humidity := strings.Split(fmt.Sprintf("Hu: %d%%", f.Humidity), "")
	line := &rgbrender.ColorCharLine{
		Chars: humidity,
		Clrs:  colors(color.White, humidity),
	}
	return line
}

func colors(clr color.Color, chars []string) []color.Color {
	c := []color.Color{}
	for range chars {
		c = append(c, clr)
	}

	return c
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
