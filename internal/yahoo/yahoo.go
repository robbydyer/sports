package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/robfig/cron/v3"

	stockboard "github.com/robbydyer/sports/internal/board/stocks"
)

const baseURL = "https://query2.finance.yahoo.com"

// API is used for accessing the Yahoo Finance API
type API struct {
	log               *zap.Logger
	cache             map[string]*cache
	cacheLock         *sync.RWMutex
	afterHoursUpdated *atomic.Bool
}

type cache struct {
	time  time.Time
	stock *stockboard.Stock
}

type chartDat struct {
	Chart *struct {
		Result []struct {
			Meta       *chart  `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators *struct {
				Quote []*struct {
					Close []*float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

type chart struct {
	Symbol             string  `json:"symbol"`
	RegularMarketPrice float64 `json:"regularMarketPrice"`
	ChartPreviousClose float64 `json:"chartPreviousClose"`
}

// New ...
func New(log *zap.Logger) (*API, error) {
	a := &API{
		log:               log,
		cache:             make(map[string]*cache),
		cacheLock:         &sync.RWMutex{},
		afterHoursUpdated: atomic.NewBool(false),
	}

	c := cron.New()

	if _, err := c.AddFunc("00 01 * * *", func() {
		a.afterHoursUpdated.Store(false)
	}); err != nil {
		return nil, fmt.Errorf("failed to set cron for afterHoursUpdated: %w", err)
	}

	return a, nil
}

// Get fetch data about a list of given stock symbols
func (a *API) Get(ctx context.Context, symbols []string, interval time.Duration) ([]*stockboard.Stock, error) {
	if interval.Hours() > 1 || interval.Minutes() > 60 {
		interval = interval.Truncate(1 * time.Hour)
	}
	interval = interval.Truncate(1 * time.Minute)

	stocks := []*stockboard.Stock{}

	a.log.Debug("get stock",
		zap.Duration("interval", interval),
	)

	for _, s := range symbols {
		stock, err := a.getTicker(ctx, s, interval)
		if err != nil {
			a.log.Error("error pulling stock info",
				zap.Error(err),
				zap.String("symbol", s),
			)
			continue
		}
		stocks = append(stocks, stock)
	}

	return stocks, nil
}

// TradingOpen ...
func (a *API) TradingOpen() (time.Time, error) {
	return tradingBegin()
}

// TradingClose ...
func (a *API) TradingClose() (time.Time, error) {
	return tradingEnd()
}

func (a *API) getTicker(ctx context.Context, ticker string, interval time.Duration) (*stockboard.Stock, error) {
	cacheExpire := interval
	if stock := a.getCache(ticker, cacheExpire); stock != nil {
		a.log.Debug("get stock from cache",
			zap.String("symbol", ticker),
		)
		return stock, nil
	}

	uri, err := url.Parse(fmt.Sprintf("%s/v8/finance/chart/%s", baseURL, ticker))
	if err != nil {
		return nil, err
	}

	v := uri.Query()
	v.Set("interval", durationToAPIInterval(interval))
	v.Set("period", "1d")
	// v.Set("period1", fmt.Sprintf("%d", a.start.Unix()))
	// v.Set("period2", fmt.Sprintf("%d", time.Now().Local().Unix()))

	uri.RawQuery = v.Encode()

	a.log.Debug("get stock data from API",
		zap.String("url", uri.String()),
		zap.Duration("interval", interval),
	)
	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, err
	}
	client := http.DefaultClient

	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var c *chartDat

	if err := json.Unmarshal(body, &c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chart data: %w", err)
	}

	stock, err := a.stockFromData(c)
	if err != nil {
		return nil, err
	}

	a.setCache(stock)

	return stock, nil
}

func (a *API) stockFromData(data *chartDat) (*stockboard.Stock, error) {
	if len(data.Chart.Result) < 1 {
		return nil, fmt.Errorf("no data found")
	}
	c := data.Chart.Result[0]
	s := &stockboard.Stock{
		Symbol:    c.Meta.Symbol,
		OpenPrice: c.Meta.ChartPreviousClose,
		Price:     c.Meta.RegularMarketPrice,
	}

	s.Change = ((s.Price - s.OpenPrice) / s.OpenPrice) * 100.0

	fltPtr := func(f float64) *float64 {
		return &f
	}

	lastPrice := s.OpenPrice
	for i, ts := range c.Timestamp {
		t := time.Unix(ts, 0)

		price := c.Indicators.Quote[0].Close[i]
		// If price doesn't change from previous period, it returns as `null`
		if price == nil {
			price = fltPtr(lastPrice)
		} else {
			lastPrice = *price
		}

		p := &stockboard.Price{
			Time:  t,
			Price: *price,
		}

		a.log.Debug("add price",
			zap.String("symbol", s.Symbol),
			zap.Float64("price", p.Price),
		)

		s.Prices = append(s.Prices, p)
	}

	// sort prices
	sort.SliceStable(s.Prices, func(i int, j int) bool {
		return s.Prices[i].Time.Before(s.Prices[j].Time)
	})

	return s, nil
}
