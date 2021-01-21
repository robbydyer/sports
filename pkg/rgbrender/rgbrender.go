package rgbrender

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
	"time"

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

func PlayImages(canvas *rgb.Canvas, images []image.Image, delay []time.Duration, loop int) (chan bool, chan error) {
	quit := make(chan bool, 0)
	errChan := make(chan error)

	go func() {
		l := len(images)
		i := 0
		for {
			select {
			case <-quit:
				return
			default:
				if err := ShowImage(canvas, images[i]); err != nil {
					errChan <- err
				}
				time.Sleep(delay[i])
			}

			i++
			if i >= l {
				if loop == 0 {
					i = 0
					continue
				}

				break
			}
		}
	}()

	return quit, errChan
}

// PlayGIF reads and draw a gif file from r. It use the contained images and
// delays and loops over it, until a true is sent to the returned chan
func PlayGIF(canvas *rgb.Canvas, r io.Reader) (chan bool, chan error) {
	errChan := make(chan error)
	gif, err := gif.DecodeAll(r)
	if err != nil {
		defer func() { errChan <- err }()
		return nil, errChan
	}

	delay := make([]time.Duration, len(gif.Delay))
	images := make([]image.Image, len(gif.Image))
	for i, image := range gif.Image {
		images[i] = image
		delay[i] = time.Millisecond * time.Duration(gif.Delay[i]) * 10
	}

	return PlayImages(canvas, images, delay, gif.LoopCount)
}
