package here

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Package(t *testing.T) {
	r := require.New(t)

	h := New()

	info, err := h.Package("github.com/gobuffalo/here")
	r.NoError(err)
	sanityCheck(t, info)

	info, err = h.Package("github.com/gobuffalo/buffalo")
	r.NoError(err)
	r.NotZero(info)
	r.Equal("github.com/gobuffalo/buffalo", info.ImportPath)

	_, err = h.Package("")
	r.Error(err)

	_, err = h.Package(".")
	r.Error(err)

	_, err = h.Package("./cmd")
	r.Error(err)

}
