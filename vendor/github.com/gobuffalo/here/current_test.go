package here

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Current(t *testing.T) {
	r := require.New(t)

	h := New()

	info, err := h.Current()
	r.NoError(err)
	sanityCheck(t, info)

}
