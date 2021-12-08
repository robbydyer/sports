package rgbrender

import (
	"image"
	"image/color"
	"math"

	"go.uber.org/zap"
)

// GradientXRectangle creates a rectangle that has a solid fill in the center of the rect
// of the given fillPercentage, then gradually increases transparency beyond those bounds in both directions
// on the X axis
func GradientXRectangle(bounds image.Rectangle, fillPercentage float64, baseColor color.Color, logger *zap.Logger) image.Image {
	grad := image.NewNRGBA(bounds)

	centerWidth := int(math.Ceil((float64(bounds.Dx()) * fillPercentage)))
	outerWidth := bounds.Dx() - centerWidth
	minFull := bounds.Max.X - centerWidth - (outerWidth / 2)
	maxFull := bounds.Max.X - (outerWidth / 2)

	// gradientStep := uint8(255 / (outerWidth / 2))
	gradientStep := uint8(1)
	if outerWidth < 128 {
		gradientStep = uint8(255 / (outerWidth / 2))
	}
	if gradientStep == 0 {
		gradientStep = 1
	}
	r1, g1, b1, _ := baseColor.RGBA()
	r := uint8(r1)
	g := uint8(g1)
	b := uint8(b1)
	leftGradient := uint8(0)
	rightGradient := uint8(255)

	if logger != nil {
		logger.Debug("gradient",
			zap.Int("fill start X", minFull),
			zap.Int("fill end X", maxFull),
			zap.Int("gradient start", bounds.Min.X),
			zap.Int("gradient end", bounds.Max.X),
			zap.Uint8("gradient step", gradientStep),
		)
	}

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		bumpLeft := false
		bumpRight := false
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			if x < minFull {
				grad.Set(x, y, color.NRGBA{r, g, b, leftGradient})
				bumpLeft = true
			} else if x > maxFull {
				grad.Set(x, y, color.NRGBA{r, g, b, rightGradient})
				bumpRight = true
			} else {
				grad.Set(x, y, baseColor)
			}
		}
		if bumpLeft {
			if logger != nil {
				logger.Debug("gradient fill",
					zap.String("side", "left"),
					zap.Uint8("gradient", leftGradient),
					zap.Int("X", x),
				)
			}
			if (255 - leftGradient) < gradientStep {
				leftGradient = 255
			} else {
				leftGradient += gradientStep
			}
		}
		if bumpRight {
			if gradientStep > rightGradient {
				rightGradient = 0
			} else {
				rightGradient -= gradientStep
			}
		}
	}

	return grad
}
