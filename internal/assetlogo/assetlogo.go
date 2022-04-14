package assetlogo

import (
	"bytes"
	"context"
	"embed"
	"image"
	"path/filepath"

	"github.com/disintegration/imaging"

	"github.com/robbydyer/sports/internal/logo"
)

//go:embed assets
var assets embed.FS

// GetLogo gets a logo based on a filename of an asset
func GetLogo(fileName string, bounds image.Rectangle) (*logo.Logo, error) {
	getter := func(ctx context.Context) (image.Image, error) {
		b, err := assets.ReadFile(filepath.Join("assets", fileName))
		if err != nil {
			return nil, err
		}

		reader := bytes.NewReader(b)
		return imaging.Decode(reader)
	}
	return logo.New(
		fileName,
		getter,
		"/tmp/sportsmatrix/assetlogos",
		bounds,
		&logo.Config{
			FitImage: true,
			Abbrev:   fileName,
			XSize:    bounds.Dx(),
			YSize:    bounds.Dy(),
			Pt: &logo.Pt{
				Zoom: 1,
			},
		},
	), nil
}
