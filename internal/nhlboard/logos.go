package nhlboard

import (
	"fmt"
	"image"
	"image/png"
	"os"

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
	"NYI_HOME": {
		TeamAbbreviation: "NYI",
		Zoom:             1,
		XPosition:        -3,
		YPosition:        0,
	},
	"NYI_AWAY": {
		TeamAbbreviation: "NYI",
		Zoom:             1,
		XPosition:        3,
		YPosition:        0,
	},
	"COL_HOME": {
		TeamAbbreviation: "COL",
		Zoom:             1,
		XPosition:        -5,
		YPosition:        0,
	},
	"COL_AWAY": {
		TeamAbbreviation: "COL",
		Zoom:             1,
		XPosition:        -5,
		YPosition:        0,
	},
	"ANA_HOME": {
		TeamAbbreviation: "ANA",
		Zoom:             0.8,
		XPosition:        -22,
		YPosition:        3,
	},
	"ANA_AWAY": {
		TeamAbbreviation: "ANA",
		Zoom:             0.8,
		XPosition:        7,
		YPosition:        3,
	},
	"MTL_HOME": {
		TeamAbbreviation: "MTL",
		Zoom:             0.8,
		XPosition:        -22,
		YPosition:        3,
	},
	"MTL_AWAY": {
		TeamAbbreviation: "MTL",
		Zoom:             0.8,
		XPosition:        7,
		YPosition:        3,
	},
}

func imageRootDir() (string, error) {
	d := "/tmp/sportsmatrix"
	if err := os.MkdirAll(d, 0755); err != nil {
		return "", err
	}
	return d, nil
	/*
		u, err := user.Current()
		if err != nil {
			return "", err
		}
		return u.HomeDir, nil
	*/
}

func GetLogo(logo *LogoInfo, bounds image.Rectangle) (image.Image, error) {
	f, err := pkger.Open(fmt.Sprintf("github.com/robbydyer/sports:/assets/logos/%s/%s.png", logo.TeamAbbreviation, logo.TeamAbbreviation))
	if err != nil {
		return nil, fmt.Errorf("failed to locate logo asset: %w", err)
	}

	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode logo png: %w", err)
	}

	imgRoot, err := imageRootDir()
	if err != nil {
		return nil, err
	}

	thumbFile := fmt.Sprintf("%s/%s_%dx%d.png", imgRoot, logo.TeamAbbreviation, bounds.Dx(), bounds.Dy())

	var thumbnail image.Image

	_, err = os.Stat(thumbFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the thumbnail
			thumbnail = rgbrender.ResizeImage(img, bounds, logo.Zoom)

			fmt.Printf("Saving thumbnail logo for %s\n", logo.TeamAbbreviation)
			if err := rgbrender.SavePng(thumbnail, thumbFile); err != nil {
				return nil, fmt.Errorf("failed to save logo %s: %w", thumbFile, err)
			}

			return thumbnail, nil
		}
	}

	t, err := os.Open(thumbFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open logo %s: %w", thumbFile, err)
	}
	defer t.Close()

	thumbnail, err = png.Decode(t)
	if err != nil {
		return nil, fmt.Errorf("failed to decode logo %s: %w", thumbFile, err)
	}

	return thumbnail, nil
}
