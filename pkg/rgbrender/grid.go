package rgbrender

import (
	"fmt"
	"image"
	"image/color"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/imgcanvas"
	"github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"go.uber.org/zap"
)

const maxAllowedCols = 10
const maxAllowedRows = 10

type GridOption func(grid *Grid) error

type Canvaser func(bounds image.Rectangle) (board.Canvas, error)

type Grid struct {
	log        *zap.Logger
	cells      []*Cell
	baseCanvas board.Canvas
	canvaser   Canvaser
	cols       int
	rows       int
	cellX      int
	cellY      int
	padding    int
	paddedPix  []image.Point
}

type Cell struct {
	Canvas board.Canvas
	Bounds image.Rectangle
}

func NewGrid(canvas board.Canvas, canvaser Canvaser, colWidth int, rowHeight int, log *zap.Logger, opts ...GridOption) (*Grid, error) {
	if log == nil {
		var err error
		log, err = zap.NewDevelopment()
		if err != nil {
			return nil, err
		}
	}

	numCols := canvas.Bounds().Dx() / colWidth
	numRows := canvas.Bounds().Dy() / rowHeight

	if numCols > maxAllowedCols {
		return nil, fmt.Errorf("unsupported number of columns %d", numCols)
	}
	if numRows > maxAllowedRows {
		return nil, fmt.Errorf("unsupported number of rows %d", numRows)
	}

	grid := &Grid{
		log:        log,
		baseCanvas: canvas,
		canvaser:   canvaser,
		cols:       numCols,
		rows:       numRows,
		cellX:      canvas.Bounds().Dx() / numCols,
		cellY:      canvas.Bounds().Dy() / numRows,
	}

	for _, f := range opts {
		if err := f(grid); err != nil {
			return nil, err
		}
	}

	if grid.padding > 0 && grid.padding%2 != 0 {
		grid.padding++
	}

	grid.cells = make([]*Cell, numCols*numRows)
	grid.log.Info("new grid", zap.Int("num cols", numCols), zap.Int("num rows", numRows), zap.Int("padding", grid.padding))

	if err := grid.generateCells(); err != nil {
		return nil, err
	}

	return grid, nil
}

func (g *Grid) generateCells() error {
	cellIndex := 0
	for r := 0; r < g.rows; r++ {
		for c := 0; c < g.cols; c++ {
			halfPad := g.padding / 2
			startX := (c * g.cellX) + halfPad
			startY := (r * g.cellY) + halfPad
			endX := (startX + g.cellX) - halfPad
			endY := (startY + g.cellY) - halfPad

			for x := startX - halfPad; x < endX+halfPad; x++ {
				for y := startY - halfPad; y < endY+halfPad; y++ {
					if x < startX || y < startY || x > endX || y > endY {
						g.paddedPix = append(g.paddedPix, image.Pt(x, y))
					}
				}
			}

			newC, err := g.canvaser(image.Rect(startX, startY, endX, endY))
			if err != nil {
				return err
			}
			if newC == nil {
				return fmt.Errorf("cell canvas was nil")
			}
			g.log.Debug("new cell",
				zap.Int("index", cellIndex),
				zap.Int("start X", newC.Bounds().Min.X),
				zap.Int("start Y", newC.Bounds().Min.Y),
				zap.Int("end X", newC.Bounds().Max.X),
				zap.Int("end Y", newC.Bounds().Max.Y),
			)
			g.cells[cellIndex] = &Cell{
				Canvas: newC,
				Bounds: image.Rect(startX, startY, endX, endY),
			}
			cellIndex++
		}
	}

	return nil
}

func (g *Grid) Clear() error {
	g.cells = make([]*Cell, g.cols*g.rows)
	return g.generateCells()
}

func (g *Grid) Canvases() []board.Canvas {
	canvases := []board.Canvas{}
	for _, c := range g.cells {
		canvases = append(canvases, c.Canvas)
	}

	return canvases
}

func (g *Grid) Cells() []*Cell {
	return g.cells
}

func (g *Grid) Cell(index int) (*Cell, error) {
	if index > len(g.cells)-1 {
		return nil, fmt.Errorf("no cell at index %d, max of %d", index, len(g.cells)-1)
	}
	return g.cells[index], nil
}

func (g *Grid) FillPadded(canvas board.Canvas, clr color.Color) {
	for _, pt := range g.paddedPix {
		canvas.Set(pt.X, pt.Y, clr)
	}
}

func (g *Grid) DrawToBase(base board.Canvas) error {

	return nil
}

func WithPadding(pad int) GridOption {
	return func(g *Grid) error {
		g.padding = pad
		return nil
	}
}

func GetCanvaser(canvas board.Canvas, logger *zap.Logger) (Canvaser, error) {
	switch canvas.(type) {
	case *imgcanvas.ImgCanvas:
		return func(bounds image.Rectangle) (board.Canvas, error) {
			return imgcanvas.New(bounds.Dx(), bounds.Dy(), logger), nil
		}, nil
	case *rgbmatrix.Canvas:
		c := canvas.(*rgbmatrix.Canvas)
		mtrx := c.Matrix()
		w, h := mtrx.Geometry()
		var newM rgbmatrix.Matrix
		switch mtrx.(type) {
		case *rgbmatrix.ConsoleMatrix:
			newM = rgbmatrix.NewConsoleMatrix(w, h, mtrx.Writer(), logger)
		}
		return func(bounds image.Rectangle) (board.Canvas, error) {
			return rgbmatrix.NewCanvas(newM), nil
		}, nil
	}

	return nil, fmt.Errorf("unsupported board.Canvas for grid layout")
}
