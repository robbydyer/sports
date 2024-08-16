package statboard

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"math"
	"strings"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/rgbrender"
)

const padSize = float64(0.005)

func (s *StatBoard) getWriter(bounds image.Rectangle) (*rgbrender.TextWriter, error) {
	s.Lock()
	defer s.Unlock()

	bounds = rgbrender.ZeroedBounds(bounds)

	k := fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy())
	w, ok := s.writers[k]
	if ok {
		s.log.Debug("using cached writer")
		return w, nil
	}

	writer, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return nil, err
	}

	writer.FontSize = getReadableFontSize(bounds)

	writer.YStartCorrection = (-1 * int(padSize*float64(bounds.Dy()))) + writer.YStartCorrection

	s.log.Debug("statboard writer font",
		zap.Float64("size", writer.FontSize),
		zap.String("canvas", fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy())),
		zap.Int("Y correction", writer.YStartCorrection),
	)

	s.writers[k] = writer

	return writer, nil
}

func getReadableFontSize(bounds image.Rectangle) float64 {
	if bounds.Dy() > 128 && bounds.Dx() > 128 {
		return 0.08 * float64(bounds.Dy())
	}
	return 8.0
}

func getGridRatios(writer StringMeasurer, canvas draw.Image, strs []string) ([]float64, error) {
	if len(strs) < 1 {
		return []float64{}, nil
	}

	bounds := rgbrender.ZeroedBounds(canvas.Bounds())

	zeroedCanvas := image.NewRGBA(image.Rect(0, 0, bounds.Max.X, bounds.Max.Y))

	widths, err := writer.MeasureStrings(zeroedCanvas, strs)
	if err != nil {
		return nil, err
	}

	pad := bounds.Dx() / 64

	for i := range widths {
		widths[i] += pad
	}

	ratios := []float64{
		float64(widths[0]) / float64(bounds.Dx()),
	}

	total := widths[0]
	statCols := []int{}
	for i, w := range widths {
		if i == 0 {
			continue
		}
		if total+w > bounds.Dx() {
			break
		}
		total += w
		statCols = append(statCols, w)
	}

	if len(statCols) == 0 {
		return []float64{}, nil
	}

	leftOver := bounds.Dx() - total

	if leftOver/len(statCols) >= 1 {
		for i := range statCols {
			statCols[i] += leftOver / len(statCols)
		}
	}

	for _, w := range statCols {
		ratios = append(ratios, float64(w)/float64(bounds.Dx()))
		total += w
	}

	return ratios, nil
}

// getStatPlaceholders gets a list of strings with longest stat lengths
func (s *StatBoard) getStatPlaceholders(ctx context.Context, bounds image.Rectangle, players []Player, stats []string) ([]string, error) {
	maxName := 0
	prefixCol := 0
	statCols := make([]int, len(stats))

	for _, player := range players {
		select {
		case <-ctx.Done():
			return nil, context.Canceled
		default:
		}

		if len(player.LastName()) > maxName {
			maxName = len(player.LastName())
		}

		if s.withPrefixCol {
			prefix := player.PrefixCol()
			if len(prefix) > prefixCol {
				prefixCol = len(prefix)
			}
		}

		for i, stat := range stats {
			val := player.GetStat(stat)
			if len(val) > statCols[i] {
				statCols[i] = len(val)
			}
			if s.withTitleRow && len(s.api.StatShortName(stat)) > statCols[i] {
				statCols[i] = len(s.api.StatShortName(stat))
			}
		}
	}

	nameMax := maxNameLength(bounds)

	if maxName > nameMax {
		maxName = nameMax
	}

	var strs []string
	if prefixCol > 0 {
		strs = []string{strings.Repeat("0", prefixCol), strings.Repeat("0", maxName)}
		for _, s := range statCols {
			strs = append(strs, strings.Repeat("0", s))
		}
	} else {
		strs = []string{strings.Repeat("0", maxName)}
		for _, s := range statCols {
			strs = append(strs, strings.Repeat("0", s))
		}
	}

	return strs, nil
}

func (s *StatBoard) getStatGrid(ctx context.Context, canvas board.Canvas, players []Player, writer *rgbrender.TextWriter, stats []string) (*rgbrender.Grid, error) {
	strs, err := s.getStatPlaceholders(ctx, rgbrender.ZeroedBounds(canvas.Bounds()), players, stats)
	if err != nil {
		return nil, err
	}

	s.log.Debug("cell X Maxes",
		zap.Strings("strs", strs),
	)

	cellXRatios, err := getGridRatios(writer, canvas, strs)
	if err != nil {
		return nil, err
	}

	numRows := int(math.Floor((float64(rgbrender.ZeroedBounds(canvas.Bounds()).Dy()) / writer.FontSize)))

	var pad int

	heightRatio := float64(1.0) / float64(numRows)

	s.log.Debug("statboard rows",
		zap.Int("num rows", numRows),
		zap.Float64("row height ratio", heightRatio),
		zap.Float64("font size", writer.FontSize),
		zap.Int("pad", pad),
	)

	cellYRatios := make([]float64, numRows)
	for i := range cellYRatios {
		cellYRatios[i] = heightRatio
	}

	return rgbrender.NewGrid(
		canvas,
		len(cellXRatios),
		numRows,
		s.log,
		rgbrender.WithPadding(padSize),
		rgbrender.WithCellColRatios(cellXRatios),
		rgbrender.WithUniformRows(),
	)
}

func maxNameLength(canvas image.Rectangle) int {
	return canvas.Dx() / 8
}

func maxedStr(str string, maxLen int) string {
	if maxLen <= 0 || len(str) <= maxLen {
		return str
	}

	start := float64(maxLen) / 2.0
	i := int(start)
	j := int(start) - 1
	if math.Trunc(start) != start {
		i = int(math.Ceil(start))
		j = int(math.Floor(start)) - 1
	}

	return fmt.Sprintf("%s..%s", str[0:i], str[len(str)-j:])
}
