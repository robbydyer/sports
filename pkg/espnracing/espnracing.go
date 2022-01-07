package espnracing

import (
	"context"
	"image"

	"github.com/robbydyer/sports/pkg/logo"
)

type API struct {
	myLogo *logo.Logo
}

func (a *API) GetLogo(ctx context.Context) (*logo.Logo, error) {
	if a.myLogo != nil {
		return a.myLogo, nil
	}
	getter := func(ctx context.Context) (image.Image, error) {}
	return nil, nil
}
