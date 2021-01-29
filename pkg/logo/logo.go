package logo

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"

	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

type Logo struct {
	key             string
	sourceLogo      image.Image
	bounds          image.Rectangle
	targetDirectory string
	config          *Config
	thumbnail       image.Image
}

type Config struct {
	Abbrev string `json:"abbrev"`
	Pt     *Pt    `json:"pt"`
}

type Pt struct {
	X    int     `json:"xShift"`
	Y    int     `json:"yShift"`
	Zoom float64 `json:"zoom"`
}

type OptionFunc func(*Logo) error

func New(key string, sourceLogo image.Image, targetDirectory string, matrixBounds image.Rectangle, conf *Config) *Logo {
	return &Logo{
		key:             key,
		targetDirectory: targetDirectory,
		sourceLogo:      sourceLogo,
		config:          conf,
		bounds:          matrixBounds,
	}
}

func (l *Logo) ThumbnailFilename(size image.Rectangle) string {
	return fmt.Sprintf("%s/%s_%dx%d_%f.png", l.targetDirectory, l.key, size.Dx(), size.Dy(), l.config.Pt.Zoom)
}

func (l *Logo) GetThumbnail(size image.Rectangle) (image.Image, error) {
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
			// Create the thumbnail
			fmt.Printf("Saving thumbnail logo %s\n", thumbFile)
			l.thumbnail = rgbrender.ResizeImage(l.sourceLogo, size, l.config.Pt.Zoom)

			if err := rgbrender.SavePng(l.thumbnail, thumbFile); err != nil {
				return nil, fmt.Errorf("failed to save logo %s: %w", thumbFile, err)
			}

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

func (l *Logo) RenderLeftAligned(canvas *rgb.Canvas, width int) (image.Image, error) {
	thumb, err := l.GetThumbnail(l.bounds)
	if err != nil {
		return nil, err
	}

	startX := width - thumb.Bounds().Dx() + l.config.Pt.X
	startY := 0 + l.config.Pt.Y

	bounds := image.Rect(startX, startY, canvas.Bounds().Dx()-1, canvas.Bounds().Dy()-1)

	i := image.NewRGBA(bounds)
	draw.Draw(i, bounds, thumb, image.ZP, draw.Over)

	return i, nil
}

func (l *Logo) RenderRightAligned(canvas *rgb.Canvas, width int) (image.Image, error) {
	thumb, err := l.GetThumbnail(l.bounds)
	if err != nil {
		return nil, err
	}

	startX := width + l.config.Pt.X
	startY := 0 + l.config.Pt.Y

	bounds := image.Rect(startX, startY, canvas.Bounds().Dx()-1, canvas.Bounds().Dy()-1)

	i := image.NewRGBA(bounds)
	draw.Draw(i, bounds, thumb, image.ZP, draw.Over)

	return i, nil
}

func RenderLeftAligned(canvas *rgb.Canvas, img image.Image, width int, xShift int, yShift int) (image.Image, error) {
	startX := width - img.Bounds().Dx() + xShift
	startY := 0 + yShift

	bounds := image.Rect(startX, startY, canvas.Bounds().Dx()-1, canvas.Bounds().Dy()-1)

	i := image.NewRGBA(bounds)
	draw.Draw(i, bounds, img, image.ZP, draw.Over)

	return i, nil
}

func RenderRightAligned(canvas *rgb.Canvas, img image.Image, width int, xShift int, yShift int) (image.Image, error) {
	startX := width + xShift
	startY := 0 + yShift

	bounds := image.Rect(startX, startY, canvas.Bounds().Dx()-1, canvas.Bounds().Dy()-1)

	i := image.NewRGBA(bounds)
	draw.Draw(i, bounds, img, image.ZP, draw.Over)

	return i, nil
}
