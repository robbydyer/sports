package sportsmatrix

import (
	"context"
	"image"
	_ "image/png"
	"time"

	"github.com/gobuffalo/packr/v2"
	rgb "github.com/robbydyer/rgbmatrix-rpi"

	"github.com/robbydyer/sports/pkg/nhl"
)

type SportsMatrix struct {
	nhlAPI   *nhl.Nhl
	cfg      *Config
	matrix   rgb.Matrix
	canvas   *rgb.Canvas
	toolkit  *rgb.ToolKit
	imageBox *packr.Box
}

type Config struct {
	EnableNHL bool
}

func New(ctx context.Context, cfg Config) (*SportsMatrix, error) {
	s := &SportsMatrix{
		cfg: &cfg,
	}

	s.imageBox = packr.New("images", "./images")

	var err error

	if cfg.EnableNHL {
		s.nhlAPI, err = nhl.New(ctx)
		if err != nil {
			return nil, err
		}
	}

	c := &rgb.HardwareConfig{
		Rows:       64,
		Cols:       32,
		Brightness: 60,
	}
	rt := &rgb.DefaultRuntimeOptions
	s.matrix, err = rgb.NewRGBLedMatrix(c, rt)

	s.canvas = rgb.NewCanvas(s.matrix)

	s.toolkit = rgb.NewToolKit(s.matrix)

	return s, nil
}

func (s *SportsMatrix) Close() {
	_ = s.canvas.Close()
	_ = s.matrix.Close()
	_ = s.toolkit.Close()
}

func (s *SportsMatrix) RenderGoal() error {
	f, err := s.imageBox.Open("goal_light.png")
	if err != nil {
		return err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	return s.toolkit.PlayImage(img, 60*time.Second)
}
