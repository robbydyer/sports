package rgbmatrix

import (
	"fmt"
	"image/color"
	"io"
	"strings"

	"go.uber.org/zap"
)

type ConsoleMatrix struct {
	matrix []uint32
	width  int
	height int
	out    io.Writer
	log    *zap.Logger
}

func NewConsoleMatrix(width int, height int, out io.Writer, logger *zap.Logger) *ConsoleMatrix {
	c := &ConsoleMatrix{
		width:  width,
		height: height,
		matrix: make([]uint32, width*height),
		out:    out,
		log:    logger,
	}

	c.Reset()

	return c
}

func (c *ConsoleMatrix) Reset() {
	for i := range c.matrix {
		c.matrix[i] = colorToUint32(color.Black)
	}
}

func (c *ConsoleMatrix) Geometry() (int, int) {
	return c.width, c.height
}
func (c *ConsoleMatrix) At(position int) color.Color {
	if position > len(c.matrix)-1 || position < 0 {
		return color.Black
	}

	return uint32ToColorGo(c.matrix[position])
}
func (c *ConsoleMatrix) Set(position int, clr color.Color) {
	if position > len(c.matrix)-1 || position < 0 {
		return
	}

	c.matrix[position] = colorToUint32(clr)
}
func (c *ConsoleMatrix) Apply(leds []color.Color) error {
	for position, clr := range leds {
		c.Set(position, clr)
	}

	return c.Render()
}
func (c *ConsoleMatrix) Render() error {
	rendered := []string{
		strings.Repeat("_ ", c.width+1),
	}
	row := ""
	for index, clrint := range c.matrix {
		clr := uint32ToColorGo(clrint)
		if (index)%c.width == 0 {
			// This is a new row
			row += "|"
			rendered = append(rendered, row)
			row = "|"
		}
		if clr == nil {
			row += "  "
			continue
		}

		r, g, b, _ := clr.RGBA()

		if r > g && r > b {
			row += "R "
		} else if g > r && g > b {
			row += "G "
		} else if b > r && b > g {
			row += "B "
		} else if r < 40 && g < 40 && b < 40 {
			row += "_ "
		} else if r > 240 && g > 240 && b > 240 {
			row += "W "
		} else {
			row += "0 "
		}
	}

	rendered = append(rendered, strings.Repeat("_ ", c.width+1)+"|")

	fmt.Fprintln(c.out, strings.Join(rendered, "\n"))

	c.Reset()

	return nil
}

func (c *ConsoleMatrix) Close() error {
	return nil
}
func (c *ConsoleMatrix) SetBrightness(brightness int) {
	return
}
