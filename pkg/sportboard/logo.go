package sportboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *SportBoard) logoConfig(logoKey string) (*logo.Config, error) {
	for _, conf := range s.config.LogoConfigs {
		if conf.Abbrev == logoKey {
			return conf, nil
		}
	}

	s.log.Warn("no logo config defined, defaults will be used", zap.String("logo key", logoKey))
	return nil, fmt.Errorf("no logo config for %s", logoKey)
}

// RenderHomeLogo ...
func (s *SportBoard) RenderHomeLogo(ctx context.Context, canvas board.Canvas, abbreviation string) error {
	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}
	logoKey := fmt.Sprintf("%s_HOME_%dx%d", abbreviation, canvas.Bounds().Dx(), canvas.Bounds().Dy())

	i, ok := s.logoDrawCache[logoKey]
	if ok {
		s.log.Debug("drawing logo with drawCache", zap.String("logo key", logoKey))
		draw.Draw(canvas, canvas.Bounds(), i, image.Point{}, draw.Over)
		return nil
	}

	l, ok := s.logos[logoKey]
	if !ok {
		var err error
		logoConf, _ := s.logoConfig(logoKey)

		s.log.Debug("fetching logo",
			zap.String("logoKey", logoKey),
			zap.Int("X", canvas.Bounds().Dx()),
			zap.Int("Y", canvas.Bounds().Dy()),
		)
		l, err = s.api.GetLogo(ctx, logoKey, logoConf, canvas.Bounds())
		if err != nil {
			s.log.Error("failed to get logo", zap.Error(err))
		} else {
			s.logos[logoKey] = l
		}
	} else {
		s.log.Debug("using logo cache", zap.String("logo key", logoKey))
	}

	textWidth := s.textAreaWidth(canvas.Bounds())
	logoWidth := (canvas.Bounds().Dx() - textWidth) / 2

	var renderErr error
	if l != nil {
		var renderedLogo image.Image
		renderedLogo, renderErr = l.RenderLeftAligned(canvas.Bounds(), logoWidth)
		if renderErr != nil {
			s.log.Error("failed to render home logo", zap.Error(renderErr))
		} else {
			s.logoDrawCache[logoKey] = renderedLogo
			draw.Draw(canvas, canvas.Bounds(), renderedLogo, image.Point{}, draw.Over)

			return nil
		}
	}

	endX := ((canvas.Bounds().Dx() - textWidth) / 2)
	writeBounds := image.Rect(0, 0, endX, canvas.Bounds().Dy())
	writer, err := missingLogoWriter(writeBounds)
	if err != nil {
		return err
	}
	return writer.Write(
		canvas,
		writeBounds,
		[]string{
			abbreviation,
		},
		color.White,
	)
}

// RenderAwayLogo ...
func (s *SportBoard) RenderAwayLogo(ctx context.Context, canvas board.Canvas, abbreviation string) error {
	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}
	logoKey := fmt.Sprintf("%s_AWAY_%dx%d", abbreviation, canvas.Bounds().Dx(), canvas.Bounds().Dy())

	i, ok := s.logoDrawCache[logoKey]
	if ok {
		s.log.Debug("drawing logo with drawCache", zap.String("logo key", logoKey))
		draw.Draw(canvas, canvas.Bounds(), i, image.Point{}, draw.Over)
		return nil
	}

	var l *logo.Logo

	l, ok = s.logos[logoKey]
	if !ok {
		var err error
		logoConf, _ := s.logoConfig(logoKey)

		s.log.Debug("fetching logo",
			zap.String("abbreviation", abbreviation),
			zap.Int("X", canvas.Bounds().Dx()),
			zap.Int("Y", canvas.Bounds().Dy()),
		)
		l, err = s.api.GetLogo(ctx, logoKey, logoConf, canvas.Bounds())
		if err != nil {
			s.log.Error("failed to get away logo", zap.Error(err))
		} else {
			s.logos[logoKey] = l
		}
	}

	textWidth := s.textAreaWidth(canvas.Bounds())
	logoWidth := (canvas.Bounds().Dx() - textWidth) / 2

	var renderErr error
	if l != nil {
		var renderedLogo image.Image
		renderedLogo, renderErr = l.RenderRightAligned(canvas.Bounds(), logoWidth+textWidth)
		if renderErr != nil {
			s.log.Error("failed to render away logo", zap.Error(renderErr))
		} else {
			s.logoDrawCache[logoKey] = renderedLogo
			draw.Draw(canvas, canvas.Bounds(), renderedLogo, image.Point{}, draw.Over)

			return nil
		}
	}

	startX := ((canvas.Bounds().Dx() - textWidth) / 2) + textWidth
	writeBounds := image.Rect(startX, 0, canvas.Bounds().Dx(), canvas.Bounds().Dy())
	writer, err := missingLogoWriter(writeBounds)
	if err != nil {
		return err
	}
	return writer.WriteRight(
		canvas,
		writeBounds,
		[]string{
			abbreviation,
		},
		color.White,
	)
}

func missingLogoWriter(bounds image.Rectangle) (*rgbrender.TextWriter, error) {
	fnt, err := rgbrender.GetFont("score.ttf")
	if err != nil {
		return nil, err
	}

	size := 0.25 * float64(bounds.Dx())

	writer, err := rgbrender.NewTextWriter(fnt, size), nil
	if err != nil {
		return nil, err
	}

	return writer, nil
}
