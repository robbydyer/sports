package rgbrender

import (
	"testing"
	"time"

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
