package sportsmatrix

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

//go:embed assets
var assets embed.FS

// EmbedDir is a wrapper to return index.html by default
type EmbedDir struct {
	http.FileSystem
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

	router.HandleFunc("/api/screenoff", s.turnScreenOff)
	router.HandleFunc("/api/screenon", s.turnScreenOn)
	router.HandleFunc("/api/board", s.webCanvas)

	for _, b := range s.boards {
		handlers, err := b.GetHTTPHandlers()
		if err != nil {
			errChan <- err
			return errChan
		}
		for _, h := range handlers {
			if !strings.HasPrefix(h.Path, "/api") {
				h.Path = filepath.Join("/api", h.Path)
			}
			s.log.Info("registering http handler", zap.String("board", b.Name()), zap.String("path", h.Path))
			router.HandleFunc(h.Path, h.Handler)
		}
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

func (s *SportsMatrix) turnScreenOff(respWriter http.ResponseWriter, req *http.Request) {
	s.Lock()
	defer s.Unlock()
	s.screenOff <- struct{}{}
}

func (s *SportsMatrix) turnScreenOn(respWriter http.ResponseWriter, req *http.Request) {
	s.Lock()
	defer s.Unlock()
	s.screenOn <- struct{}{}
}

func (s *SportsMatrix) webCanvas(w http.ResponseWriter, req *http.Request) {
	i, err := s.GetImgCanvas()
	if err != nil || i == nil {
		s.log.Error("could not get ImgCanvas", zap.Error(err))
		return
	}

	s.log.Debug("getting image for web board")

	w.Header().Set("Content-Type", "image/png")
	board := i.LastPng()

	if board == nil {
		s.log.Error("no board has been rendered")
	}

	s.log.Debug("reading web board")
	boardBytes, err := io.ReadAll(board)
	if err != nil {
		s.log.Error("failed to read board", zap.Error(err))
		return
	}

	if len(boardBytes) == 0 {
		s.log.Debug("web board has already been read, using cache")
		boardBytes = s.webBoardCache
	} else {
		s.log.Debug("first time reading web board, caching")
		s.webBoardCache = boardBytes
	}

	if _, err := w.Write(boardBytes); err != nil {
		s.log.Error("failed to copy png for /api/board", zap.Error(err))
	}
}
