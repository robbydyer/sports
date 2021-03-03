package rgbrender

import (
	"fmt"
	"image"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/imgcanvas"
	"github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"go.uber.org/zap"
)

const maxAllowedCols = 10
const maxAllowedRows = 10

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
}

type Cell struct {
	Canvas board.Canvas
	Bounds image.Rectangle
}

func NewGrid(canvas board.Canvas, canvaser Canvaser, colWidth int, rowHeight int, log *zap.Logger) (*Grid, error) {
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

	grid.cells = make([]*Cell, numCols*numRows)
	grid.log.Info("new grid", zap.Int("num cols", numCols), zap.Int("num rows", numRows))

	if err := grid.generateCells(); err != nil {
		return nil, err
	}

	return grid, nil
}

func (g *Grid) generateCells() error {
	cellIndex := 0
	for c := 0; c < g.cols; c++ {
		for r := 0; r < g.rows; r++ {
			startX := c * g.cellX
			startY := r * g.cellY
			endX := startX + g.cellX
			endY := startY + g.cellY

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

func (g *Grid) Clear() {
	//g.canvases = []board.Canvas{}
}

func (g *Grid) Canvases() []board.Canvas {
	canvases := []board.Canvas{}
	for _, c := range g.cells {
		canvases = append(canvases, c.Canvas)
	}

	return canvases
}

func (g *Grid) Cell(index int) *Cell {
	return g.cells[index]
}

func (g *Grid) DrawToBase(base board.Canvas) error {

	return nil
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
