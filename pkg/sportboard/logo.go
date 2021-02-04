package sportboard

import (
	"context"
	"fmt"
	"image"
	"image/draw"

	"github.com/robbydyer/sports/pkg/logo"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
)

func (s *SportBoard) logoConfig(logoKey string) (*logo.Config, error) {
	for _, conf := range s.config.LogoConfigs {
		if conf.Abbrev == logoKey {
			return conf, nil
		}
	}

	s.log.Warnf("no logo config defined in config file for %s. Defaults will be used", logoKey)
	return nil, fmt.Errorf("no logo config for %s", logoKey)
}

// RenderHomeLogo ...
func (s *SportBoard) RenderHomeLogo(ctx context.Context, canvas *rgb.Canvas, abbreviation string) error {
	logoKey := fmt.Sprintf("%s_HOME", abbreviation)

	i, ok := s.logoDrawCache[logoKey]
	if ok {
		s.log.Debugf("drawing %s logo with drawCache", logoKey)
		draw.Draw(canvas, canvas.Bounds(), i, image.Point{}, draw.Over)
		return nil
	}

	l, ok := s.logos[logoKey]
	if !ok {
		var err error
		logoConf, _ := s.logoConfig(logoKey)

		s.log.Debugf("fetching logo for %s %dx%d", abbreviation, s.matrixBounds.Dx(), s.matrixBounds.Dy())
		l, err = s.api.GetLogo(ctx, logoKey, logoConf, s.matrixBounds)
		if err != nil {
			return err
		}

		s.logos[logoKey] = l
	} else {
		s.log.Debugf("using logo cache for %s", logoKey)
	}

	textWdith := s.textAreaWidth()
	logoWidth := (s.matrixBounds.Dx() - textWdith) / 2
	renderedLogo, err := l.RenderLeftAligned(canvas, logoWidth)
	if err != nil {
		return err
	}

	s.logoDrawCache[logoKey] = renderedLogo

	draw.Draw(canvas, canvas.Bounds(), renderedLogo, image.Point{}, draw.Over)

	return nil
}

// RenderAwayLogo ...
func (s *SportBoard) RenderAwayLogo(ctx context.Context, canvas *rgb.Canvas, abbreviation string) error {
	logoKey := fmt.Sprintf("%s_AWAY", abbreviation)

	i, ok := s.logoDrawCache[logoKey]
	if ok {
		draw.Draw(canvas, canvas.Bounds(), i, image.Point{}, draw.Over)
		return nil
	}

	l, ok := s.logos[logoKey]
	if !ok {
		var err error
		logoConf, _ := s.logoConfig(logoKey)

		s.log.Debugf("fetching logo for %s %dx%d", abbreviation, s.matrixBounds.Dx(), s.matrixBounds.Dy())
		l, err = s.api.GetLogo(ctx, logoKey, logoConf, s.matrixBounds)
		if err != nil {
			return err
		}

		s.logos[logoKey] = l
	}

	textWdith := s.textAreaWidth()
	logoWidth := (s.matrixBounds.Dx() - textWdith) / 2

	renderedLogo, err := l.RenderRightAligned(canvas, logoWidth+textWdith)
	if err != nil {
		return err
	}

	s.logoDrawCache[logoKey] = renderedLogo

	draw.Draw(canvas, canvas.Bounds(), renderedLogo, image.Point{}, draw.Over)

	return nil
}
