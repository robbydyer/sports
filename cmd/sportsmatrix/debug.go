package main

import (
	_ "expvar" // Register the expvar handlers
	"net/http"
	_ "net/http/pprof" // Register the pprof handlers
)

type DebugServer struct {
	*http.Server
}

// NewDebugServer provides new debug http server
func NewDebugServer(address string) *DebugServer {
	return &DebugServer{
		&http.Server{
			Addr:    address,
			Handler: http.DefaultServeMux,
		},
	}
}
