package here

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Module_IsZero(t *testing.T) {
	r := require.New(t)

	var m Module
	r.True(m.IsZero())

	m.Path = "foo"
	r.False(m.IsZero())
}
