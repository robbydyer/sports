package main

import (
	"context"
	"fmt"
	"image"
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/nhlmock"
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

	c.rArgs.config.NHLConfig.Enabled = true

	if len(c.rArgs.config.NHLConfig.WatchTeams) < 1 {
		c.rArgs.config.NHLConfig.WatchTeams = []string{"NYI", "NJD", "CBJ", "MIN"}
	}

	bounds := image.Rect(0, 0, c.rArgs.config.SportsMatrixConfig.HardwareConfig.Cols, c.rArgs.config.SportsMatrixConfig.HardwareConfig.Rows)

	logger := log.New()
	logger.Level = c.rArgs.logLevel

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
