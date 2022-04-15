package calendarboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/rgbmatrix-rpi"
	"github.com/robbydyer/sports/internal/rgbrender"
)

// ScrollRender ...
func (s *CalendarBoard) ScrollRender(ctx context.Context, canvas board.Canvas, padding int) (board.Canvas, error) {
	origScrollMode := s.config.ScrollMode.Load()
	origPad := s.config.TightScrollPadding
	defer func() {
		s.config.ScrollMode.Store(origScrollMode)
		s.config.TightScrollPadding = origPad
	}()

	s.config.ScrollMode.Store(true)
	s.config.TightScrollPadding = padding

	return s.render(ctx, canvas)
}

// Render ...
func (s *CalendarBoard) Render(ctx context.Context, canvas board.Canvas) error {
	c, err := s.render(ctx, canvas)
	if err != nil {
		return err
	}
	if c != nil {
		return c.Render(ctx)
	}

	return nil
}

// Render ...
func (s *CalendarBoard) render(ctx context.Context, canvas board.Canvas) (board.Canvas, error) {
	s.boardCtx, s.boardCancel = context.WithCancel(ctx)

	events, err := s.api.DailyEvents(ctx, time.Now())
	if err != nil {
		return nil, err
	}

	scheduleWriter, err := s.getScheduleWriter(rgbrender.ZeroedBounds(canvas.Bounds()))
	if err != nil {
		return nil, err
	}

	if s.logo == nil {
		var err error
		s.logo, err = s.api.CalendarIcon(ctx, canvas.Bounds())
		if err != nil {
			return nil, err
		}
	}

	var scrollCanvas *rgbmatrix.ScrollCanvas
	if canvas.Scrollable() && s.config.ScrollMode.Load() {
		base, ok := canvas.(*rgbmatrix.ScrollCanvas)
		if !ok {
			return nil, fmt.Errorf("invalid scroll canvas")
		}

		var err error
		scrollCanvas, err = rgbmatrix.NewScrollCanvas(base.Matrix, s.log)
		if err != nil {
			return nil, err
		}
		scrollCanvas.SetScrollSpeed(s.config.scrollDelay)
		scrollCanvas.SetScrollDirection(rgbmatrix.RightToLeft)
	}

	s.log.Debug("calendar events",
		zap.Int("number", len(s.events)),
	)

EVENTS:
	for _, event := range events {
		select {
		case <-s.boardCtx.Done():
			return nil, context.Canceled
		default:
		}
		if err := s.renderEvent(s.boardCtx, canvas, event, scheduleWriter); err != nil {
			s.log.Error("failed to render racing event",
				zap.Error(err),
			)
			continue EVENTS
		}

		if scrollCanvas != nil && s.config.ScrollMode.Load() {
			scrollCanvas.AddCanvas(canvas)
			draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Over)
			continue EVENTS
		}

		if err := canvas.Render(s.boardCtx); err != nil {
			s.log.Error("failed to render racing board",
				zap.Error(err),
			)
			continue EVENTS
		}

		if !s.config.ScrollMode.Load() {
			select {
			case <-ctx.Done():
				return nil, context.Canceled
			case <-time.After(s.config.boardDelay):
			}
		}
	}

	if canvas.Scrollable() && scrollCanvas != nil {
		scrollCanvas.Merge(s.config.TightScrollPadding)
		return scrollCanvas, nil
	}

	return nil, nil
}

func (s *CalendarBoard) renderEvent(ctx context.Context, canvas board.Canvas, event *Event, writer *rgbrender.TextWriter) error {
	canvasBounds := rgbrender.ZeroedBounds(canvas.Bounds())

	logoHeight := int(writer.FontSize * 2.0)
	logoBounds := image.Rect(canvasBounds.Min.X, canvasBounds.Min.Y, canvasBounds.Min.X+logoHeight, canvasBounds.Min.Y+logoHeight)

	dateBounds := image.Rect(canvasBounds.Min.X+logoHeight+2, canvasBounds.Min.Y, canvasBounds.Max.X, canvasBounds.Min.Y+logoHeight)

	titleBounds := image.Rect(canvasBounds.Min.X, canvasBounds.Min.Y+logoHeight+2, canvasBounds.Max.X, canvasBounds.Max.Y)

	logoImg, err := s.logo.RenderLeftAlignedWithStart(ctx, logoBounds, 0)
	if err != nil {
		return err
	}

	pt := image.Pt(logoImg.Bounds().Min.X, logoImg.Bounds().Min.Y)
	draw.Draw(canvas, logoImg.Bounds(), logoImg, pt, draw.Over)

	if err := writer.WriteAligned(
		rgbrender.CenterCenter,
		canvas,
		dateBounds,
		[]string{
			event.Time.Format("Mon Jan 2"),
		},
		color.White,
	); err != nil {
		return err
	}

	lines, err := writer.BreakText(canvas, titleBounds.Max.X-titleBounds.Min.X, event.Title)
	if err != nil {
		return err
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
		canvas,
		titleBounds,
		lines,
		color.White,
	); err != nil {
		return err
	}

	return nil
}
