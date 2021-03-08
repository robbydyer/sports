package rgbrender

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
)

func TestGridLayout(t *testing.T) {
	tests := []struct {
		name         string
		canvasWidth  int
		canvasHeight int
		cols         int
		rows         int
		expectedNum  int
	}{
		{
			name:         "square",
			canvasWidth:  100,
			canvasHeight: 100,
			cols:         2,
			rows:         2,
			expectedNum:  4,
		},
	}

	log, err := zap.NewDevelopment()
	require.NoError(t, err)

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			canvas := board.NewBlankCanvas(100, 100, log)
			grid, err := NewGrid(canvas, test.cols, test.rows, nil, WithUniformCells())
			require.NoError(t, err)
			require.Equal(t, test.expectedNum, len(grid.cells))

			for _, cell := range grid.Cells() {
				c := cell.Canvas
				require.NotNil(t, c)
				require.Equal(t, test.canvasWidth/test.cols, c.Bounds().Dx())
				require.Equal(t, test.canvasHeight/test.rows, c.Bounds().Dy())
			}
		})
	}
}

func TestGrid(t *testing.T) {
	log, err := zap.NewDevelopment()
	require.NoError(t, err)
	canvas := board.NewBlankCanvas(100, 100, log)
	grid, err := NewGrid(canvas, 2, 2, log, WithUniformCells())
	require.NoError(t, err)

	clr := color.RGBA{
		R: 50,
		G: 50,
		B: 50,
		A: 255,
	}

	for _, cell := range grid.Cells() {
		c := cell.Canvas
		draw.Draw(c, c.Bounds(), image.NewUniform(clr), image.Point{}, draw.Over)
		checkColors(t, c, clr)
	}

	err = grid.DrawToBase(canvas)
	require.NoError(t, err)

	checkColors(t, canvas, clr)
}

func checkColors(t *testing.T, canvas board.Canvas, clr color.Color) {
	for x := 0; x < canvas.Bounds().Dx(); x++ {
		for y := 0; y < canvas.Bounds().Dy(); y++ {
			if y == 50 {
				y += 2
			}
			require.Equal(t, clr, canvas.At(x, y), fmt.Sprintf("x: %d, y: %d", x, y))
		}
	}
}
