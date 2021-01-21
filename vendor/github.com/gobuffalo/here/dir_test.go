package here

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Dir(t *testing.T) {
	r := require.New(t)

	root, err := os.Getwd()
	r.NoError(err)

	h := New()

	info, err := h.Dir(root)
	r.NoError(err)
	sanityCheck(t, info)

	cmd := filepath.Join(root, "cmd")
	info, err = h.Dir(cmd)
	r.NoError(err)
	r.NotZero(info)

	r.Equal(cmd, info.Dir)
	r.Equal(filepath.Join(root, "go.mod"), info.Module.GoMod)
	r.Equal("", info.ImportPath)
	r.Equal("cmd", info.Name)
	r.Empty(info.GoFiles)
	r.Empty(info.Imports)

	cmd = filepath.Join(cmd, "here")
	info, err = h.Dir(cmd)
	r.NoError(err)
	r.NotZero(info)

	r.Equal(cmd, info.Dir)
	r.Equal(filepath.Join(root, "go.mod"), info.Module.GoMod)
	r.Equal("github.com/gobuffalo/here/cmd/here", info.ImportPath)
	r.Equal("main", info.Name)
	r.NotEmpty(info.GoFiles)
	r.NotEmpty(info.Imports)

	cmd = filepath.Join(cmd, "main.go")
	info, err = h.Dir(cmd)
	r.NoError(err)
	r.NotZero(info)

	r.Equal(filepath.Dir(cmd), info.Dir)
	r.Equal(filepath.Join(root, "go.mod"), info.Module.GoMod)
	r.Equal("github.com/gobuffalo/here/cmd/here", info.ImportPath)
	r.Equal("main", info.Name)
	r.NotEmpty(info.GoFiles)
	r.NotEmpty(info.Imports)

	info, err = h.Dir("/unknown")
	r.Error(err)
	r.Zero(info)
}
