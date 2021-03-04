package main

import (
	"github.com/spf13/cobra"
)

type mlbCmd struct {
	rArgs *rootArgs
}

func newMlbCmd(args *rootArgs) *cobra.Command {
	c := mlbCmd{
		rArgs: args,
	}

	cmd := &cobra.Command{
		Use:   "mlbtest",
		Short: "runs some MLB board layout tests",
		RunE:  c.run,
	}

	return cmd
}

func (c *mlbCmd) run(cmd *cobra.Command, args []string) error {
	/*
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

		c.rArgs.config.MLBConfig.Enabled.Store(true)

		bounds := image.Rect(0, 0, c.rArgs.config.SportsMatrixConfig.HardwareConfig.Cols, c.rArgs.config.SportsMatrixConfig.HardwareConfig.Rows)

		logger, err := c.rArgs.getLogger(c.rArgs.logLevel)
		if err != nil {
			return err
		}
		defer func() {
			if c.rArgs.writer != nil {
				c.rArgs.writer.Close()
			}
		}()

		api, err := mlb.NewMock(logger)
		if err != nil {
			return fmt.Errorf("failed to init mlbmock: %w", err)
		}

		b, err := sportboard.New(ctx, api, bounds, logger, c.rArgs.config.MLBConfig)
		if err != nil {
			return err
		}

		var boards []board.Board

		boards = append(boards, b)

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

		var canvases []board.Canvas
		canvases = append(canvases, rgb.NewCanvas(matrix))

		mtrx, err := sportsmatrix.New(ctx, logger, c.rArgs.config.SportsMatrixConfig, canvases, boards...)
		if err != nil {
			return err
		}
		defer mtrx.Close()

		if err := mtrx.Serve(ctx); err != nil {
			return err
		}
	*/
	return nil
}
