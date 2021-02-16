package main

import (
	"context"
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"time"

	yaml "github.com/ghodss/yaml"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/robbydyer/sports/internal/config"
	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/clock"
	"github.com/robbydyer/sports/pkg/imageboard"
	"github.com/robbydyer/sports/pkg/mlb"
	"github.com/robbydyer/sports/pkg/nhl"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/sportboard"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
	"github.com/robbydyer/sports/pkg/sysboard"
)

const defaultConfigFile = "/etc/sportsmatrix.conf"

type rootArgs struct {
	level      string
	logLevel   zapcore.Level
	configFile string
	config     *config.Config
	test       bool
	today      string
}

func main() {
	args := &rootArgs{}

	rootCmd := newRootCmd(args)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err.Error())

		os.Exit(1)
	}

	os.Exit(0)
}

func newRootCmd(args *rootArgs) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "sports",
		Short: "Sports info",
		PersistentPreRunE: func(cmd *cobra.Command, a []string) error {
			configFile := viper.GetString("config")

			if configFile == defaultConfigFile {
				if _, err := os.Stat(configFile); err != nil && os.IsNotExist(err) {
					fmt.Println("Using default config")
					args.config = &config.Config{}
				} else {
					fmt.Printf("Loading config from file %s\n", configFile)
					if err := args.setConfig(configFile); err != nil {
						return fmt.Errorf("failed to load config file: %w", err)
					}
				}
			} else if configFile != "" {
				fmt.Printf("Loading config from file %s\n", configFile)
				if err := args.setConfig(configFile); err != nil {
					return fmt.Errorf("failed to load config file: %w", err)
				}
			} else {
				fmt.Println("Using default config")
				args.config = &config.Config{}
			}

			lvl := viper.GetString("log-level")

			if lvl == "" {
				args.logLevel = zapcore.InfoLevel
			} else {
				var l zapcore.Level
				if err := l.Set(lvl); err != nil {
					return err
				}
				args.logLevel = l
			}

			args.setConfigDefaults()

			if err := args.setTodayFuncs(viper.GetString("date-str")); err != nil {
				return err
			}

			return nil
		},
	}

	f := rootCmd.PersistentFlags()

	f.StringVarP(&args.configFile, "config", "c", defaultConfigFile, "Config filename")
	f.StringVarP(&args.level, "log-level", "l", "info", "Log level. 'info', 'warn', 'debug'")
	f.BoolVarP(&args.test, "test", "t", false, "uses a test console matrix")
	f.StringVar(&args.today, "date-str", "", "Set the date of 'Today' for testing past days. Format 2020-01-30")

	_ = viper.BindPFlags(f)

	rootCmd.AddCommand(newMlbCmd(args))
	rootCmd.AddCommand(newNhlCmd(args))
	rootCmd.AddCommand(newRunCmd(args))

	return rootCmd
}

func (r *rootArgs) setConfig(filename string) error {
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var c *config.Config

	if err := yaml.Unmarshal(f, &c); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	r.config = c
	return nil
}

func (r *rootArgs) setConfigDefaults() {
	if r.config.SportsMatrixConfig == nil {
		r.config.SportsMatrixConfig = &sportsmatrix.Config{}
	}
	r.config.SportsMatrixConfig.Defaults()

	if r.config.NHLConfig == nil {
		r.config.NHLConfig = &sportboard.Config{
			Enabled: atomic.NewBool(false),
		}
	}

	r.config.NHLConfig.SetDefaults()

	if r.config.ImageConfig == nil {
		r.config.ImageConfig = &imageboard.Config{
			Enabled: atomic.NewBool(false),
		}
	}
	r.config.ImageConfig.SetDefaults()

	if r.config.ClockConfig == nil {
		r.config.ClockConfig = &clock.Config{
			Enabled: atomic.NewBool(false),
		}
	}
	r.config.ClockConfig.SetDefaults()

	if r.config.MLBConfig == nil {
		r.config.MLBConfig = &sportboard.Config{
			Enabled: atomic.NewBool(false),
		}
	}
	r.config.MLBConfig.SetDefaults()

	if r.config.SysConfig == nil {
		r.config.SysConfig = &sysboard.Config{
			Enabled: atomic.NewBool(false),
		}
	}
	r.config.SysConfig.SetDefaults()
}

func (r *rootArgs) getRGBMatrix(logger *zap.Logger) (rgb.Matrix, error) {
	var matrix rgb.Matrix
	logger.Info("initializing matrix",
		zap.Int("Cols", r.config.SportsMatrixConfig.HardwareConfig.Cols),
		zap.Int("Rows", r.config.SportsMatrixConfig.HardwareConfig.Rows),
		zap.Int("Brightness", r.config.SportsMatrixConfig.HardwareConfig.Brightness),
		zap.String("Mapping", r.config.SportsMatrixConfig.HardwareConfig.HardwareMapping),
	)

	rt := &rgb.DefaultRuntimeOptions

	// If we have configured the http server to listen on a privileged port (like 80),
	// we need to maintain root permissions
	if r.config.SportsMatrixConfig.HTTPListenPort < 1024 {
		rt.DropPrivileges = -1
	}
	var err error
	matrix, err = rgb.NewRGBLedMatrix(r.config.SportsMatrixConfig.HardwareConfig, rt)

	return matrix, err
}

func (r *rootArgs) getTestMatrix(logger *zap.Logger) rgb.Matrix {
	logger.Info("initializing console matrix",
		zap.Int("Cols", r.config.SportsMatrixConfig.HardwareConfig.Cols),
		zap.Int("Rows", r.config.SportsMatrixConfig.HardwareConfig.Rows),
	)
	return rgb.NewConsoleMatrix(r.config.SportsMatrixConfig.HardwareConfig.Cols, r.config.SportsMatrixConfig.HardwareConfig.Rows, os.Stdout, logger)
}

func (r *rootArgs) getBoards(ctx context.Context, logger *zap.Logger) ([]board.Board, error) {
	bounds := image.Rect(0, 0, r.config.SportsMatrixConfig.HardwareConfig.Cols, r.config.SportsMatrixConfig.HardwareConfig.Rows)

	var boards []board.Board

	if r.config.NHLConfig != nil {
		api, err := nhl.New(ctx, logger)
		if err != nil {
			return boards, err
		}

		b, err := sportboard.New(ctx, api, bounds, logger, r.config.NHLConfig)
		if err != nil {
			return boards, err
		}

		boards = append(boards, b)
	}
	if r.config.MLBConfig != nil {
		api, err := mlb.New(ctx, logger)
		if err != nil {
			return boards, err
		}

		b, err := sportboard.New(ctx, api, bounds, logger, r.config.MLBConfig)
		if err != nil {
			return boards, err
		}

		boards = append(boards, b)
	}

	if r.config.ImageConfig != nil {
		b, err := imageboard.New(afero.NewOsFs(), bounds, r.config.ImageConfig, logger)
		if err != nil {
			return boards, err
		}
		boards = append(boards, b)
	}

	if r.config.ClockConfig != nil {
		b, err := clock.New(r.config.ClockConfig, logger)
		if err != nil {
			return boards, err
		}
		boards = append(boards, b)
	}

	if r.config.SysConfig != nil {
		b, err := sysboard.New(logger, r.config.SysConfig)
		if err != nil {
			return boards, err
		}
		boards = append(boards, b)
	}

	return boards, nil
}

func (r *rootArgs) setTodayFuncs(today string) error {
	if today == "" {
		return nil
	}

	t, err := time.Parse("2006-01-02", today)
	if err != nil {
		return err
	}

	f := func() time.Time {
		return t
	}

	r.config.NHLConfig.TodayFunc = f
	r.config.MLBConfig.TodayFunc = f

	return nil
}
