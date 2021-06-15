package sportboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

const mls = "MLS"
const scrollModeBuffer = 10
const teamInfoArea = 22

var red = color.RGBA{255, 0, 0, 255}

func (s *SportBoard) homeSide() string {
	if s.api.League() == mls {
		return "left"
	}
	return "right"
}

func (s *SportBoard) renderLiveGame(ctx context.Context, canvas board.Canvas, liveGame Game, counter image.Image) error {
	s.logCanvas(canvas, "render live canvas size")

	layers, err := rgbrender.NewLayerDrawer(60*time.Second, s.log)
	if err != nil {
		return err
	}

	logos, err := s.logoLayers(liveGame, canvas.Bounds())
	if err != nil {
		return err
	}

	var infos []*rgbrender.TextLayer
	if s.config.ShowRecord.Load() || s.config.GamblingSpread.Load() {
		var err error
		infos, err = s.teamInfoLayers(liveGame, canvas.Bounds())
		if err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		default:
		}

		for _, l := range logos {
			layers.AddLayer(rgbrender.BackgroundPriority, l)
		}

		layers.AddTextLayer(rgbrender.BackgroundPriority+1,
			rgbrender.NewTextLayer(
				func(ctx context.Context) (*rgbrender.TextWriter, []string, error) {
					quarter, err := liveGame.GetQuarter()
					if err != nil {
						return nil, nil, err
					}

					clock, err := liveGame.GetClock()
					if err != nil {
						return nil, nil, err
					}
					writer, err := s.getTimeWriter(canvas.Bounds())
					if err != nil {
						return nil, nil, err
					}
					return writer, []string{quarter, clock}, nil
				},
				func(canvas board.Canvas, writer *rgbrender.TextWriter, text []string) error {
					return writer.WriteAlignedBoxed(
						rgbrender.CenterTop,
						canvas,
						rgbrender.ZeroedBounds(canvas.Bounds()),
						text,
						s.config.TimeColor,
						color.Black,
					)
				},
			),
		)

		layers.AddTextLayer(rgbrender.BackgroundPriority+1,
			rgbrender.NewTextLayer(
				func(ctx context.Context) (*rgbrender.TextWriter, []string, error) {
					isFavorite, err := s.isFavoriteGame(liveGame)
					if err != nil {
						return nil, nil, err
					}
					score := []string{}
					if s.config.HideFavoriteScore.Load() && isFavorite {
						s.log.Warn("hiding score for favorite team")
						score = []string{}
					} else {
						str, err := scoreStr(liveGame, s.homeSide())
						if err != nil {
							return nil, nil, err
						}
						score = append(score, str)
					}
					writer, err := s.getScoreWriter(canvas.Bounds())
					if err != nil {
						return nil, nil, err
					}
					return writer, score, nil
				},
				func(canvas board.Canvas, writer *rgbrender.TextWriter, text []string) error {
					return writer.WriteAlignedBoxed(
						rgbrender.CenterBottom,
						canvas,
						rgbrender.ZeroedBounds(canvas.Bounds()),
						text,
						s.config.ScoreColor,
						color.Black,
					)
				},
			),
		)

		if counter != nil {
			layers.AddLayer(rgbrender.ForegroundPriority, counterLayer(counter))
		}

		for _, i := range infos {
			layers.AddTextLayer(rgbrender.ForegroundPriority, i)
		}

		if err := layers.Draw(ctx, canvas); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		default:
		}

		isFavorite, err := s.isFavoriteGame(liveGame)
		if err != nil {
			return err
		}
		if !(isFavorite && s.config.FavoriteSticky.Load()) {
			return nil
		}

		if err := canvas.Render(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(s.config.boardDelay - 3*time.Second):
		}

		liveGame, err = liveGame.GetUpdate(ctx)
		if err != nil {
			return fmt.Errorf("failed to update sticky game: %w", err)
		}
		s.log.Debug("updated live sticky game")

		over, err := liveGame.IsComplete()
		if err != nil {
			return err
		}
		if over {
			s.log.Info("live game is over", zap.Int("game ID", liveGame.GetID()))
			return nil
		}

		layers.ClearLayers()
	}
}

func (s *SportBoard) renderUpcomingGame(ctx context.Context, canvas board.Canvas, liveGame Game, counter image.Image) error {
	layers, err := rgbrender.NewLayerDrawer(60*time.Second, s.log)
	if err != nil {
		return err
	}

	logos, err := s.logoLayers(liveGame, canvas.Bounds())
	if err != nil {
		return err
	}

	if s.config.ShowRecord.Load() || s.config.GamblingSpread.Load() {
		infos, err := s.teamInfoLayers(liveGame, canvas.Bounds())
		if err != nil {
			return err
		}
		for _, i := range infos {
			layers.AddTextLayer(rgbrender.ForegroundPriority, i)
		}
	}

	for _, l := range logos {
		layers.AddLayer(rgbrender.BackgroundPriority, l)
	}

	layers.AddTextLayer(rgbrender.BackgroundPriority+1,
		rgbrender.NewTextLayer(
			func(ctx context.Context) (*rgbrender.TextWriter, []string, error) {
				timeWriter, err := s.getTimeWriter(canvas.Bounds())
				if err != nil {
					return nil, nil, err
				}
				gameTimeStr := ""
				if is, err := liveGame.IsPostponed(); err == nil && is {
					s.log.Debug("game was postponed", zap.Int("game ID", liveGame.GetID()))
					gameTimeStr = "PPD"
				} else {
					gameTime, err := liveGame.GetStartTime(ctx)
					if err != nil {
						return nil, nil, err
					}
					gameTimeStr = gameTime.Local().Format("3:04PM")
				}
				return timeWriter, []string{gameTimeStr}, nil
			},
			func(canvas board.Canvas, writer *rgbrender.TextWriter, text []string) error {
				return writer.WriteAlignedBoxed(
					rgbrender.CenterTop,
					canvas,
					rgbrender.ZeroedBounds(canvas.Bounds()),
					text,
					s.config.TimeColor,
					color.Black,
				)
			},
		),
	)
	layers.AddTextLayer(rgbrender.BackgroundPriority+1,
		rgbrender.NewTextLayer(
			func(ctx context.Context) (*rgbrender.TextWriter, []string, error) {
				scoreWriter, err := s.getScoreWriter(canvas.Bounds())
				if err != nil {
					return nil, nil, err
				}
				return scoreWriter, []string{"VS"}, nil
			},
			func(canvas board.Canvas, writer *rgbrender.TextWriter, text []string) error {
				return writer.WriteAlignedBoxed(
					rgbrender.CenterCenter,
					canvas,
					rgbrender.ZeroedBounds(canvas.Bounds()),
					text,
					s.config.ScoreColor,
					color.Black,
				)
			},
		),
	)

	if counter != nil {
		layers.AddLayer(rgbrender.ForegroundPriority, counterLayer(counter))
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
	default:
	}

	return layers.Draw(ctx, canvas)
}

func (s *SportBoard) renderCompleteGame(ctx context.Context, canvas board.Canvas, liveGame Game, counter image.Image) error {
	layers, err := rgbrender.NewLayerDrawer(60*time.Second, s.log)
	if err != nil {
		return err
	}

	// Give the logoLayers the real bounds, as it already accounts for scroll mode itself
	logos, err := s.logoLayers(liveGame, canvas.Bounds())
	if err != nil {
		return err
	}

	if s.config.ShowRecord.Load() || s.config.GamblingSpread.Load() {
		infos, err := s.teamInfoLayers(liveGame, canvas.Bounds())
		if err != nil {
			return err
		}
		for _, i := range infos {
			layers.AddTextLayer(rgbrender.ForegroundPriority, i)
		}
	}

	for _, l := range logos {
		layers.AddLayer(rgbrender.BackgroundPriority, l)
	}

	layers.AddTextLayer(rgbrender.BackgroundPriority+1,
		rgbrender.NewTextLayer(
			func(ctx context.Context) (*rgbrender.TextWriter, []string, error) {
				writer, err := s.getTimeWriter(canvas.Bounds())
				if err != nil {
					return nil, nil, err
				}
				return writer, []string{"FINAL"}, nil
			},
			func(canvas board.Canvas, writer *rgbrender.TextWriter, text []string) error {
				return writer.WriteAlignedBoxed(
					rgbrender.CenterTop,
					canvas,
					rgbrender.ZeroedBounds(canvas.Bounds()),
					text,
					s.config.TimeColor,
					color.Black,
				)
			},
		),
	)

	layers.AddTextLayer(rgbrender.BackgroundPriority+1,
		rgbrender.NewTextLayer(
			func(ctx context.Context) (*rgbrender.TextWriter, []string, error) {
				isFavorite, err := s.isFavoriteGame(liveGame)
				if err != nil {
					return nil, nil, err
				}
				score := []string{}
				if s.config.HideFavoriteScore.Load() && isFavorite {
					s.log.Warn("hiding score for favorite team")
					score = []string{}
				} else {
					str, err := scoreStr(liveGame, s.homeSide())
					if err != nil {
						return nil, nil, err
					}
					score = append(score, str)
				}
				writer, err := s.getScoreWriter(canvas.Bounds())
				if err != nil {
					return nil, nil, err
				}
				return writer, score, nil
			},
			func(canvas board.Canvas, writer *rgbrender.TextWriter, text []string) error {
				return writer.WriteAlignedBoxed(
					rgbrender.CenterBottom,
					canvas,
					rgbrender.ZeroedBounds(canvas.Bounds()),
					text,
					s.config.ScoreColor,
					color.Black,
				)
			},
		),
	)

	if s.config.ShowRecord.Load() {
		infos, err := s.teamInfoLayers(liveGame, canvas.Bounds())
		if err != nil {
			return err
		}
		for _, i := range infos {
			layers.AddTextLayer(rgbrender.ForegroundPriority, i)
		}
	}

	if counter != nil {
		layers.AddLayer(rgbrender.ForegroundPriority, counterLayer(counter))
	}

	return layers.Draw(ctx, canvas)
}

func counterLayer(counter image.Image) *rgbrender.Layer {
	if counter == nil {
		return nil
	}
	return rgbrender.NewLayer(
		nil,
		func(canvas board.Canvas, img image.Image) error {
			draw.Draw(canvas, counter.Bounds(), counter, image.Point{}, draw.Over)
			return nil
		},
	)
}

func (s *SportBoard) logoLayers(liveGame Game, bounds image.Rectangle) ([]*rgbrender.Layer, error) {
	rightTeam, err := liveGame.HomeTeam()
	if err != nil {
		return nil, err
	}
	leftTeam, err := liveGame.AwayTeam()
	if err != nil {
		return nil, err
	}

	if s.api.League() == mls {
		// MLS does Home team on left
		leftTeam, rightTeam = rightTeam, leftTeam
	}

	return []*rgbrender.Layer{
		rgbrender.NewLayer(
			func(ctx context.Context) (image.Image, error) {
				return s.RenderLeftLogo(ctx, bounds, leftTeam.GetAbbreviation())
			},
			func(canvas board.Canvas, img image.Image) error {
				pt := image.Pt(img.Bounds().Min.X, img.Bounds().Min.Y)
				b := canvas.Bounds()
				s.log.Debug("draw left team logo",
					zap.Int("pt X", pt.X),
					zap.Int("pt Y", pt.Y),
					zap.Int("canvas min X", b.Bounds().Min.X),
					zap.Int("canvas min Y", b.Bounds().Min.Y),
					zap.Int("canvas max X", b.Bounds().Max.X),
					zap.Int("canvas max Y", b.Bounds().Max.Y),
					zap.Int("img min X", img.Bounds().Min.X),
					zap.Int("img min Y", img.Bounds().Min.Y),
					zap.Int("img max X", img.Bounds().Max.X),
					zap.Int("img max Y", img.Bounds().Max.Y),
				)
				draw.Draw(canvas, img.Bounds(), img, pt, draw.Over)
				return nil
			},
		),

		rgbrender.NewLayer(
			func(ctx context.Context) (image.Image, error) {
				return s.RenderRightLogo(ctx, bounds, rightTeam.GetAbbreviation())
			},
			func(canvas board.Canvas, img image.Image) error {
				pt := image.Pt(img.Bounds().Min.X, img.Bounds().Min.Y)
				draw.Draw(canvas, img.Bounds(), img, pt, draw.Over)
				return nil
			},
		),
	}, nil
}

/*
func (s *SportBoard) GamblingLayers(liveGame Game, bounds image.Rectangle) ([]*rgbrender.TextLayer, error) {
	rightTeam, err := liveGame.HomeTeam()
	if err != nil {
		return nil, err
	}
	leftTeam, err := liveGame.AwayTeam()
	if err != nil {
		return nil, err
	}

	if s.api.League() == mls {
		// MLS does Home team on left
		leftTeam, rightTeam = rightTeam, leftTeam
	}

	underAbbrev, odds, err := liveGame.GetOdds()
	if err != nil {
		return nil, err
	}
}
*/

func (s *SportBoard) teamInfoLayers(liveGame Game, bounds image.Rectangle) ([]*rgbrender.TextLayer, error) {
	rightTeam, err := liveGame.HomeTeam()
	if err != nil {
		return nil, err
	}
	leftTeam, err := liveGame.AwayTeam()
	if err != nil {
		return nil, err
	}
	if s.api.League() == mls {
		// MLS does Home team on left
		leftTeam, rightTeam = rightTeam, leftTeam
	}

	s.log.Debug("showing team records",
		zap.String("left", leftTeam.GetAbbreviation()),
		zap.String("right", rightTeam.GetAbbreviation()),
	)

	z := rgbrender.ZeroedBounds(bounds)
	textWidth := s.textAreaWidth(z)
	rightBounds := bounds
	if s.config.ScrollMode.Load() {
		logoWidth := (z.Dx() - textWidth) / 2
		startX := textWidth + logoWidth
		rightBounds = image.Rect(startX, z.Min.Y, startX+teamInfoArea, z.Max.Y)
	}

	leftBounds := bounds
	if s.config.ScrollMode.Load() {
		endX := (z.Dx() - textWidth) / 2
		leftBounds = image.Rect(endX-teamInfoArea, z.Min.Y, endX, z.Max.Y)
	}

	oddStr := ""
	underDog := ""

	if s.config.GamblingSpread.Load() {
		var err error
		underDog, oddStr, err = liveGame.GetOdds()
		if err != nil {
			s.log.Error("failed to get gambling odds for game",
				zap.Error(err),
				zap.String("left team", leftTeam.GetAbbreviation()),
				zap.String("right team", rightTeam.GetAbbreviation()),
			)
		} else {
			underDog = strings.ToUpper(underDog)
			s.log.Info("gambling odds",
				zap.String("underdog", underDog),
				zap.String("odds", oddStr),
			)
		}
	}

	return []*rgbrender.TextLayer{
		rgbrender.NewTextLayer(
			func(ctx context.Context) (*rgbrender.TextWriter, []string, error) {
				rank := s.api.TeamRank(ctx, leftTeam)
				record := s.api.TeamRecord(ctx, leftTeam)

				writer, err := s.getTimeWriter(bounds)
				if err != nil {
					return nil, nil, err
				}

				return writer, []string{rank, record}, nil
			},
			func(canvas board.Canvas, writer *rgbrender.TextWriter, text []string) error {
				if len(text) != 2 {
					return fmt.Errorf("invalid rank/record input")
				}
				rank := text[0]
				record := text[1]
				if rank != "" {
					_ = writer.WriteAlignedBoxed(
						rgbrender.LeftTop,
						canvas,
						leftBounds,
						[]string{rank},
						color.White,
						color.Black,
					)
				}
				if record != "" {
					_ = writer.WriteAlignedBoxed(
						rgbrender.LeftBottom,
						canvas,
						leftBounds,
						[]string{record},
						color.White,
						color.Black,
					)
				}
				if oddStr != "" && strings.ToUpper(leftTeam.GetAbbreviation()) == underDog {
					_ = writer.WriteAlignedBoxed(
						rgbrender.LeftCenter,
						canvas,
						leftBounds,
						[]string{oddStr},
						red,
						color.Black,
					)
				}
				return nil
			},
		),
		rgbrender.NewTextLayer(
			func(ctx context.Context) (*rgbrender.TextWriter, []string, error) {
				rank := s.api.TeamRank(ctx, rightTeam)
				record := s.api.TeamRecord(ctx, rightTeam)

				writer, err := s.getTimeWriter(bounds)
				if err != nil {
					return nil, nil, err
				}

				return writer, []string{rank, record}, nil
			},
			func(canvas board.Canvas, writer *rgbrender.TextWriter, text []string) error {
				if len(text) != 2 {
					return fmt.Errorf("invalid rank/record input")
				}
				rank := text[0]
				record := text[1]
				if rank != "" {
					_ = writer.WriteAlignedBoxed(
						rgbrender.RightTop,
						canvas,
						rightBounds,
						[]string{rank},
						color.White,
						color.Black,
					)
				}
				if record != "" {
					_ = writer.WriteAlignedBoxed(
						rgbrender.RightBottom,
						canvas,
						rightBounds,
						[]string{record},
						color.White,
						color.Black,
					)
				}
				if oddStr != "" && strings.ToUpper(rightTeam.GetAbbreviation()) == underDog {
					_ = writer.WriteAlignedBoxed(
						rgbrender.RightCenter,
						canvas,
						rightBounds,
						[]string{oddStr},
						red,
						color.Black,
					)
				}
				return nil
			},
		),
	}, nil
}
