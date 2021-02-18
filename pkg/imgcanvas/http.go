package imgcanvas

import (
	_ "embed"
	"net/http"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
)

//go:embed assets/loading.gif
var loading []byte

// GetHTTPHandlers ...
func (i *ImgCanvas) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	enable := &board.HTTPHandler{
		Path: "/api/imgcanvas/enable",
		Handler: func(w http.ResponseWriter, req *http.Request) {
			i.log.Info("enabling ImgCanvas")
			i.Enable()
		},
	}
	disable := &board.HTTPHandler{
		Path: "/api/imgcanvas/disable",
		Handler: func(w http.ResponseWriter, req *http.Request) {
			i.log.Info("disabling ImgCanvas")
			i.Disable()
		},
	}
	render := &board.HTTPHandler{
		Path: "/api/imgcanvas/board",
		Handler: func(w http.ResponseWriter, req *http.Request) {
			i.Enable()

			i.log.Debug("getting image for web board")

			if len(i.lastPng) == 0 {
				i.log.Warn("web board is not ready yet, loading")
				w.Header().Set("Content-Type", "image/gif")
				if _, err := w.Write(loading); err != nil {
					i.log.Error("failed to copy loading.gif", zap.Error(err))
				}
				return
			}

			w.Header().Set("Content-Type", "image/png")

			i.Lock()
			defer i.Unlock()
			if _, err := w.Write(i.lastPng); err != nil {
				i.log.Error("failed to copy png for /api/imgcanvas/board", zap.Error(err))
			}
		},
	}

	return []*board.HTTPHandler{
		enable,
		disable,
		render,
	}, nil
}
