package espnracing

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	"github.com/robbydyer/sports/pkg/logo"
	"go.uber.org/zap"
)

//go:embed assets
var assets embed.FS

const cacheDir = "/tmp/sportsmatrix/racing"

type API struct {
	myLogo   *image.Image
	leaguer  Leaguer
	schedule *Scoreboard
	log      *zap.Logger
}

type Leaguer interface {
	ShortName() string
	LogoSourceURL() string
	HTTPPathPrefix() string
	APIPath() string
	LogoAsset() string
}

func New(leaguer Leaguer, log *zap.Logger) (*API, error) {
	return &API{
		leaguer: leaguer,
		log:     log,
	}, nil
}

// GetLogo ...
func (a *API) GetLogo(ctx context.Context, matrixBounds image.Rectangle) (*logo.Logo, error) {
	return logo.New(
		a.leaguer.ShortName(),
		a.logoSourceGetter,
		cacheDir,
		matrixBounds,
		&logo.Config{
			FitImage: true,
			Abbrev:   a.leaguer.ShortName(),
			XSize:    matrixBounds.Dx(),
			YSize:    matrixBounds.Dy(),
			Pt: &logo.Pt{
				Zoom: 1,
			},
		},
	), nil
}

// LeagueShortName ...
func (a *API) LeagueShortName() string {
	return a.leaguer.ShortName()
}

func (a *API) logoSourceGetter(ctx context.Context) (image.Image, error) {
	if a.myLogo != nil {
		return *a.myLogo, nil
	}

	cacheFile := filepath.Join(cacheDir, a.leaguer.ShortName())

	if _, err := os.Stat(cacheFile); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to detect if logo cache exists: %w", err)
		}
	} else {
		return imaging.Open(cacheFile)
	}

	b, err := assets.ReadFile(filepath.Join("assets", a.leaguer.LogoAsset()))
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(b)
	return imaging.Decode(reader)
}
