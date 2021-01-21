package there

import "github.com/gobuffalo/here"

var cache = here.New()

// Dir attempts to gather info for the requested directory.
// Results are cached globally inside the package.
func Dir(p string) (here.Info, error) {
	return cache.Dir(p)
}

// Package attempts to gather info for the requested package.
//
// From the `go help list` docs:
//	The -find flag causes list to identify the named packages but not
//	resolve their dependencies: the Imports and Deps lists will be empty.
//
// A workaround for this issue is to use the `Dir` field in the
// returned `Info` value and pass it to the `Dir(string) (Info, error)`
// function to return the complete data.
// Results are cached globally inside the package.
func Package(p string) (here.Info, error) {
	return cache.Package(p)
}

// Results are cached globally inside the package.
// Current returns the Info representing the current Go module
func Current() (here.Info, error) {
	return cache.Current()
}
