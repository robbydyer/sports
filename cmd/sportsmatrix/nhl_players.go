package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/robbydyer/sports/pkg/nhl"
)

type nhlPlayersCmd struct {
	rArgs *rootArgs
	team  string
}

func newNhlPlayersCmd(args *rootArgs) *cobra.Command {
	c := nhlPlayersCmd{
		rArgs: args,
	}

	cmd := &cobra.Command{
		Use:   "players",
		Short: "List NHL players for a given team",
		RunE:  c.run,
	}

	f := cmd.Flags()

	f.StringVar(&c.team, "team", "NYI", "Team abbreviation")

	return cmd
}

func (c *nhlPlayersCmd) run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger, err := c.rArgs.getLogger(c.rArgs.logLevel)
	if err != nil {
		return err
	}
	api, err := nhl.New(ctx, logger)
	if err != nil {
		return err
	}

	players, err := api.ListPlayers(ctx, c.team)
	if err != nil {
		return err
	}

	for _, p := range players {
		fmt.Println(p.FirstName())
	}

	return nil
}
