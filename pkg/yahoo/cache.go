package yahoo

import (
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/stockboard"
)

// CacheClear clears the cache
func (a *API) CacheClear() {
	for k := range a.cache {
		delete(a.cache, k)
	}
}

func (a *API) getCache(symbol string, expire time.Duration) *stockboard.Stock {
	a.cacheLock.RLock()
	defer a.cacheLock.RUnlock()

	c, ok := a.cache[symbol]
	if !ok || c == nil || c.stock == nil {
		return nil
	}

	// Don't expire cache before or after trading hours
	begin, beginErr := tradingBegin()
	if beginErr != nil {
		a.log.Error("failed to get trading day begin time",
			zap.Error(beginErr),
		)
	}
	end, endErr := tradingEnd()
	if endErr != nil {
		a.log.Error("failed to get trading day end time",
			zap.Error(endErr),
		)
	}

	if beginErr == nil && endErr == nil {
		t := time.Now()
		loc, err := tradingLocation()
		if err != nil {
			a.log.Error("failed to get trading day location",
				zap.Error(err),
			)
		} else {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
			if t.After(end) {
				// Do at least one update after trading hours end
				if !a.afterHoursUpdated.Load() {
					a.afterHoursUpdated.Store(true)
					return nil
				}
				a.log.Info("outside trading hours, not expiring cache",
					zap.Time("begin", begin),
					zap.Time("end", end),
					zap.Time("current", t),
				)
				return c.stock
			}
			if t.Before(begin) {
				a.log.Info("outside trading hours, not expiring cache",
					zap.Time("begin", begin),
					zap.Time("end", end),
					zap.Time("current", t),
				)
				return c.stock
			}
		}
	}

	if c.time.Add(expire).Before(time.Now()) {
		a.log.Info("cache expired",
			zap.String("symbol", symbol),
			zap.String("since", time.Since(c.time).String()),
		)
		return nil
	}

	return c.stock
}

func (a *API) setCache(stock *stockboard.Stock) {
	a.cacheLock.Lock()
	defer a.cacheLock.Unlock()

	a.cache[stock.Symbol] = &cache{
		time:  time.Now(),
		stock: stock,
	}
}
