package here_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gobuffalo/here"
	"github.com/stretchr/testify/require"
)

func Test_Info_Parse(t *testing.T) {
	const name = "/public/index.html"

	r := require.New(t)

	info, err := here.Current()
	r.NoError(err)

	ip := info.ImportPath
	ip2 := "another/app"

	table := []struct {
		in  string
		exp here.Path
		err bool
	}{
		{in: name, exp: here.Path{Pkg: ip, Name: name}},
		{in: "", exp: here.Path{Pkg: ip, Name: "/"}},
		{in: "/", exp: here.Path{Pkg: ip, Name: "/"}},
		{in: filepath.Join(info.Dir, name), exp: here.Path{Pkg: ip, Name: name}},
		{in: ":" + name, exp: here.Path{Pkg: ip, Name: name}},
		{in: ip + ":" + name, exp: here.Path{Pkg: ip, Name: name}},
		{in: ip, exp: here.Path{Pkg: ip, Name: "/"}},
		{in: ":", exp: here.Path{Pkg: ip, Name: "/"}},
		{in: ip2 + ":" + name, exp: here.Path{Pkg: ip2, Name: name}},
		{in: ip2, exp: here.Path{Pkg: ip2, Name: "/"}},
		{in: ip2 + ":", exp: here.Path{Pkg: ip2, Name: "/"}},
		{in: filepath.Join(info.Dir, "public"), exp: here.Path{Pkg: ip, Name: "/public"}},
	}

	for _, tt := range table {
		for _, in := range []string{tt.in, strings.ReplaceAll(tt.in, "/", "\\")} {
			t.Run(in, func(st *testing.T) {
				r := require.New(st)

				pt, err := info.Parse(in)

				if tt.err {
					r.Error(err)
					return
				}
				r.NoError(err)

				r.Equal(tt.exp, pt)

			})
		}
	}
}
