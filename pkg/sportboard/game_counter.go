package sportboard

import (
	"image"
	"image/color"

	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *SportBoard) RenderGameCounter(canvas *rgb.Canvas, numGames int, activeIndex int, spacing int) (image.Image, error) {
	totalWidth := (numGames * spacing) + numGames - 1

	aligned, err := rgbrender.AlignPosition(rgbrender.CenterBottom, canvas.Bounds(), totalWidth, 1)
	if err != nil {
		return nil, err
	}

	realActive := activeIndex + (activeIndex * spacing)

	s.log.Debugf("Rendering counter: ActiveIndex: %d -> %d. From %d, %d to %d, %d",
		activeIndex,
		realActive,
		aligned.Min.X,
		aligned.Min.Y,
		aligned.Max.X,
		aligned.Max.Y,
	)

	img := image.NewRGBA(canvas.Bounds())

	yPix := aligned.Max.Y - 1
	for i := 0; i < totalWidth; i += spacing + 1 {
		xPix := aligned.Min.X + i
		if i == realActive || (i == 0 && activeIndex == 0) {
			s.log.Debugf("Setting pixel %d, %d to red", xPix, yPix)
			img.Set(xPix, yPix, color.RGBA{255, 0, 0, 255})
			continue
		}
		s.log.Debugf("Setting pixel %d, %d to white", xPix, yPix)
		img.Set(xPix, yPix, color.White)
	}

	s.counter = img
	return img, nil
}
