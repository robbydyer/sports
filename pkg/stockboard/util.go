package stockboard

import (
	"fmt"
	"image"
	"math"
	"sort"

	"github.com/robbydyer/sports/pkg/rgbrender"
	"go.uber.org/zap"
)

func (s *Stock) minPrice() *Price {
	if len(s.Prices) < 1 {
		return nil
	}
	min := &Price{}
	for _, p := range s.Prices {
		if p != nil {
			min = p
			break
		}
	}

	for _, price := range s.Prices {
		if price != nil && price.Price < min.Price && price.Price > 0 {
			min = price
		}
	}

	return min
}

func (s *Stock) maxPrice() *Price {
	if len(s.Prices) < 1 {
		return nil
	}
	max := &Price{}
	for _, p := range s.Prices {
		if p != nil {
			max = p
			break
		}
	}

	for _, price := range s.Prices {
		if price != nil && price.Price > max.Price {
			max = price
		}
	}

	return max
}

func (s *StockBoard) getChartPrices(maxPix int, stock *Stock) []*Price {
	if maxPix < 1 {
		return nil
	}
	if len(stock.Prices) < maxPix {
		return stock.Prices
	}

	sampled := make(map[int64]*Price)

	latest := stock.Prices[0]
	for _, p := range stock.Prices {
		if latest.Time.Before(p.Time) {
			latest = p
		}
	}
	sampled[latest.Time.Unix()] = latest

	if len(sampled) >= maxPix || len(sampled) == len(stock.Prices) {
		return getSortedPriceList(sampled)
	}

	first := stock.Prices[0]
	for _, p := range stock.Prices {
		if first.Time.After(p.Time) {
			first = p
		}
	}
	sampled[first.Time.Unix()] = first

	if len(sampled) >= maxPix || len(sampled) == len(stock.Prices) {
		return getSortedPriceList(sampled)
	}

	// For accurate chart, make sure open, min and max are in sample
	sampled[stock.Prices[0].Time.Unix()] = stock.Prices[0]
	if len(stock.Prices) >= 2 && maxPix >= 2 {
		min := stock.minPrice()
		max := stock.maxPrice()
		s.log.Debug("stock min/max prices",
			zap.String("symbol", stock.Symbol),
			zap.Float64("min", min.Price),
			zap.Float64("max", max.Price),
		)
		sampled[max.Time.Unix()] = max
		if len(sampled) >= maxPix || len(sampled) == len(stock.Prices) {
			return getSortedPriceList(sampled)
		}
		sampled[min.Time.Unix()] = min
	}

	if len(sampled) >= maxPix || len(sampled) == len(stock.Prices) {
		return getSortedPriceList(sampled)
	}

	granularity := len(stock.Prices) / maxPix
	s.log.Debug("stock chart granularity",
		zap.String("symbol", stock.Symbol),
		zap.Int("granularity", granularity),
		zap.Int("num prices", len(stock.Prices)),
	)

	samples := func(start int) {
		for i := start; i < len(stock.Prices); i++ {
			if len(stock.Prices) < i+granularity || len(sampled) >= maxPix || len(sampled) == len(stock.Prices) {
				return
			}
			p := stock.Prices[i+granularity]
			if p == nil {
				continue
			}
			if _, ok := sampled[p.Time.Unix()]; ok {
				continue
			}
			sampled[p.Time.Unix()] = p
		}
	}

	for i := 0; i < granularity; i++ {
		if len(sampled) >= maxPix || len(sampled) == len(stock.Prices) {
			return getSortedPriceList(sampled)
		}
		samples(i * -1)
	}

	return getSortedPriceList(sampled)
}

func getSortedPriceList(sampled map[int64]*Price) []*Price {
	keys := []int64{}
	for k := range sampled {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	ret := make([]*Price, len(keys))
	for i := 0; i < len(keys); i++ {
		ret[i] = sampled[keys[i]]
	}

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
