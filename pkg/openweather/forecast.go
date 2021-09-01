package openweather

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/robbydyer/sports/pkg/weatherboard"
	"go.uber.org/zap"
)

func (w *weather) expired(refresh time.Duration) bool {
	if w.lastUpdate.Add(refresh).Before(w.lastUpdate) {
		return false
	}

	return true
}

func (a *API) boardForecast(ctx context.Context, f *forecast, bounds image.Rectangle) (*weatherboard.Forecast, error) {
	icon, err := a.getIcon(ctx, f.Weather[0].Icon, bounds)
	if err != nil {
		return nil, err
	}
	w := &weatherboard.Forecast{
		Time:        time.Unix(int64(f.Dt), 0),
		Temperature: f.Temp,
		Humidity:    f.Humidity,
		TempUnit:    "F",
		Icon:        icon,
	}

	return w, nil
}

func (a *API) weatherFromCache(key string) *weather {
	a.forecastLock.RLock()
	defer a.forecastLock.RUnlock()

	w, ok := a.cache[key]
	if ok {
		return w
	}

	return nil
}

func (a *API) setWeatherCache(key string, w *weather) {
	a.forecastLock.Lock()
	defer a.forecastLock.Unlock()

	a.cache[key] = w
}

func (a *API) getWeather(ctx context.Context, city string, state string, country string, bounds image.Rectangle) (*weather, error) {
	g, err := a.getLocation(ctx, city, state, country)
	if err != nil {
		return nil, err
	}

	uri, err := url.Parse(fmt.Sprintf("%s/data/2.5/onecall", baseURL))
	if err != nil {
		return nil, err
	}

	v := uri.Query()
	v.Set("appid", a.apiKey)
	//v.Set("id", cityID)
	v.Set("units", "imperial")
	v.Set("lat", fmt.Sprintf("%f", g.Lat))
	v.Set("lon", fmt.Sprintf("%f", g.Lon))
	v.Set("exclude", "minutely")

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

	var w *weather

	if err := json.Unmarshal(body, &w); err != nil {
		return nil, err
	}

	return w, nil
}
