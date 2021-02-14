package sportsmatrix

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

//go:embed assets
var assets embed.FS

type EmbedDir struct {
	http.FileSystem
}

// Open implementation of http.FileSystem that falls back to serving /index.html
func (d EmbedDir) Open(name string) (http.File, error) {
	if f, err := d.FileSystem.Open(name); err == nil {
		return f, nil
	} else {
		return d.FileSystem.Open("/index.html")
	}
}

func (s *SportsMatrix) startHTTP() chan error {
	errChan := make(chan error, 1)

	router := mux.NewRouter()

	if s.cfg.ServeWebUI {
		filesys := fs.FS(assets)
		static, err := fs.Sub(filesys, "assets/web")
		if err != nil {
			s.log.Error("failed to get sub filesystem", zap.Error(err))
			errChan <- err
			return errChan
		}
		ed := EmbedDir{
			http.FS(static),
		}
		router.PathPrefix("/").Handler(http.FileServer(ed))
	}

	router.HandleFunc("/screenoff", s.turnScreenOff)
	router.HandleFunc("/screenon", s.turnScreenOn)

	for _, b := range s.boards {
		handlers, err := b.GetHTTPHandlers()
		if err != nil {
			errChan <- err
			return errChan
		}
		for _, h := range handlers {
			s.log.Info("registering http handler", zap.String("board", b.Name()), zap.String("path", h.Path))
			router.HandleFunc(h.Path, h.Handler)
		}
	}

	s.server = http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.HTTPListenPort),
		Handler: router,
	}

	s.log.Info("Starting http server")
	go func() {
		errChan <- s.server.ListenAndServe()
	}()

	// We could manually manage the listener and check for readiness that way,
	// but laziness instead leads to this arbitrary wait time to let the server
	// get ready or potentially return an error to the channel
	time.Sleep(3 * time.Second)

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
