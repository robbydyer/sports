package rgbrender

import (
	"image"

	"github.com/robbydyer/sports/pkg/board"
	"go.uber.org/zap"
)

type Grid struct {
	log   *zap.Logger
	cells []image.Rectangle
}

func NewGrid(canvas board.Canvas, colWidth int, rowHeight int, log *zap.Logger) (*Grid, error) {
	if log == nil {
		var err error
		log, err = zap.NewDevelopment()
		if err != nil {
			return nil, err
		}
	}
	numCols := canvas.Bounds().Dx() / colWidth
	numRows := canvas.Bounds().Dy() / rowHeight
	grid := &Grid{
		log:   log,
		cells: []image.Rectangle{},
	}

	grid.log.Info("new grid", zap.Int("num cols", numCols), zap.Int("num rows", numRows))

	cellX := canvas.Bounds().Dx() / numCols
	cellY := canvas.Bounds().Dy() / numRows

	for c := 0; c < numCols; c++ {
		for r := 0; r < numRows; r++ {
			startX := c * cellX
			startY := r * cellY
			endX := startX + cellX
			endY := startY + cellY

			grid.log.Debug("new cell",
				zap.Int("start X", startX),
				zap.Int("start Y", startY),
				zap.Int("end X", endX),
				zap.Int("end Y", endY),
			)
			grid.cells = append(grid.cells, image.Rect(startX, startY, endX, endY))
		}
	}

	return grid, nil
}

func (g *Grid) Cells() []image.Rectangle {
	return g.cells
}
