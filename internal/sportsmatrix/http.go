package sportsmatrix

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/twitchtv/twirp"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	pb "github.com/robbydyer/sports/internal/proto/sportsmatrix"
	"github.com/robbydyer/sports/internal/twirphelpers"
)

//go:embed assets
var assets embed.FS

// EmbedDir is a wrapper to return index.html by default
type EmbedDir struct {
	http.FileSystem
}

type jumpRequest struct {
	Board string `json:"board"`
}

// Open implementation of http.FileSystem that falls back to serving /index.html
func (d EmbedDir) Open(name string) (http.File, error) {
	if f, err := d.FileSystem.Open(name); err == nil {
		return f, nil
	}

	return d.FileSystem.Open("/index.html")
}

func (s *SportsMatrix) startHTTP() chan error {
	errChan := make(chan error, 1)

	router := mux.NewRouter()

	registeredPaths := make(map[string]struct{})

	register := func(name string, h *board.HTTPHandler) {
		if _, ok := registeredPaths[h.Path]; ok {
			// This path already registered
			return
		}
		registeredPaths[h.Path] = struct{}{}

		if !strings.HasPrefix(h.Path, "/api") {
			h.Path = filepath.Join("/api", h.Path)
		}
		s.log.Info("registering http handler", zap.String("name", name), zap.String("path", h.Path))
		router.HandleFunc(h.Path, h.Handler)
		s.httpEndpoints = append(s.httpEndpoints, h.Path)
	}

	for _, h := range s.httpHandlers() {
		register("sportsmatrix", h)
	}

	allBoards := append(s.boards, s.betweenBoards...)

	rpcPaths := make(map[string]struct{})

	for _, b := range allBoards {
		s.log.Info("register HTTP/RPC handlers for board",
			zap.String("board", b.Name()),
		)
		handlers, err := b.GetHTTPHandlers()
		if err != nil {
			errChan <- err
			return errChan
		}
		for _, h := range handlers {
			register(b.Name(), h)
		}

		// RPC handlers
		if path, h := b.GetRPCHandler(); h != nil && path != "" {
			if _, ok := rpcPaths[path]; !ok {
				s.log.Info("register RPC Handler",
					zap.String("path", path),
					zap.String("board", b.Name()),
				)
				router.PathPrefix(path).Handler(h)
				rpcPaths[path] = struct{}{}
			}
		}
	}

	for _, c := range s.canvases {
		handlers, err := c.GetHTTPHandlers()
		if err != nil {
			errChan <- err
			return errChan
		}
		for _, h := range handlers {
			register(c.Name(), h)
		}
	}

	// Ensure we didn't dupe any endpoints
	dupe := make(map[string]struct{}, len(s.httpEndpoints))
	for _, e := range s.httpEndpoints {
		if _, exists := dupe[e]; exists {
			errChan <- fmt.Errorf("duplicate HTTP endpoint '%s'", e)
			return errChan
		}
		dupe[e] = struct{}{}
	}

	s.server = http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.HTTPListenPort),
		Handler: router,
	}

	// RPC server
	svr := &Server{
		sm: s,
	}

	twirpHandler := pb.NewSportsmatrixServer(svr,
		twirp.WithServerPathPrefix(""),
		twirp.ChainHooks(
			twirphelpers.GetDefaultHooks(nil, s.log),
		),
	)
	s.log.Info("register RPC Handler",
		zap.String("board", "Sportsmatrix"),
		zap.String("path", twirpHandler.PathPrefix()),
	)
	// router.Handle(twirpHandler.PathPrefix(), twirpHandler)
	router.PathPrefix(twirpHandler.PathPrefix()).Handler(twirpHandler)

	if s.cfg.ServeWebUI {
		filesys := fs.FS(assets)
		web, err := fs.Sub(filesys, "assets/web")
		if err != nil {
			s.log.Error("failed to get sub filesystem", zap.Error(err))
			errChan <- err
			return errChan
		}
		s.log.Info("serving web UI", zap.Int("port", s.cfg.HTTPListenPort))
		router.PathPrefix("/").Handler(http.FileServer(EmbedDir{http.FS(web)}))
	}

	s.log.Info("Starting http server")
	go func() {
		errChan <- s.server.ListenAndServe()
	}()

	time.Sleep(1 * time.Second)

	return errChan
}

func (s *SportsMatrix) httpHandlers() []*board.HTTPHandler {
	return []*board.HTTPHandler{
		{
			Path: "/api/version",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				_, _ = w.Write([]byte(version))
			},
		},
		{
			Path: "/api/screenon",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := s.ScreenOn(ctx); err != nil {
					s.log.Error("failed /api/screenon",
						zap.Error(err),
					)
				}
			},
		},
		{
			Path: "/api/screenoff",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := s.ScreenOff(ctx); err != nil {
					s.log.Error("failed /api/screenoff",
						zap.Error(err),
					)
				}
			},
		},
		{
			Path: "/api/status",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				if s.screenIsOn.Load() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
		{
			Path: "/api/webboardon",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.Lock()
				defer s.Unlock()
				s.webBoardOn <- struct{}{}
			},
		},
		{
			Path: "/api/webboardoff",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.Lock()
				defer s.Unlock()
				s.webBoardOff <- struct{}{}
			},
		},
		{
			Path: "/api/webboardstatus",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				if s.webBoardIsOn.Load() {
					_, _ = w.Write([]byte("true"))
					return
				}
				_, _ = w.Write([]byte("false"))
			},
		},
		{
			Path: "/api/disableall",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.Lock()
				defer s.Unlock()
				s.log.Info("disabling all boards")
				for _, board := range s.boards {
					board.Enabler().Disable()
				}
				s.log.Info("all boards disabled")
			},
		},
		{
			Path: "/api/enableall",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.Lock()
				defer s.Unlock()
				for _, board := range s.boards {
					board.Enabler().Enable()
				}
				s.log.Info("all boards enabled")
			},
		},
		{
			Path: "/api/jump",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				if s.jumping.Load() {
					return
				}

				d := json.NewDecoder(req.Body)
				var j *jumpRequest
				if err := d.Decode(&j); err != nil {
					s.log.Error("failed to process /api/jump request",
						zap.Error(err),
					)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if err := s.JumpTo(ctx, j.Board); err != nil {
					s.log.Error(err.Error(), zap.Error(err))
					http.Error(w, err.Error(), http.StatusRequestTimeout)
				}
			},
		},
		{
			Path: "/api/nextboard",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.currentBoardCancel()
			},
		},
	}
}
