package main

import (
	"fmt"
	"io/ioutil"
	"os"

	yaml "github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/atomic"
	"go.uber.org/zap/zapcore"

	"github.com/robbydyer/sports/internal/config"
	"github.com/robbydyer/sports/pkg/clock"
	"github.com/robbydyer/sports/pkg/imageboard"
	"github.com/robbydyer/sports/pkg/sportboard"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
	"github.com/robbydyer/sports/pkg/sysboard"
)

type rootArgs struct {
	level      string
	logLevel   zapcore.Level
	configFile string
	config     *config.Config
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

			if configFile != "" {
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

	f.StringVarP(&args.configFile, "config", "c", "", "Config filename")
	f.StringVarP(&args.level, "log-level", "l", "info", "Log level. 'info', 'warn', 'debug'")

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
