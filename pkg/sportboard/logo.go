package sportboard

import (
	"context"
	"fmt"
	"image"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

const scrollLogoBufferRatio = float64(0.05)

func (s *SportBoard) logoConfig(logoKey string, bounds image.Rectangle) *logo.Config {
	for _, conf := range s.config.LogoConfigs {
		if conf.Abbrev == logoKey {
			return conf
		}
	}

	s.log.Warn("no logo config defined, defaults will be used", zap.String("logo key", logoKey))

	zoom := float64(1)

	if bounds.Dx() == bounds.Dy() {
		zoom = 0.8
	}

	return &logo.Config{
		Abbrev: logoKey,
		XSize:  bounds.Dx(),
		YSize:  bounds.Dy(),
		Pt: &logo.Pt{
			X:    0,
			Y:    0,
			Zoom: zoom,
		},
	}
}

func (s *SportBoard) getLogoDrawCache(logoKey string) (image.Image, error) {
	s.drawLock.RLock()
	defer s.drawLock.RUnlock()
	l, ok := s.logoDrawCache[logoKey]
	if ok {
		return l, nil
	}

	return nil, fmt.Errorf("no cache for %s", logoKey)
}

func (s *SportBoard) setLogoDrawCache(logoKey string, img image.Image) {
	s.drawLock.Lock()
	defer s.drawLock.Unlock()

	s.logoDrawCache[logoKey] = img
}

func (s *SportBoard) setLogoCache(logoKey string, l *logo.Logo) {
	s.logoLock.Lock()
	defer s.logoLock.Unlock()

	s.logos[logoKey] = l
}

func (s *SportBoard) getLogoCache(logoKey string) (*logo.Logo, error) {
	s.logoLock.RLock()
	defer s.logoLock.RUnlock()

	l, ok := s.logos[logoKey]
	if ok {
		return l, nil
	}

	return nil, fmt.Errorf("no cache for %s", logoKey)
}

// RenderLeftLogo ...
func (s *SportBoard) RenderLeftLogo(ctx context.Context, canvasBounds image.Rectangle, abbreviation string) (image.Image, error) {
	select {
	case <-ctx.Done():
		return nil, context.Canceled
	default:
	}
	bounds := rgbrender.ZeroedBounds(canvasBounds)
	logoKey := fmt.Sprintf("%s_HOME_%dx%d", abbreviation, bounds.Dx(), bounds.Dy())

	i, err := s.getLogoDrawCache(logoKey)
	if err == nil && i != nil {
		s.log.Debug("drawing logo with drawCache", zap.String("logo key", logoKey))
		return i, nil
	}

	l, err := s.getLogoCache(logoKey)
	if err != nil {
		var err error
		logoConf := s.logoConfig(logoKey, bounds)

		s.log.Debug("fetching logo",
			zap.String("logoKey", logoKey),
			zap.Int("X", bounds.Dx()),
			zap.Int("Y", bounds.Dy()),
		)
		l, err = s.api.GetLogo(ctx, logoKey, logoConf, bounds)
		if err != nil {
			s.log.Error("failed to get logo", zap.Error(err))
		} else {
			s.setLogoCache(logoKey, l)
		}
	} else {
		s.log.Debug("using logo cache", zap.String("logo key", logoKey))
	}

	textWidth := s.textAreaWidth(bounds)
	logoEndX := (bounds.Dx() - textWidth) / 2

	setCache := true

	if s.config.ScrollMode.Load() {
		if bounds.Dx() >= 64 && bounds.Dy() <= 64 {
			if bounds.Dx() < 64 {
				logoEndX -= 3
			} else {
				logoEndX -= 6
			}
		} else {
			logoEndX -= int(float64(bounds.Dx()) * scrollLogoBufferRatio)
		}
		//logoEndX -= int(float64(bounds.Dx()) * scrollLogoBufferRatio)
	}

	if s.config.ShowRecord.Load() || s.config.GamblingSpread.Load() {
		w, err := s.getTeamInfoWidth(s.api.League(), abbreviation)
		if err != nil {
			w = defaultTeamInfoArea
			setCache = false
			s.log.Error("failed to get team info width",
				zap.Error(err),
			)
		}
		logoEndX -= w
	}

	var renderErr error
	if l != nil {
		var renderedLogo image.Image
		renderedLogo, renderErr = l.RenderRightAlignedWithEnd(ctx, bounds, logoEndX)
		if renderErr != nil {
			s.log.Error("failed to render home logo", zap.Error(renderErr))
		} else {
			if setCache {
				s.setLogoDrawCache(logoKey, renderedLogo)
			}
			return renderedLogo, nil
		}
	}

	return nil, fmt.Errorf("no logo %s", logoKey)
}

// RenderRightLogo ...
func (s *SportBoard) RenderRightLogo(ctx context.Context, canvasBounds image.Rectangle, abbreviation string) (image.Image, error) {
	select {
	case <-ctx.Done():
		return nil, context.Canceled
	default:
	}
	bounds := rgbrender.ZeroedBounds(canvasBounds)
	logoKey := fmt.Sprintf("%s_AWAY_%dx%d", abbreviation, bounds.Dx(), bounds.Dy())

	i, err := s.getLogoDrawCache(logoKey)
	if err == nil && i != nil {
		s.log.Debug("drawing logo with drawCache", zap.String("logo key", logoKey))
		return i, nil
	}

	l, err := s.getLogoCache(logoKey)
	if err != nil {
		var err error
		logoConf := s.logoConfig(logoKey, bounds)

		s.log.Debug("fetching logo",
			zap.String("abbreviation", abbreviation),
			zap.Int("X", bounds.Dx()),
			zap.Int("Y", bounds.Dy()),
		)
		l, err = s.api.GetLogo(ctx, logoKey, logoConf, bounds)
		if err != nil {
			s.log.Error("failed to get away logo", zap.Error(err))
		} else {
			s.setLogoCache(logoKey, l)
		}
	}

	textWidth := s.textAreaWidth(bounds)
	logoWidth := (bounds.Dx() - textWidth) / 2
	recordAdder := 0

	setCache := true

	if s.config.ScrollMode.Load() {
		logoWidth += int(float64(bounds.Dx()) * scrollLogoBufferRatio)
	}

	if s.config.ShowRecord.Load() || s.config.GamblingSpread.Load() {
		recordAdder, err = s.getTeamInfoWidth(s.api.League(), abbreviation)
		if err != nil {
			s.log.Error("failed to get team info width",
				zap.Error(err),
			)
			recordAdder = defaultTeamInfoArea
			setCache = false
		}
	}

	startX := textWidth + logoWidth + recordAdder

	var renderErr error
	if l != nil {
		var renderedLogo image.Image

		renderedLogo, renderErr = l.RenderLeftAlignedWithStart(ctx, bounds, startX)
		if renderErr != nil {
			s.log.Error("failed to render away logo", zap.Error(renderErr))
		} else {
			if setCache {
				s.setLogoDrawCache(logoKey, renderedLogo)
			}
			return renderedLogo, nil
		}
	}

	return nil, fmt.Errorf("no logo %s", logoKey)
}
