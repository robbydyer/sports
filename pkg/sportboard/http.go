package sportboard

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
)

// GetHTTPHandlers ...
func (s *SportBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	return []*board.HTTPHandler{
		{
			Path: fmt.Sprintf("/%s/hidefavoritescore", s.api.HTTPPathPrefix()),
			Handler: func(http.ResponseWriter, *http.Request) {
				s.log.Info("hiding favorite team scores")
				s.config.HideFavoriteScore.Store(true)
			},
		},
		{
			Path: fmt.Sprintf("/%s/showfavoritescore", s.api.HTTPPathPrefix()),
			Handler: func(http.ResponseWriter, *http.Request) {
				s.log.Info("showing favorite team scores")
				s.config.HideFavoriteScore.Store(false)
			},
		},
		{
			Path: fmt.Sprintf("/%s/favoritescorestatus", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				if s.config.HideFavoriteScore.Load() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
		{
			Path: fmt.Sprintf("/%s/favoritesticky", s.api.HTTPPathPrefix()),
			Handler: func(wrter http.ResponseWriter, req *http.Request) {
				s.log.Info("setting favorite teams to sticky")
				s.config.FavoriteSticky.Store(true)
			},
		},
		{
			Path: fmt.Sprintf("/%s/favoriteunstick", s.api.HTTPPathPrefix()),
			Handler: func(wrter http.ResponseWriter, req *http.Request) {
				s.log.Info("setting favorite teams to not stick")
				s.config.FavoriteSticky.Store(false)
			},
		},
		{
			Path: fmt.Sprintf("/%s/favoritestickystatus", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				if s.config.FavoriteSticky.Load() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
		{
			Path: fmt.Sprintf("/%s/disable", s.api.HTTPPathPrefix()),
			Handler: func(wrter http.ResponseWriter, req *http.Request) {
				s.log.Info("disabling board", zap.String("board", s.Name()))
				s.Disable()
				s.cacheClear()
			},
		},
		{
			Path: fmt.Sprintf("/%s/enable", s.api.HTTPPathPrefix()),
			Handler: func(wrter http.ResponseWriter, req *http.Request) {
				s.log.Info("enabling board", zap.String("board", s.Name()))
				s.Enable()
			},
		},
		{
			Path: fmt.Sprintf("/%s/status", s.api.HTTPPathPrefix()),
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
			Path: fmt.Sprintf("/%s/clearcache", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Info("clearing sportboard cache", zap.String("board", s.api.League()))
				s.cacheClear()
				s.api.CacheClear(context.Background())
			},
		},
		{
			Path: fmt.Sprintf("/%s/scrollon", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Info("enabling scroll mode", zap.String("board", s.api.League()))
				select {
				case s.cancelBoard <- struct{}{}:
				default:
				}
				s.config.ScrollMode.Store(true)
				s.cacheClear()
				s.api.CacheClear(context.Background())
			},
		},
		{
			Path: fmt.Sprintf("/%s/scrolloff", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Info("disabling scroll mode", zap.String("board", s.api.League()))
				select {
				case s.cancelBoard <- struct{}{}:
				default:
				}
				s.config.ScrollMode.Store(false)
				s.cacheClear()
				s.api.CacheClear(context.Background())
			},
		},
		{
			Path: fmt.Sprintf("/%s/scrollstatus", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Debug("get board scroll status", zap.String("board", s.Name()))
				w.Header().Set("Content-Type", "text/plain")
				if s.config.ScrollMode.Load() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
	}, nil
}
