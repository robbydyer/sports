package sportboard

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
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
				s.Enabler().Disable()
			},
		},
		{
			Path: fmt.Sprintf("/%s/enable", s.api.HTTPPathPrefix()),
			Handler: func(wrter http.ResponseWriter, req *http.Request) {
				s.log.Info("enabling board", zap.String("board", s.Name()))
				s.Enabler().Enable()
			},
		},
		{
			Path: fmt.Sprintf("/%s/status", s.api.HTTPPathPrefix()),
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
				s.callCancelBoard()
				s.config.ScrollMode.Store(true)
				s.api.CacheClear(context.Background())
			},
		},
		{
			Path: fmt.Sprintf("/%s/scrolloff", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Info("disabling scroll mode", zap.String("board", s.api.League()))
				s.callCancelBoard()
				s.config.ScrollMode.Store(false)
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
		{
			Path: fmt.Sprintf("/%s/tightscrollon", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Info("enabling tight scroll mode", zap.String("board", s.api.League()))
				s.callCancelBoard()
				s.config.TightScroll.Store(true)
				s.api.CacheClear(context.Background())
			},
		},
		{
			Path: fmt.Sprintf("/%s/tightscrolloff", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Info("disabling scroll mode", zap.String("board", s.api.League()))
				s.callCancelBoard()
				s.config.TightScroll.Store(false)
				s.api.CacheClear(context.Background())
			},
		},
		{
			Path: fmt.Sprintf("/%s/tightscrollstatus", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Debug("get board tight scroll status", zap.String("board", s.Name()))
				w.Header().Set("Content-Type", "text/plain")
				if s.config.TightScroll.Load() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
		{
			Path: fmt.Sprintf("/%s/recordrankon", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Info("enabling team record/rank mode", zap.String("board", s.api.League()))
				s.callCancelBoard()
				s.config.ShowRecord.Store(true)
				s.api.CacheClear(context.Background())
			},
		},
		{
			Path: fmt.Sprintf("/%s/recordrankoff", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Info("disabling team record/rank mode", zap.String("board", s.api.League()))
				s.callCancelBoard()
				s.config.ShowRecord.Store(false)
				s.api.CacheClear(context.Background())
			},
		},
		{
			Path: fmt.Sprintf("/%s/recordrankstatus", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Debug("get team record/rank status", zap.String("board", s.Name()))
				w.Header().Set("Content-Type", "text/plain")
				if s.config.ShowRecord.Load() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
		{
			Path: fmt.Sprintf("/%s/oddson", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Info("enabling odds mode", zap.String("board", s.api.League()))
				s.callCancelBoard()
				s.config.GamblingSpread.Store(true)
				s.api.CacheClear(context.Background())
			},
		},
		{
			Path: fmt.Sprintf("/%s/oddsoff", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Info("disabling odds mode", zap.String("board", s.api.League()))
				s.callCancelBoard()
				s.config.GamblingSpread.Store(false)
				s.api.CacheClear(context.Background())
			},
		},
		{
			Path: fmt.Sprintf("/%s/oddsstatus", s.api.HTTPPathPrefix()),
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.log.Debug("get odds status", zap.String("board", s.Name()))
				w.Header().Set("Content-Type", "text/plain")
				if s.config.GamblingSpread.Load() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
	}, nil
}
