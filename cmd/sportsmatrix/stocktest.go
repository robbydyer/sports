package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/spf13/cobra"

	"github.com/robbydyer/sports/pkg/board"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
	"github.com/robbydyer/sports/pkg/stockboard"
)

type stockCmd struct {
	rArgs *rootArgs
}

func newStockCmd(args *rootArgs) *cobra.Command {
	s := stockCmd{
		rArgs: args,
	}

	cmd := &cobra.Command{
		Use:   "stocktest",
		Short: "Tests stock board",
		RunE:  s.run,
	}

	return cmd
}

func (s *stockCmd) run(cmd *cobra.Command, args []string) error {
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

	api := &fakeStocks{}

	b, err := stockboard.New(api, s.rArgs.config.StocksConfig, logger)
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

type fakeStocks struct {
	num int
}

func (f *fakeStocks) Get(ctx context.Context, symbols []string, cacheExpire time.Duration) ([]*stockboard.Stock, error) {
	s := &stockboard.Stock{
		Symbol:    "VTI",
		OpenPrice: 232.5,
		Price:     299.0,
		Prices: []*stockboard.Price{
			{
				Time:  time.Now(),
				Price: 299.0,
			},
		},
	}

	if f.num < 100 {
		f.num++
	}

	for i := 1; i < f.num; i++ {
		var p float64
		r := float64(rand.Intn(10))
		if i%4 == 0 {
			p = s.OpenPrice - r + rand.Float64()
		} else {
			p = s.OpenPrice + r + rand.Float64()
		}
		s.Price = p
		s.Change = ((p - s.OpenPrice) / s.OpenPrice) * 100.0
		s.Prices = append(s.Prices,
			&stockboard.Price{
				Time:  s.Prices[i-1].Time.Add(1 * time.Minute),
				Price: p,
			},
		)
	}

	return []*stockboard.Stock{s}, nil
}

func (f *fakeStocks) CacheClear() {
}

func (f *fakeStocks) TradingClose() (time.Time, error) {
	return time.Now(), fmt.Errorf("no")
}

func (f *fakeStocks) TradingOpen() (time.Time, error) {
	return time.Now(), fmt.Errorf("no")
}
