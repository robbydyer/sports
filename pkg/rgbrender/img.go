package rgbrender

import (
	"context"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"time"

	"github.com/nfnt/resize"
	"github.com/spf13/afero"

	"github.com/robbydyer/sports/pkg/board"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
)

// ResizeImage ...
func ResizeImage(img image.Image, bounds image.Rectangle, zoom float64) image.Image {
	// Ignore Y, so that we maintain aspect ratio
	sizeX, sizeY := ZoomImageSize(bounds, zoom)

	return resize.Thumbnail(uint(sizeX), uint(sizeY), img, resize.Lanczos3)
}

// ResizeGIF ...
func ResizeGIF(g *gif.GIF, bounds image.Rectangle, zoom float64) error {
	var newPals []*image.Paletted
	for _, i := range g.Image {
		resizedI := ResizeImage(i, bounds, 1)

		/*
			resizedPal := image.NewPaletted(resizedI.Bounds(), nil)
			quantizer := gogif.MedianCutQuantizer{NumColor: 64}
			quantizer.Quantize(resizedPal, resizedI.Bounds(), resizedI, image.Point{})
		*/

		resizedPal := image.NewPaletted(resizedI.Bounds(), palette.Plan9)
		draw.Draw(resizedPal, resizedPal.Bounds(), resizedI, resizedPal.Bounds().Min, draw.Over)
		newPals = append(newPals, resizedPal)
	}

	g.Image = newPals

	return nil
}

// SavePng ...
func SavePng(img image.Image, fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// SaveGif ...
func SaveGif(img *gif.GIF, fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	return gif.EncodeAll(f, img)
}

// SavePngAfero ...
func SavePngAfero(fs afero.Fs, img image.Image, fileName string) error {
	f, err := fs.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// SaveGifAfero ...
func SaveGifAfero(fs afero.Fs, img *gif.GIF, fileName string) error {
	f, err := fs.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	return gif.EncodeAll(f, img)
}

// ShiftedSize shifts an image's start location and returns its resulting bounds
func ShiftedSize(xStart int, yStart int, bounds image.Rectangle) image.Rectangle {
	startX := bounds.Min.X + xStart
	startY := bounds.Min.Y + yStart
	endX := bounds.Dx() + startX
	endY := bounds.Dy() + startY

	return image.Rect(startX, startY, endX, endY)
}

// DrawImageAligned draws an image aligned within the given bounds
func DrawImageAligned(canvas *rgb.Canvas, bounds image.Rectangle, img *image.RGBA, align Align) error {
	aligned, err := AlignPosition(align, bounds, img.Bounds().Dx(), img.Bounds().Dy())
	if err != nil {
		return err
	}

	img.Rect.Min = aligned.Min
	img.Rect.Max = aligned.Max

	return DrawImage(canvas, canvas.Bounds(), img)
}

// DrawImage draws an image
func DrawImage(canvas draw.Image, bounds image.Rectangle, img image.Image) error {
	draw.Draw(canvas, bounds, img, img.Bounds().Min, draw.Over)
	return nil
}

// PlayImages plays s series of images. If loop == 0, it will play forever until the context is canceled
func PlayImages(ctx context.Context, canvas board.Canvas, images []image.Image, delay []time.Duration, loop int) error {
	center, err := AlignPosition(CenterCenter, canvas.Bounds(), images[0].Bounds().Dx(), images[0].Bounds().Dy())
	if err != nil {
		return err
	}

	l := len(images)
	i := 0
	for {
		select {
		case <-ctx.Done():
			// no error, since this is how we stop the GIF
			return nil
		default:
		}

		if canvas == nil {
			return fmt.Errorf("nil canvas passed to PlayImages")
		}

		draw.Draw(canvas, center, images[i], image.Point{}, draw.Over)

		if err := canvas.Render(); err != nil {
			return err
		}

		time.Sleep(delay[i])

		i++
		if i >= l {
			if loop == 0 {
				i = 0
				continue
			}

			break
		}
	}

	return nil
}

// PlayGIF reads and draw a gif file from r. It use the contained images and
// delays and loops over it, until a true is sent to the returned chan
func PlayGIF(ctx context.Context, canvas board.Canvas, gif *gif.GIF) error {
	delay := make([]time.Duration, len(gif.Image))
	images := make([]image.Image, len(gif.Image))
	for i, image := range gif.Image {
		images[i] = image
		delay[i] = time.Millisecond * time.Duration(gif.Delay[i]) * 10
	}

	return PlayImages(ctx, canvas, images, delay, gif.LoopCount)
}
