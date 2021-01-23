package rgbrender

import (
	"image"
	"image/png"
	"os"

	"github.com/nfnt/resize"
)

func ResizeImage(img image.Image, zoom float64) image.Image {
	// Ignore Y, so that we maintain aspect ratio
	sizeX, _ := ZoomImageSize(img, zoom)

	return resize.Resize(uint(sizeX), 0, img, resize.Lanczos3)
}

func SavePng(img image.Image, fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}
