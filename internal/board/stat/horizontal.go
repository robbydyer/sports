package statboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/rgbrender"
)

func (s *StatBoard) doHorizontal(ctx context.Context, canvas board.Canvas, players map[string][]Player) (draw.Image, error) {
	zeroed := rgbrender.ZeroedBounds(canvas.Bounds())

	clrLine := &rgbrender.ColorCharLine{}

	for _, playerList := range players {
		playerList = s.sorter(playerList)

		num := 0
	PLIST:
		for _, player := range playerList {
			if player == nil {
				continue PLIST
			}

			num++
			if num > s.config.HorizontalLimit {
				break PLIST
			}
			prfx := player.PrefixCol()
			fName := player.FirstName()
			lName := player.LastName()

			stats, err := s.api.AvailableStats(ctx, player.GetCategory())
			if err != nil {
				return nil, err
			}

			for _, s := range fmt.Sprintf("  %s)", prfx) {
				clrLine.Chars = append(clrLine.Chars, string(s))
				clrLine.Clrs = append(clrLine.Clrs, color.RGBA{30, 144, 255, 255})
			}

			name := ""
			if len(fName) > 0 {
				name = fName[0:1] + ". "
			} else {
				s.log.Info("first name was empty",
					zap.String("first", fName),
					zap.String("last", lName),
				)
			}
			name += lName
			for _, s := range name {
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
		return nil, err
	}

	lengths, err := writer.MeasureStrings(canvas, []string{strings.Join(clrLine.Chars, "")})
	if err != nil {
		return nil, err
	}
	if len(lengths) < 1 {
		return nil, fmt.Errorf("failed to measure text")
	}
	bounds := image.Rect(
		zeroed.Min.X,
		zeroed.Min.Y,
		zeroed.Min.X+lengths[0],
		zeroed.Max.Y,
	)

	img := image.NewRGBA(bounds)

	_ = writer.WriteAlignedColorCodes(
		rgbrender.CenterCenter,
		img,
		bounds,
		&rgbrender.ColorChar{
			BoxClr: color.Black,
			Lines: []*rgbrender.ColorCharLine{
				clrLine,
			},
		},
	)

	return img, nil
}
