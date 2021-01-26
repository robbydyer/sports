package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"os/signal"
	"time"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/nhl"
	"github.com/robbydyer/sports/pkg/nhlboard"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
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

	s.rArgs.setConfigDefaults()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		fmt.Println("Shutting down")
		cancel()
	}()

	var boards []board.Board

	nhlB, err := nhlBoards(ctx, s.rArgs)
	if err != nil {
		return err
	}
	boards = append(boards, nhlB...)

	if s.testMode {
		boards = []board.Board{&testBoard{}}
	}

	if len(boards) < 1 {
		return fmt.Errorf("WAT. No boards?")
	}

	mtrx, err := sportsmatrix.New(ctx, s.rArgs.config.SportsMatrixConfig, boards...)
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

func nhlBoards(ctx context.Context, rArgs *rootArgs) ([]board.Board, error) {
	bounds := image.Rect(0, 0, rArgs.config.SportsMatrixConfig.HardwareConfig.Cols, rArgs.config.SportsMatrixConfig.HardwareConfig.Rows)

	nhlAPI, err := nhl.New(ctx)
	if err != nil {
		return nil, err
	}

	boards, err := nhlboard.New(ctx, bounds, nhlAPI, nhl.GetLiveGame, rArgs.config.NHLConfig)
	if err != nil {
		return nil, err
	}

	return boards, nil
}

func basicTest() error {
	fmt.Println("test mode")
	cfg := &rgb.HardwareConfig{
		Rows:              32,
		Cols:              64,
		ChainLength:       1,
		Parallel:          1,
		PWMBits:           11,
		PWMLSBNanoseconds: 130,
		Brightness:        60,
		ScanMode:          rgb.Progressive,
		HardwareMapping:   "adafruit-hat",
	}
	// create a new Matrix instance with the DefaultConfig & DefaultRuntimeOptions
	m, _ := rgb.NewRGBLedMatrix(cfg, &rgb.DefaultRuntimeOptions)

	// create the Canvas, implements the image.Image interface
	c := rgb.NewCanvas(m)
	defer c.Close() // don't forgot close the Matrix, if not your leds will remain on

	// using the standard draw.Draw function we copy a white image onto the Canvas
	draw.Draw(c, c.Bounds(), &image.Uniform{color.RGBA{255, 255, 0, 255}}, image.ZP, draw.Src)

	// don't forget call Render to display the new led status
	c.Render()
	time.Sleep(10 * time.Second)
	return nil
}
