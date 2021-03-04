package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

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
	return fmt.Errorf("this is broken for now")
}
