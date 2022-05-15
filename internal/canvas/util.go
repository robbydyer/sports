package canvas

import (
	"image"
	"image/color"
)

func position(x int, y int, canvasWidth int) int {
	return x + (y * canvasWidth)
}

func isBlack(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r == 0 && b == 0 && g == 0
}

func firstNonBlankY(img image.Image) int {
	if img == nil {
		return 0
	}
	for y := img.Bounds().Min.Y; y <= img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x <= img.Bounds().Max.X; x++ {
			if !isBlack(img.At(x, y)) {
				return y
			}
		}
	}

	return img.Bounds().Min.Y
}

func firstNonBlankX(img image.Image) int {
	if img == nil {
		return 0
	}
	for x := img.Bounds().Min.X; x <= img.Bounds().Max.X; x++ {
		for y := img.Bounds().Min.Y; y <= img.Bounds().Max.Y; y++ {
			if !isBlack(img.At(x, y)) {
				return x
			}
		}
	}

	return img.Bounds().Min.X
}

func lastNonBlankY(img image.Image) int {
	if img == nil {
		return 0
	}
	for y := img.Bounds().Max.Y; y >= img.Bounds().Min.Y; y-- {
		for x := img.Bounds().Min.X; x <= img.Bounds().Max.X; x++ {
			if !isBlack(img.At(x, y)) {
				return y
			}
		}
	}

	return img.Bounds().Max.Y
}

func lastNonBlankX(img image.Image) int {
	if img == nil {
		return 0
	}
	for x := img.Bounds().Max.X; x >= img.Bounds().Min.X; x-- {
		for y := img.Bounds().Max.Y; y >= img.Bounds().Min.Y; y-- {
			if !isBlack(img.At(x, y)) {
				return x
			}
		}
	}

	return img.Bounds().Max.X
}
