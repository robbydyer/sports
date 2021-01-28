package main

import (
	"context"
	"fmt"
	"image"
	"os"
	"os/signal"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/nhlmock"
	"github.com/robbydyer/sports/pkg/sportboard"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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

	if len(c.rArgs.config.NHLConfig.WatchTeams) < 1 {
		c.rArgs.config.NHLConfig.WatchTeams = []string{"NYI", "NJD", "CBJ", "MIN"}
	}

	bounds := image.Rect(0, 0, c.rArgs.config.SportsMatrixConfig.HardwareConfig.Cols, c.rArgs.config.SportsMatrixConfig.HardwareConfig.Rows)

	logger := log.New()
	logger.Level = log.DebugLevel

	api, err := nhlmock.New()
	if err != nil {
		return err
	}

	b, err := sportboard.New(ctx, api, bounds, logger, c.rArgs.config.NHLConfig)
	if err != nil {
		return err
	}

	var boards []board.Board

	boards = append(boards, b)

	mtrx, err := sportsmatrix.New(ctx, c.rArgs.config.SportsMatrixConfig, boards...)
	if err != nil {
		return err
	}
	defer mtrx.Close()

	if err := mtrx.Serve(ctx); err != nil {
		return err
	}
	return nil
}
