package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/robbydyer/sports/pkg/nhl"
)

type nhlCmd struct {
	rArgs *rootArgs
}

func newNhlCmd(args *rootArgs) *cobra.Command {
	c := nhlCmd{
		rArgs: args,
	}

	cmd := &cobra.Command{
		Use:   "nhl",
		Short: "nhl",
		RunE:  c.run,
	}

	return cmd
}

func (c *nhlCmd) run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	n, err := nhl.New(ctx)
	if err != nil {
		return err
	}

	n.PrintTodaySchedule(ctx, os.Stdout)
	return nil
}
