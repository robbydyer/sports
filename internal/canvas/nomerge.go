package canvas

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"sort"

	"github.com/robbydyer/sports/internal/matrix"
	"go.uber.org/zap"
)

// getActualPixel returns the pixel color at virtual coordinates in unmerged canvas list
func (c *ScrollCanvas) getActualPixel(virtualX int, virtualY int) color.Color {
	if len(c.subCanvases) < 1 {
		c.PrepareSubCanvases()
	}

	for _, sub := range c.subCanvases {
		if virtualX >= sub.virtualStartX && virtualX <= sub.virtualEndX {
			actualX := (virtualX - sub.virtualStartX) + sub.actualStartX
			return sub.img.At(actualX, virtualY)
		}
	}

	return color.Black
}

// PrepareSubCanvases
func (c *ScrollCanvas) PrepareSubCanvases() {
	if len(c.actuals) < 1 {
		return
	}
	c.subCanvases = []*subCanvasHorizontal{
		{
			index:         0,
			actualStartX:  0,
			actualEndX:    c.w,
			virtualStartX: 0,
			virtualEndX:   c.w,
			img:           image.NewRGBA(image.Rect(0, 0, c.w, c.h)),
		},
	}

	index := 1
	for _, actual := range c.actuals {
		c.subCanvases = append(c.subCanvases,
			&subCanvasHorizontal{
				actualStartX: firstNonBlankX(actual),
				actualEndX:   lastNonBlankX(actual),
				img:          actual,
				index:        index,
			},
		)
		index++
		c.subCanvases = append(c.subCanvases,
			&subCanvasHorizontal{
				actualStartX: 0,
				actualEndX:   c.mergePad,
				img:          image.NewRGBA(image.Rect(0, 0, c.mergePad, c.h)),
				index:        index,
			},
		)
		index++
	}

	c.subCanvases = append(c.subCanvases,
		&subCanvasHorizontal{
			index:        index,
			actualStartX: 0,
			actualEndX:   c.w,
			img:          image.NewRGBA(image.Rect(0, 0, c.w, c.h)),
		},
	)

	sort.SliceStable(c.subCanvases, func(i int, j int) bool {
		return c.subCanvases[i].index < c.subCanvases[j].index
	})

	c.log.Debug("done initializing sub canvases",
		zap.Int("num", len(c.subCanvases)),
	)

SUBS:
	for _, sub := range c.subCanvases {
		if sub.index == 0 {
			continue SUBS
		}

		prev := c.prevSub(sub)

		if prev == nil {
			continue SUBS
		}

		sub.virtualStartX = prev.virtualEndX + 1
		diff := sub.actualEndX - sub.actualStartX
		if sub.actualStartX < 1 {
			diff = sub.actualEndX - (sub.actualStartX * -1)
		}
		sub.virtualEndX = sub.virtualStartX + diff

		c.log.Debug("define sub canvas",
			zap.Int("index", sub.index),
			zap.Int("actualstartX", sub.actualStartX),
			zap.Int("actualendX", sub.actualEndX),
			zap.Int("virtualstartX", sub.virtualStartX),
			zap.Int("virtualendx", sub.virtualEndX),
			zap.Int("actual canvas Width", c.w),
		)
	}
	c.log.Debug("done defining sub canvases")
}

func (c *ScrollCanvas) prevSub(me *subCanvasHorizontal) *subCanvasHorizontal {
	if me.index == 0 {
		return nil
	}
	for _, sub := range c.subCanvases {
		if sub.index == me.index-1 {
			return sub
		}
	}
	return nil
}

func (c *ScrollCanvas) rightToLeftNoMerge(ctx context.Context) error {
	if len(c.subCanvases) < 1 {
		c.PrepareSubCanvases()
	}
	if len(c.subCanvases) < 1 {
		return fmt.Errorf("not enough subcanvases to merge")
	}

	finish := c.subCanvases[len(c.subCanvases)-1].virtualEndX

	virtualX := c.subCanvases[0].virtualStartX

	c.log.Debug("performing right to left scroll without canvas merge",
		zap.Int("virtualX start", virtualX),
		zap.Int("finish", finish),
	)

	for {
		if virtualX == finish {
			break
		}

		loader := make([]matrix.MatrixPoint, c.w*c.h)

		index := 0
		for x := 0; x < c.w; x++ {
			for y := 0; y < c.h; y++ {
				thisVirtualX := x + virtualX

				loader[index] = matrix.MatrixPoint{
					X:     x,
					Y:     y,
					Color: c.getActualPixel(thisVirtualX, y),
				}
				index++

			}
		}
		virtualX++

		c.Matrix.PreLoad(loader)
	}

	return c.Matrix.Play(ctx, c.interval)
}
