package stockboard

import (
	"fmt"
	"image"
	"math"

	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *Stock) minPrice() *Price {
	if len(s.Prices) < 1 {
		return nil
	}
	min := s.Prices[0]
	for _, price := range s.Prices {
		if price.Price < min.Price && price.Price > 0 {
			min = price
		}
	}

	return min
}

func (s *Stock) maxPrice() *Price {
	if len(s.Prices) < 1 {
		return nil
	}
	max := s.Prices[0]
	for _, price := range s.Prices {
		if price.Price > max.Price {
			max = price
		}
	}

	return max
}

func (s *StockBoard) getChartPrices(maxPix int, stock *Stock) []*Price {
	if len(stock.Prices) < maxPix || maxPix < 1 {
		return stock.Prices
	}

	granularity := len(stock.Prices) / maxPix
	ret := make([]*Price, maxPix)

	// For accurate chart, make sure open, min and max are in sample
	startX := 1
	ret[0] = stock.Prices[0]
	if len(stock.Prices) >= 2 {
		ret[1] = stock.minPrice()
		ret[2] = stock.maxPrice()
		startX = 3
	}

	for i := startX; i < maxPix-1; i++ {
		if len(stock.Prices) < i+granularity {
			break
		}
		ret[i] = stock.Prices[i+granularity]
	}

	ret[maxPix-1] = stock.Prices[len(stock.Prices)-1]

	return ret
}

func (s *StockBoard) getPriceWriter(canvasBounds image.Rectangle) (*rgbrender.TextWriter, error) {
	s.Lock()
	defer s.Unlock()

	if s.priceWriter != nil {
		return s.priceWriter, nil
	}
	bounds := rgbrender.ZeroedBounds(canvasBounds)

	writer, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return nil, err
	}

	if bounds.Dy() <= 256 {
		writer.FontSize = 8.0
		writer.YStartCorrection = -2
	} else {
		writer.FontSize = 0.25 * float64(bounds.Dy())
		writer.YStartCorrection = -1 * ((bounds.Dy() / 32) + 1)
	}

	s.priceWriter = writer
	return writer, nil
}

func (s *StockBoard) getSymbolWriter(canvasBounds image.Rectangle) (*rgbrender.TextWriter, error) {
	s.Lock()
	defer s.Unlock()

	if s.symbolWriter != nil {
		return s.symbolWriter, nil
	}
	bounds := rgbrender.ZeroedBounds(canvasBounds)

	var writer *rgbrender.TextWriter

	if (bounds.Dx() == bounds.Dy()) && bounds.Dx() <= 32 {
		var err error
		writer, err = rgbrender.DefaultTextWriter()
		if err != nil {
			return nil, err
		}
		writer.FontSize = 0.25 * float64(bounds.Dy())
		writer.YStartCorrection = -1 * ((bounds.Dy() / 32) + 1)
	} else {
		fnt, err := rgbrender.GetFont("score.ttf")
		if err != nil {
			return nil, fmt.Errorf("failed to load font for symbol: %w", err)
		}
		size := 0.5 * float64(bounds.Dy())
		writer = rgbrender.NewTextWriter(fnt, size)
		yCorrect := math.Ceil(float64(3.0/32.0) * float64(bounds.Dy()))
		writer.YStartCorrection = int(yCorrect * -1)
	}

	s.symbolWriter = writer
	return writer, nil
}

func (s *StockBoard) specialName(symbol string) string {
	switch symbol {
	case "^GSPC":
		return "S&P500"
	case "^IXIC":
		return "NASDAQ"
	case "^DJI":
		return "DOW"
	default:
		return symbol
	}
}
