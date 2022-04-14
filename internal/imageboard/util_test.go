package imageboard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilenameCompare(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "full match",
			a:        "/src/tmp/foo.png",
			b:        "/src/tmp/foo.png",
			expected: true,
		},
		{
			name:     "relative match",
			a:        "/src/tmp/foo.png",
			b:        "foo.png",
			expected: true,
		},
		{
			name:     "relative match reverse",
			a:        "foo.png",
			b:        "/src/tmp/foo.png",
			expected: true,
		},
		{
			name:     "different dir",
			a:        "/src/tmp/foo.png",
			b:        "/src/tmp2/foo.png",
			expected: false,
		},
		{
			name:     "full not full",
			a:        "/src/tmp/foo.png",
			b:        "tmp/foo.png",
			expected: true,
		},
		{
			name:     "not match",
			a:        "/src/tmp/foo.png",
			b:        "/src/tmp/notfoo.png",
			expected: false,
		},
		{
			name:     "not match 2",
			a:        "/src/tmp/foo.png",
			b:        "/src/tmp/foo.gif",
			expected: false,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, filenameCompare(test.a, test.b))
		})
	}
}

func TestReverseStrs(t *testing.T) {
	tests := []struct {
		name     string
		in       []string
		expected []string
	}{
		{
			name:     "good",
			in:       []string{"hello", "world"},
			expected: []string{"world", "hello"},
		},
		{
			name:     "good 3",
			in:       []string{"tmp", "src", "foo"},
			expected: []string{"foo", "src", "tmp"},
		},
		{
			name:     "good 4",
			in:       []string{"tmp", "src", "foo", "bar"},
			expected: []string{"bar", "foo", "src", "tmp"},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, reverseStrs(test.in))
		})
	}
}
