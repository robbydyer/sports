package rgbrender

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/robbydyer/sports/internal/board"
	rgb "github.com/robbydyer/sports/internal/rgbmatrix-rpi"
)

// Align represents alignment vertically and horizontally
type Align int

const (
	// CenterCenter ...
	CenterCenter Align = iota
	// CenterTop ...
	CenterTop
	// CenterBottom ...
	CenterBottom
	// RightCenter ...
	RightCenter
	// RightTop ...
	RightTop
	// RightBottom ...
	RightBottom
	// LeftCenter ...
	LeftCenter
	// LeftTop ...
	LeftTop
	// LeftBottom ...
	LeftBottom
)

// SetImageAlign ...
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
	endX := 0
	endY := 0

	if align == CenterTop || align == CenterCenter || align == CenterBottom {
		startX = int(math.Ceil(float64(bounds.Max.X-sizeX) / 2))
		endX = startX + sizeX - 1
	} else if align == RightTop || align == RightCenter || align == RightBottom {
		startX = bounds.Max.X - sizeX + 1
		endX = bounds.Max.X

		if startX < bounds.Min.X {
			newStartX := bounds.Min.X
			diff := newStartX - startX
			endX += diff
			startX = newStartX
		}
	} else {
		// Default to Left
		startX = bounds.Min.X
		endX = bounds.Min.X + sizeX - 1

		if endX > bounds.Max.X {
			newEndX := bounds.Max.X
			diff := newEndX - endX
			startX += diff
			endX = newEndX
		}
	}

	if align == CenterCenter || align == RightCenter || align == LeftCenter {
		startY = int(math.Ceil(float64(bounds.Max.Y-sizeY) / 2))
		endY = startY + sizeY - 1
	} else if align == CenterBottom || align == RightBottom || align == LeftBottom {
		startY = bounds.Max.Y - sizeY + 1
		endY = bounds.Max.Y

		if startY < bounds.Min.Y {
			newStartY := bounds.Min.Y
			diff := newStartY - startY
			endY += diff
			startY = newStartY
		}
	} else {
		// defaults to Top
		startY = bounds.Min.Y
		endY = sizeY - 1

		if endY > bounds.Max.Y {
			newEndY := bounds.Max.Y
			diff := newEndY - endY
			startY += diff
			endY = newEndY
		}
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
func DrawRectangle(canvas board.Canvas, startX int, startY int, sizeX int, sizeY int, fillColor color.Color) error {
	rect := image.Rect(startX, startY, startX+sizeX, startY+sizeY)

	rgba := image.NewRGBA(rect)

	for x := rect.Min.X; x < rect.Max.X; x++ {
		for y := rect.Min.Y; y < rect.Min.Y; y++ {
			rgba.Set(x, y, fillColor)
		}
	}

	draw.Draw(canvas, canvas.Bounds(), rgba, image.Point{}, draw.Over)

	return nil
}

// ZeroedBounds returns an image.Rectangle with square padding stripped off
func ZeroedBounds(bounds image.Rectangle) image.Rectangle {
	bounds = ZeroedXBounds(bounds)
	return ZeroedYBounds(bounds)
}

// ZeroedXBounds returns an image.Rectangle with square padding stripped off
func ZeroedXBounds(bounds image.Rectangle) image.Rectangle {
	if bounds.Min.X >= 0 {
		return bounds
	}
	xPad := bounds.Min.X * -1

	return image.Rect(0, bounds.Min.Y, bounds.Max.X-xPad, bounds.Max.Y)
}

// ZeroedYBounds returns an image.Rectangle with square padding stripped off
func ZeroedYBounds(bounds image.Rectangle) image.Rectangle {
	if bounds.Min.Y >= 0 {
		return bounds
	}
	yPad := bounds.Min.Y * -1

	return image.Rect(bounds.Min.X, 0, bounds.Max.X, bounds.Max.Y-yPad)
}
