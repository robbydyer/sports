package main

import (
	"context"
	"fmt"
	"image"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/nhl"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/sportboard"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
)

type nhlCmd struct {
	rArgs *rootArgs
}

func newNhlCmd(args *rootArgs) *cobra.Command {
	c := nhlCmd{
		rArgs: args,
	}

	cmd := &cobra.Command{
		Use:   "nhltest",
		Short: "runs some NHL board layout tests",
		RunE:  c.run,
	}

	return cmd
}

func (c *nhlCmd) run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c.rArgs.setConfigDefaults()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		fmt.Println("Shutting down")
		cancel()
	}()

	if c.rArgs.config.NHLConfig == nil {
		c.rArgs.config.NHLConfig = &sportboard.Config{}
	}

	c.rArgs.config.NHLConfig.Enabled.Store(true)

	if len(c.rArgs.config.NHLConfig.WatchTeams) < 1 {
		c.rArgs.config.NHLConfig.WatchTeams = []string{"NYI", "NJD", "CBJ", "MIN"}
	}

	c.rArgs.config.MLBConfig.SetDefaults()

	bounds := image.Rect(0, 0, c.rArgs.config.SportsMatrixConfig.HardwareConfig.Cols, c.rArgs.config.SportsMatrixConfig.HardwareConfig.Rows)

	logger := getLogger(c.rArgs.logLevel)

	api, err := nhl.NewMock(logger)
	if err != nil {
		return err
	}

	b, err := sportboard.New(ctx, api, bounds, logger, c.rArgs.config.NHLConfig)
	if err != nil {
		return err
	}

	var boards []board.Board

	boards = append(boards, b)

	var matrix rgb.Matrix
	if c.rArgs.test {
		matrix = c.rArgs.getTestMatrix(logger)
	} else {
		var err error
		matrix, err = c.rArgs.getRGBMatrix(logger)
		if err != nil {
			return err
		}
	}

	canvas := rgb.NewCanvas(matrix)

	mtrx, err := sportsmatrix.New(ctx, logger, c.rArgs.config.SportsMatrixConfig, canvas, boards...)
	if err != nil {
		return err
	}
	defer mtrx.Close()

	if err := mtrx.Serve(ctx); err != nil {
		return err
	}
	return nil
}
