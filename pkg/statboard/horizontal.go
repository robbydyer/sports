package statboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *StatBoard) doHorizontal(ctx context.Context, canvas board.Canvas, players map[string][]Player) error {
	zeroed := rgbrender.ZeroedBounds(canvas.Bounds())

	clrLine := &rgbrender.ColorCharLine{}

	for _, playerList := range players {
		playerList = s.sorter(playerList)

		num := 0
	PLIST:
		for _, player := range playerList {
			num++
			if num > s.config.HorizontalLimit {
				break PLIST
			}
			prfx := player.PrefixCol()
			fName := player.FirstName()
			lName := player.LastName()

			stats, err := s.api.AvailableStats(ctx, player.GetCategory())
			if err != nil {
				return err
			}

			for _, s := range fmt.Sprintf("  %s)", prfx) {
				clrLine.Chars = append(clrLine.Chars, string(s))
				clrLine.Clrs = append(clrLine.Clrs, color.RGBA{30, 144, 255, 255})
			}

			for _, s := range fmt.Sprintf(" %s. %s  ", fName[0:1], lName) {
				clrLine.Chars = append(clrLine.Chars, string(s))
				clrLine.Clrs = append(clrLine.Clrs, color.White)
			}

		STATS:
			for i := 0; i < 2; i++ {
				if len(stats) < i+1 {
					break STATS
				}
				stat := player.GetStat(stats[i])
				clr := player.StatColor(stats[i])
				if clr == nil {
					clr = color.White
				}
				clrLine.Chars = append(clrLine.Chars, " ", " ")
				clrLine.Clrs = append(clrLine.Clrs, color.White, color.White)
				for _, s := range stat {
					clrLine.Chars = append(clrLine.Chars, string(s))
					clrLine.Clrs = append(clrLine.Clrs, clr)
				}
			}
		}
	}

	writer, err := s.getWriter(zeroed.Bounds())
	if err != nil {
		return err
	}

	lengths, err := writer.MeasureStrings(canvas, []string{strings.Join(clrLine.Chars, "")})
	if err != nil {
		return err
	}
	if len(lengths) < 1 {
		return fmt.Errorf("failed to measure text")
	}
	bounds := image.Rect(
		zeroed.Min.X,
		zeroed.Min.Y,
		zeroed.Min.X+lengths[0],
		zeroed.Max.Y,
	)

	canvas.SetWidth(bounds.Dx())

	return writer.WriteAlignedColorCodes(
		rgbrender.CenterCenter,
		canvas,
		bounds,
		&rgbrender.ColorChar{
			BoxClr: color.Black,
			Lines: []*rgbrender.ColorCharLine{
				clrLine,
			},
		},
	)
}
