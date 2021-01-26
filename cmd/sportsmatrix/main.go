package main

import (
	"fmt"
	"io/ioutil"
	"os"

	yaml "github.com/ghodss/yaml"
	"github.com/markbates/pkger"
	"github.com/robbydyer/sports/internal/config"
	"github.com/robbydyer/sports/pkg/nhlboard"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		r.config.NHLConfig = &nhlboard.Config{}
	}

	r.config.NHLConfig.Defaults()

}

func includeAssets() {
	_ = pkger.Include("/assets/logos/DAL/DAL.png")
	_ = pkger.Include("/assets/logos/OTT/OTT.png")
	_ = pkger.Include("/assets/logos/MTL/MTL.png")
	_ = pkger.Include("/assets/logos/COL/COL.png")
	_ = pkger.Include("/assets/logos/MIN/MIN.png")
	_ = pkger.Include("/assets/logos/PHI/PHI.png")
	_ = pkger.Include("/assets/logos/TOR/TOR.png")
	_ = pkger.Include("/assets/logos/WSH/WSH.png")
	_ = pkger.Include("/assets/logos/WPG/WPG.png")
	_ = pkger.Include("/assets/logos/EDM/EDM.png")
	_ = pkger.Include("/assets/logos/CBJ/CBJ.png")
	_ = pkger.Include("/assets/logos/BOS/BOS.png")
	_ = pkger.Include("/assets/logos/FLA/FLA.png")
	_ = pkger.Include("/assets/logos/ARI/ARI.png")
	_ = pkger.Include("/assets/logos/SJS/SJS.png")
	_ = pkger.Include("/assets/logos/LAK/LAK.png")
	_ = pkger.Include("/assets/logos/NYI/NYI.png")
	_ = pkger.Include("/assets/logos/STL/STL.png")
	_ = pkger.Include("/assets/logos/NYR/NYR.png")
	_ = pkger.Include("/assets/logos/VGK/VGK.png")
	_ = pkger.Include("/assets/logos/NSH/NSH.png")
	_ = pkger.Include("/assets/logos/VAN/VAN.png")
	_ = pkger.Include("/assets/logos/TBL/TBL.png")
	_ = pkger.Include("/assets/logos/BUF/BUF.png")
	_ = pkger.Include("/assets/logos/CGY/CGY.png")
	_ = pkger.Include("/assets/logos/CAR/CAR.png")
	_ = pkger.Include("/assets/logos/ANA/ANA.png")
	_ = pkger.Include("/assets/logos/CHI/CHI.png")
	_ = pkger.Include("/assets/logos/PIT/PIT.png")
	_ = pkger.Include("/assets/logos/NJD/NJD.png")
	_ = pkger.Include("/assets/logos/DET/DET.png")
	_ = pkger.Include("/assets/fonts/04b24.ttf")
}
