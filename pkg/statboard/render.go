package statboard

import (
	"context"
	"fmt"
	"image/color"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *StatBoard) enablerCancel(ctx context.Context, cancel context.CancelFunc) {
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.cancelBoard:
			cancel()
			return
		case <-ticker.C:
			if !s.config.Enabled.Load() {
				cancel()
				return
			}
		}
	}
}

// ScrollRender ...
func (s *StatBoard) ScrollRender(ctx context.Context, canvas board.Canvas, padding int) (board.Canvas, error) {
	return nil, nil
}

// Render ...
func (s *StatBoard) Render(ctx context.Context, canvas board.Canvas) error {
	if len(s.config.Players) == 0 && len(s.config.Teams) == 0 {
		return fmt.Errorf("no players or teams configured for stats %s", s.api.LeagueShortName())
	}

	boardCtx, boardCancel := context.WithCancel(ctx)
	defer boardCancel()

	go s.enablerCancel(boardCtx, boardCancel)

	doUpdate := false
	if time.Since(s.lastUpdate) > s.config.updateInterval {
		doUpdate = true
		s.lastUpdate = time.Now()
	}

	// var players []Player
	players := make(map[string][]Player)
	for _, abbrev := range s.config.Teams {
		p, err := s.api.ListPlayers(ctx, abbrev)
		if err != nil {
			return err
		}
		for _, player := range p {
			cat := player.GetCategory()
			if doUpdate {
				s.log.Debug("updating player stats",
					zap.String("league", s.api.LeagueShortName()),
					zap.String("player", player.LastName()),
				)
				if err := player.UpdateStats(ctx); err != nil {
					s.log.Error("failed to update player stats",
						zap.Error(err),
						zap.String("league", s.api.LeagueShortName()),
						zap.String("player", player.LastName()),
					)
				}
			}
			players[cat] = append(players[cat], player)
		}

		s.log.Debug("found players for team",
			zap.String("team", abbrev),
			zap.Int("num players", len(p)),
		)
	}

	for _, p := range s.config.Players {
		parts := strings.Fields(p)
		fName := ""
		lName := ""
		if len(parts) > 1 {
			fName = parts[0]
			lName = strings.Join(parts[1:], " ")
		} else {
			lName = p
		}
		player, err := s.api.FindPlayer(ctx, fName, lName)
		if err != nil {
			s.log.Error("failed to get player", zap.String("first", fName), zap.String("last", lName))
			continue
		}
		cat := player.GetCategory()
		players[cat] = append(players[cat], player)
	}

PLAYERS:
	for cat, p := range players {
		s.log.Debug("rendering category",
			zap.String("category", cat),
			zap.Int("num players", len(p)),
		)

		if s.config.ScrollMode.Load() && canvas.Scrollable() {
			if err := s.doScroll(boardCtx, canvas, p); err != nil {
				return err
			}
			continue PLAYERS
		}

		if err := s.doRender(boardCtx, canvas, p); err != nil {
			return err
		}
	}

	return nil
}

func (s *StatBoard) doRender(ctx context.Context, canvas board.Canvas, players []Player) error {
	if len(players) < 1 {
		return nil
	}
	select {
	case <-ctx.Done():
		return context.Canceled
	default:
	}

	// Assume all players passed to this func are in the same category
	playerCategory := players[0].GetCategory()

	var stats []string
	override, ok := s.config.StatOverride[playerCategory]
	if ok && len(override) > 0 {
		stats = override
	} else {
		var err error
		stats, err = s.api.AvailableStats(ctx, playerCategory)
		if err != nil {
			return err
		}
	}

	s.log.Debug("statboard stats",
		zap.String("league", s.api.LeagueShortName()),
		zap.Strings("stats", stats),
	)

	writer, err := s.getWriter(canvas.Bounds())
	if err != nil {
		return err
	}

	players = s.sorter(players)

	if s.config.LimitPlayers > 0 && len(players) > s.config.LimitPlayers {
		s.log.Warn("limiting player cound",
			zap.String("league", s.api.LeagueShortName()),
			zap.Int("limit", s.config.LimitPlayers),
		)
		players = players[0:s.config.LimitPlayers]
	}

	grid, err := s.getStatGrid(ctx, canvas, players, writer, stats)
	if err != nil {
		return err
	}

	playersPerGrid := grid.NumRows() - 1

	for i, p := range players {
		s.log.Debug("writing player stats",
			zap.Int("index", i),
			zap.String("name", p.LastName()),
			zap.Int("players per grid", playersPerGrid),
			zap.Int("num players", len(players)),
		)
	}

	delay := time.Duration(grid.NumRows()) * time.Second

	if s.config.boardDelay.Seconds() > 0 {
		delay = delay + s.config.boardDelay
	}

	s.log.Debug("setting statboard delay", zap.Int("seconds", int(delay.Seconds())))

	row := 0
	i := 0
	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}

		if row == 0 && s.withTitleRow {
			if err := s.renderTitleRow(ctx, grid.GetRow(0), writer, stats); err != nil {
				return err
			}
			row++
		}

		if row < grid.NumRows() && i < len(players) {
			s.log.Debug("render player stats",
				zap.Int("index", i),
				zap.String("name", players[i].LastName()),
				zap.Int("players per grid", playersPerGrid),
				zap.Int("num players", len(players)),
			)
			if err := s.renderPlayer(ctx, players[i], grid.GetRow(row), writer, stats, maxNameLength(canvas.Bounds())); err != nil {
				s.log.Error("failed to render player", zap.Error(err))
			}
			row++
			i++

			continue
		}

		s.log.Debug("drawing grid to base canvas")
		if err := grid.DrawToBase(canvas); err != nil {
			s.log.Error("failed to draw grid", zap.Error(err))
			return err
		}

		grid.FillPadded(canvas, color.White)

		if err := canvas.Render(ctx); err != nil {
			s.log.Error("failed to render canvas", zap.Error(err))
			return err
		}

		s.log.Debug("delaying stat board", zap.Int("seconds", int(delay.Seconds())))

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(delay):
		}

		row = 0

		if err := grid.Clear(); err != nil {
			s.log.Error("failed to clear grid", zap.Error(err))
			return err
		}

		if i >= len(players) {
			break
		}
	}

	s.log.Debug("rendered players",
		zap.Int("number players", i),
	)

	return nil
}

func (s *StatBoard) renderTitleRow(ctx context.Context, row []*rgbrender.Cell, writer *rgbrender.TextWriter, stats []string) error {
	s.log.Debug("render stat title row")
	for index, cell := range row {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}
		if (index == 0 && !s.withPrefixCol) || (index == 1 && s.withPrefixCol) {
			if err := writer.WriteAligned(
				rgbrender.LeftCenter,
				cell.Canvas,
				cell.Canvas.Bounds(),
				[]string{
					s.api.LeagueShortName(),
				},
				color.White,
			); err != nil {
				return err
			}
			continue
		}

		if err := writer.WriteAligned(
			rgbrender.LeftCenter,
			cell.Canvas,
			cell.Canvas.Bounds(),
			[]string{
				s.api.StatShortName(stats[index-1]),
			},
			color.White,
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *StatBoard) renderPlayer(ctx context.Context, player Player, row []*rgbrender.Cell, writer *rgbrender.TextWriter, stats []string, maxName int) error {
	s.log.Debug("render player",
		zap.String("name", player.LastName()),
	)
	adder := 1
	if s.withPrefixCol {
		adder = 2
	}
	for index, cell := range row {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}
		if index == 0 && s.withPrefixCol {
			if err := writer.WriteAligned(
				rgbrender.LeftCenter,
				cell.Canvas,
				cell.Canvas.Bounds(),
				[]string{
					player.PrefixCol(),
				},
				color.White,
			); err != nil {
				return err
			}
			continue
		}
		if index == 0 || (index == 1 && s.withPrefixCol) {
			if err := writer.WriteAligned(
				rgbrender.LeftCenter,
				cell.Canvas,
				cell.Canvas.Bounds(),
				[]string{
					maxedStr(player.LastName(), maxName),
				},
				color.White,
			); err != nil {
				return err
			}
			continue
		}
		stat := player.GetStat(stats[index-adder])
		clr := player.StatColor(stats[index-adder])
		if clr == nil {
			clr = color.White
		}
		if err := writer.WriteAligned(
			rgbrender.LeftCenter,
			cell.Canvas,
			cell.Canvas.Bounds(),
			[]string{
				stat,
			},
			clr,
		); err != nil {
			return err
		}
	}

	return nil
}
