package weatherboard

import (
	_ "embed"
	"os"
)

//go:embed assets/raindrop.png
var rainDrop []byte

func cacheDir() (string, error) {
	d := "/tmp/sportsmatrix_logos/weathericons"
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

/*
// raindDropSource implements logo.SourceGetter
func rainDropSource(ctx context.Context) (image.Image, error) {
	r := bytes.NewReader(rainDrop)
	return png.Decode(r)
}

func (w *WeatherBoard) rainDropIcon(ctx context.Context, bounds image.Rectangle) (image.Image, error) {
	w.iconLock.Lock()
	defer w.iconLock.Unlock()

	key := fmt.Sprintf("rainDrop_%dx%d", bounds.Dx(), bounds.Dy())

	if i, ok := w.iconCache[key]; ok {
		return i, nil
	}

	d, err := cacheDir()
	if err != nil {
		return nil, err
	}

	l := logo.New(key, rainDropSource, d, bounds, &logo.Config{
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

	w.iconCache[key] = i

	return i, nil
}
*/
