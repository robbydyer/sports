package sportboard

import (
	"image"
	"image/color"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

// RenderGameCounter ...
func (s *SportBoard) RenderGameCounter(canvas board.Canvas, numGames int, activeIndex int, spacing int) (image.Image, error) {
	totalWidth := (numGames * spacing) + numGames - 1

	aligned, err := rgbrender.AlignPosition(rgbrender.CenterBottom, canvas.Bounds(), totalWidth, 1)
	if err != nil {
		return nil, err
	}

	realActive := activeIndex + (activeIndex * spacing)

	s.log.Debug("Rendering counter",
		zap.Int("active index", activeIndex),
		zap.Int("real active", realActive),
		zap.Int("start x", aligned.Min.X),
		zap.Int("start y", aligned.Min.Y),
		zap.Int("end x", aligned.Max.X),
		zap.Int("end y", aligned.Max.Y),
	)

	img := image.NewRGBA(canvas.Bounds())

	yPix := aligned.Max.Y - 1
	for i := 0; i < totalWidth; i += spacing + 1 {
		xPix := aligned.Min.X + i
		if i == realActive || (i == 0 && activeIndex == 0) {
			img.Set(xPix, yPix, color.RGBA{255, 0, 0, 255})
			continue
		}
		img.Set(xPix, yPix, color.White)
	}

	s.counter = img
	return img, nil
}
