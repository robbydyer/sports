package rgbrender

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/imgcanvas"
)

func TestSetForegroundPriority(t *testing.T) {
	t.Parallel()
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
			l, err := NewLayerDrawer(1*time.Second, nil)
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
	t.Parallel()
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
					priority: BackgroundPriority + 1,
				},
				{
					priority: BackgroundPriority + 3,
				},
			},
		},
		{
			name:     "like render without rank",
			expected: []int{0, 2, 3},
			layers: []*Layer{
				{
					priority: ForegroundPriority,
				},
				{
					priority: BackgroundPriority,
				},
				{
					priority: BackgroundPriority + 2,
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			l, err := NewLayerDrawer(1*time.Second, nil)
			require.NoError(t, err)
			for _, layer := range test.layers {
				l.AddLayer(layer.priority, layer)
			}
			require.Equal(t, test.expected, l.priorities())
		})
	}
}

func TestRender(t *testing.T) {
	layers, err := NewLayerDrawer(60*time.Second, nil)
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

	require.NoError(t, layers.Draw(context.Background(), imgcanvas.New(1, 1, nil)))
	require.True(t, layer1)
	require.True(t, layer2)
	require.Equal(t, []string{"layer", "text"}, renderedLayers)
	require.NotEqual(t, []string{"text", "layer"}, renderedLayers)
}

func TestBadPrepare(t *testing.T) {
	layers, err := NewLayerDrawer(60*time.Second, nil)
	require.NoError(t, err)

	layers.AddLayer(BackgroundPriority, NewLayer(
		func(ctx context.Context) (image.Image, error) {
			return nil, fmt.Errorf("prep failed")
		},
		nil,
	))

	err = layers.Prepare(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "prep failed")
}

func TestBadRender(t *testing.T) {
	layers, err := NewLayerDrawer(60*time.Second, nil)
	require.NoError(t, err)

	layers.AddLayer(BackgroundPriority, NewLayer(
		func(ctx context.Context) (image.Image, error) {
			return image.NewUniform(color.White), nil
		},
		func(canvas board.Canvas, i image.Image) error {
			return fmt.Errorf("render failed")
		},
	))

	err = layers.Draw(context.Background(), imgcanvas.New(1, 2, nil))
	require.Error(t, err)
	require.Contains(t, err.Error(), "render failed")
}
