package imageboard

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
)

type jumpRequest struct {
	Name string `json:"name"`
}

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
				_ = os.RemoveAll(filepath.Join(diskCacheDir))
			},
		},
		{
			Path: "/img/diskcachestatus",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				if i.config.UseDiskCache.Load() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
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
		{
			Path: "/img/memcachestatus",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				if i.config.UseMemCache.Load() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
		{
			Path: "/img/status",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				if i.Enabled() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
		{
			Path: "/img/jump",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				i.jumpLock.Lock()
				defer i.jumpLock.Unlock()

				// Clear the channel
				select {
				case <-i.jumpTo:
				default:
				}

				d := json.NewDecoder(req.Body)
				var j *jumpRequest
				if err := d.Decode(&j); err != nil {
					i.log.Error("failed to process /api/img/jump request",
						zap.Error(err),
					)
					http.Error(w, "failed to process /api/img/jump request", http.StatusBadRequest)
					return
				}
				select {
				case i.jumpTo <- j.Name:
				case <-time.After(5 * time.Second):
					i.log.Error("timed out waiting to jump image")
					http.Error(w, "timed out waiting to jump iamge", http.StatusRequestTimeout)
					return
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if i.jumpTo != nil {
					if err := i.jumper(ctx, i.Name()); err != nil {
						i.log.Error("failed to jump to image board",
							zap.Error(err),
							zap.String("file name", j.Name),
						)
						http.Error(w, "failed to jump to image board", http.StatusInternalServerError)
						return
					}
				}
			},
		},
	}, nil
}
