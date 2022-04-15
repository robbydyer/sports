package sportboard

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func TestPreviousScore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		max  int32
	}{
		{
			name: "three",
			max:  3,
		},
		{
			name: "none",
			max:  0,
		},
		{
			name: "one",
			max:  1,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			team := &previousTeam{
				init:       atomic.NewBool(false),
				previous:   atomic.NewInt32(0),
				repeats:    atomic.NewInt32(0),
				maxRepeats: test.max,
			}

			// First one should init
			require.False(t, team.hasScored(1))

			require.True(t, team.hasScored(2))

			count := int32(0)
			for i := 0; i < int(test.max)+1; i++ {
				if i < int(test.max) {
					require.True(t, team.hasScored(2))
					count++
					continue
				}
				require.False(t, team.hasScored(2))
			}

			require.Equal(t, count, test.max)
		})
	}
}
