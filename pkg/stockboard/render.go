package stockboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *StockBoard) renderStock(ctx context.Context, stock *Stock, canvas board.Canvas) error {
	canvasBounds := rgbrender.ZeroedBounds(canvas.Bounds())
	chartWidth, _ := s.chartWidth(canvasBounds.Dx())
	chartBounds := rgbrender.ZeroedBounds(image.Rect(canvasBounds.Min.X, canvasBounds.Max.Y/2, chartWidth+canvasBounds.Min.X, canvasBounds.Max.Y))
	symbolBounds := rgbrender.ZeroedBounds(image.Rect(canvasBounds.Min.X, canvasBounds.Min.Y, canvasBounds.Max.X/2, canvasBounds.Max.Y/2))
	priceBounds := rgbrender.ZeroedBounds(image.Rect(canvasBounds.Max.X/2, canvasBounds.Min.Y, canvasBounds.Max.X, canvasBounds.Max.Y/2))

	chartPrices := s.getChartPrices(chartBounds.Dx()/s.config.ChartResolution, stock)

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
	if err := symbolWriter.WriteAligned(
		rgbrender.CenterCenter,
		canvas,
		symbolBounds,
		[]string{symbol},
		color.White,
	); err != nil {
		return err
	}

	if err := priceWriter.WriteAligned(
		rgbrender.RightCenter,
		canvas,
		priceBounds,
		[]string{
			fmt.Sprintf("%.2f ", stock.Price),
			fmt.Sprintf("%.2f%% ", stock.Change),
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

	s.log.Debug("draw chart",
		zap.Int("mid", midY),
		zap.Float64("open price", stock.OpenPrice),
		zap.Float64("max price", maxPrice.Price),
		zap.Float64("min price", minPrice.Price),
		zap.Float64("deviator", deviator),
	)

	x := bounds.Min.X - 1
	lastY := midY

	resolution := s.config.ChartResolution

	if len(prices) < bounds.Dx()/resolution {
		resolution = int(math.Ceil(float64(bounds.Dx()) / float64(len(prices))))
	}

	for _, price := range prices {
		lastX := x
		x += resolution

		var y int
		clr := green
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
			y = midY + int(math.Ceil((stock.OpenPrice-price.Price)/deviator))
			if y == midY {
				y++
			}
		}

		if bounds.Dx()/resolution != bounds.Dx() && resolution > 1 {
			fillChartGaps(img, midY, image.Pt(lastX, lastY), image.Pt(x, y))
		}

		img.Set(x, y, clr)
		s.log.Debug("set pt",
			zap.Int("X", x),
			zap.Int("Y", y),
			zap.Float64("price", price.Price),
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
func fillChartGaps(img draw.Image, midY int, previous image.Point, current image.Point) {
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
	for thisX := x - 1; thisX > lastX; thisX-- {
		if thisY <= midY {
			for myY := thisY; myY <= midY; myY++ {
				if myY == thisY {
					img.Set(thisX, myY, green)
				} else {
					img.Set(thisX, myY, lightGreen)
				}
			}
		} else {
			for myY := thisY; myY > midY; myY-- {
				if myY == thisY {
					img.Set(thisX, myY, red)
				} else {
					img.Set(thisX, myY, lightRed)
				}
			}
		}
		thisY = thisY + (multiplier * fill)
	}
}
