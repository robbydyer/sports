package openweather

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
	"net/url"
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
	log       *zap.Logger
	apiKey    string
	icons     map[string]image.Image
	current   *weatherboard.Forecast
	lastCheck time.Time
	refresh   time.Duration
	sync.Mutex
}

type forecast struct {
	Weather []*struct {
		ID   int    `json:"id"`
		Icon string `json:"icon"`
	} `json:"weather"`
	Main *struct {
		Temp     float64 `json:"temp"`
		Humidity int     `json:"humidity"`
	} `json:"main"`
}

func New(apiKey string, refresh time.Duration, log *zap.Logger) (*API, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("must pass a valid API key from openweathermap.org")
	}
	a := &API{
		apiKey:  apiKey,
		log:     log,
		icons:   make(map[string]image.Image),
		refresh: refresh,
	}

	return a, nil
}

func (a *API) CacheClear() {
	a.current = nil
}

func (a *API) CurrentForecast(ctx context.Context, cityID string, bounds image.Rectangle) (*weatherboard.Forecast, error) {
	if a.current != nil && !time.Now().Local().Add(a.refresh).Before(time.Now().Local()) {
		a.log.Debug("using cached weather data")
		return a.current, nil
	}

	uri, err := url.Parse(fmt.Sprintf("%s/data/2.5/weather", baseURL))
	if err != nil {
		return nil, err
	}

	v := uri.Query()
	v.Set("appid", a.apiKey)
	v.Set("id", cityID)
	v.Set("units", "imperial")

	uri.RawQuery = v.Encode()

	a.log.Debug("fetching weather from API",
		zap.String("url", uri.String()),
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

	var w *forecast

	if err := json.Unmarshal(body, &w); err != nil {
		return nil, err
	}

	weather := &weatherboard.Forecast{
		TempUnit: "F",
	}

	if w.Main != nil {
		weather.Temperature = w.Main.Temp
	}

	if len(w.Weather) > 0 {
		weather.Icon, err = a.getIcon(ctx, w.Weather[0].Icon, bounds)
		if err != nil {
			return nil, err
		}
	}

	a.current = weather

	return weather, nil

}

func (a *API) forecastFromData(f *forecast) (*weatherboard.Forecast, error) {
	w := &weatherboard.Forecast{
		TempUnit: "F",
	}

	if f.Main != nil {
		w.Temperature = f.Main.Temp
	}

	return w, nil
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
