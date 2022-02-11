package weatherboard

import (
	"bytes"
	"context"
	"embed"
	_ "embed"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/robbydyer/sports/pkg/logo"
)

//go:embed assets
var assets embed.FS

func cacheDir() (string, error) {
	d := "/tmp/sportsmatrix_logos/weathericons"
	if _, err := os.Stat(d); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0o755); err != nil {
				return "", err
			}
			return d, nil
		}
	}
	return d, nil
}

func customImgSource(iconCode string) (logo.SourceGetter, error) {
	f := ""

	// These conditions match https://openweathermap.org/weather-conditions
	switch strings.ToLower(iconCode) {
	case "01d":
		f = "sun.png"
	case "01n":
		f = "moon.png"
	case "02d":
		f = "partcloud.png"
	case "02n":
		f = "partcloud.png"
	case "03d", "03n", "04d", "04n":
		f = "cloudy.png"
	case "09d", "09n", "10d", "10n":
		f = "rain.png"
	case "11d", "11n":
		f = "storm.png"
	case "13d", "13n":
		f = "snowflake.png"
	case "50d", "50n":
		f = "mist.png"
	default:
		return nil, fmt.Errorf("no custom img source for %s", iconCode)
	}

	b, err := assets.ReadFile(filepath.Join("assets", f))
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(b)

	return func(ctx context.Context) (image.Image, error) {
		return png.Decode(r)
	}, nil
}

func (w *WeatherBoard) customIcon(iconCode string, bounds image.Rectangle) (*logo.Logo, error) {
	w.iconLock.Lock()
	defer w.iconLock.Unlock()

	key := fmt.Sprintf("%s_cust_%dx%d", iconCode, bounds.Dx(), bounds.Dy())

	if i, ok := w.iconCache[key]; ok {
		return i, nil
	}

	getter, err := customImgSource(iconCode)
	if err != nil {
		return nil, err
	}

	d, err := cacheDir()
	if err != nil {
		return nil, err
	}

	l := logo.New(key, getter, d, bounds, &logo.Config{
		Abbrev: key,
		XSize:  bounds.Dx(),
		YSize:  bounds.Dy(),
		Pt: &logo.Pt{
			X:    0,
			Y:    0,
			Zoom: 1,
		},
	})

	l.SetLogger(w.log)

	w.iconCache[key] = l

	return l, nil
}
