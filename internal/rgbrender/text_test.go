package rgbrender

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBreakText(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		max      int
		expected []string
	}{
		{
			name:     "no wrap",
			in:       "foo bar",
			max:      7,
			expected: []string{"foo bar"},
		},
		{
			name: "one wrap",
			in:   "foo bar",
			max:  3,
			expected: []string{
				"foo",
				"bar",
			},
		},
		{
			name: "multi wrap",
			in:   "foo bar baz",
			max:  3,
			expected: []string{
				"foo",
				"bar",
				"baz",
			},
		},
		{
			name: "whitespace",
			in:   " foo  bar  baz  ",
			max:  3,
			expected: []string{
				"foo",
				"bar",
				"baz",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, breakText(test.max, test.in))
		})
	}
}
