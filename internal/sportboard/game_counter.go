package sportboard

import (
	"image"
	"image/color"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/rgbrender"
)

// RenderGameCounter ...
func (s *SportBoard) RenderGameCounter(canvas board.Canvas, numGames int, activeIndex int) (image.Image, error) {
	bounds := rgbrender.ZeroedBounds(canvas.Bounds())
	spacing := bounds.Dy() / 32
	pixSize := bounds.Dy() / 32
	totalWidth := (numGames * spacing) + (pixSize * (numGames - 1))

	aligned, err := rgbrender.AlignPosition(rgbrender.CenterBottom, bounds, totalWidth, 1)
	if err != nil {
		return nil, err
	}

	realActive := activeIndex + (activeIndex * (spacing + pixSize - 1))

	s.log.Debug("Rendering counter",
		zap.Int("active index", activeIndex),
		zap.Int("num games", numGames),
		zap.Int("real active", realActive),
		zap.Int("start x", aligned.Min.X),
		zap.Int("start y", aligned.Min.Y),
		zap.Int("end x", aligned.Max.X),
		zap.Int("end y", aligned.Max.Y),
		zap.Int("spacing", spacing),
		zap.Int("pix size", pixSize),
	)

	img := image.NewRGBA(bounds)

	yPix := aligned.Max.Y - 1
	for i := 0; i < totalWidth; i += spacing + 1 {
		xPix := aligned.Min.X + i
		if i == realActive || (i == 0 && activeIndex == 0) {
			for x := 0; x < pixSize; x++ {
				firstY := yPix
				for y := 0; y < pixSize; y++ {
					img.Set(xPix, yPix, red)
					yPix--
				}
				yPix = firstY
				xPix++
				i++
			}
			i--
			continue
		}
		for x := 0; x < pixSize; x++ {
			firstY := yPix
			for y := 0; y < pixSize; y++ {
				img.Set(xPix, yPix, color.White)
				yPix--
			}
			yPix = firstY
			xPix++
			i++
		}
		i--
	}

	return img, nil
}
