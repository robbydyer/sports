package rgbrender

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/golang/freetype"
	rgb "github.com/robbydyer/rgbmatrix-rpi"
)

type Layout struct {
}

func DrawText(canvas *rgb.Canvas, layout Layout, text string) error {
	c := freetype.NewContext()
	c.SetDst(canvas)

	return nil
}

func DrawRectangle(canvas *rgb.Canvas, startX int, startY int, endX int, endY int, fillColor color.Color) error {
	rect := image.Rect(startX, startY, endX, endY)

	rgba := image.NewRGBA(rect)

	for x := rect.Min.X; x < rect.Max.X; x++ {
		for y := rect.Min.Y; y < rect.Min.Y; y++ {
			rgba.Set(x, y, fillColor)
		}
	}

	return nil
}

func ShowImage(canvas *rgb.Canvas, img image.Image) error {
	draw.Draw(canvas, canvas.Bounds(), img, image.ZP, draw.Over)
	return canvas.Render()
}
