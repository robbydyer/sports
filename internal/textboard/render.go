package textboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/logo"
	"github.com/robbydyer/sports/internal/rgbrender"
)

const logoCacheDir = "/tmp/sportsmatrix_logos/newslogos"

func (s *TextBoard) renderLogo(ctx context.Context, canvas board.Canvas) error {
	s.Lock()
	defer s.Unlock()

	zeroed := rgbrender.ZeroedBounds(canvas.Bounds())

	if s.config.halfSizeLogo {
		var err error
		zeroed, err = rgbrender.AlignPosition(rgbrender.CenterCenter, zeroed, zeroed.Dx()/2, zeroed.Dy()/2)
		if err != nil {
			return err
		}
	}

	key := fmt.Sprintf("%s_%dx%d", strings.ReplaceAll(s.api.HTTPPathPrefix(), "/", ""), zeroed.Dx(), zeroed.Dy())
	l, ok := s.logos[key]
	if !ok {
		g := func(ctx context.Context) (image.Image, error) {
			return s.api.GetLogo(ctx)
		}
		l = logo.New(key, g, logoCacheDir, zeroed, &logo.Config{
			Abbrev: "news",
			XSize:  zeroed.Dx(),
			YSize:  zeroed.Dy(),
			Pt: &logo.Pt{
				X:    0,
				Y:    0,
				Zoom: 1,
			},
		})
	}

	i, err := l.GetThumbnail(ctx, zeroed)
	if err != nil {
		return err
	}

	draw.Draw(canvas, zeroed, i, image.Point{}, draw.Over)

	return nil
}

func (s *TextBoard) doRender(canvas board.Canvas, text string) error {
	zeroed := rgbrender.ZeroedBounds(canvas.Bounds())
	lengths, err := s.writer.MeasureStrings(canvas, []string{text})
	if err != nil {
		return err
	}
	if len(lengths) < 1 {
		return fmt.Errorf("failed to measure text")
	}
	bounds := image.Rect(zeroed.Min.X, zeroed.Min.Y, zeroed.Min.X+lengths[0], zeroed.Max.Y)

	s.log.Debug("writing headline",
		zap.String("text", text),
		zap.Int("pix length", lengths[0]),
		zap.Int("X", bounds.Min.X),
		zap.Int("Y", bounds.Min.Y),
		zap.Int("X", bounds.Max.X),
		zap.Int("Y", bounds.Max.Y),
		zap.Int("canvas X", zeroed.Min.X),
		zap.Int("canvas Y", zeroed.Min.Y),
		zap.Int("canvas X", zeroed.Max.X),
		zap.Int("canvas Y", zeroed.Max.Y),
	)

	canvas.SetWidth(bounds.Dx())

	_ = s.writer.WriteAligned(
		rgbrender.CenterCenter,
		canvas,
		bounds,
		[]string{text},
		color.White,
	)

	return nil
}
