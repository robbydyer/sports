package main

import (
	"fmt"
	"io/ioutil"
	"os"

	yaml "github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/robbydyer/sports/internal/config"
	"github.com/robbydyer/sports/pkg/clock"
	"github.com/robbydyer/sports/pkg/imageboard"
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
				if configFile == defaultConfigFile {
				}
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

			return nil
		},
	}

	f := rootCmd.PersistentFlags()

	f.StringVarP(&args.configFile, "config", "c", defaultConfigFile, "Config filename")
	f.StringVarP(&args.level, "log-level", "l", "info", "Log level. 'info', 'warn', 'debug'")
	f.BoolVarP(&args.test, "test", "t", false, "uses a test console matrix")

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
