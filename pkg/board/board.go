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

// Enabler is an interface for basic Enable/Disable functions
type Enabler interface {
	Enabled() bool
	Enable()
	Disable()
}

// Board is the interface to implement for displaying on the matrix
type Board interface {
	Enabler
	Name() string
	Render(ctx context.Context, canvases Canvas) error
	GetHTTPHandlers() ([]*HTTPHandler, error)
}

// Canvas ...
type Canvas interface {
	image.Image
	draw.Image
	Enabler
	Name() string
	Clear() error
	Render() error
	GetHTTPHandlers() ([]*HTTPHandler, error)
	Close() error
	Scrollable() bool
	PaddedBounds() image.Rectangle
}
