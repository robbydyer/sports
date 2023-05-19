package sportboard

import (
	"context"
	"fmt"
	"image"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/logo"
	"github.com/robbydyer/sports/internal/rgbrender"
)

func (s *SportBoard) logoConfig(logoKey string, bounds image.Rectangle) *logo.Config {
	for _, conf := range s.config.LogoConfigs {
		if conf.Abbrev == logoKey {
			return conf
		}
	}

	s.log.Debug("no logo config defined, defaults will be used", zap.String("logo key", logoKey))

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

func (s *SportBoard) clearDrawCache() {
	s.drawLock.Lock()
	defer s.drawLock.Unlock()
	s.logoDrawCache = make(map[string]image.Image)
}

func (s *SportBoard) getLogoDrawCache(logoKey string) (image.Image, error) {
	s.drawLock.RLock()
	defer s.drawLock.RUnlock()
	l, ok := s.logoDrawCache[logoKey]
	if ok {
		if l == nil {
			s.log.Warn("logo draw cache was nil",
				zap.String("league", s.api.League()),
				zap.String("key", logoKey),
			)
			delete(s.logoDrawCache, logoKey)
			return nil, fmt.Errorf("no cache for %s", logoKey)
		}
		return l, nil
	}

	return nil, fmt.Errorf("no cache for %s", logoKey)
}

func (s *SportBoard) setLogoDrawCache(logoKey string, img image.Image) {
	s.drawLock.Lock()
	defer s.drawLock.Unlock()

	s.log.Debug("set logo draw cache",
		zap.String("league", s.api.League()),
		zap.String("key", logoKey),
	)
	s.logoDrawCache[logoKey] = img
}

func (s *SportBoard) setLogoCache(logoKey string, l *logo.Logo) {
	s.logoLock.Lock()
	defer s.logoLock.Unlock()

	s.log.Debug("set logo cache",
		zap.String("league", s.api.League()),
		zap.String("key", logoKey),
	)
	s.logos[logoKey] = l
}

func (s *SportBoard) getLogoCache(logoKey string) (*logo.Logo, error) {
	s.logoLock.RLock()
	defer s.logoLock.RUnlock()

	l, ok := s.logos[logoKey]
	if ok {
		if l == nil {
			s.log.Warn("logo cache was nil",
				zap.String("league", s.api.League()),
				zap.String("key", logoKey),
			)
			delete(s.logos, logoKey)
			return nil, fmt.Errorf("no cache for %s", logoKey)
		}
		return l, nil
	}

	s.log.Debug("no logo cache",
		zap.String("league", s.api.League()),
		zap.String("key", logoKey),
	)
	return nil, fmt.Errorf("no cache for %s", logoKey)
}

// RenderLeftLogo ...
func (s *SportBoard) RenderLeftLogo(ctx context.Context, canvasBounds image.Rectangle, teamID string) (image.Image, error) {
	select {
	case <-ctx.Done():
		return nil, context.Canceled
	default:
	}
	bounds := rgbrender.ZeroedBounds(canvasBounds)
	logoKey := fmt.Sprintf("%s_HOME_%dx%d", teamID, bounds.Dx(), bounds.Dy())

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
			s.log.Error("failed to get left logo", zap.Error(err))
			return nil, fmt.Errorf("failed to get left logo: %w", err)
		}
		l.SetLogger(s.log)
		s.setLogoCache(logoKey, l)
	} else {
		s.log.Debug("using logo cache", zap.String("logo key", logoKey))
	}

	if l == nil {
		s.log.Error("logo was nil",
			zap.String("league", s.api.League()),
			zap.String("key", logoKey),
		)
		return nil, fmt.Errorf("logo %s was nil", logoKey)
	}

	textWidth := s.textAreaWidth(bounds)
	logoEndX := (bounds.Dx() - textWidth) / 2
	s.log.Debug("starting logoWidth",
		zap.Int("logoWidth", logoEndX),
	)

	setCache := true

	if s.config.ShowRecord.Load() {
		w, err := s.getTeamInfoWidth(s.api.League(), teamID)
		if err != nil {
			w = defaultTeamInfoArea
			setCache = false
			s.log.Error("failed to get team info width",
				zap.Error(err),
			)
		}
		logoEndX -= w
	}

	s.log.Debug("scroll mode added width",
		zap.String("side", "left"),
		zap.Int("textWidth", textWidth),
		zap.Int("logoEndX", logoEndX),
	)

	var renderErr error
	var renderedLogo image.Image
	renderedLogo, renderErr = l.RenderRightAlignedWithEnd(ctx, bounds, logoEndX)
	if renderErr != nil {
		s.log.Error("failed to render left logo", zap.Error(renderErr))
		return nil, fmt.Errorf("failed to render left logo: %w", renderErr)
	}
	if setCache {
		s.setLogoDrawCache(logoKey, renderedLogo)
	}
	return renderedLogo, nil
}

// RenderRightLogo ...
func (s *SportBoard) RenderRightLogo(ctx context.Context, canvasBounds image.Rectangle, teamID string) (image.Image, error) {
	select {
	case <-ctx.Done():
		return nil, context.Canceled
	default:
	}
	bounds := rgbrender.ZeroedBounds(canvasBounds)
	logoKey := fmt.Sprintf("%s_AWAY_%dx%d", teamID, bounds.Dx(), bounds.Dy())

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
			zap.String("abbreviation", teamID),
			zap.Int("X", bounds.Dx()),
			zap.Int("Y", bounds.Dy()),
		)
		l, err = s.api.GetLogo(ctx, logoKey, logoConf, bounds)
		if err != nil {
			s.log.Error("failed to get right logo", zap.Error(err))
			return nil, fmt.Errorf("failed to get right logo: %w", err)
		}
		l.SetLogger(s.log)
		s.setLogoCache(logoKey, l)
	}

	if l == nil {
		s.log.Error("logo was nil",
			zap.String("league", s.api.League()),
			zap.String("key", logoKey),
		)
		return nil, fmt.Errorf("logo %s was nil", logoKey)
	}

	textWidth := s.textAreaWidth(bounds)
	logoWidth := (bounds.Dx() - textWidth) / 2
	s.log.Debug("starting logoWidth",
		zap.Int("logoWidth", logoWidth),
	)
	recordAdder := 0

	setCache := true

	if s.config.ShowRecord.Load() {
		recordAdder, err = s.getTeamInfoWidth(s.api.League(), teamID)
		if err != nil {
			s.log.Error("failed to get team info width",
				zap.Error(err),
			)
			recordAdder = defaultTeamInfoArea
			setCache = false
		}
	}

	startX := textWidth + logoWidth + recordAdder

	s.log.Debug("scroll mode added width",
		zap.String("side", "right"),
		zap.Int("logoWidth", logoWidth),
		zap.Int("textWidth", textWidth),
		zap.Int("recordAdder", recordAdder),
		zap.Int("total", startX),
	)

	var renderErr error
	var renderedLogo image.Image

	renderedLogo, renderErr = l.RenderLeftAlignedWithStart(ctx, bounds, startX)
	if renderErr != nil {
		s.log.Error("failed to render right logo", zap.Error(renderErr))
		return nil, fmt.Errorf("failed to render right logo: %w", err)
	}
	if setCache {
		s.setLogoDrawCache(logoKey, renderedLogo)
	}
	return renderedLogo, nil
}
