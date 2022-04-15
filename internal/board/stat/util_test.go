package statboard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaxedStr(t *testing.T) {
	t.Parallel()
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
			expected: "bi..f",
		},
		{
			name:     "exact",
			in:       "bill",
			len:      4,
			expected: "bill",
		},
		{
			name:     "one less",
			in:       "bill",
			len:      3,
			expected: "bi..",
		},
		{
			name:     "longer",
			in:       "billygoatandgruff",
			len:      8,
			expected: "bill..uff",
		},
		{
			name:     "odd",
			in:       "billygoatandgruff",
			len:      7,
			expected: "bill..ff",
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
