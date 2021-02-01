package imageboard

import (
	"net/http"

	"github.com/robbydyer/sports/pkg/board"
)

// GetHTTPHandlers ...
func (i *ImageBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	disable := &board.HTTPHandler{
		Path: "/img/disable",
		Handler: func(http.ResponseWriter, *http.Request) {
			i.log.Info("disabling image board")
			i.config.Enabled = false
		},
	}
	enable := &board.HTTPHandler{
		Path: "/img/disable",
		Handler: func(http.ResponseWriter, *http.Request) {
			i.log.Info("enabling image board")
			i.config.Enabled = true
		},
	}
	cache := &board.HTTPHandler{
		Path: "/img/clearcache",
		Handler: func(http.ResponseWriter, *http.Request) {
			i.log.Info("clearing image board cache")
			i.cacheClear()
		},
	}

	return []*board.HTTPHandler{
		disable,
		enable,
		cache,
	}, nil
}
