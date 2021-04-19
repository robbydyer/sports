package logo

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/rgbrender"
)

// SourceGetter is a func type that retrieves a source logo image.Image
type SourceGetter func(ctx context.Context) (image.Image, error)

// Logo is used to manage logo rendering
type Logo struct {
	key              string
	sourceLogoGetter SourceGetter
	bounds           image.Rectangle
	targetDirectory  string
	config           *Config
	thumbnail        image.Image
	log              *zap.Logger
}

// Config ...
type Config struct {
	Abbrev string `json:"abbrev"`
	Pt     *Pt    `json:"pt"`
	XSize  int    `json:"xSize"`
	YSize  int    `json:"ySize"`
}

// Pt defines the x, y shift and zoom values for a logo
type Pt struct {
	X    int     `json:"xShift"`
	Y    int     `json:"yShift"`
	Zoom float64 `json:"zoom"`
}

// New ...
func New(key string, getter SourceGetter, targetDirectory string, matrixBounds image.Rectangle, conf *Config) *Logo {
	return &Logo{
		key:              key,
		targetDirectory:  targetDirectory,
		sourceLogoGetter: getter,
		config:           conf,
		bounds:           matrixBounds,
	}
}

// Key returns the key name of the logo
func (l *Logo) Key() string {
	return l.key
}

// SetLogger ...
func (l *Logo) SetLogger(logger *zap.Logger) {
	l.log = logger
}

func (l *Logo) ensureLogger() {
	if l.log == nil {
		l.log, _ = zap.NewDevelopment()
	}
}

// ThumbnailFilename returns the filname for the resized thumbnail to use
func (l *Logo) ThumbnailFilename(size image.Rectangle) string {
	return filepath.Join(l.targetDirectory, fmt.Sprintf("%s.png", l.key))
}

// GetThumbnail returns the resized image
func (l *Logo) GetThumbnail(ctx context.Context, size image.Rectangle) (image.Image, error) {
	if l.thumbnail != nil {
		return l.thumbnail, nil
	}

	thumbFile := l.ThumbnailFilename(size)

	if _, err := os.Stat(thumbFile); err != nil {
		if os.IsNotExist(err) {
			if _, err := os.Stat(l.targetDirectory); err != nil {
				if os.IsNotExist(err) {
					if err := os.MkdirAll(l.targetDirectory, 0755); err != nil {
						return nil, fmt.Errorf("failed to create logo cache dir: %w", err)
					}
				}
			}

			src, err := l.sourceLogoGetter(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get logo source for %s: %w", l.config.Abbrev, err)
			}

			if src == nil {
				return nil, fmt.Errorf("failed to get logo source for %s", l.config.Abbrev)
			}

			// Create the thumbnail
			l.thumbnail = rgbrender.ResizeImage(src, size, l.config.Pt.Zoom)

			go func() {
				l.ensureLogger()
				l.log.Info("saving thumbnail logo", zap.String("filename", thumbFile))
				if err := rgbrender.SavePng(l.thumbnail, thumbFile); err != nil {
					l.log.Error("failed to save logo PNG", zap.Error(err))
				}
			}()

			return l.thumbnail, nil
		}

		return nil, err
	}

	t, err := os.Open(thumbFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open logo %s: %w", thumbFile, err)
	}
	defer t.Close()

	l.thumbnail, err = png.Decode(t)
	if err != nil {
		return nil, fmt.Errorf("failed to decode logo %s: %w", thumbFile, err)
	}

	return l.thumbnail, nil
}

// RenderLeftAligned renders the logo on the left side of the matrix
func (l *Logo) RenderLeftAligned(ctx context.Context, bounds image.Rectangle, endX int) (image.Image, error) {
	thumb, err := l.GetThumbnail(ctx, l.bounds)
	if err != nil {
		return nil, err
	}

	startX := 0

	if thumb.Bounds().Dx() > endX {
		startX = endX - thumb.Bounds().Dx()
	}

	startX += l.config.Pt.X

	startY := 0 + l.config.Pt.Y
	newBounds := image.Rect(startX, startY, bounds.Dx()-1, bounds.Dy()-1)
	align, err := rgbrender.AlignPosition(rgbrender.LeftCenter, newBounds, thumb.Bounds().Dx(), thumb.Bounds().Dy())
	if err != nil {
		return nil, err
	}

	i := image.NewRGBA(newBounds)
	draw.Draw(i, align, thumb, image.Point{}, draw.Over)

	if l.log != nil {
		l.log.Debug("logo left alignment",
			zap.Int("end X", endX),
			zap.Int("size X", bounds.Dx()),
			zap.Int("size Y", bounds.Dy()),
			zap.Int("newBounds min X", newBounds.Min.X),
			zap.Int("newBounds min Y", newBounds.Min.Y),
			zap.Int("newBounds max X", newBounds.Max.X),
			zap.Int("newBounds max Y", newBounds.Max.Y),
			zap.Int("align min X", align.Min.X),
			zap.Int("align min Y", align.Min.Y),
			zap.Int("align max X", align.Max.X),
			zap.Int("align max Y", align.Max.Y),
			zap.Int("img min X", i.Bounds().Min.X),
			zap.Int("img min Y", i.Bounds().Min.Y),
			zap.Int("img max X", i.Bounds().Max.X),
			zap.Int("img max Y", i.Bounds().Max.Y),
		)
	}

	return i, nil
}

// RenderRightAligned renders the logo on the right side of the matrix
func (l *Logo) RenderRightAligned(ctx context.Context, bounds image.Rectangle, startX int) (image.Image, error) {
	thumb, err := l.GetThumbnail(ctx, l.bounds)
	if err != nil {
		return nil, err
	}

	startX = startX + l.config.Pt.X
	startY := 0 + l.config.Pt.Y

	newBounds := image.Rect(startX, startY, thumb.Bounds().Dx()+startX, thumb.Bounds().Dy()+startY)

	align, err := rgbrender.AlignPosition(rgbrender.RightCenter, newBounds, thumb.Bounds().Dx(), thumb.Bounds().Dy())
	if err != nil {
		return nil, err
	}

	i := image.NewRGBA(newBounds)
	draw.Draw(i, align, thumb, image.Point{}, draw.Over)

	return i, nil
}
