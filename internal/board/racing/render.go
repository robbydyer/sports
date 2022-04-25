package racingboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	cnvs "github.com/robbydyer/sports/internal/canvas"
	"github.com/robbydyer/sports/internal/logo"
	"github.com/robbydyer/sports/internal/rgbrender"
)

// ScrollRender ...
func (s *RacingBoard) ScrollRender(ctx context.Context, canvas board.Canvas, padding int) (board.Canvas, error) {
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
func (s *RacingBoard) Render(ctx context.Context, canvas board.Canvas) error {
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
func (s *RacingBoard) render(ctx context.Context, canvas board.Canvas) (board.Canvas, error) {
	s.boardCtx, s.boardCancel = context.WithCancel(ctx)

	s.log.Debug("render racing board",
		zap.String("league", s.api.LeagueShortName()),
	)
	if s.leagueLogo == nil {
		var err error
		s.leagueLogo, err = s.api.GetLogo(ctx, canvas.Bounds())
		if err != nil {
			return nil, err
		}
	}

	if len(s.events) < 1 {
		var err error
		s.events, err = s.api.GetScheduledEvents(ctx)
		if err != nil {
			return nil, err
		}
	}

	scheduleWriter, err := s.getScheduleWriter(rgbrender.ZeroedBounds(canvas.Bounds()))
	if err != nil {
		return nil, err
	}

	var scrollCanvas *cnvs.ScrollCanvas
	if canvas.Scrollable() && s.config.ScrollMode.Load() {
		base, ok := canvas.(*cnvs.ScrollCanvas)
		if !ok {
			return nil, fmt.Errorf("invalid scroll canvas")
		}

		var err error
		scrollCanvas, err = cnvs.NewScrollCanvas(base.Matrix, s.log,
			cnvs.WithMergePadding(s.config.TightScrollPadding),
		)
		if err != nil {
			return nil, err
		}
		scrollCanvas.SetScrollSpeed(s.config.scrollDelay)
		scrollCanvas.SetScrollDirection(cnvs.RightToLeft)
		go scrollCanvas.MatchScroll(ctx, base)
	}

	s.log.Debug("racing events",
		zap.String("league", s.api.LeagueShortName()),
		zap.Int("number", len(s.events)),
	)

EVENTS:
	for _, event := range s.events {
		select {
		case <-s.boardCtx.Done():
			return nil, context.Canceled
		default:
		}
		if err := s.renderEvent(s.boardCtx, canvas, event, s.leagueLogo, scheduleWriter); err != nil {
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
		return scrollCanvas, nil
	}

	return nil, nil
}

func (s *RacingBoard) renderEvent(ctx context.Context, canvas board.Canvas, event *Event, leagueLogo *logo.Logo, scheduleWriter *rgbrender.TextWriter) error {
	canvasBounds := rgbrender.ZeroedBounds(canvas.Bounds())

	logoImg, err := leagueLogo.RenderRightAlignedWithEnd(ctx, canvasBounds, (canvasBounds.Max.X-canvasBounds.Min.X)/2)
	if err != nil {
		return err
	}

	pt := image.Pt(logoImg.Bounds().Min.X, logoImg.Bounds().Min.Y)
	draw.Draw(canvas, logoImg.Bounds(), logoImg, pt, draw.Over)

	gradient := rgbrender.GradientXRectangle(
		canvasBounds,
		0.1,
		color.Black,
		s.log,
	)
	pt = image.Pt(gradient.Bounds().Min.X, gradient.Bounds().Min.Y)
	draw.Draw(canvas, gradient.Bounds(), gradient, pt, draw.Over)

	event.Date = event.Date.Local()

	tz, _ := event.Date.Zone()
	txt := []string{
		event.Name,
		event.Date.Format("01/02/2006"),
		fmt.Sprintf("%s %s", event.Date.Format("3:04PM"), tz),
	}

	lengths, err := scheduleWriter.MeasureStrings(canvas, txt)
	if err != nil {
		return err
	}
	max := canvasBounds.Dx() / 2

	for _, length := range lengths {
		if length > max {
			max = length
		}
	}

	s.log.Debug("max racing schedule text length",
		zap.Int("max", max),
		zap.Int("half bounds", canvasBounds.Dy()/2),
	)

	scheduleBounds := image.Rect(
		canvasBounds.Max.X/2,
		canvasBounds.Min.Y,
		(canvasBounds.Max.X/2)+max,
		canvasBounds.Max.Y,
	)

	if err := scheduleWriter.WriteAligned(
		rgbrender.LeftCenter,
		canvas,
		scheduleBounds,
		txt,
		color.White,
	); err != nil {
		return fmt.Errorf("failed to write schedule: %w", err)
	}

	return nil
}
