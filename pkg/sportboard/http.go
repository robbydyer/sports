package sportboard

import (
	"fmt"
	"net/http"

	"github.com/robbydyer/sports/pkg/board"
	"go.uber.org/zap"
)

// GetHTTPHandlers ...
func (s *SportBoard) GetHTTPHandlers() ([]*board.HTTPHandler, error) {
	hideFav := &board.HTTPHandler{
		Path: fmt.Sprintf("/%s/hidefavoritescore", s.api.HTTPPathPrefix()),
		Handler: func(http.ResponseWriter, *http.Request) {
			s.log.Info("hiding favorite team scores")
			s.config.HideFavoriteScore = true
		},
	}
	showFav := &board.HTTPHandler{
		Path: fmt.Sprintf("/%s/showfavoritescore", s.api.HTTPPathPrefix()),
		Handler: func(http.ResponseWriter, *http.Request) {
			s.log.Info("showing favorite team scores")
			s.config.HideFavoriteScore = false
		},
	}
	stick := &board.HTTPHandler{
		Path: fmt.Sprintf("/%s/favoritesticky", s.api.HTTPPathPrefix()),
		Handler: func(wrter http.ResponseWriter, req *http.Request) {
			s.log.Info("setting favorite teams to sticky")
			s.config.FavoriteSticky = true
		},
	}
	unstick := &board.HTTPHandler{
		Path: fmt.Sprintf("/%s/favoriteunstick", s.api.HTTPPathPrefix()),
		Handler: func(wrter http.ResponseWriter, req *http.Request) {
			s.log.Info("setting favorite teams to not stick")
			s.config.FavoriteSticky = false
		},
	}
	disable := &board.HTTPHandler{
		Path: fmt.Sprintf("/%s/disable", s.api.HTTPPathPrefix()),
		Handler: func(wrter http.ResponseWriter, req *http.Request) {
			s.log.Info("disabling board", zap.String("board", s.Name()))
			s.config.Enabled = false
		},
	}
	enable := &board.HTTPHandler{
		Path: fmt.Sprintf("/%s/enable", s.api.HTTPPathPrefix()),
		Handler: func(wrter http.ResponseWriter, req *http.Request) {
			s.log.Info("enabling board", zap.String("board", s.Name()))
			s.config.Enabled = true
		},
	}

	return []*board.HTTPHandler{
		hideFav,
		showFav,
		stick,
		unstick,
		disable,
		enable,
	}, nil
}
