package imageboard

import (
	"net/http"

	"github.com/robbydyer/sports/pkg/board"
)

// GetHTTPHandlers ...
func (i *ImageBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return []*board.HTTPHandler{
		{
			Path: "/img/disable",
			Handler: func(http.ResponseWriter, *http.Request) {
				i.log.Info("disabling image board")
				i.Disable()
			},
		},
		{
			Path: "/img/enable",
			Handler: func(http.ResponseWriter, *http.Request) {
				i.log.Info("enabling image board")
				i.Enable()
			},
		},
		{
			Path: "/img/clearcache",
			Handler: func(http.ResponseWriter, *http.Request) {
				i.log.Info("clearing image board cache")
				i.cacheClear()
			},
		},
		{
			Path: "/img/enablediskcache",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				i.log.Info("enabled disk cache for image board")
				i.config.UseDiskCache.Store(true)
			},
		},
		{
			Path: "/img/disablediskcache",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				i.log.Info("disabling disk cache for image board")
				i.config.UseDiskCache.Store(false)
			},
		},
		{
			Path: "/img/enablememcache",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				i.log.Info("enabled memory cache for image board")
				i.config.UseMemCache.Store(true)
			},
		},
		{
			Path: "/img/disablememcache",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				i.log.Info("disabling memory cache for image board")
				i.config.UseMemCache.Store(false)
				i.cacheClear()
			},
		},
	}, nil
}
