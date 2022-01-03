package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/imageboard"
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
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-ch
		fmt.Println("Got OS interrupt signal, Shutting down")
		cancel()
	}()

	logger, err := s.rArgs.getLogger(s.rArgs.logLevel)
	if err != nil {
		return err
	}
	defer func() {
		if s.rArgs.writer != nil {
			s.rArgs.writer.Close()
		}
	}()

	boards, err := s.rArgs.getBoards(ctx, logger)
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

	newBoards := []board.Board{}
	inBetweenBoards := []board.Board{}

	for _, b := range boards {
		if b.InBetween() {
			fmt.Printf("Removing %s from board list- in-between setting enabled\n", b.Name())
			inBetweenBoards = append(inBetweenBoards, b)
		} else {
			newBoards = append(newBoards, b)
		}
	}

	boards = newBoards

	mtrx, err := sportsmatrix.New(ctx, logger, s.rArgs.config.SportsMatrixConfig, canvases, boards...)
	if err != nil {
		return err
	}
	defer mtrx.Close()

	for _, b := range boards {
		if strings.EqualFold(b.Name(), "img") {
			if i, ok := b.(*imageboard.ImageBoard); ok {
				i.SetJumper(mtrx.JumpTo)
			}
		}
	}

	for _, brd := range inBetweenBoards {
		fmt.Printf("Registering %s as in-between board\n", brd.Name())
		mtrx.AddBetweenBoard(brd)
		break
	}

	fmt.Println("Starting matrix service")
	if err := mtrx.Serve(ctx); err != nil {
		fmt.Printf("Matrix returned an error: %s", err.Error())
		return err
	}

	return nil
}
