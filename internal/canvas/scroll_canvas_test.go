package canvas

import (
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/robbydyer/sports/internal/matrix"
)

func TestScrollCanvas(t *testing.T) {
	l := zaptest.NewLogger(t)
	m := matrix.NewConsoleMatrix(64, 32, ioutil.Discard, l)
	c, err := NewScrollCanvas(m, l)
	require.NoError(t, err)

	defaultPad := 64 + int(float64(64)*0.25)

	require.Equal(t, image.Rect(defaultPad*-1, defaultPad*-1, 64+defaultPad, 32+defaultPad), c.Bounds())
	require.Equal(t, 64, c.w)
	require.Equal(t, 32, c.h)
}

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
			pt:       image.Pt(2, 5),
			expected: 2,
		},
	}

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
