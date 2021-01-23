package nhlboard

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"os/user"

	"github.com/markbates/pkger"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

type LogoInfo struct {
	TeamAbbreviation string
	Zoom             float64
	XPosition        int
	YPosition        int
}

var logos = map[string]*LogoInfo{
	"NYI_HOME": &LogoInfo{
		Zoom:      1,
		XPosition: -3,
		YPosition: 0,
	},
	"NYI_AWAY": &LogoInfo{
		Zoom:      1,
		XPosition: 3,
		YPosition: 0,
	},
}

func imageRootDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}

func GetLogo(logo *LogoInfo, bounds image.Rectangle) (image.Image, error) {
	f, err := pkger.Open(fmt.Sprintf("github.com/robbydyer/sports:/assets/logos/svg/%s_light.svg", logo.TeamAbbreviation))
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	imgRoot, err := imageRootDir()
	if err != nil {
		return nil, err
	}

	thumbFile := fmt.Sprintf("%s/.sportsmatrix/%s_%dx%d.png", imgRoot, logo.TeamAbbreviation, bounds.Dx(), bounds.Dy())

	var thumbnail image.Image

	_, err = os.Stat(thumbFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the thumbnail
			thumbnail = rgbrender.ResizeImage(img, logo.Zoom)

			fmt.Printf("Saving thumbnail logo for %s\n", logo.TeamAbbreviation)
			if err := rgbrender.SavePng(thumbnail, thumbFile); err != nil {
				return nil, err
			}

			return thumbnail, nil
		}
	}

	t, err := os.Open(thumbFile)
	if err != nil {
		return nil, err
	}
	defer t.Close()

	thumbnail, err = png.Decode(t)
	if err != nil {
		return nil, err
	}

	return thumbnail, nil
}
