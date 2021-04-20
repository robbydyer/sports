package statboard

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
)

// GetHTTPHandlers ...
func (s *StatBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return []*board.HTTPHandler{
		{
			Path: fmt.Sprintf("/%s/stats/disable", s.api.HTTPPathPrefix()),
			Handler: func(wrter http.ResponseWriter, req *http.Request) {
				s.log.Info("disabling board", zap.String("board", s.Name()))
				s.Disable()
			},
		},
		{
			Path: fmt.Sprintf("/%s/stats/enable", s.api.HTTPPathPrefix()),
			Handler: func(wrter http.ResponseWriter, req *http.Request) {
				s.log.Info("enabling board", zap.String("board", s.Name()))
				s.Enable()
			},
		},
		{
			Path: fmt.Sprintf("/%s/stats/status", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Debug("get board status", zap.String("board", s.Name()))
				w.Header().Set("Content-Type", "text/plain")
				if s.Enabled() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
		{
			Path: fmt.Sprintf("/%s/stats/scrolloff", s.api.HTTPPathPrefix()),
			Handler: func(wrter http.ResponseWriter, req *http.Request) {
				s.log.Info("disabling board scroll", zap.String("board", s.Name()))
				s.config.ScrollMode.Store(false)
			},
		},
		{
			Path: fmt.Sprintf("/%s/stats/scrollon", s.api.HTTPPathPrefix()),
			Handler: func(wrter http.ResponseWriter, req *http.Request) {
				s.log.Info("enabling board scroll", zap.String("board", s.Name()))
				s.config.ScrollMode.Store(true)
			},
		},
		{
			Path: fmt.Sprintf("/%s/stats/scrollstatus", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Debug("get board status", zap.String("board", s.Name()))
				w.Header().Set("Content-Type", "text/plain")
				if s.Enabled() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
	}, nil
}
