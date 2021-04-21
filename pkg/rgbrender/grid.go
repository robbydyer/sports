package rgbrender

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
)

// GridOption is an option for a Grid
type GridOption func(grid *Grid) error

// Grid manages sub-canvas "cells" of a larger canvas
type Grid struct {
	baseCanvas   board.Canvas
	log          *zap.Logger
	cells        []*Cell
	cols         int
	rows         int
	cellX        []int
	cellY        []int
	padRatio     float64
	padding      int
	paddedPix    map[string]image.Point
	cellStyleSet bool
}

// Cell contains a canvas and it's bounds related to it's parent canvas
type Cell struct {
	Canvas board.Canvas
	Bounds image.Rectangle
	Col    int
	Row    int
}

// NewGrid ...
func NewGrid(canvas board.Canvas, numCols int, numRows int, log *zap.Logger, opts ...GridOption) (*Grid, error) {
	if log == nil {
		var err error
		log, err = zap.NewDevelopment()
		if err != nil {
			return nil, err
		}
	}

	/*
		if numCols > maxAllowedCols {
			return nil, fmt.Errorf("unsupported number of columns %d", numCols)
		}
		if numRows > maxAllowedRows {
			return nil, fmt.Errorf("unsupported number of rows %d", numRows)
		}
	*/

	grid := &Grid{
		baseCanvas: canvas,
		log:        log,
		cols:       numCols,
		rows:       numRows,
		paddedPix:  make(map[string]image.Point),
		cellX:      make([]int, numCols),
		cellY:      make([]int, numRows),
	}

	for _, f := range opts {
		if err := f(grid); err != nil {
			return nil, err
		}
	}

	if !grid.cellStyleSet {
		f := WithUniformCells()
		if err := f(grid); err != nil {
			return nil, err
		}
	}

	if grid.padRatio > 0 {
		grid.padding = int(grid.padRatio * float64(ZeroedBounds(canvas.Bounds()).Dx()))
		if grid.padding < 1 {
			grid.padding = 2
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
	if len(g.cellX) != g.cols {
		return fmt.Errorf("invalid number of cell width settings")
	}
	if len(g.cellY) != g.rows {
		return fmt.Errorf("invalid number of cell height settings")
	}
	cellIndex := 0
	for r := 0; r < g.rows; r++ {
		for c := 0; c < g.cols; c++ {
			halfPad := g.padding / 2
			realStartX := 0
			if c > 0 {
				for i := c - 1; i >= 0; i-- {
					realStartX += g.cellX[i]
				}
			}

			realStartY := 0

			if r > 0 {
				for i := r - 1; i >= 0; i-- {
					realStartY += g.cellY[i]
				}
			}

			realEndX := realStartX + g.cellX[c]
			realEndY := realStartY + g.cellY[r]
			startX := realStartX + halfPad
			startY := realStartY + halfPad
			endX := realEndX - halfPad
			endY := realEndY - halfPad

			// Save padded pixels, exluding the outermost regions of the canvas
			for x := realStartX; x < realEndX; x++ {
				if (realStartX == 0 && x < startX) || (realEndX == (g.cols*g.cellX[c]) && x > endX) {
					continue
				}
				for y := realStartY; y < realEndY; y++ {
					if (realStartY == 0 && y < startY) || (realEndY == (g.rows*g.cellY[r]) && y > endY) {
						continue
					}
					if x < startX || y < startY || x > endX || y > endY {
						g.paddedPix[fmt.Sprintf("%dx%d", x, y)] = image.Pt(x, y)
					}
				}
			}

			newC := board.NewBlankCanvas(g.cellX[c], g.cellY[r], g.log)
			if newC == nil {
				return fmt.Errorf("cell canvas was nil")
			}
			/*
				g.log.Debug("new cell",
					zap.Int("index", cellIndex),
					zap.Int("start X", startX),
					zap.Int("start Y", startY),
					zap.Int("end X", endX),
					zap.Int("end Y", endY),
				)
			*/
			g.cells[cellIndex] = &Cell{
				Canvas: newC,
				Row:    r,
				Col:    c,
				Bounds: image.Rect(startX, startY, endX, endY),
			}
			cellIndex++
		}
	}

	return nil
}

// NumRows ...
func (g *Grid) NumRows() int {
	return g.rows
}

// NumCols ...
func (g *Grid) NumCols() int {
	return g.cols
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

// GetRow get all the cells in the given row
func (g *Grid) GetRow(row int) []*Cell {
	cells := []*Cell{}
	for _, c := range g.cells {
		if c.Row == row {
			cells = append(cells, c)
		}
	}

	return cells
}

// GetCol gets all the cells in a given column
func (g *Grid) GetCol(col int) []*Cell {
	cells := []*Cell{}
	for _, c := range g.cells {
		if c.Col == col {
			cells = append(cells, c)
		}
	}

	return cells
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

// WithPadding is an option to specify padding width between cells as a percentage of canvas width
func WithPadding(pad float64) GridOption {
	return func(g *Grid) error {
		g.padRatio = pad
		return nil
	}
}

// WithUniformCells sets all cell sizes to a uniform size
func WithUniformCells() GridOption {
	return func(g *Grid) error {
		if g.baseCanvas == nil {
			return fmt.Errorf("base canvas not set")
		}
		g.log.Debug("uniform grid")
		cellX := g.baseCanvas.Bounds().Dx() / g.cols
		cellY := g.baseCanvas.Bounds().Dy() / g.rows

		for i := 0; i < g.cols; i++ {
			g.cellX[i] = cellX
		}

		for i := 0; i < g.rows; i++ {
			g.cellY[i] = cellY
		}

		g.cellStyleSet = true

		return nil
	}
}

// WithCellRatios sets col/row sizes with ratios
func WithCellRatios(colRatios []float64, rowRatios []float64) GridOption {
	return func(g *Grid) error {
		g.log.Debug("grid with col/row ratios")
		if len(colRatios) != g.cols {
			return fmt.Errorf("invalid number of col ratios, must match number of cols")
		}
		if len(rowRatios) != g.rows {
			return fmt.Errorf("invalid number of row ratios, must match number of rows")
		}

		bounds := ZeroedBounds(g.baseCanvas.Bounds())

		for i, r := range colRatios {
			g.cellX[i] = int(math.Floor(r * float64(bounds.Dx())))
			g.log.Debug("cellX",
				zap.Int("index", i),
				zap.Float64("ratio", r),
				zap.Int("size", g.cellX[i]),
			)
		}

		for i, r := range rowRatios {
			g.cellY[i] = int(math.Floor(r * float64(g.baseCanvas.Bounds().Dy())))
			g.log.Debug("cellY",
				zap.Int("index", i),
				zap.Float64("ratio", r),
				zap.Int("size", g.cellY[i]),
			)
		}

		g.cellStyleSet = true

		return nil
	}
}
