package statboard

import (
	"context"
	"fmt"
	"image/color"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	cnvs "github.com/robbydyer/sports/internal/canvas"
	"github.com/robbydyer/sports/internal/rgbrender"
)

func (s *StatBoard) doScroll(ctx context.Context, canvas board.Canvas, players []Player) error {
	if len(players) < 1 {
		return nil
	}

	scrollCanvas, ok := canvas.(*cnvs.ScrollCanvas)
	if !ok {
		return fmt.Errorf("incorrect canvas type given for scrolling")
	}

	origDir := scrollCanvas.GetScrollDirection()
	defer scrollCanvas.SetScrollDirection(origDir)
	scrollCanvas.SetScrollDirection(cnvs.BottomToTop)

	origPadding := scrollCanvas.GetPadding()
	defer scrollCanvas.SetPadding(origPadding)

	origSpeed := scrollCanvas.GetScrollSpeed()
	defer scrollCanvas.SetScrollSpeed(origSpeed)
	scrollCanvas.SetScrollSpeed(200 * time.Millisecond)

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

	row := 0
	if s.withTitleRow {
		if err := s.renderTitleRow(ctx, grid.GetRow(row), writer, stats); err != nil {
			return err
		}
		row++
	}

	for _, player := range players {
		if err := s.renderPlayer(ctx, player, grid.GetRow(row), writer, stats, maxNameLength(rgbrender.ZeroedBounds(canvas.Bounds()))); err != nil {
			return err
		}
		row++
	}

	grid.FillPadded(scrollCanvas, color.White)

	if err := grid.DrawToBase(scrollCanvas); err != nil {
		return err
	}

	return scrollCanvas.Render(ctx)
}
