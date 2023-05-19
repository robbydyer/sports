package statboard

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
)

// GetHTTPHandlers ...
func (s *StatBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return []*board.HTTPHandler{
		{
			Path: fmt.Sprintf("/%s/stats/disable", s.api.HTTPPathPrefix()),
			Handler: func(wrter http.ResponseWriter, req *http.Request) {
				s.log.Info("disabling board", zap.String("board", s.Name()))
				s.Enabler().Disable()
			},
		},
		{
			Path: fmt.Sprintf("/%s/stats/enable", s.api.HTTPPathPrefix()),
			Handler: func(wrter http.ResponseWriter, req *http.Request) {
				s.log.Info("enabling board", zap.String("board", s.Name()))
				s.Enabler().Enable()
			},
		},
		{
			Path: fmt.Sprintf("/%s/stats/status", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Debug("get board status", zap.String("board", s.Name()))
				w.Header().Set("Content-Type", "text/plain")
				if s.Enabler().Enabled() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
	}, nil
}
