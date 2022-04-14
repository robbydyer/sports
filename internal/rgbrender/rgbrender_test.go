package rgbrender

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAlignPosition(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		bounds   image.Rectangle
		align    Align
		sizeX    int
		sizeY    int
		expected image.Rectangle
	}{
		{
			name:  "centercenter",
			align: CenterCenter,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 5,
			sizeY: 5,
			expected: image.Rectangle{
				Min: image.Point{
					X: 2,
					Y: 2,
				},
				Max: image.Point{
					X: 6,
					Y: 6,
				},
			},
		},
		{
			name:  "centercenter larger than bounds",
			align: CenterCenter,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 20,
			sizeY: 20,
			expected: image.Rectangle{
				Min: image.Point{
					X: -5,
					Y: -5,
				},
				Max: image.Point{
					X: 14,
					Y: 14,
				},
			},
		},
		{
			name:  "centertop",
			align: CenterTop,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 5,
			sizeY: 5,
			expected: image.Rectangle{
				Min: image.Point{
					X: 2,
					Y: 0,
				},
				Max: image.Point{
					X: 6,
					Y: 4,
				},
			},
		},
		{
			name:  "centertop larger than bounds",
			align: CenterTop,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 20,
			sizeY: 20,
			expected: image.Rectangle{
				Min: image.Point{
					X: -5,
					Y: -10,
				},
				Max: image.Point{
					X: 14,
					Y: 9,
				},
			},
		},
		{
			name:  "centerbottom",
			align: CenterBottom,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 5,
			sizeY: 5,
			expected: image.Rectangle{
				Min: image.Point{
					X: 2,
					Y: 5,
				},
				Max: image.Point{
					X: 6,
					Y: 9,
				},
			},
		},
		{
			name:  "centerbottom larger than bounds",
			align: CenterBottom,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 20,
			sizeY: 20,
			expected: image.Rectangle{
				Min: image.Point{
					X: -5,
					Y: 0,
				},
				Max: image.Point{
					X: 14,
					Y: 19,
				},
			},
		},
		{
			name:  "righttop",
			align: RightTop,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 5,
			sizeY: 5,
			expected: image.Rectangle{
				Min: image.Point{
					X: 5,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 4,
				},
			},
		},
		{
			name:  "righttop larger than bounds",
			align: RightTop,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 20,
			sizeY: 20,
			expected: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: -10,
				},
				Max: image.Point{
					X: 19,
					Y: 9,
				},
			},
		},
		{
			name:  "rightcenter",
			align: RightCenter,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 5,
			sizeY: 5,
			expected: image.Rectangle{
				Min: image.Point{
					X: 5,
					Y: 2,
				},
				Max: image.Point{
					X: 9,
					Y: 6,
				},
			},
		},
		{
			name:  "rightcenter larger than bounds",
			align: RightCenter,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 20,
			sizeY: 20,
			expected: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: -5,
				},
				Max: image.Point{
					X: 19,
					Y: 14,
				},
			},
		},
		{
			name:  "rightbottom",
			align: RightBottom,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 5,
			sizeY: 5,
			expected: image.Rectangle{
				Min: image.Point{
					X: 5,
					Y: 5,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
		},
		{
			name:  "rightbottom larger than bounds",
			align: RightBottom,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 20,
			sizeY: 20,
			expected: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 19,
					Y: 19,
				},
			},
		},
		{
			name:  "lefttop",
			align: LeftTop,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 5,
			sizeY: 5,
			expected: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 4,
					Y: 4,
				},
			},
		},
		{
			name:  "lefttop larger than bounds",
			align: LeftTop,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 20,
			sizeY: 20,
			expected: image.Rectangle{
				Min: image.Point{
					X: -10,
					Y: -10,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
		},
		{
			name:  "leftcenter",
			align: LeftCenter,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 5,
			sizeY: 5,
			expected: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 2,
				},
				Max: image.Point{
					X: 4,
					Y: 6,
				},
			},
		},
		{
			name:  "leftcenter larger than bounds",
			align: LeftCenter,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 20,
			sizeY: 20,
			expected: image.Rectangle{
				Min: image.Point{
					X: -10,
					Y: -5,
				},
				Max: image.Point{
					X: 9,
					Y: 14,
				},
			},
		},
		{
			name:  "leftbottom",
			align: LeftBottom,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 5,
			sizeY: 5,
			expected: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 5,
				},
				Max: image.Point{
					X: 4,
					Y: 9,
				},
			},
		},
		{
			name:  "leftbottom larger than bounds",
			align: LeftBottom,
			bounds: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 9,
				},
			},
			sizeX: 20,
			sizeY: 20,
			expected: image.Rectangle{
				Min: image.Point{
					X: -10,
					Y: 0,
				},
				Max: image.Point{
					X: 9,
					Y: 19,
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual, err := AlignPosition(test.align, test.bounds, test.sizeX, test.sizeY)
			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		})
	}
}

func TestZoomImageSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		img       image.Image
		zoom      float64
		expectedX int
		expectedY int
	}{
		{
			name: "full size",
			img: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 5,
					Y: 5,
				},
			},
			zoom:      1,
			expectedX: 6,
			expectedY: 6,
		},
		{
			name: "half square, even",
			img: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 5,
					Y: 5,
				},
			},
			zoom:      0.5,
			expectedX: 3,
			expectedY: 3,
		},
		{
			name: "half square, odd",
			img: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 4,
					Y: 4,
				},
			},
			zoom:      0.5,
			expectedX: 3,
			expectedY: 3,
		},
		{
			name: "half rectangle",
			img: image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 0,
				},
				Max: image.Point{
					X: 5,
					Y: 11,
				},
			},
			zoom:      0.5,
			expectedX: 3,
			expectedY: 6,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actualX, actualY := ZoomImageSize(test.img, test.zoom)
			require.Equal(t, test.expectedX, actualX)
			require.Equal(t, test.expectedY, actualY)
		})
	}
}

func TestNegativeImagePoint(t *testing.T) {
	i := image.NewRGBA(image.Rect(-10, -10, 10, 10))

	i.Set(-5, -5, color.Gray16{0xffff})

	require.Equal(t, i.At(0, 0), color.RGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x0}, "Default to black on zero point")
	require.NotEqual(t, i.At(-5, -5), color.RGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x0}, "Changing color at negative point should work")
}

func TestZeroedBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		in       image.Rectangle
		expected image.Rectangle
	}{
		{
			name:     "no change",
			in:       image.Rect(0, 0, 1, 1),
			expected: image.Rect(0, 0, 1, 1),
		},
		{
			name:     "changed",
			in:       image.Rect(-1, -1, 2, 2),
			expected: image.Rect(0, 0, 1, 1),
		},
		{
			name:     "changed large",
			in:       image.Rect(-10, -10, 20, 20),
			expected: image.Rect(0, 0, 10, 10),
		},
		{
			name:     "changed rectangle",
			in:       image.Rect(-10, -10, 74, 42),
			expected: image.Rect(0, 0, 64, 32),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.expected, ZeroedBounds(test.in))
		})
	}
}
