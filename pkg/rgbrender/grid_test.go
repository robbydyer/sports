package rgbrender

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/imgcanvas"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGridLayout(t *testing.T) {
	tests := []struct {
		name         string
		canvasWidth  int
		canvasHeight int
		colWidth     int
		rowHeight    int
		expectedNum  int
	}{
		{
			name:         "square",
			canvasWidth:  100,
			canvasHeight: 100,
			colWidth:     50,
			rowHeight:    50,
			expectedNum:  4,
		},
	}

	log, err := zap.NewDevelopment()
	require.NoError(t, err)

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			canvas := imgcanvas.New(100, 100, log)
			canvaser, err := GetCanvaser(canvas, log)
			require.NoError(t, err)
			require.NotNil(t, canvaser)
			grid, err := NewGrid(canvas, canvaser, test.colWidth, test.rowHeight, nil)
			require.NoError(t, err)
			require.Equal(t, test.expectedNum, len(grid.canvases))

			for _, c := range grid.Canvases() {
				require.NotNil(t, c)
				require.Equal(t, test.colWidth, c.Bounds().Dx())
				require.Equal(t, test.rowHeight, c.Bounds().Dy())
			}
		})
	}
}

func TestGrid(t *testing.T) {
	log, err := zap.NewDevelopment()
	require.NoError(t, err)
	canvas := imgcanvas.New(100, 100, log)
	canvaser, err := GetCanvaser(canvas, log)
	require.NoError(t, err)
	require.NotNil(t, canvaser)
	grid, err := NewGrid(canvas, canvaser, 50, 50, log)
	require.NoError(t, err)

	clr := color.RGBA{
		R: 50,
		G: 50,
		B: 50,
		A: 255,
	}

	for _, c := range grid.Canvases() {
		draw.Draw(c, c.Bounds(), image.NewUniform(clr), image.Point{}, draw.Over)
		checkColors(t, c, clr)
	}

	err = grid.DrawToBase()
	require.NoError(t, err)

	checkColors(t, canvas, clr)
}

func checkColors(t *testing.T, canvas board.Canvas, clr color.Color) {
	for x := 0; x < canvas.Bounds().Dx(); x++ {
		for y := 0; y < canvas.Bounds().Dy(); y++ {
			if y == 50 {
				y += 2
			}
			//fmt.Printf("x: %d, y: %d\n", x, y)
			require.Equal(t, clr, canvas.At(x, y), fmt.Sprintf("x: %d, y: %d", x, y))
		}
	}
}
