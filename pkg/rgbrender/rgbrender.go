package rgbrender

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
)

type Align int

const (
	CenterCenter Align = iota
	CenterTop
	CenterBottom
	RightCenter
	RightTop
	RightBottom
	LeftCenter
	LeftTop
	LeftBottom
)

// SetImageAlign
func SetImageAlign(canvas *rgb.Canvas, align Align, img image.Image) (image.Image, error) {
	rect, err := AlignPosition(align, canvas.Bounds(), img.Bounds().Dx(), img.Bounds().Dy())
	if err != nil {
		return nil, err
	}

	n := image.NewRGBA(rect)

	draw.Draw(n, rect.Bounds(), img, image.Pt(rect.Min.X, rect.Min.Y), draw.Over)

	return n, nil
}

// AlignPosition returns image.Rectangle bounds for an image within a given bounds
func AlignPosition(align Align, bounds image.Rectangle, sizeX int, sizeY int) (image.Rectangle, error) {
	startX := 0
	startY := 0
	endX := sizeX
	endY := sizeY
	if align == CenterTop || align == CenterCenter || align == CenterBottom {
		startX = int(math.Ceil(float64(bounds.Max.X-sizeX) / 2))
		endX = startX + sizeX - 1
	} else if align == RightTop || align == RightCenter || align == RightBottom {
		startX = bounds.Max.X - sizeX + 1
		endX = bounds.Max.X
	} else {
		// Default to Left
		startX = bounds.Min.X
		endX = bounds.Min.X + sizeX - 1
	}

	if align == CenterCenter || align == RightCenter || align == LeftCenter {
		startY = int(math.Ceil(float64(bounds.Max.Y-sizeY) / 2))
		endY = startY + sizeY - 1
	} else if align == CenterBottom || align == RightBottom || align == LeftBottom {
		startY = bounds.Max.Y - sizeY + 1
		endY = bounds.Max.Y
	} else {
		// defaults to Top
		startY = bounds.Min.Y
		endY = sizeY - 1
	}

	return image.Rectangle{
		Min: image.Point{
			X: startX,
			Y: startY,
		},
		Max: image.Point{
			X: endX,
			Y: endY,
		},
	}, nil
}

// ZoomImageSize takes a zoom percentage and returns the resulting image size.
func ZoomImageSize(img image.Image, zoom float64) (int, int) {
	fullX := img.Bounds().Dx() + 1
	fullY := img.Bounds().Dy() + 1

	if zoom <= 0 {
		return 0, 0
	}

	return int(math.Round(float64(fullX) * zoom)), int(math.Round(float64(fullY) * zoom))
}

// DrawRectangle ...
func DrawRectangle(canvas *rgb.Canvas, startX int, startY int, sizeX int, sizeY int, fillColor color.Color) error {
	rect := image.Rect(startX, startY, startX+sizeX, startY+sizeY)

	rgba := image.NewRGBA(rect)

	for x := rect.Min.X; x < rect.Max.X; x++ {
		for y := rect.Min.Y; y < rect.Min.Y; y++ {
			rgba.Set(x, y, fillColor)
		}
	}

	draw.Draw(canvas, canvas.Bounds(), rgba, image.ZP, draw.Over)

	return nil
}
