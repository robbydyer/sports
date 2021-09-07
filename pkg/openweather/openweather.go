package openweather

import (
	"context"
	"fmt"
	"image"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/util"
	"github.com/robbydyer/sports/pkg/weatherboard"
)

const (
	baseURL = "https://api.openweathermap.org"
	imgURL  = "http://openweathermap.org/img/wn"
)

// API ...
type API struct {
	log          *zap.Logger
	apiKey       string
	icons        map[string]*logo.Logo
	refresh      time.Duration
	coordinates  map[string]*geo
	geoLock      sync.RWMutex
	forecastLock sync.RWMutex
	cache        map[string]*weather
	lastAPICall  *time.Time
	callLimit    time.Duration
	sync.Mutex
}

type weather struct {
	lastUpdate time.Time
	Current    *forecast   `json:"current"`
	Hourly     []*forecast `json:"hourly"`
	Daily      []*daily    `json:"daily"`
}

type baseForecast struct {
	Dt      int `json:"dt"`
	Weather []*struct {
		ID   int    `json:"id"`
		Icon string `json:"icon"`
	} `json:"weather"`
	Humidity int      `json:"humidity"`
	Pop      *float64 `json:"pop"`
}

type daily struct {
	baseForecast
	Temp *struct {
		Day   float64 `json:"day"`
		Min   float64 `json:"min"`
		Max   float64 `json:"max"`
		Night float64 `json:"night"`
		Eve   float64 `json:"eve"`
		Morn  float64 `json:"morn"`
	} `json:"temp"`
}

type forecast struct {
	baseForecast
	Temp     float64 `json:"temp"`
	isHourly bool
}

// New ...
func New(apiKey string, refresh time.Duration, log *zap.Logger) (*API, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("must pass a valid API key from openweathermap.org")
	}
	a := &API{
		apiKey:      apiKey,
		log:         log,
		icons:       make(map[string]*logo.Logo),
		refresh:     refresh,
		coordinates: make(map[string]*geo),
		callLimit:   30 * time.Minute,
		cache:       make(map[string]*weather),
	}

	return a, nil
}

// CacheClear ...
func (a *API) CacheClear() {
}

func weatherKey(zipCode string, country string) string {
	return fmt.Sprintf("%s_%s", zipCode, country)
}

// CurrentForecast ...
func (a *API) CurrentForecast(ctx context.Context, zipCode string, country string, bounds image.Rectangle) (*weatherboard.Forecast, error) {
	w, err := a.getWeather(ctx, zipCode, country)
	if err != nil {
		return nil, err
	}

	forecasts, err := a.boardForecastFromForecast([]*forecast{w.Current}, bounds)
	if err != nil {
		return nil, err
	}

	if len(forecasts) < 1 {
		return nil, fmt.Errorf("could not get forecast from data")
	}

	return forecasts[0], nil
}

// DailyForecasts ...
func (a *API) DailyForecasts(ctx context.Context, zipCode string, country string, bounds image.Rectangle) ([]*weatherboard.Forecast, error) {
	w, err := a.getWeather(ctx, zipCode, country)
	if err != nil {
		return nil, err
	}

	return a.boardForecastFromDaily(w.Daily, bounds)
}

// HourlyForecasts ...
func (a *API) HourlyForecasts(ctx context.Context, zipCode string, country string, bounds image.Rectangle) ([]*weatherboard.Forecast, error) {
	w, err := a.getWeather(ctx, zipCode, country)
	if err != nil {
		return nil, err
	}

	return a.boardForecastFromForecast(w.Hourly, bounds)
}

func (a *API) getIcon(icon string, bounds image.Rectangle) (*logo.Logo, error) {
	a.Lock()
	defer a.Unlock()

	url := ""
	key := ""
	if icon != "" {
		url = fmt.Sprintf("%s/%s@4x.png", imgURL, icon)
		key = fmt.Sprintf("%s_%dx%d", icon, bounds.Dx(), bounds.Dy())
	}

	a.log.Debug("fetching weather icon",
		zap.String("key", key),
	)

	if i, ok := a.icons[key]; ok {
		return i, nil
	}

	logoGetter := func(ctx context.Context) (image.Image, error) {
		return util.PullPng(ctx, url)
	}

	d, err := cacheDir()
	if err != nil {
		return nil, err
	}

	l := logo.New(key, logoGetter, d, bounds, &logo.Config{
		Abbrev: key,
		XSize:  bounds.Dx(),
		YSize:  bounds.Dy(),
		Pt: &logo.Pt{
			X:    0,
			Y:    0,
			Zoom: 1,
		},
	})

	a.icons[key] = l

	return l, nil
}

func cacheDir() (string, error) {
	d := "/tmp/sportsmatrix_logos/weather"
	if _, err := os.Stat(d); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0755); err != nil {
				return "", err
			}
			return d, nil
		}
	}
	return d, nil
}
