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
	teamKey         string
	zoom            float64
	xPosition       int
	yPosition       int
	sourceLogo      image.Image
	bounds          image.Rectangle
	targetDirectory string
}

type OptionFunc func(*Logo) error

func New(teamKey string, sourceLogo image.Image, targetDirectory string, zoom float64) *Logo {
	return &Logo{
		teamKey:         teamKey,
		targetDirectory: targetDirectory,
		zoom:            zoom,
		sourceLogo:      sourceLogo,
	}
}

func (l *Logo) ThumbnailFilename(size image.Rectangle) string {
	return fmt.Sprintf("%s/%s_%dx%d_%f.png", l.targetDirectory, l.teamKey, size.Dx(), size.Dy(), l.zoom)
}

func (l *Logo) GetThumbnail(size image.Rectangle) (image.Image, error) {
	thumbFile := l.ThumbnailFilename(size)

	var thumbnail image.Image

	if _, err := os.Stat(thumbFile); err != nil {
		if os.IsNotExist(err) {
			// Create the thumbnail
			fmt.Printf("Saving thumbnail logo %s\n", thumbFile)
			thumbnail = rgbrender.ResizeImage(l.sourceLogo, size, l.zoom)

			if err := rgbrender.SavePng(thumbnail, thumbFile); err != nil {
				return nil, fmt.Errorf("failed to save logo %s: %w", thumbFile, err)
			}

			return thumbnail, nil
		}

		return nil, err
	}

	t, err := os.Open(thumbFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open logo %s: %w", thumbFile, err)
	}
	defer t.Close()

	thumbnail, err = png.Decode(t)
	if err != nil {
		return nil, fmt.Errorf("failed to decode logo %s: %w", thumbFile, err)
	}

	return thumbnail, nil
}

func RenderLeftAligned(canvas *rgb.Canvas, img image.Image, width int, xShift int, yShift int) (image.Image, error) {
	startX := width - img.Bounds().Dx() + xShift
	startY := 0 + yShift

	bounds := image.Rect(startX, startY, canvas.Bounds().Dx()-1, canvas.Bounds().Dy()-1)

	i := image.NewRGBA(bounds)
	draw.Draw(i, bounds, img, image.ZP, draw.Over)

	return i, nil

	/*
		draw.Draw(canvas, canvas.Bounds(), i, image.ZP, draw.Over)

		return nil
	*/
}

func RenderRightAligned(canvas *rgb.Canvas, img image.Image, width int, xShift int, yShift int) (image.Image, error) {
	startX := width + xShift
	startY := 0 + yShift

	bounds := image.Rect(startX, startY, canvas.Bounds().Dx()-1, canvas.Bounds().Dy()-1)

	i := image.NewRGBA(bounds)
	draw.Draw(i, bounds, img, image.ZP, draw.Over)

	return i, nil

	/*
		draw.Draw(canvas, canvas.Bounds(), i, image.ZP, draw.Over)

		return nil
	*/
}
