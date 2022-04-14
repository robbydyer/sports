package yahoo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDurationToAPIInterval(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		in       time.Duration
		expected string
	}{
		{
			name:     "1m",
			in:       1 * time.Minute,
			expected: "1m",
		},
		{
			name:     "5m",
			in:       5 * time.Minute,
			expected: "5m",
		},
		{
			name:     "60",
			in:       60 * time.Minute,
			expected: "1h",
		},
		{
			name:     "90",
			in:       90 * time.Minute,
			expected: "90m",
		},
		{
			name:     "1d",
			in:       24 * time.Hour,
			expected: "24h",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.expected, durationToAPIInterval(test.in))
		})
	}
}
