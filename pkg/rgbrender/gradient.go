package rgbrender

import (
	"image"
	"image/color"
	"math"
)

// GradientRectangle creates a rectangle that has a solid fill in the center of the rect
// of the given fillPercentage, then gradually increases transparency beyond those bounds
func GradientRectangle(bounds image.Rectangle, fillPercentage float64, baseColor color.Color) image.Image {
	grad := image.NewNRGBA(bounds)

	centerWidth := int(math.Ceil((float64(bounds.Dx()) * fillPercentage)))
	outerWidth := bounds.Dx() - centerWidth
	minFull := bounds.Max.X - centerWidth - (outerWidth / 2)
	maxFull := bounds.Max.X - (outerWidth / 2)

	gradientStep := uint8(255 / (outerWidth / 2))
	r1, g1, b1, _ := baseColor.RGBA()
	r := uint8(r1)
	g := uint8(g1)
	b := uint8(b1)
	leftGradient := uint8(0)
	rightGradient := uint8(255)
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			if x < minFull {
				grad.Set(x, y, color.NRGBA{r, g, b, leftGradient})
				leftGradient += gradientStep
			} else if x > maxFull {
				grad.Set(x, y, color.NRGBA{r, g, b, rightGradient})
				rightGradient -= gradientStep
			} else {
				grad.Set(x, y, baseColor)
			}
		}
	}

	return grad
}
