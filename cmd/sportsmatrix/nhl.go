package main

import (
	"github.com/spf13/cobra"
)

func newNhlCmd(args *rootArgs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nhl",
		Short: "Get NHL specific information",
	}

	cmd.AddCommand(newNhlPlayersCmd(args))

	return cmd
}
