package openweather

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/robbydyer/sports/pkg/weatherboard"
	"go.uber.org/zap"
)

const baseURL = "https://api.openweathermap.org"

type API struct {
	log    *zap.Logger
	apiKey string
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

func New(apiKey string, log *zap.Logger) (*API, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("must pass a valid API key from openweathermap.org")
	}
	a := &API{
		apiKey: apiKey,
		log:    log,
	}

	return a, nil
}

func (a *API) CurrentForecast(ctx context.Context, cityID string) (*weatherboard.Forecast, error) {
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

	return a.forecastFromData(w)
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
