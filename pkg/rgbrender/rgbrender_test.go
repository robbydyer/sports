package rgbrender

import (
	"image"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAlignPosition(t *testing.T) {
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
