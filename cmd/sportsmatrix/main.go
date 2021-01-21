package main

import (
	"fmt"
	"os"

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
