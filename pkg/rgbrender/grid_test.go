package rgbrender

import (
	"testing"

	"github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/stretchr/testify/require"
)

func TestGrid(t *testing.T) {
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

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			mtrx := rgbmatrix.NewConsoleMatrix(test.canvasWidth, test.canvasHeight, nil, nil)
			canvas := rgbmatrix.NewCanvas(mtrx)
			grid, err := NewGrid(canvas, test.colWidth, test.rowHeight, nil)
			require.NoError(t, err)
			require.Equal(t, test.expectedNum, len(grid.cells))
		})
	}
}
