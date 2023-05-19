package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/robbydyer/sports/internal/espnboard"
)

type ncaamCmd struct {
	rArgs *rootArgs
}

func newNcaaMCmd(args *rootArgs) *cobra.Command {
	c := ncaamCmd{
		rArgs: args,
	}

	cmd := &cobra.Command{
		Use:   "ncaamtest",
		Short: "runs some MLB board layout tests",
		RunE:  c.run,
	}

	return cmd
}

func (c *ncaamCmd) run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := c.rArgs.getLogger(c.rArgs.logLevel)
	if err != nil {
		return err
	}
	defer func() {
		if c.rArgs.writer != nil {
			c.rArgs.writer.Close()
		}
	}()

	n, err := espnboard.NewNCAAMensBasketball(ctx, logger)
	if err != nil {
		return err
	}

	games, err := n.GetScheduledGames(ctx, []time.Time{time.Now().Local()})
	if err != nil {
		return err
	}

	fmt.Printf("%d games today\n", len(games))

	for _, g := range games {
		live, err := g.GetUpdate(ctx)
		if err != nil {
			fmt.Printf("could not get %d\n", g.GetID())
			continue
		}
		h, err := live.HomeTeam()
		if err != nil {
			return err
		}
		a, err := live.AwayTeam()
		if err != nil {
			return err
		}

		fmt.Printf("%s vs. %s -> %d - %d\n",
			h.GetName(),
			a.GetName(),
			h.Score(),
			a.Score(),
		)
	}

	return err
}
