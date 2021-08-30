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
	if !ok {
		return nil
	}

	if c.time.Add(expire).Before(time.Now()) {
		a.log.Debug("cache expired",
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
