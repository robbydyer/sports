package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
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

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		fmt.Println("Got OS interrupt signal, Shutting down")
		cancel()
	}()

	logger := getLogger(s.rArgs.logLevel)

	boards, err := s.rArgs.getBoards(ctx, logger)
	if err != nil {
		return err
	}

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

	canvas := rgb.NewCanvas(matrix)

	mtrx, err := sportsmatrix.New(ctx, logger, s.rArgs.config.SportsMatrixConfig, canvas, boards...)
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
