package main

import (
	"context"
	"embed"
	"fmt"
	"image"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	sportboard "github.com/robbydyer/sports/internal/board/sport"
	cnvs "github.com/robbydyer/sports/internal/canvas"
	"github.com/robbydyer/sports/internal/espnboard"
	"github.com/robbydyer/sports/internal/logo"
	"github.com/robbydyer/sports/internal/matrix"
	"github.com/robbydyer/sports/internal/mlblive"
	scrcnvs "github.com/robbydyer/sports/internal/scrollcanvas"
	"github.com/robbydyer/sports/internal/sportsmatrix"
)

//go:embed assets
var assets embed.FS

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c.rArgs.setConfigDefaults()

	logger, err := c.rArgs.getLogger(c.rArgs.logLevel)
	if err != nil {
		return err
	}
	defer func() {
		if c.rArgs.writer != nil {
			c.rArgs.writer.Close()
		}
	}()

	mockLiveGames := make(map[string][]byte, 2)

	mockSchedule, err := assets.ReadFile(filepath.Join("assets", "mlb_espn_schedule.json"))
	if err != nil {
		return err
	}
	dirs, err := assets.ReadDir("assets")
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		if !dir.IsDir() && strings.Contains(dir.Name(), "mlb_espn_live_") {
			logger.Info("register MLB test file",
				zap.String("file", dir.Name()),
			)
			parts := strings.Split(dir.Name(), "_")
			idParts := strings.Split(parts[3], ".")
			mockLiveGames[idParts[0]], err = assets.ReadFile(filepath.Join("assets", dir.Name()))
			if err != nil {
				return err
			}
		}
	}

	var opts []espnboard.Option

	opts = append(opts, espnboard.WithMockData(mockSchedule, mockLiveGames))

	api, err := espnboard.NewMLB(ctx, logger, opts...)
	if err != nil {
		return err
	}

	var canvases []board.Canvas
	var matrix matrix.Matrix
	if c.rArgs.test {
		matrix = c.rArgs.getTestMatrix(logger)
	} else {
		var err error
		matrix, err = c.rArgs.getRGBMatrix(logger)
		if err != nil {
			return err
		}
	}

	scroll, err := scrcnvs.NewScrollCanvas(matrix, logger)
	if err != nil {
		return err
	}
	canvases = append(canvases, scroll)

	canvases = append(canvases, cnvs.NewCanvas(matrix))

	bounds := image.Rect(0, 0, c.rArgs.config.SportsMatrixConfig.HardwareConfig.Cols, c.rArgs.config.SportsMatrixConfig.HardwareConfig.Rows)

	m := &mlblive.MlbLive{
		Logger: logger,
	}

	dR := func(ctx context.Context, canvas board.Canvas, game sportboard.Game, hLogo *logo.Logo, aLogo *logo.Logo) error {
		logger.Info("render detailed live view")
		mlbGame, ok := game.(*espnboard.Game)
		if !ok {
			return fmt.Errorf("unsupported sport for detailed renderer")
		}
		return m.RenderLive(ctx, canvas, mlbGame, hLogo, aLogo)
	}

	c.rArgs.config.MLBConfig.ScrollMode.Store(false)
	c.rArgs.config.MLBConfig.DetailedLive.Store(true)
	c.rArgs.config.MLBConfig.GridCols = 0
	c.rArgs.config.MLBConfig.GridRows = 0
	if err := c.rArgs.setTodayFuncs("2022-05-23"); err != nil {
		return err
	}

	b, err := sportboard.New(ctx, api, bounds, logger, c.rArgs.config.MLBConfig, sportboard.WithDetailedLiveRenderer(dR))
	if err != nil {
		return err
	}

	b.Enabler().Enable()

	c.rArgs.config.SportsMatrixConfig.CombinedScroll.Store(false)
	mtrx, err := sportsmatrix.New(ctx, logger, c.rArgs.config.SportsMatrixConfig, canvases, b)
	if err != nil {
		return err
	}
	defer mtrx.Close()

	return mtrx.Serve(ctx)
}
