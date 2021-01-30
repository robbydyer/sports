package main

import (
	"context"
	"fmt"
	"image"
	"os"
	"os/signal"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/mlbmock"
	"github.com/robbydyer/sports/pkg/sportboard"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type mlbCmd struct {
	rArgs *rootArgs
}

func newMlbCmd(args *rootArgs) *cobra.Command {
	c := mlbCmd{
		rArgs: args,
	}

	cmd := &cobra.Command{
		Use:   "mlbtest",
		Short: "runs some MLB board layout tests",
		RunE:  c.run,
	}

	return cmd
}

func (c *mlbCmd) run(cmd *cobra.Command, args []string) error {
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

	if len(c.rArgs.config.MLBConfig.WatchTeams) < 1 {
		c.rArgs.config.MLBConfig.WatchTeams = []string{"ATL"}
	}

	bounds := image.Rect(0, 0, c.rArgs.config.SportsMatrixConfig.HardwareConfig.Cols, c.rArgs.config.SportsMatrixConfig.HardwareConfig.Rows)

	logger := log.New()
	logger.Level = c.rArgs.logLevel

	api, err := mlbmock.New()
	if err != nil {
		return err
	}

	b, err := sportboard.New(ctx, api, bounds, logger, c.rArgs.config.MLBConfig)
	if err != nil {
		return err
	}

	var boards []board.Board

	boards = append(boards, b)

	mtrx, err := sportsmatrix.New(ctx, logger, c.rArgs.config.SportsMatrixConfig, boards...)
	if err != nil {
		return err
	}
	defer mtrx.Close()

	if err := mtrx.Serve(ctx); err != nil {
		return err
	}
	return nil
}
