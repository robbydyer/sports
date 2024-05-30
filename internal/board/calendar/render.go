package calendarboard

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"math"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/rgbrender"
)

// Render ...
func (s *CalendarBoard) Render(ctx context.Context, canvas board.Canvas) error {
	err := s.render(ctx, canvas)
	if err != nil {
		return err
	}

	return nil
}

// Render ...
// nolint:contextcheck
func (s *CalendarBoard) render(ctx context.Context, canvas board.Canvas) error {
	s.boardCtx, s.boardCancel = context.WithCancel(ctx)

	events, err := s.api.DailyEvents(ctx, time.Now())
	if err != nil {
		return err
	}

	s.log.Debug("calendar events",
		zap.Int("number", len(events)),
	)

	if len(events) < 1 {
		return nil
	}

	scheduleWriter, err := s.getScheduleWriter(rgbrender.ZeroedBounds(canvas.Bounds()))
	if err != nil {
		return err
	}

	if s.logo == nil {
		var err error
		s.logo, err = s.api.CalendarIcon(ctx, canvas.Bounds())
		if err != nil {
			return err
		}
	}

EVENTS:
	for _, event := range events {
		select {
		case <-s.boardCtx.Done():
			return context.Canceled
		default:
		}
		img, err := s.renderEvent(s.boardCtx, canvas.Bounds(), event, scheduleWriter)
		if err != nil {
			s.log.Error("failed to render calendar event",
				zap.Error(err),
			)
			continue EVENTS
		}

		draw.Draw(canvas, img.Bounds(), img, image.Point{}, draw.Over)

		if err := canvas.Render(s.boardCtx); err != nil {
			s.log.Error("failed to render calendar board",
				zap.Error(err),
			)
			continue EVENTS
		}

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(s.config.boardDelay):
		}
	}

	return nil
}

func (s *CalendarBoard) renderEvent(ctx context.Context, bounds image.Rectangle, event *Event, writer *rgbrender.TextWriter) (draw.Image, error) {
	img := image.NewRGBA(bounds)
	canvasBounds := rgbrender.ZeroedBounds(bounds)

	logoHeight := int(writer.FontSize * 2.0)
	logoBounds := image.Rect(canvasBounds.Min.X, canvasBounds.Min.Y, canvasBounds.Min.X+logoHeight, canvasBounds.Min.Y+logoHeight)

	dateBounds := image.Rect(canvasBounds.Min.X+logoHeight+2, canvasBounds.Min.Y, canvasBounds.Max.X, canvasBounds.Min.Y+logoHeight)

	titleBounds := image.Rect(canvasBounds.Min.X, canvasBounds.Min.Y+logoHeight+2, canvasBounds.Max.X, canvasBounds.Max.Y)

	logoImg, err := s.logo.RenderLeftAlignedWithStart(ctx, logoBounds, 0)
	if err != nil {
		return nil, err
	}

	pt := image.Pt(logoImg.Bounds().Min.X, logoImg.Bounds().Min.Y)
	draw.Draw(img, logoImg.Bounds(), logoImg, pt, draw.Over)

	if err := writer.WriteAligned(
		rgbrender.CenterCenter,
		img,
		dateBounds,
		[]string{
			event.Time.Format("Mon Jan 2"),
			event.Time.Format("03:04PM"),
		},
		color.White,
	); err != nil {
		return nil, err
	}

	lines, err := writer.BreakText(img, titleBounds.Max.X-titleBounds.Min.X, event.Title)
	if err != nil {
		return nil, err
	}

	maxLines := int(math.Ceil(float64(titleBounds.Dy()) / writer.FontSize))

	if len(lines) > maxLines {
		lines = lines[0:maxLines]
	}

	s.log.Debug("calendar event",
		zap.Strings("titles", lines),
		zap.Int("max lines", maxLines),
		zap.Int("X min", titleBounds.Min.X),
		zap.Int("Y min", titleBounds.Min.Y),
		zap.Int("X max", titleBounds.Max.X),
		zap.Int("Y max", titleBounds.Max.Y),
	)

	if err := writer.WriteAligned(
		rgbrender.LeftBottom,
		img,
		titleBounds,
		lines,
		color.White,
	); err != nil {
		return nil, err
	}

	return img, nil
}
