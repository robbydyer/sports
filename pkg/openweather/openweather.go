package openweather

import (
	"context"
	"fmt"
	"image"
	"os"
	"sync"
	"time"

	"github.com/robbydyer/sports/pkg/logo"
	"github.com/robbydyer/sports/pkg/util"
	"github.com/robbydyer/sports/pkg/weatherboard"
	"go.uber.org/zap"
)

const baseURL = "https://api.openweathermap.org"
const imgURL = "http://openweathermap.org/img/wn"

type API struct {
	log          *zap.Logger
	apiKey       string
	icons        map[string]image.Image
	refresh      time.Duration
	coordinates  map[string]*geo
	geoLock      sync.RWMutex
	forecastLock sync.RWMutex
	cache        map[string]*weather
	sync.Mutex
}

type weather struct {
	lastUpdate time.Time
	Current    *forecast `json:"current"`
	Hourly     *forecast `json:"hourly"`
	Daily      *forecast `json:"daily"`
}

type forecast struct {
	Dt      int `json:"dt"`
	Weather []*struct {
		ID   int    `json:"id"`
		Icon string `json:"icon"`
	} `json:"weather"`
	Temp     float64 `json:"temp"`
	Humidity int     `json:"humidity"`
}

func New(apiKey string, refresh time.Duration, log *zap.Logger) (*API, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("must pass a valid API key from openweathermap.org")
	}
	a := &API{
		apiKey:      apiKey,
		log:         log,
		icons:       make(map[string]image.Image),
		refresh:     refresh,
		coordinates: make(map[string]*geo),
	}

	return a, nil
}

func (a *API) CacheClear() {
}

func weatherKey(city string, state string, bounds image.Rectangle) string {
	return fmt.Sprintf("%s_%s_%dx%d", city, state, bounds.Dx(), bounds.Dy())
}

func (a *API) CurrentForecast(ctx context.Context, city string, state string, country string, bounds image.Rectangle) (*weatherboard.Forecast, error) {
	key := weatherKey(city, state, bounds)
	w := a.weatherFromCache(key)
	if w != nil && w.expired(a.refresh) {
		w = nil
	}

	if w == nil {
		var err error
		w, err = a.getWeather(ctx, city, state, country, bounds)
		if err != nil {
			return nil, err
		}
	}

	return a.boardForecast(ctx, w.Current, bounds)
}

func (a *API) getIcon(ctx context.Context, icon string, bounds image.Rectangle) (image.Image, error) {
	a.Lock()
	defer a.Unlock()

	url := ""
	key := ""
	if icon != "" {
		url = fmt.Sprintf("%s/%s@4x.png", imgURL, icon)
		key = fmt.Sprintf("%s_%dx%d", icon, bounds.Dx(), bounds.Dy())
	}

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

	i, err := l.GetThumbnail(ctx, bounds)
	if err != nil {
		return nil, err
	}

	a.icons[key] = i

	return i, nil
}

func cacheDir() (string, error) {
	d := "/tmp/sportsmatrix_logos/weather"
	if _, err := os.Stat(d); err != nil {
		if os.IsNotExist(err) {
			return d, os.MkdirAll(d, 0755)
		}
	}
	return d, nil
}
