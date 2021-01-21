package main

import (
	"context"

	"github.com/robbydyer/sports/internal/sportsmatrix"
	"github.com/spf13/cobra"
)

type runCmd struct {
	rArgs *rootArgs
	port  int
}

func newRunCmd(args *rootArgs) *cobra.Command {
	c := runCmd{
		rArgs: args,
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Runs the matrix",
		RunE:  c.run,
	}

	return cmd
}

func (s *runCmd) run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	mtrx, err := sportsmatrix.New(ctx, sportsmatrix.Config{
		EnableNHL: true,
	})
	if err != nil {
		return err
	}

	return mtrx.RenderGoal()
	//return nil
}
