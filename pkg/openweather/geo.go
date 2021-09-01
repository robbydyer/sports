package openweather

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type geo struct {
	Name    string  `json:"name"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Country string  `json:"country"`
	State   string  `json:"state"`
}

func (a *API) getLocation(ctx context.Context, city string, state string, country string) (*geo, error) {
	gKey := fmt.Sprintf("%s_%s", city, state)
	a.geoLock.RLock()
	if g, ok := a.coordinates[gKey]; ok {
		a.geoLock.RUnlock()
		return g, nil
	}
	a.geoLock.RUnlock()

	uri, err := url.Parse(fmt.Sprintf("%s/geo/1.0/direct", baseURL))
	if err != nil {
		return nil, err
	}

	v := uri.Query()
	v.Set("q", fmt.Sprintf("%s,%s,%s", city, state, country))
	v.Set("appid", a.apiKey)

	uri.RawQuery = v.Encode()

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	client := http.DefaultClient

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var geos []*geo

	if err := json.Unmarshal(body, &geos); err != nil {
		return nil, err
	}

	if len(geos) < 1 {
		return nil, fmt.Errorf("failed to get location for %s, %s", city, state)
	}

	var g *geo
	for _, geo := range geos {
		if strings.ToLower(geo.Country) == strings.ToLower(country) && strings.ToLower(geo.State) == strings.ToLower(state) {
			g = geo
		}
	}

	a.geoLock.Lock()
	defer a.geoLock.Unlock()
	a.coordinates[gKey] = g

	return g, nil
}
