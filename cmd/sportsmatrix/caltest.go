package main

import (
	"context"
	"image"
	"time"

	"github.com/robbydyer/sports/pkg/assetlogo"
	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/calendarboard"
	"github.com/robbydyer/sports/pkg/logo"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
	"github.com/spf13/cobra"
)

type calCmd struct {
	rArgs *rootArgs
}

type cal struct{}

func (c *cal) CalendarIcon(ctx context.Context, bounds image.Rectangle) (*logo.Logo, error) {
	return assetlogo.GetLogo("schedule.png", bounds)
}

func (c *cal) HTTPPathPrefix() string {
	return "caltest"
}

func (c *cal) DailyEvents(ctx context.Context, date time.Time) ([]*calendarboard.Event, error) {
	return []*calendarboard.Event{
		{
			Time:  time.Now().Add(24 * time.Hour),
			Title: "A Meeting Tomorrow that is long A B C D E F G H I J K L M N O P Q R S T U V W X Y Z",
		},
		{
			Time:  time.Now(),
			Title: "A Meeting",
		},
	}, nil
}

func newCalCmd(args *rootArgs) *cobra.Command {
	c := calCmd{
		rArgs: args,
	}

	cmd := &cobra.Command{
		Use:   "caltest",
		Short: "Runs a calendar test",
		RunE:  c.run,
	}

	return cmd
}

func (c *calCmd) run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := c.rArgs.getLogger(c.rArgs.logLevel)
	if err != nil {
		return err
	}
	defer func() {
		if c.rArgs.writer != nil {
			c.rArgs.writer.Close()
		}
	}()

	cfg := &calendarboard.Config{}
	cfg.SetDefaults()
	calBoard, err := calendarboard.New(&cal{}, logger, cfg)
	if err != nil {
		return err
	}

	calBoard.Enable()

	var canvases []board.Canvas
	var matrix rgb.Matrix
	if c.rArgs.test {
		matrix = c.rArgs.getTestMatrix(logger)
	} else {
		var err error
		matrix, err = c.rArgs.getRGBMatrix(logger)
		if err != nil {
			return err
		}
	}

	scroll, err := rgb.NewScrollCanvas(matrix, logger)
	if err != nil {
		return err
	}

	canvases = append(canvases, rgb.NewCanvas(matrix), scroll)

	mtrx, err := sportsmatrix.New(ctx, logger, c.rArgs.config.SportsMatrixConfig, canvases, calBoard)
	if err != nil {
		return err
	}
	defer mtrx.Close()

	return mtrx.Serve(ctx)
}
