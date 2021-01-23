package main

import (
	"fmt"
	"os"

	"github.com/markbates/pkger"
	"github.com/spf13/cobra"
)

type rootArgs struct {
	logLevel string
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
	}

	/*
		f := rootCmd.PersistentFlags()

		f.StringVarP(&args.logLevel, "log-level", "l", "info", "Logging level. One of info, warn, debug")

	*/
	rootCmd.AddCommand(newNhlCmd(args))
	rootCmd.AddCommand(newRunCmd(args))

	return rootCmd
}

func includeAssets() {
	_ = pkger.Include("/assets/fonts/04b24.ttf")
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
}
