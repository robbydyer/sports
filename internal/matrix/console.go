package matrix

import (
	"context"
	"fmt"
	"image/color"
	"io"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ConsoleMatrix prints a representation of a matrix to a terminal.
// Useful for testing layouts without a Pi or an LED matrix.
type ConsoleMatrix struct {
	matrix []uint32
	width  int
	height int
	out    io.Writer
	log    *zap.Logger
}

// NewConsoleMatrix ...
func NewConsoleMatrix(width int, height int, out io.Writer, logger *zap.Logger) *ConsoleMatrix {
	c := &ConsoleMatrix{
		width:  width,
		height: height,
		matrix: make([]uint32, (width * height)),
		out:    out,
		log:    logger,
	}

	c.Reset()

	return c
}

// Reset ...
func (c *ConsoleMatrix) Reset() {
	for i := range c.matrix {
		c.matrix[i] = colorToUint32(color.Black)
	}
}

// Geometry ...
func (c *ConsoleMatrix) Geometry() (int, int) {
	return c.width, c.height
}

func (c *ConsoleMatrix) position(x int, y int) int {
	return x + (y * c.width)
}

// At ...
func (c *ConsoleMatrix) At(x int, y int) color.Color {
	position := c.position(x, y)
	if position > len(c.matrix)-1 || position < 0 {
		return color.Black
	}

	return uint32ToColorGo(c.matrix[position])
}

// Set ...
func (c *ConsoleMatrix) Set(x int, y int, clr color.Color) {
	position := c.position(x, y)
	if position > len(c.matrix)-1 || position < 0 {
		return
	}

	c.matrix[position] = colorToUint32(clr)
}

func (c *ConsoleMatrix) PreLoad(points []MatrixPoint) {
	for _, pt := range points {
		c.Set(pt.X, pt.Y, pt.Color)
	}
}

func (c *ConsoleMatrix) Play(ctx context.Context, defInt time.Duration, intervalCh chan time.Duration) error {
	return nil
}

// Render ...
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
			row += "  "
		} else if r > 240 && g > 240 && b > 240 {
			row += "W "
		} else {
			row += "0 "
		}
	}
	rendered = append(rendered, row)

	rendered = append(rendered, strings.Repeat("_ ", c.width+1)+"|")

	fmt.Fprintln(c.out, strings.Join(rendered, "\n"))

	c.Reset()

	return nil
}

// Close ...
func (c *ConsoleMatrix) Close() error {
	return nil
}

// SetBrightness does nothing
func (c *ConsoleMatrix) SetBrightness(brightness int) {
}
