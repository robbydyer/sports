package main

import (
	"context"
	"fmt"
	"image"
	"os"
	"os/signal"
	"time"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/nhlboard"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
	"github.com/spf13/cobra"
)

type runCmd struct {
	rArgs    *rootArgs
	port     int
	testMode bool
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

	f := cmd.Flags()

	f.BoolVarP(&c.testMode, "test-mode", "t", false, "test mode")

	return cmd
}

func (s *runCmd) run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		fmt.Println("Shutting down")
		cancel()
	}()

	cfg := sportsmatrix.DefaultConfig()

	var boards []board.Board

	if s.testMode {
		boards = append(boards, &testBoard{})
	}

	/*
		clockBoard, err := clock.New()
		if err != nil {
			return err
		}

		boards = append(boards, clockBoard)
	*/

	bounds := image.Rect(0, 0, cfg.HardwareConfig.Cols, cfg.HardwareConfig.Rows)

	nhlBoards, err := nhlboard.New(ctx, bounds, &nhlboard.Config{
		Delay: 30 * time.Second,
	})
	if err != nil {
		return err
	}
	boards = append(boards, nhlBoards...)

	if len(boards) < 1 {
		return fmt.Errorf("WAT. No boards?")
	}

	mtrx, err := sportsmatrix.New(ctx, cfg, boards...)
	if err != nil {
		return err
	}
	defer mtrx.Close()

	fmt.Println("Starting matrix service")
	if err := mtrx.Serve(ctx); err != nil {
		return err
	}

	return nil
}
