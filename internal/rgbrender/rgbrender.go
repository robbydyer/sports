package rgbrender

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"

	cnvs "github.com/robbydyer/sports/internal/canvas"
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
func SetImageAlign(canvas *cnvs.Canvas, align Align, img image.Image) (image.Image, error) {
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
func DrawRectangle(canvas draw.Image, startX int, startY int, sizeX int, sizeY int, fillColor color.Color) error {
	for x := startX; x < startX+sizeX; x++ {
		for y := startY; y < startY+sizeY; y++ {
			canvas.Set(x, y, fillColor)
		}
	}

	return nil
}

func DrawSquare(canvas draw.Image, start image.Point, width int, outlineClr color.Color, fillClr color.Color) {
	for x := start.X; x <= start.X+width; x++ {
		for y := start.Y; y <= start.Y+width; y++ {
			if x == start.X || x == start.X+width || y == start.Y || y == start.Y+width {
				canvas.Set(x, y, outlineClr)
			} else {
				canvas.Set(x, y, fillClr)
			}
		}
	}
}

func DrawVerticalLine(canvas draw.Image, start image.Point, end image.Point, clr color.Color) {
	for x := start.X; x <= end.X; x++ {
		for y := start.Y; y <= end.Y; y++ {
			canvas.Set(x, y, clr)
		}
	}
}

func DrawUpTriangle(canvas draw.Image, start image.Point, width int, height int, outlineColor color.Color, fillColor color.Color) {
	canvas.Set(start.X, start.Y, outlineColor)
	topY := start.Y
	for x := start.X; x < start.X+(width/2); x++ {
		canvas.Set(x, topY, outlineColor)
		for y := start.Y - (height / 2); y < start.Y+(height/2); y++ {
			if y > topY {
				canvas.Set(x, y, fillColor)
			}
		}
		topY--
	}
	for x := start.X + (width / 2); x <= start.X+width; x++ {
		canvas.Set(x, topY, outlineColor)
		for y := start.Y - (height / 2); y < start.Y+(height/2); y++ {
			if y > topY {
				canvas.Set(x, y, fillColor)
			}
		}
		topY++
	}
}

func DrawDownTriangle(canvas draw.Image, start image.Point, width int, height int, outlineColor color.Color, fillColor color.Color) {
	canvas.Set(start.X, start.Y, outlineColor)
	botY := start.Y
	for x := start.X; x < start.X+(width/2); x++ {
		canvas.Set(x, botY, outlineColor)
		for y := start.Y; y <= start.Y+(height/2); y++ {
			if y < botY {
				canvas.Set(x, y, fillColor)
			}
		}
		botY++
	}
	for x := start.X + (width / 2); x <= start.X+width; x++ {
		canvas.Set(x, botY, outlineColor)
		for y := start.Y; y <= start.Y+(height/2); y++ {
			if y < botY {
				canvas.Set(x, y, fillColor)
			}
		}
		botY--
	}
}

func DrawDiamond(canvas draw.Image, start image.Point, width int, height int, outlineColor color.Color, fillColor color.Color) {
	canvas.Set(start.X, start.Y, outlineColor)
	topY := start.Y
	botY := start.Y
	for x := start.X; x < start.X+(width/2); x++ {
		canvas.Set(x, topY, outlineColor)
		canvas.Set(x, botY, outlineColor)
		for y := start.Y - (height / 2); y < start.Y+(height/2); y++ {
			if y > topY && y < botY {
				canvas.Set(x, y, fillColor)
			}
		}
		topY--
		botY++
	}
	for x := start.X + (width / 2); x <= start.X+width; x++ {
		canvas.Set(x, topY, outlineColor)
		canvas.Set(x, botY, outlineColor)
		for y := start.Y - (height / 2); y < start.Y+(height/2); y++ {
			if y > topY && y < botY {
				canvas.Set(x, y, fillColor)
			}
		}
		topY++
		botY--
	}
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

func HexToRGB(hexClr string) (uint8, uint8, uint8, error) {
	values, err := strconv.ParseUint(hexClr, 16, 32)
	if err != nil {
		return 0, 0, 0, err
	}

	return uint8(values >> 16), uint8(values>>8) & 0xFF, uint8(values & 0xFF), nil
}
