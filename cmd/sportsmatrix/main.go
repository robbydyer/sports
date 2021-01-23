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

	a := []string{
		"/assets/fonts/04b24.ttf",
		"/assets/logos/DAL/DAL.png",
		"/assets/logos/OTT/OTT.png",
		"/assets/logos/MTL/MTL.png",
		"/assets/logos/COL/COL.png",
		"/assets/logos/MIN/MIN.png",
		"/assets/logos/PHI/PHI.png",
		"/assets/logos/TOR/TOR.png",
		"/assets/logos/WSH/WSH.png",
		"/assets/logos/WPG/WPG.png",
		"/assets/logos/EDM/EDM.png",
		"/assets/logos/CBJ/CBJ.png",
		"/assets/logos/BOS/BOS.png",
		"/assets/logos/FLA/FLA.png",
		"/assets/logos/ARI/ARI.png",
		"/assets/logos/SJS/SJS.png",
		"/assets/logos/LAK/LAK.png",
		"/assets/logos/NYI/NYI.png",
		"/assets/logos/STL/STL.png",
		"/assets/logos/NYR/NYR.png",
		"/assets/logos/VGK/VGK.png",
		"/assets/logos/NSH/NSH.png",
		"/assets/logos/VAN/VAN.png",
		"/assets/logos/TBL/TBL.png",
		"/assets/logos/BUF/BUF.png",
		"/assets/logos/CGY/CGY.png",
		"/assets/logos/CAR/CAR.png",
		"/assets/logos/ANA/ANA.png",
		"/assets/logos/CHI/CHI.png",
		"/assets/logos/PIT/PIT.png",
		"/assets/logos/NJD/NJD.png",
		"/assets/logos/DET/DET.png",
	}

	for _, f := range a {
		_ = pkger.Include(f)
	}
}
