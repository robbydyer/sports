package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/robbydyer/sports/internal/espnboard"
	"github.com/robbydyer/sports/internal/sportboard"
)

type abbrevCmd struct {
	rArgs *rootArgs
}

func newAbbrevCmd(args *rootArgs) *cobra.Command {
	c := abbrevCmd{
		rArgs: args,
	}

	cmd := &cobra.Command{
		Use:   "abbrev",
		Short: "Prints team abbreviations",
		RunE:  c.run,
	}

	return cmd
}

func (s *abbrevCmd) run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-ch
		fmt.Println("Got OS interrupt signal, Shutting down")
		cancel()
	}()

	logger, err := s.rArgs.getLogger(s.rArgs.logLevel)
	if err != nil {
		return err
	}
	defer func() {
		if s.rArgs.writer != nil {
			s.rArgs.writer.Close()
		}
	}()

	var sports []*espnboard.ESPNBoard

	ncaaf, err := espnboard.NewNCAAF(ctx, logger)
	if err != nil {
		return err
	}
	nba, err := espnboard.NewNBA(ctx, logger)
	if err != nil {
		return err
	}
	nfl, err := espnboard.NewNFL(ctx, logger)
	if err != nil {
		return err
	}
	ncaam, err := espnboard.NewNCAAMensBasketball(ctx, logger)
	if err != nil {
		return err
	}
	mlb, err := espnboard.NewMLB(ctx, logger)
	if err != nil {
		return err
	}
	mls, err := espnboard.NewMLS(ctx, logger)
	if err != nil {
		return err
	}
	nhl, err := espnboard.NewNHL(ctx, logger)
	if err != nil {
		return err
	}
	sports = append(sports, ncaaf, nba, nfl, ncaam, mlb, mls, nhl)

	for _, e := range sports {
		if err := printESPNAbbrev(ctx, e); err != nil {
			return err
		}
	}

	for _, e := range sports {
		if err := printESPNConf(ctx, e); err != nil {
			return err
		}
	}

	return nil
}

func printESPNAbbrev(ctx context.Context, e *espnboard.ESPNBoard) error {
	teams, err := e.GetTeams(ctx)
	if err != nil {
		return err
	}
	sort.SliceStable(teams, func(i, j int) bool {
		return strings.ToLower(teams[i].GetAbbreviation()) < strings.ToLower(teams[j].GetAbbreviation())
	})
	fmt.Println(e.League())
	seent := make(map[string]struct{})

TEAM:
	for _, t := range teams {
		if _, ok := seent[t.GetAbbreviation()]; ok {
			continue TEAM
		}
		seent[t.GetAbbreviation()] = struct{}{}
		if t.GetDisplayName() != "" {
			fmt.Printf("  %s => %s\n", t.GetAbbreviation(), t.GetDisplayName())
		} else {
			fmt.Printf("  %s => %s\n", t.GetAbbreviation(), t.GetName())
		}
	}
	fmt.Println("")

	return nil
}

func printESPNConf(ctx context.Context, e *espnboard.ESPNBoard) error {
	teams, err := e.GetTeams(ctx)
	if err != nil {
		return err
	}
	sort.SliceStable(teams, func(i, j int) bool {
		return strings.ToLower(teams[i].GetAbbreviation()) < strings.ToLower(teams[j].GetAbbreviation())
	})
	seent := make(map[string]struct{})
	confs := make(map[string][]sportboard.Team)

TEAM:
	for _, t := range teams {
		if _, ok := seent[t.GetAbbreviation()]; ok {
			continue TEAM
		}
		seent[t.GetAbbreviation()] = struct{}{}
		if t.ConferenceName() == "" {
			continue TEAM
		}
		confs[t.ConferenceName()] = append(confs[t.ConferenceName()], t)
	}

	fmt.Printf("%s Conferences/Divisions\n", e.League())
	if len(confs) < 1 {
		fmt.Printf("  Conferences currently unsupported\n\n")
		return nil
	}

	for conf, teams := range confs {
		fmt.Printf("  %s\n", conf)
		for _, t := range teams {
			if t.GetDisplayName() != "" {
				fmt.Printf("    %s => %s\n", t.GetAbbreviation(), t.GetDisplayName())
			} else {
				fmt.Printf("    %s => %s\n", t.GetAbbreviation(), t.GetName())
			}
		}
	}

	return nil
}
