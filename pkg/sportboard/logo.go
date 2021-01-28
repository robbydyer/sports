package sportboard

import (
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

	return nil, fmt.Errorf("no logo config for %s", logoKey)
}

func (s *SportBoard) RenderHomeLogo(canvas *rgb.Canvas, abbreviation string) error {
	logoKey := fmt.Sprintf("%s_HOME", abbreviation)

	i, ok := s.logoDrawCache[logoKey]
	if ok {
		draw.Draw(canvas, canvas.Bounds(), i, image.ZP, draw.Over)
		return nil
	}

	l, ok := s.logos[logoKey]
	if !ok {
		var err error
		logoConf, _ := s.logoConfig(logoKey)

		l, err = s.api.GetLogo(logoKey, logoConf, s.matrixBounds)
		if err != nil {
			return err
		}

		s.logos[logoKey] = l
	}

	textWdith := s.textAreaWidth()
	logoWidth := (s.matrixBounds.Dx() - textWdith) / 2
	renderedLogo, err := l.RenderLeftAligned(canvas, logoWidth)
	if err != nil {
		return err
	}

	s.logoDrawCache[logoKey] = renderedLogo

	draw.Draw(canvas, canvas.Bounds(), renderedLogo, image.ZP, draw.Over)

	return nil
}

func (s *SportBoard) RenderAwayLogo(canvas *rgb.Canvas, abbreviation string) error {
	logoKey := fmt.Sprintf("%s_AWAY", abbreviation)

	i, ok := s.logoDrawCache[logoKey]
	if ok {
		draw.Draw(canvas, canvas.Bounds(), i, image.ZP, draw.Over)
		return nil
	}

	l, ok := s.logos[logoKey]
	if !ok {
		var err error
		logoConf, _ := s.logoConfig(logoKey)

		l, err = s.api.GetLogo(logoKey, logoConf, s.matrixBounds)
		if err != nil {
			return err
		}

		s.logos[logoKey] = l
	}

	textWdith := s.textAreaWidth()
	logoWidth := (s.matrixBounds.Dx() - textWdith) / 2

	renderedLogo, err := l.RenderRightAligned(canvas, logoWidth)
	if err != nil {
		return err
	}

	s.logoDrawCache[logoKey] = renderedLogo

	draw.Draw(canvas, canvas.Bounds(), renderedLogo, image.ZP, draw.Over)

	return nil
}
