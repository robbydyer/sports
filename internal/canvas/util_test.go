package canvas

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFirstNonBlankY(t *testing.T) {
	tests := []struct {
		name     string
		pt       image.Point
		expected int
	}{
		{
			name:     "first line",
			pt:       image.Pt(2, 0),
			expected: 0,
		},
		{
			name:     "last line",
			pt:       image.Pt(2, 10),
			expected: 10,
		},
		{
			name:     "middle",
			pt:       image.Pt(2, 5),
			expected: 5,
		},
	}

	t.Parallel()
	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			img := image.NewRGBA(image.Rect(0, 0, 11, 11))
			draw.Draw(img, img.Bounds(), image.NewUniform(black), image.Point{}, draw.Over)
			img.Set(test.pt.X, test.pt.Y, color.White)

			require.Equal(t, test.expected, firstNonBlankY(img))
		})
	}
}

func TestFirstNonBlankX(t *testing.T) {
	tests := []struct {
		name     string
		pt       image.Point
		expected int
	}{
		{
			name:     "first line",
			pt:       image.Pt(2, 0),
			expected: 2,
		},
		{
			name:     "last line",
			pt:       image.Pt(2, 10),
			expected: 2,
		},
		{
			name:     "middle",
			pt:       image.Pt(2, 5),
			expected: 2,
		},
	}

	t.Parallel()
	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			img := image.NewRGBA(image.Rect(0, 0, 11, 11))
			draw.Draw(img, img.Bounds(), image.NewUniform(black), image.Point{}, draw.Over)
			img.Set(test.pt.X, test.pt.Y, color.White)

			require.Equal(t, test.expected, firstNonBlankX(img))
		})
	}
}

func TestLastNonBlankY(t *testing.T) {
	tests := []struct {
		name     string
		pt       image.Point
		expected int
	}{
		{
			name:     "first line",
			pt:       image.Pt(0, 0),
			expected: 0,
		},
		{
			name:     "last line",
			pt:       image.Pt(2, 10),
			expected: 10,
		},
		{
			name:     "middle",
			pt:       image.Pt(2, 5),
			expected: 5,
		},
	}

	t.Parallel()
	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			img := image.NewRGBA(image.Rect(0, 0, 10, 10))
			draw.Draw(img, img.Bounds(), image.NewUniform(black), image.Point{}, draw.Over)
			img.Set(test.pt.X, test.pt.Y, color.White)

			require.Equal(t, test.expected, lastNonBlankY(img))
		})
	}
}

func TestLastNonBlankX(t *testing.T) {
	tests := []struct {
		name     string
		pt       image.Point
		expected int
	}{
		{
			name:     "first line",
			pt:       image.Pt(0, 0),
			expected: 0,
		},
		{
			name:     "last line",
			pt:       image.Pt(2, 9),
			expected: 2,
		},
		{
			name:     "middle",
			pt:       image.Pt(5, 5),
			expected: 5,
		},
		{
			name:     "last X",
			pt:       image.Pt(10, 5),
			expected: 10,
		},
	}

	t.Parallel()
	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			img := image.NewRGBA(image.Rect(0, 0, 10, 10))
			draw.Draw(img, img.Bounds(), image.NewUniform(black), image.Point{}, draw.Over)
			img.Set(test.pt.X, test.pt.Y, color.White)

			require.Equal(t, test.expected, lastNonBlankX(img))
		})
	}
}
