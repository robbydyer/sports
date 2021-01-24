package rgbrender

import (
	"image"
	"image/png"
	"os"

	"github.com/nfnt/resize"
)

func ResizeImage(img image.Image, bounds image.Rectangle, zoom float64) image.Image {
	// Ignore Y, so that we maintain aspect ratio
	sizeX, sizeY := ZoomImageSize(bounds, zoom)

	return resize.Thumbnail(uint(sizeX), uint(sizeY), img, resize.Lanczos3)
}

func SavePng(img image.Image, fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// ShiftedSize shifts an image's start location and returns its resulting bounds
func ShiftedSize(xStart int, yStart int, bounds image.Rectangle) image.Rectangle {
	startX := bounds.Min.X + xStart
	startY := bounds.Min.Y + yStart
	endX := bounds.Dx() + startX
	endY := bounds.Dy() + startY

	return image.Rect(startX, startY, endX, endY)
}
