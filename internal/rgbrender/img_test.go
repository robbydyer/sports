package rgbrender

import (
	"image"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShiftedSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		x        int
		y        int
		bounds   image.Rectangle
		expected image.Rectangle
	}{
		{
			name:     "no shift",
			x:        0,
			y:        0,
			bounds:   image.Rect(0, 0, 9, 9),
			expected: image.Rect(0, 0, 9, 9),
		},
		{
			name:     "negative shift",
			x:        -2,
			y:        -2,
			bounds:   image.Rect(0, 0, 9, 9),
			expected: image.Rect(-2, -2, 7, 7),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := ShiftedSize(test.x, test.y, test.bounds)

			require.Equal(t, test.expected.Min.X, actual.Min.X)
			require.Equal(t, test.expected.Min.Y, actual.Min.Y)
			require.Equal(t, test.expected.Max.X, actual.Max.X)
			require.Equal(t, test.expected.Max.Y, actual.Max.Y)
		})
	}
}
