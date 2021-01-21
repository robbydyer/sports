package here

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_nonGoDirRx(t *testing.T) {
	r := require.New(t)
	r.False(nonGoDirRx.MatchString(""))
	r.False(nonGoDirRx.MatchString("hello"))

	table := []string{
		"go: cannot find main module; see 'go help modules'",
		"go help modules",
		"go: ",
		"build .:",
		"no Go files",
		"can't load package",
	}

	for _, tt := range table {
		t.Run(tt, func(st *testing.T) {
			r := require.New(st)

			b := nonGoDirRx.MatchString(tt)
			r.True(b)

		})
	}

}

func sanityCheck(t *testing.T, info Info) {
	t.Helper()
	r := require.New(t)

	root, err := os.Getwd()
	r.NoError(err)
	r.NotZero(info)
	r.Equal(root, info.Dir)
	r.Equal(filepath.Join(root, "go.mod"), info.Module.GoMod)
	r.Equal("github.com/gobuffalo/here", info.ImportPath)
	r.Equal("here", info.Name)
}
