package main

import (
	"context"
	"fmt"
	"image"
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/clock"
	"github.com/robbydyer/sports/pkg/imageboard"
	"github.com/robbydyer/sports/pkg/mlb"
	"github.com/robbydyer/sports/pkg/nhl"
	"github.com/robbydyer/sports/pkg/sportboard"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
	"github.com/robbydyer/sports/pkg/sysboard"
)

type runCmd struct {
	rArgs *rootArgs
}

func newRunCmd(args *rootArgs) *cobra.Command {
	c := runCmd{
		rArgs: args,
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Runs the matrix",
		RunE:  c.run,
	}

	return cmd
}

func (s *runCmd) run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.rArgs.setConfigDefaults()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		fmt.Println("Got OS interrupt signal, Shutting down")
		cancel()
	}()

	logger := log.New()
	logger.Level = s.rArgs.logLevel
	logger.Out = os.Stdout

	bounds := image.Rect(0, 0, s.rArgs.config.SportsMatrixConfig.HardwareConfig.Cols, s.rArgs.config.SportsMatrixConfig.HardwareConfig.Rows)

	var boards []board.Board

	if s.rArgs.config.NHLConfig != nil {
		api, err := nhl.New(ctx, logger)
		if err != nil {
			return err
		}

		b, err := sportboard.New(ctx, api, bounds, logger, s.rArgs.config.NHLConfig)
		if err != nil {
			return err
		}

		boards = append(boards, b)
	}

	if s.rArgs.config.MLBConfig != nil {
		api, err := mlb.NewMock(logger)
		if err != nil {
			return err
		}

		b, err := sportboard.New(ctx, api, bounds, logger, s.rArgs.config.MLBConfig)
		if err != nil {
			return err
		}

		boards = append(boards, b)
	}

	if s.rArgs.config.ImageConfig != nil {
		b, err := imageboard.New(afero.NewOsFs(), bounds, s.rArgs.config.ImageConfig, logger)
		if err != nil {
			return err
		}
		boards = append(boards, b)
	}

	if s.rArgs.config.ClockConfig != nil {
		b, err := clock.New(s.rArgs.config.ClockConfig, logger)
		if err != nil {
			return err
		}
		boards = append(boards, b)
	}

	if s.rArgs.config.SysConfig != nil {
		b, err := sysboard.New(logger, s.rArgs.config.SysConfig)
		if err != nil {
			return err
		}
		boards = append(boards, b)
	}

	mtrx, err := sportsmatrix.New(ctx, logger, s.rArgs.config.SportsMatrixConfig, boards...)
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
