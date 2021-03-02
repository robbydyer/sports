package sportboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *SportBoard) renderLiveGame(ctx context.Context, canvas board.Canvas, liveGame Game, counter image.Image) error {
	layers, err := rgbrender.NewLayerRenderer(60*time.Second, s.log)
	if err != nil {
		return err
	}

	logos, err := s.logoLayers(liveGame, canvas.Bounds())
	if err != nil {
		return err
	}

	var infos []*rgbrender.TextLayer
	if s.config.ShowRecord.Load() {
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

		layers.AddTextLayer(rgbrender.ForegroundPriority,
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
					return writer.WriteAligned(
						rgbrender.CenterTop,
						canvas,
						canvas.Bounds(),
						text,
						s.config.TimeColor,
					)
				},
			),
		)

		layers.AddTextLayer(rgbrender.ForegroundPriority,
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
						str, err := scoreStr(liveGame)
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
					return writer.WriteAligned(
						rgbrender.CenterBottom,
						canvas,
						canvas.Bounds(),
						text,
						s.config.ScoreColor,
					)
				},
			),
		)

		layers.AddLayer(rgbrender.ForegroundPriority, counterLayer(counter))

		for _, i := range infos {
			layers.AddTextLayer(rgbrender.ForegroundPriority, i)
		}

		if err := layers.Render(ctx, canvas); err != nil {
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

		select {
		case <-ctx.Done():
			return context.Canceled
		case <-time.After(s.config.boardDelay - 3*time.Second):
		}

		liveGame, err = liveGame.GetUpdate(ctx)
		if err != nil {
			return fmt.Errorf("failed to update stickey game: %w", err)
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
	layers, err := rgbrender.NewLayerRenderer(60*time.Second, s.log)
	if err != nil {
		return err
	}

	logos, err := s.logoLayers(liveGame, canvas.Bounds())
	if err != nil {
		return err
	}

	for _, l := range logos {
		layers.AddLayer(rgbrender.BackgroundPriority, l)
	}

	layers.AddTextLayer(rgbrender.ForegroundPriority,
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
				return writer.WriteAligned(
					rgbrender.CenterTop,
					canvas,
					canvas.Bounds(),
					text,
					s.config.TimeColor,
				)
			},
		),
	)
	layers.AddTextLayer(rgbrender.ForegroundPriority,
		rgbrender.NewTextLayer(
			func(ctx context.Context) (*rgbrender.TextWriter, []string, error) {
				scoreWriter, err := s.getScoreWriter(canvas.Bounds())
				if err != nil {
					return nil, nil, err
				}
				return scoreWriter, []string{"VS"}, nil
			},
			func(canvas board.Canvas, writer *rgbrender.TextWriter, text []string) error {
				return writer.WriteAligned(
					rgbrender.CenterCenter,
					canvas,
					canvas.Bounds(),
					text,
					s.config.ScoreColor,
				)
			},
		),
	)

	layers.AddLayer(rgbrender.ForegroundPriority, counterLayer(counter))

	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
	default:
	}

	return layers.Render(ctx, canvas)
}

func (s *SportBoard) renderCompleteGame(ctx context.Context, canvas board.Canvas, liveGame Game, counter image.Image) error {
	layers, err := rgbrender.NewLayerRenderer(60*time.Second, s.log)
	if err != nil {
		return err
	}

	logos, err := s.logoLayers(liveGame, canvas.Bounds())
	if err != nil {
		return err
	}

	for _, l := range logos {
		layers.AddLayer(rgbrender.BackgroundPriority, l)
	}

	layers.AddTextLayer(rgbrender.ForegroundPriority,
		rgbrender.NewTextLayer(
			func(ctx context.Context) (*rgbrender.TextWriter, []string, error) {
				writer, err := s.getTimeWriter(canvas.Bounds())
				if err != nil {
					return nil, nil, err
				}
				return writer, []string{"FINAL"}, nil
			},
			func(canvas board.Canvas, writer *rgbrender.TextWriter, text []string) error {
				return writer.WriteAligned(
					rgbrender.CenterTop,
					canvas,
					canvas.Bounds(),
					text,
					s.config.TimeColor,
				)
			},
		),
	)

	layers.AddTextLayer(rgbrender.ForegroundPriority,
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
					str, err := scoreStr(liveGame)
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
				return writer.WriteAligned(
					rgbrender.CenterBottom,
					canvas,
					canvas.Bounds(),
					text,
					s.config.ScoreColor,
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

	layers.AddLayer(rgbrender.ForegroundPriority, counterLayer(counter))

	layers.AddLayer(rgbrender.ForegroundPriority,
		rgbrender.NewLayer(
			nil,
			func(canvas board.Canvas, img image.Image) error {
				draw.Draw(canvas, canvas.Bounds(), counter, image.Point{}, draw.Over)
				return nil
			},
		),
	)

	return layers.Render(ctx, canvas)
}

func counterLayer(counter image.Image) *rgbrender.Layer {
	return rgbrender.NewLayer(
		nil,
		func(canvas board.Canvas, img image.Image) error {
			draw.Draw(canvas, canvas.Bounds(), counter, image.Point{}, draw.Over)
			return nil
		},
	)
}

func (s *SportBoard) logoLayers(liveGame Game, bounds image.Rectangle) ([]*rgbrender.Layer, error) {
	homeTeam, err := liveGame.HomeTeam()
	if err != nil {
		return nil, err
	}
	awayTeam, err := liveGame.AwayTeam()
	if err != nil {
		return nil, err
	}

	return []*rgbrender.Layer{
		rgbrender.NewLayer(
			func(ctx context.Context) (image.Image, error) {
				return s.RenderHomeLogo(ctx, bounds, homeTeam.GetAbbreviation())
			},
			func(canvas board.Canvas, img image.Image) error {
				draw.Draw(canvas, canvas.Bounds(), img, image.Point{}, draw.Over)
				return nil
			},
		),

		rgbrender.NewLayer(
			func(ctx context.Context) (image.Image, error) {
				return s.RenderAwayLogo(ctx, bounds, awayTeam.GetAbbreviation())
			},
			func(canvas board.Canvas, img image.Image) error {
				draw.Draw(canvas, canvas.Bounds(), img, image.Point{}, draw.Over)
				return nil
			},
		),
	}, nil
}

func (s *SportBoard) teamInfoLayers(liveGame Game, bounds image.Rectangle) ([]*rgbrender.TextLayer, error) {
	homeTeam, err := liveGame.HomeTeam()
	if err != nil {
		return nil, err
	}
	awayTeam, err := liveGame.AwayTeam()
	if err != nil {
		return nil, err
	}

	return []*rgbrender.TextLayer{
		rgbrender.NewTextLayer(
			func(ctx context.Context) (*rgbrender.TextWriter, []string, error) {
				rank := s.api.TeamRank(ctx, homeTeam)
				record := s.api.TeamRecord(ctx, homeTeam)

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
					_ = writer.WriteAligned(rgbrender.LeftTop, canvas, canvas.Bounds(), []string{rank}, color.White)
				}
				if record != "" {
					_ = writer.WriteAligned(rgbrender.LeftBottom, canvas, canvas.Bounds(), []string{record}, color.White)
				}
				return nil
			},
		),
		rgbrender.NewTextLayer(
			func(ctx context.Context) (*rgbrender.TextWriter, []string, error) {
				rank := s.api.TeamRank(ctx, awayTeam)
				record := s.api.TeamRecord(ctx, awayTeam)

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
					_ = writer.WriteAligned(rgbrender.RightTop, canvas, canvas.Bounds(), []string{rank}, color.White)
				}
				if record != "" {
					_ = writer.WriteAligned(rgbrender.RightBottom, canvas, canvas.Bounds(), []string{record}, color.White)
				}
				return nil
			},
		),
	}, nil
}
