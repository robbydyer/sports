package stockboard

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/logo"
)

func (s *StockBoard) logoSource(symbol string) (logo.SourceGetter, error) {
	symbol = strings.ReplaceAll(symbol, "^", "")
	b, err := assets.ReadFile(filepath.Join("assets", "logos", fmt.Sprintf("%s.PNG", strings.ToUpper(symbol))))
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(b)

	return func(ctx context.Context) (image.Image, error) {
		return png.Decode(r)
	}, nil
}

func (s *StockBoard) drawLogo(ctx context.Context, canvas draw.Image, bounds image.Rectangle, symbol string) error {
	l, err := s.getOrCreateLogo(symbol, canvas, bounds)
	if err != nil {
		return err
	}

	s.log.Debug("draw stock logo",
		zap.Int("width", bounds.Dx()),
		zap.Int("height", bounds.Dy()),
		zap.Int("endX", bounds.Max.X-2),
	)

	i, err := l.RenderRightAlignedWithEnd(ctx, bounds, bounds.Max.X-4)
	if err != nil {
		return err
	}

	draw.Draw(canvas, i.Bounds(), i, image.Pt(i.Bounds().Min.X, i.Bounds().Min.Y), draw.Over)

	return nil
}

func (s *StockBoard) getOrCreateLogo(symbol string, canvas draw.Image, bounds image.Rectangle) (*logo.Logo, error) {
	s.logoLock.Lock()
	defer s.logoLock.Unlock()

	key := fmt.Sprintf("%s_%dx%d", strings.ToLower(symbol), bounds.Dx(), bounds.Dy())

	l, ok := s.logos[strings.ToLower(symbol)]
	if ok {
		return l, nil
	}

	src, err := s.logoSource(symbol)
	if err != nil {
		return nil, err
	}

	lo := logo.New(key, src, "/tmp/sportsmatrix_logos/stocks", canvas.Bounds(),
		&logo.Config{
			FitImage: true,
			Abbrev:   key,
			XSize:    bounds.Dx(),
			YSize:    bounds.Dy(),
			Pt: &logo.Pt{
				X:    0,
				Y:    0,
				Zoom: 1.0,
			},
		},
	)

	lo.SetLogger(s.log)

	s.logos[strings.ToLower(symbol)] = lo

	return lo, nil
}
