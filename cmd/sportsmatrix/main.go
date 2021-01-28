package main

import (
	"fmt"
	"io/ioutil"
	"os"

	yaml "github.com/ghodss/yaml"
	"github.com/markbates/pkger"
	"github.com/robbydyer/sports/internal/config"

	//"github.com/robbydyer/sports/pkg/nhlboard"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/robbydyer/sports/pkg/sportboard"
)

type rootArgs struct {
	logLevel   string
	configFile string
	config     *config.Config
}

func main() {
	includeAssets()
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
					return err
				}
			} else {
				fmt.Println("Using default config")
				args.config = &config.Config{}
			}

			return nil
		},
	}

	f := rootCmd.PersistentFlags()

	f.StringVarP(&args.configFile, "config", "c", "", "Config filename")

	_ = viper.BindPFlags(f)

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
		r.config.NHLConfig = &sportboard.Config{}
	}

	r.config.NHLConfig.SetDefaults()
}

func includeAssets() {
	_ = pkger.Include("/assets/fonts/04b24.ttf")
	_ = pkger.Include("/assets/fonts/score.ttf")
	_ = pkger.Include("/assets/fonts/BlockStockRegular-A71p.ttf")
}
