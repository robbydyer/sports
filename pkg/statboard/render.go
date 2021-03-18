package statboard

import (
	"context"
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
		case <-ticker.C:
			if !s.config.Enabled.Load() {
				cancel()
				return
			}
		}
	}
}

// Render ...
func (s *StatBoard) Render(ctx context.Context, canvas board.Canvas) error {
	if !s.config.Enabled.Load() || len(s.config.Players) == 0 && len(s.config.Teams) == 0 {
		return nil
	}

	boardCtx, boardCancel := context.WithCancel(ctx)
	defer boardCancel()

	go s.enablerCancel(boardCtx, boardCancel)

	// var players []Player
	players := make(map[string][]Player)
	for _, abbrev := range s.config.Teams {
		p, err := s.api.ListPlayers(ctx, abbrev)
		if err != nil {
			return err
		}
		for _, player := range p {
			cat := player.GetCategory()
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

	for _, cat := range s.api.PlayerCategories() {
		if err := s.doRender(boardCtx, canvas, players[cat]); err != nil {
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

	if s.config.LimitPlayers > 0 {
		players = players[0:s.config.LimitPlayers]
	}

	grid, err := s.getStatGrid(ctx, canvas, players, writer, stats)
	if err != nil {
		return err
	}

	playersPerGrid := grid.NumRows() - 1

	s.log.Debug("writing player stats",
		zap.Int("players per grid", playersPerGrid),
		zap.Int("num players", len(players)),
	)

	row := 0
	for i := 0; i < len(players); i++ {
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

		if row < grid.NumRows() {
			if err := s.renderPlayer(ctx, players[i], grid.GetRow(row), writer, stats); err != nil {
				return err
			}
			row++

			if i < len(players)-1 {
				continue
			}
		}

		s.log.Debug("drawing grid to base canvas")
		if err := grid.DrawToBase(canvas); err != nil {
			return err
		}

		grid.FillPadded(canvas, color.White)

		if err := canvas.Render(); err != nil {
			return err
		}

		delay := time.Duration(row*2) * time.Second

		if s.config.boardDelay.Seconds() > 0 {
			delay = delay + s.config.boardDelay
		}

		s.log.Debug("delaying stat board", zap.Int("seconds", int(delay.Seconds())))

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(delay):
		}

		row = 0

		if err := grid.Clear(); err != nil {
			return err
		}
	}

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
		if index == 0 {
			if err := writer.WriteAligned(
				rgbrender.CenterTop,
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
			rgbrender.CenterTop,
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

func (s *StatBoard) renderPlayer(ctx context.Context, player Player, row []*rgbrender.Cell, writer *rgbrender.TextWriter, stats []string) error {
	s.log.Debug("render player",
		zap.String("name", player.LastName(false)),
	)
	for index, cell := range row {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}
		if index == 0 {
			if err := writer.WriteAligned(
				rgbrender.LeftCenter,
				cell.Canvas,
				cell.Canvas.Bounds(),
				[]string{
					player.LastName(true),
				},
				color.White,
			); err != nil {
				return err
			}
			continue
		}
		stat := player.GetStat(stats[index-1])
		clr := player.StatColor(stats[index-1])
		if clr == nil {
			clr = color.White
		}
		if err := writer.WriteAligned(
			rgbrender.CenterCenter,
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
