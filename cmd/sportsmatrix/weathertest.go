package main

import (
	"context"
	"fmt"
	"image"
	"time"

	"github.com/spf13/cobra"

	"github.com/robbydyer/sports/pkg/board"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
	"github.com/robbydyer/sports/pkg/weatherboard"
)

type weatherCmd struct {
	rArgs *rootArgs
}

func newWeatherCmd(args *rootArgs) *cobra.Command {
	s := weatherCmd{
		rArgs: args,
	}

	cmd := &cobra.Command{
		Use:   "weathertest",
		Short: "Tests weather board",
		RunE:  s.run,
	}

	return cmd
}

func (s *weatherCmd) run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.rArgs.setConfigDefaults()

	logger, err := s.rArgs.getLogger(s.rArgs.logLevel)
	if err != nil {
		return err
	}
	defer func() {
		if s.rArgs.writer != nil {
			s.rArgs.writer.Close()
		}
	}()

	api := &fakeWeather{}
	f, _ := api.DailyForecasts(ctx, "", "", image.Rectangle{})
	s.rArgs.config.WeatherConfig.DailyNumber = len(f)

	b, err := weatherboard.New(api, s.rArgs.config.WeatherConfig, logger)
	if err != nil {
		return err
	}

	var canvases []board.Canvas
	var matrix rgb.Matrix
	if s.rArgs.test {
		matrix = s.rArgs.getTestMatrix(logger)
	} else {
		var err error
		matrix, err = s.rArgs.getRGBMatrix(logger)
		if err != nil {
			return err
		}
	}

	scroll, err := rgb.NewScrollCanvas(matrix, logger)
	if err != nil {
		return err
	}

	canvases = append(canvases, rgb.NewCanvas(matrix), scroll)

	mtrx, err := sportsmatrix.New(ctx, logger, s.rArgs.config.SportsMatrixConfig, canvases, b)
	if err != nil {
		return err
	}
	defer mtrx.Close()

	fmt.Println("Starting matrix service")
	if err := mtrx.Serve(ctx); err != nil {
		fmt.Printf("Matrix returned an error: %s", err.Error())
		return err
	}

	return nil
}

type fakeWeather struct {
}

func fltPtr(f float64) *float64 {
	return &f
}
func intPtr(i int) *int {
	return &i
}

func (f *fakeWeather) CurrentForecast(ctx context.Context, zipCode string, country string, bounds image.Rectangle) (*weatherboard.Forecast, error) {
	return &weatherboard.Forecast{
		Time:         time.Now().Local(),
		Temperature:  fltPtr(72),
		Humidity:     50,
		TempUnit:     "F",
		IconCode:     "01d",
		PrecipChance: intPtr(0),
	}, nil
}
func (f *fakeWeather) DailyForecasts(ctx context.Context, zipCode string, country string, bounds image.Rectangle) ([]*weatherboard.Forecast, error) {
	return []*weatherboard.Forecast{
		{
			Time:     time.Now().Local().Add(24 * time.Hour),
			HighTemp: fltPtr(90),
			LowTemp:  fltPtr(70),
			Humidity: 50,
			TempUnit: "F",
			IconCode: "01n",
		},
		{
			Time:     time.Now().Local().Add(24 * time.Hour),
			HighTemp: fltPtr(90),
			LowTemp:  fltPtr(70),
			Humidity: 50,
			TempUnit: "F",
			IconCode: "02d",
		},
		{
			Time:     time.Now().Local().Add(24 * time.Hour),
			HighTemp: fltPtr(90),
			LowTemp:  fltPtr(70),
			Humidity: 50,
			TempUnit: "F",
			IconCode: "02n",
		},
		{
			Time:     time.Now().Local().Add(24 * time.Hour),
			HighTemp: fltPtr(90),
			LowTemp:  fltPtr(70),
			Humidity: 50,
			TempUnit: "F",
			IconCode: "03d",
		},
		{
			Time:     time.Now().Local().Add(48 * time.Hour),
			HighTemp: fltPtr(90),
			LowTemp:  fltPtr(70),
			Humidity: 50,
			TempUnit: "F",
			IconCode: "09d",
		},
		{
			Time:     time.Now().Local().Add(96 * time.Hour),
			HighTemp: fltPtr(90),
			LowTemp:  fltPtr(70),
			Humidity: 50,
			TempUnit: "F",
			IconCode: "11d",
		},
		{
			Time:     time.Now().Local().Add(120 * time.Hour),
			HighTemp: fltPtr(90),
			LowTemp:  fltPtr(70),
			Humidity: 50,
			TempUnit: "F",
			IconCode: "13d",
		},
		{
			Time:     time.Now().Local().Add(120 * time.Hour),
			HighTemp: fltPtr(90),
			LowTemp:  fltPtr(70),
			Humidity: 50,
			TempUnit: "F",
			IconCode: "50d",
		},
	}, nil
}
func (f *fakeWeather) HourlyForecasts(ctx context.Context, zipCode string, country string, bounds image.Rectangle) ([]*weatherboard.Forecast, error) {
	return []*weatherboard.Forecast{}, nil
}
func (f *fakeWeather) CacheClear() {}
