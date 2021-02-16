package sportboard

import (
	"context"
	"fmt"
	"image"
	"image/draw"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/logo"
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
	logoKey := fmt.Sprintf("%s_HOME", abbreviation)

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
			zap.String("abbreviation", abbreviation),
			zap.Int("X", s.matrixBounds.Dx()),
			zap.Int("Y", s.matrixBounds.Dy()),
		)
		l, err = s.api.GetLogo(ctx, logoKey, logoConf, s.matrixBounds)
		if err != nil {
			return err
		}

		s.logos[logoKey] = l
	} else {
		s.log.Debug("using logo cache", zap.String("logo key", logoKey))
	}

	textWdith := s.textAreaWidth()
	logoWidth := (s.matrixBounds.Dx() - textWdith) / 2
	renderedLogo, err := l.RenderLeftAligned(canvas.Bounds(), logoWidth)
	if err != nil {
		return err
	}

	s.logoDrawCache[logoKey] = renderedLogo

	draw.Draw(canvas, canvas.Bounds(), renderedLogo, image.Point{}, draw.Over)

	return nil
}

// RenderAwayLogo ...
func (s *SportBoard) RenderAwayLogo(ctx context.Context, canvas board.Canvas, abbreviation string) error {
	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}
	logoKey := fmt.Sprintf("%s_AWAY", abbreviation)

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
			zap.String("abbreviation", abbreviation),
			zap.Int("X", s.matrixBounds.Dx()),
			zap.Int("Y", s.matrixBounds.Dy()),
		)
		l, err = s.api.GetLogo(ctx, logoKey, logoConf, s.matrixBounds)
		if err != nil {
			return err
		}

		s.logos[logoKey] = l
	}

	textWdith := s.textAreaWidth()
	logoWidth := (s.matrixBounds.Dx() - textWdith) / 2

	renderedLogo, err := l.RenderRightAligned(canvas.Bounds(), logoWidth+textWdith)
	if err != nil {
		return err
	}

	s.logoDrawCache[logoKey] = renderedLogo

	draw.Draw(canvas, canvas.Bounds(), renderedLogo, image.Point{}, draw.Over)

	return nil
}
