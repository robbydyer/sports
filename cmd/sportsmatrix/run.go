package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/imageboard"
	rgb "github.com/robbydyer/sports/internal/rgbmatrix-rpi"
	"github.com/robbydyer/sports/internal/sportsmatrix"
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
			logger.Info("Removing board from list, in-between setting enabled",
				zap.String("board", b.Name()),
			)
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
		if strings.EqualFold(b.Name(), imageboard.Name) {
			if i, ok := b.(*imageboard.ImageBoard); ok {
				i.SetJumper(mtrx.JumpTo)
			}
		}
	}

	for _, brd := range inBetweenBoards {
		logger.Info("Registering in-between board",
			zap.String("board", brd.Name()),
		)
		mtrx.AddBetweenBoard(brd)
	}

	logger.Info("Starting matrix service")
	if err := mtrx.Serve(ctx); err != nil {
		logger.Error("Matrix returned an error",
			zap.Error(err),
		)
		return err
	}

	return nil
}
