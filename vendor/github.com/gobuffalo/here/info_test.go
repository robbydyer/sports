package here

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Info_IsZero(t *testing.T) {
	r := require.New(t)

	var i Info
	r.True(i.IsZero())

	i.Name = "foo"
	r.False(i.IsZero())
}
