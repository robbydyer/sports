package statboard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaxedStr(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		len      int
		expected string
	}{
		{
			name:     "short",
			in:       "short",
			len:      10,
			expected: "short",
		},
		{
			name:     "long",
			in:       "billygoatandgruff",
			len:      4,
			expected: "b..f",
		},
		{
			name:     "longer",
			in:       "billygoatandgruff",
			len:      8,
			expected: "bil..uff",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.expected, maxedStr(test.in, test.len))
		})
	}
}
