package openweather

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"go.uber.org/zap"
)

type geo struct {
	Name    string  `json:"name"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Country string  `json:"country"`
	State   string  `json:"state,omitempty"`
}

func (a *API) getLocation(ctx context.Context, zipCode string, country string) (*geo, error) {
	gKey := fmt.Sprintf("%s_%s", zipCode, country)
	a.geoLock.RLock()
	if g, ok := a.coordinates[gKey]; ok && g != nil {
		a.geoLock.RUnlock()
		return g, nil
	}
	a.geoLock.RUnlock()

	uri, err := url.Parse(fmt.Sprintf("%s/geo/1.0/zip", baseURL))
	if err != nil {
		return nil, err
	}

	v := uri.Query()
	v.Set("zip", fmt.Sprintf("%s,%s", zipCode, country))
	v.Set("appid", a.apiKey)

	uri.RawQuery = v.Encode()

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	client := http.DefaultClient

	a.log.Info("querying geolocation",
		zap.String("url", uri.String()),
	)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var g *geo

	if err := json.Unmarshal(body, &g); err != nil {
		return nil, err
	}

	if g == nil {
		a.log.Error("failed to get geolocation",
			zap.String("zip", zipCode),
			zap.String("country", country),
			zap.ByteString("geo data", body),
		)

		return nil, fmt.Errorf("failed to get geolocation")
	}

	a.geoLock.Lock()
	defer a.geoLock.Unlock()
	a.coordinates[gKey] = g

	return g, nil
}
