package rgbrender

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
)

const (
	maxAllowedCols = 10
	maxAllowedRows = 10
)

// GridOption is an option for a Grid
type GridOption func(grid *Grid) error

// Grid manages sub-canvas "cells" of a larger canvas
type Grid struct {
	log       *zap.Logger
	cells     []*Cell
	cols      int
	rows      int
	cellX     int
	cellY     int
	padding   int
	paddedPix map[string]image.Point
}

// Cell contains a canvas and it's bounds related to it's parent canvas
type Cell struct {
	Canvas board.Canvas
	Bounds image.Rectangle
}

// NewGrid ...
func NewGrid(canvas board.Canvas, colWidth int, rowHeight int, log *zap.Logger, opts ...GridOption) (*Grid, error) {
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
		log:       log,
		cols:      numCols,
		rows:      numRows,
		cellX:     canvas.Bounds().Dx() / numCols,
		cellY:     canvas.Bounds().Dy() / numRows,
		paddedPix: make(map[string]image.Point),
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
			realStartX := c * g.cellX
			realStartY := r * g.cellY
			realEndX := realStartX + g.cellX
			realEndY := realStartY + g.cellY
			startX := realStartX + halfPad
			startY := realStartY + halfPad
			endX := realEndX - halfPad
			endY := realEndY - halfPad

			// Save padded pixels, exluding the outermost regions of the canvas
			for x := realStartX; x < realEndX; x++ {
				if (realStartX == 0 && x < startX) || (realEndX == (g.cols*g.cellX) && x > endX) {
					continue
				}
				for y := realStartY; y < realEndY; y++ {
					if (realStartY == 0 && y < startY) || (realEndY == (g.rows*g.cellY) && y > endY) {
						continue
					}
					if x < startX || y < startY || x > endX || y > endY {
						g.paddedPix[fmt.Sprintf("%dx%d", x, y)] = image.Pt(x, y)
					}
				}
			}

			newC := board.NewBlankCanvas(g.cellX, g.cellY, g.log)
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

// Clear removes cells and regenerates them
func (g *Grid) Clear() error {
	g.cells = make([]*Cell, g.cols*g.rows)
	return g.generateCells()
}

// Cells returns all the cells
func (g *Grid) Cells() []*Cell {
	return g.cells
}

// Cell returns a cell at a given index
func (g *Grid) Cell(index int) (*Cell, error) {
	if index > len(g.cells)-1 {
		return nil, fmt.Errorf("no cell at index %d, max of %d", index, len(g.cells)-1)
	}
	return g.cells[index], nil
}

// FillPadded fills the cell padding with a color
func (g *Grid) FillPadded(canvas board.Canvas, clr color.Color) {
	for _, pt := range g.paddedPix {
		canvas.Set(pt.X, pt.Y, clr)
	}
}

// DrawToBase draws the cells onto a base parent canvas
func (g *Grid) DrawToBase(base board.Canvas) error {
	for _, cell := range g.cells {
		draw.Draw(base, cell.Bounds, cell.Canvas, image.Point{}, draw.Over)
	}
	return nil
}

// WithPadding is an option to specify padding width between cells
func WithPadding(pad int) GridOption {
	return func(g *Grid) error {
		g.padding = pad
		return nil
	}
}
