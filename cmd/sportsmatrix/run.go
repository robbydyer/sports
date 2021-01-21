package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/sportsmatrix"
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

	mtrx, err := sportsmatrix.New(ctx, cfg, boards...)
	if err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		fmt.Println("Shutting down")
		cancel()
		mtrx.Close()
	}()

	fmt.Println("Starting matrix service")
	if err := mtrx.Serve(ctx); err != nil {
		return err
	}

	return nil
}
