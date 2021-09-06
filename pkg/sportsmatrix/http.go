package sportsmatrix

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
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

	register := func(name string, h *board.HTTPHandler) {
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

	for _, b := range s.boards {
		handlers, err := b.GetHTTPHandlers()
		if err != nil {
			errChan <- err
			return errChan
		}
		for _, h := range handlers {
			register(b.Name(), h)
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

	s.server = http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.HTTPListenPort),
		Handler: router,
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
				s.Lock()
				defer s.Unlock()
				s.screenOn <- struct{}{}
			},
		},
		{
			Path: "/api/screenoff",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.Lock()
				defer s.Unlock()
				s.screenOff <- struct{}{}
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
				for _, board := range s.boards {
					board.Disable()
				}
			},
		},
		{
			Path: "/api/enableall",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.Lock()
				defer s.Unlock()
				for _, board := range s.boards {
					board.Enable()
				}
			},
		},
		{
			Path: "/api/jump",
			Handler: func(w http.ResponseWriter, req *http.Request) {
				s.jumpLock.Lock()
				defer s.jumpLock.Unlock()

				d := json.NewDecoder(req.Body)
				var j *jumpRequest
				if err := d.Decode(&j); err != nil {
					s.log.Error("failed to process /api/jump request",
						zap.Error(err),
					)
				}
				for _, b := range s.boards {
					if strings.ToLower(b.Name()) == j.Board {
						if !b.Enabled() {
							b.Enable()
						}

						select {
						case s.screenOff <- struct{}{}:
						case <-time.After(5 * time.Second):
							http.Error(w, "timed out", http.StatusRequestTimeout)
							return
						}

						defer func() {
							select {
							case s.screenOn <- struct{}{}:
							case <-time.After(5 * time.Second):
								s.log.Error("failed to turn screen back on after /api/jump")
							}
						}()

						select {
						case s.jumpTo <- j.Board:
						case <-time.After(5 * time.Second):
							http.Error(w, "timed out", http.StatusRequestTimeout)
							return
						}

						return
					}
				}
			},
		},
	}
}
