package rgbrender

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/imgcanvas"
	"github.com/stretchr/testify/require"
)

func TestSetForegroundPriority(t *testing.T) {
	tests := []struct {
		name     string
		layers   []*Layer
		expected int
	}{
		{
			name:     "default single layer",
			expected: 1,
			layers: []*Layer{
				{
					priority: ForegroundPriority,
				},
			},
		},
		{
			name:     "default with foreground",
			expected: 1,
			layers: []*Layer{
				{
					priority: ForegroundPriority,
				},
				{
					priority: BackgroundPriority,
				},
			},
		},
		{
			name:     "multiple priorities",
			expected: 4,
			layers: []*Layer{
				{
					priority: ForegroundPriority,
				},
				{
					priority: BackgroundPriority,
				},
				{
					priority: 1,
				},
				{
					priority: 3,
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			l, err := NewLayerRenderer(1*time.Second, nil)
			require.NoError(t, err)
			for _, layer := range test.layers {
				l.AddLayer(layer.priority, layer)
			}
			l.setForegroundPriority()
			require.Equal(t, test.expected, l.maxLayer)
		})
	}
}

func TestPriorities(t *testing.T) {
	tests := []struct {
		name     string
		layers   []*Layer
		expected []int
	}{
		{
			name:     "default single layer",
			expected: []int{1},
			layers: []*Layer{
				{
					priority: ForegroundPriority,
				},
			},
		},
		{
			name:     "default with foreground",
			expected: []int{0, 1},
			layers: []*Layer{
				{
					priority: ForegroundPriority,
				},
				{
					priority: BackgroundPriority,
				},
			},
		},
		{
			name:     "multiple priorities",
			expected: []int{0, 1, 3, 4},
			layers: []*Layer{
				{
					priority: ForegroundPriority,
				},
				{
					priority: BackgroundPriority,
				},
				{
					priority: 1,
				},
				{
					priority: 3,
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			l, err := NewLayerRenderer(1*time.Second, nil)
			require.NoError(t, err)
			for _, layer := range test.layers {
				l.AddLayer(layer.priority, layer)
			}
			require.Equal(t, test.expected, l.priorities())
		})
	}
}

func TestRender(t *testing.T) {
	layers, err := NewLayerRenderer(60*time.Second, nil)
	require.NoError(t, err)

	renderedLayers := []string{}

	i := image.NewUniform(color.White)
	layer1 := false
	layers.AddLayer(BackgroundPriority, NewLayer(
		func(ctx context.Context) (image.Image, error) {
			return i, nil
		},
		func(canvas board.Canvas, img image.Image) error {
			defer func() { renderedLayers = append(renderedLayers, "layer") }()
			if img == i {
				layer1 = true
				return nil
			}
			return fmt.Errorf("wrong image")
		},
	))

	writer, err := DefaultTextWriter()
	require.NoError(t, err)

	layer2 := false
	layers.AddTextLayer(ForegroundPriority, NewTextLayer(
		func(ctx context.Context) (*TextWriter, []string, error) {
			return writer, []string{"hello"}, nil
		},
		func(canvas board.Canvas, writer *TextWriter, text []string) error {
			defer func() { renderedLayers = append(renderedLayers, "text") }()
			require.NotNil(t, writer)
			require.Equal(t, []string{"hello"}, text)
			layer2 = true
			return nil
		},
	))

	require.NoError(t, layers.Render(context.Background(), imgcanvas.New(1, 1, nil)))
	require.True(t, layer1)
	require.True(t, layer2)
	require.Equal(t, []string{"layer", "text"}, renderedLayers)
	require.NotEqual(t, []string{"text", "layer"}, renderedLayers)
}
