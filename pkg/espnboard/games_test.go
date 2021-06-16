package espnboard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractOverUnder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		in          string
		expected    []string
		expectedErr string
	}{
		{
			name: "roll tide",
			in:   "ALA -10.0",
			expected: []string{
				"ALA",
				"-10.0",
			},
			expectedErr: "",
		},
		{
			name:        "invalid",
			in:          "123 NOT",
			expected:    []string{},
			expectedErr: "no match",
		},
		{
			name: "positive",
			in:   "ALA 10.0",
			expected: []string{
				"ALA",
				"10.0",
			},
			expectedErr: "",
		},
		{
			name: "int",
			in:   "ALA -10",
			expected: []string{
				"ALA",
				"-10",
			},
			expectedErr: "",
		},
		{
			name: "positive int",
			in:   "ALA 10",
			expected: []string{
				"ALA",
				"10",
			},
			expectedErr: "",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			team, spread, err := extractOverUnder(test.in)
			if test.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expected[0], team)
			require.Equal(t, test.expected[1], spread)
		})
	}
}
