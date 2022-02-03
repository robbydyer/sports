package stockboard

import (
	"context"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

//go:embed assets
var assets embed.FS

func (s *StockBoard) renderStock(ctx context.Context, stock *Stock, canvas board.Canvas) error {
	canvasBounds := rgbrender.ZeroedBounds(canvas.Bounds())

	maxChartWidth := canvasBounds.Dx() / 2

	if s.config.MaxChartWidthRatio > 0 {
		maxChartWidth = int(math.Ceil(float64(canvasBounds.Dx()) * 0.5 * s.config.MaxChartWidthRatio))
	}

	chartWidth, _ := s.chartWidth(maxChartWidth)

	var chartBounds image.Rectangle
	var symbolBounds image.Rectangle
	var priceBounds image.Rectangle
	if s.config.UseLogos.Load() {
		chartBounds = rgbrender.ZeroedBounds(
			image.Rect(
				canvasBounds.Max.X/2,
				canvasBounds.Max.Y/2,
				chartWidth+(canvasBounds.Max.X/2),
				canvasBounds.Max.Y),
		)
		symbolBounds = rgbrender.ZeroedBounds(
			image.Rect(
				canvasBounds.Min.X,
				canvasBounds.Min.Y,
				(canvasBounds.Max.X/2)+1,
				canvasBounds.Max.Y))
		priceBounds = rgbrender.ZeroedBounds(
			image.Rect(
				canvasBounds.Max.X/2,
				canvasBounds.Min.Y,
				canvasBounds.Max.X,
				canvasBounds.Max.Y/2))
	} else {
		chartBounds = rgbrender.ZeroedBounds(
			image.Rect(
				canvasBounds.Min.X,
				canvasBounds.Max.Y/2,
				chartWidth+canvasBounds.Min.X,
				canvasBounds.Max.Y,
			),
		)
		symbolBounds = rgbrender.ZeroedBounds(
			image.Rect(
				canvasBounds.Min.X,
				canvasBounds.Min.Y,
				canvasBounds.Max.X/2,
				canvasBounds.Max.Y/2,
			),
		)
		priceBounds = rgbrender.ZeroedBounds(
			image.Rect(canvasBounds.Max.X/2,
				canvasBounds.Min.Y,
				canvasBounds.Max.X,
				canvasBounds.Max.Y/2,
			),
		)
	}

	var chartPrices []*Price
	if len(stock.Prices) >= chartBounds.Dx() {
		chartPrices = s.getChartPrices(chartBounds.Dx(), stock)
		s.config.adjustedResolution = 1
	} else {
		s.config.adjustedResolution = int(math.Ceil(float64(chartBounds.Dx()) / float64(len(stock.Prices))))
		chartPrices = s.getChartPrices(chartBounds.Dx()/s.config.adjustedResolution, stock)
	}

	s.log.Debug("stock prices",
		zap.String("symbol", stock.Symbol),
		zap.Float64("open", stock.OpenPrice),
		zap.Int("total prices", len(stock.Prices)),
		zap.Int("sampled prices", len(chartPrices)),
		zap.Int("canvas width", canvasBounds.Dx()),
		zap.Int("max pix", chartWidth),
		zap.Int("max allowed pixels", chartBounds.Dx()/s.config.adjustedResolution),
		zap.Int("configured resolution", s.config.ChartResolution),
		zap.Int("adjusted resolution", s.config.adjustedResolution),
		zap.Float64s("prices", prices(chartPrices)),
	)

	chart := s.getChart(chartBounds, stock, chartPrices)

	draw.Draw(canvas, canvasBounds, chart, image.Point{}, draw.Over)

	priceWriter, err := s.getPriceWriter(canvasBounds)
	if err != nil {
		return err
	}

	symbolWriter, err := s.getSymbolWriter(canvasBounds)
	if err != nil {
		return err
	}

	var clr color.Color
	if stock.Price > stock.OpenPrice {
		clr = green
	} else {
		clr = red
	}

	symbol := s.specialName(stock.Symbol)

	if len(symbol) > 4 {
		symbolWriter = priceWriter
	}

	writeSymbol := func() {
		if err := symbolWriter.WriteAligned(
			rgbrender.RightCenter,
			canvas,
			symbolBounds,
			[]string{
				fmt.Sprintf("%s  ", symbol),
			},
			color.White,
		); err != nil {
			s.log.Error("failed to write symbol",
				zap.Error(err),
			)
		}
	}

	if s.config.UseLogos.Load() {
		s.log.Debug("attempting to draw logo for stock",
			zap.String("symbol", stock.Symbol),
		)
		if err := s.drawLogo(ctx, canvas, symbolBounds, stock.Symbol); err != nil {
			s.log.Error("failed to draw logo for stock",
				zap.Error(err),
				zap.String("symbol", stock.Symbol),
			)
			writeSymbol()
		}
	} else {
		writeSymbol()
	}

	if err := priceWriter.WriteAligned(
		rgbrender.LeftCenter,
		canvas,
		priceBounds,
		[]string{
			fmt.Sprintf("  %.2f ", stock.Price),
			fmt.Sprintf("  %.2f%% ", stock.Change),
		},
		clr,
	); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}

	s.log.Debug("rendering stock",
		zap.String("symbol", stock.Symbol),
		zap.Int("total prices", len(stock.Prices)),
		zap.Int("charted prices", len(chartPrices)),
	)

	return nil
}

func (s *StockBoard) getChart(bounds image.Rectangle, stock *Stock, prices []*Price) image.Image {
	img := image.NewRGBA(bounds)

	maxPrice := stock.maxPrice()
	if maxPrice == nil {
		maxPrice = &Price{
			Price: stock.Price,
		}
	}
	minPrice := stock.minPrice()
	if minPrice == nil {
		minPrice = &Price{
			Price: stock.Price,
		}
	}

	// deviator represents the maximum deviation between min/max price and open price
	var deviator float64

	if maxPrice.Price-stock.OpenPrice > stock.OpenPrice-minPrice.Price {
		deviator = (maxPrice.Price - stock.OpenPrice) / float64(bounds.Dy()/2)
	} else {
		deviator = (stock.OpenPrice - minPrice.Price) / float64(bounds.Dy()/2)
	}

	midY := ((bounds.Max.Y - bounds.Min.Y) / 2) + bounds.Min.Y

	x := bounds.Min.X - 1
	lastY := midY

	s.log.Debug("draw chart",
		zap.Int("mid", midY),
		zap.Float64("open price", stock.OpenPrice),
		zap.Float64("max price", maxPrice.Price),
		zap.Float64("min price", minPrice.Price),
		zap.Float64("deviator", deviator),
		zap.Int("num prices", len(prices)),
		zap.Int("configured resolution", s.config.ChartResolution),
		zap.Int("adjusted resolution", s.config.adjustedResolution),
	)

	for _, price := range prices {
		lastX := x
		x += s.config.adjustedResolution

		var y int
		clr := green
		logClr := "green"
		if price.Price == stock.OpenPrice {
			y = midY
		} else if price.Price > stock.OpenPrice {
			clr = green
			y = midY - int(math.Ceil((price.Price-stock.OpenPrice)/deviator))
			if y == midY {
				y--
			}
		} else {
			clr = red
			logClr = "red"
			y = midY + int(math.Ceil((stock.OpenPrice-price.Price)/deviator))
			if y == midY {
				y++
			}
		}

		if s.config.adjustedResolution > 1 {
			s.fillChartGaps(img, midY, image.Pt(lastX, lastY), image.Pt(x, y))
		}

		img.Set(x, y, clr)
		s.log.Debug("set pt",
			zap.Int("X", x),
			zap.Int("Y", y),
			zap.Int("midY", midY),
			zap.Float64("price", price.Price),
			zap.String("color", logClr),
		)

		if y > midY {
			for thisY := y; thisY > midY; thisY-- {
				if thisY == y {
					img.Set(x, thisY, red)
				} else {
					img.Set(x, thisY, lightRed)
				}
			}
		} else {
			for thisY := y; thisY <= midY; thisY++ {
				if thisY == y {
					img.Set(x, thisY, green)
				} else {
					img.Set(x, thisY, lightGreen)
				}
			}
		}
		lastY = y
	}

	return img
}

// fillChartGaps fills in a draw.Image chart with a mid line Y value with corresponding
// colors above/below the mid line
func (s *StockBoard) fillChartGaps(img draw.Image, midY int, previous image.Point, current image.Point) {
	lastX := previous.X
	lastY := previous.Y
	x := current.X
	y := current.Y

	diff := y - lastY
	steps := x - lastX
	fill := (diff / steps) + 1
	multiplier := -1
	if diff < 0 {
		fill = ((-1 * diff) / steps) + 1
		multiplier = 1
	}
	thisY := y + fill
	if y > midY {
		thisY = y - fill
	}

	s.log.Debug("fill chart gaps",
		zap.Int("prevX", previous.X),
		zap.Int("prevY", previous.Y),
		zap.Int("X", current.X),
		zap.Int("Y", current.X),
		zap.Int("steps", steps),
		zap.Int("fill", fill),
		zap.Int("multiplier", multiplier),
		zap.Int("thisX", x-1),
		zap.Int("thisY", thisY),
	)
	for thisX := x - 1; thisX > lastX; thisX-- {
		if thisY <= midY {
			for myY := thisY; myY <= midY; myY++ {
				if myY == thisY {
					img.Set(thisX, myY, green)
					s.log.Debug("fill",
						zap.Int("X", thisX),
						zap.Int("Y", myY),
						zap.String("color", "green"),
					)
				} else {
					img.Set(thisX, myY, lightGreen)
				}
			}
		} else {
			for myY := thisY; myY > midY; myY-- {
				if myY == thisY {
					img.Set(thisX, myY, red)
					s.log.Debug("fill",
						zap.Int("X", thisX),
						zap.Int("Y", myY),
						zap.String("color", "red"),
					)
				} else {
					img.Set(thisX, myY, lightRed)
				}
			}
		}
		thisY = thisY + (multiplier * fill)
	}
}
