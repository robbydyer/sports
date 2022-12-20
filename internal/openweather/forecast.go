package openweather

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	weatherboard "github.com/robbydyer/sports/internal/board/weather"
	"github.com/robbydyer/sports/internal/rgbrender"
	"github.com/robbydyer/sports/internal/util"
)

func (w *weather) expired(refresh time.Duration) bool {
	return w.LastUpdate.Add(refresh).Before(time.Now().Local())
}

func (a *API) boardForecastFromForecast(forecasts []*forecast, bounds image.Rectangle, metric bool) ([]*weatherboard.Forecast, error) {
	var ws []*weatherboard.Forecast

	for _, f := range forecasts {
		if f == nil || len(f.Weather) < 1 {
			return nil, fmt.Errorf("no weather found in forecast")
		}
		icon, err := a.getIcon(f.Weather[0].Icon, rgbrender.ZeroedBounds(bounds))
		if err != nil {
			return nil, err
		}
		w := &weatherboard.Forecast{
			Time:        time.Unix(int64(f.Dt), 0),
			Temperature: &f.Temp,
			Humidity:    f.Humidity,
			TempUnit:    "F",
			Icon:        icon,
			IconCode:    f.Weather[0].Icon,
			IsHourly:    f.isHourly,
		}

		if metric {
			w.TempUnit = "C"
		}

		if f.Pop != nil {
			c := int(*f.Pop * 100)
			w.PrecipChance = &c
		}
		ws = append(ws, w)
	}

	return ws, nil
}

func (a *API) boardForecastFromDaily(forecasts []*daily, bounds image.Rectangle, metric bool) ([]*weatherboard.Forecast, error) {
	var ws []*weatherboard.Forecast

	for _, f := range forecasts {
		if f.Weather == nil || len(f.Weather) < 1 {
			continue
		}
		icon, err := a.getIcon(f.Weather[0].Icon, rgbrender.ZeroedBounds(bounds))
		if err != nil {
			return nil, err
		}
		w := &weatherboard.Forecast{
			Time:     time.Unix(int64(f.Dt), 0),
			HighTemp: &f.Temp.Max,
			LowTemp:  &f.Temp.Min,
			Humidity: f.Humidity,
			TempUnit: "F",
			Icon:     icon,
			IconCode: f.Weather[0].Icon,
		}

		if metric {
			w.TempUnit = "C"
		}

		ws = append(ws, w)
		if f.Pop != nil {
			c := int(*f.Pop * 100)
			w.PrecipChance = &c
		}
	}

	return ws, nil
}

func (a *API) weatherFromCache(key string) *weather {
	a.forecastLock.RLock()
	defer a.forecastLock.RUnlock()

	w, ok := a.cache[key]
	if ok {
		return w
	}

	// Try disk cache
	cDir, err := cacheDir(dataCacheDir)
	if err != nil {
		a.log.Error("failed to get cache dir for weather",
			zap.Error(err),
		)
		return nil
	}

	cacheFile := filepath.Join(cDir, key)
	exists, err := util.FileExists(cacheFile)
	if err != nil {
		a.log.Error("failed to read weather cache file",
			zap.String("file", cacheFile),
			zap.Error(err),
		)
		return nil
	}
	if exists {
		dat, err := os.ReadFile(cacheFile)
		if err != nil {
			a.log.Error("failed to read weather cache file",
				zap.String("file", cacheFile),
				zap.Error(err),
			)
			return nil
		}
		var w *weather

		if err := json.Unmarshal(dat, &w); err != nil {
			a.log.Error("failed to unmarshal cached weather data",
				zap.String("file", cacheFile),
				zap.Error(err),
			)
		}
		if w == nil {
			a.log.Error("weather cache data was nil",
				zap.String("file", cacheFile),
			)

			return w
		}

		a.log.Info("got weather data from cache file",
			zap.String("file", cacheFile),
			zap.String("key", key),
		)

		// cache to memory so we don't read the file next time
		a.cache[key] = w
		return w
	}

	a.log.Error("cache miss on weather",
		zap.String("key", key),
	)

	return nil
}

func (a *API) setWeatherCache(key string, w *weather) {
	a.forecastLock.Lock()
	defer a.forecastLock.Unlock()

	// Save cache to file
	cDir, err := cacheDir(dataCacheDir)
	if err != nil {
		a.log.Error("failed to get weather data cache dir",
			zap.Error(err),
		)
		a.cache[key] = w
		return
	}

	dat, err := json.Marshal(w)
	if err != nil {
		a.log.Error("failed marshal weather for caching",
			zap.Error(err),
		)
	} else {
		cacheFile := filepath.Join(cDir, key)
		if err := os.WriteFile(cacheFile, dat, 0o644); err != nil {
			a.log.Error("failed to write weather cache to file",
				zap.Error(err),
			)
		}
		a.log.Info("wrote weather data to cache file",
			zap.String("file", cacheFile),
		)
	}

	a.cache[key] = w
}

func (a *API) getWeather(ctx context.Context, zipCode string, country string, metric bool) (*weather, error) {
	var w *weather
	key := weatherKey(zipCode, country)
	w = a.weatherFromCache(key)
	if w != nil {
		// Check if cache expired
		if w.expired(a.refresh) {
			a.log.Info("weather cache expired",
				zap.Float64("minutes", a.refresh.Minutes()),
				zap.Time("updated", w.LastUpdate),
			)
			w = nil
		} else {
			a.log.Info("using weather data from cache",
				zap.String("key", key),
			)
			return w, nil
		}
	}

	if a.lastAPICall == nil {
		t := time.Now().Local()
		a.lastAPICall = &t
	} else {
		if a.lastAPICall.Add(a.callLimit).After(time.Now().Local()) {
			a.log.Info("refusing weather API call",
				zap.Time("last call", *a.lastAPICall),
				zap.Duration("timeout", a.callLimit),
			)

			return nil, fmt.Errorf("refusing weather API call")
		}
	}

	g, err := a.getLocation(ctx, zipCode, country)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, fmt.Errorf("failed to geolocate")
	}

	uri, err := url.Parse(fmt.Sprintf("%s/data/%s/onecall", baseURL, a.apiVersion))
	if err != nil {
		return nil, err
	}

	v := uri.Query()
	v.Set("appid", a.apiKey)
	// v.Set("id", cityID)
	if metric {
		v.Set("units", "metric")
	} else {
		v.Set("units", "imperial")
	}
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		a.log.Error("failed to auth to openweathermap.org",
			zap.Int("status", resp.StatusCode),
			zap.String("status message", resp.Status),
			zap.ByteString("data", body),
		)
		return nil, fmt.Errorf("failed to get weather data")
	}

	a.log.Debug("weather data",
		zap.ByteString("data", body),
	)

	if err := json.Unmarshal(body, &w); err != nil {
		return nil, err
	}

	t := time.Now().Local()
	a.lastAPICall = &t

	w.LastUpdate = time.Now().Local()

	a.setWeatherCache(key, w)

	for _, f := range w.Hourly {
		f.isHourly = true
	}

	return w, nil
}
