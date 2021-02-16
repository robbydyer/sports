package board

import (
	"context"
	"image"
	"image/draw"
	"net/http"
)

// HTTPHandler is the type returned to the sportsmatrix for HTTP endpoints
type HTTPHandler struct {
	Handler func(http.ResponseWriter, *http.Request)
	Path    string
}

// Board is the interface to implement for displaying on the matrix
type Board interface {
	Name() string
	Render(ctx context.Context, canvas Canvas) error
	Enabled() bool
	GetHTTPHandlers() ([]*HTTPHandler, error)
}

// Canvas ...
type Canvas interface {
	image.Image
	draw.Image
	Clear() error
	Render() error
}
