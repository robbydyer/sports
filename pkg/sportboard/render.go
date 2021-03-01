package sportboard

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/board"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

func (s *SportBoard) renderLiveGame(ctx context.Context, canvas board.Canvas, liveGame Game, counter image.Image) error {
	awayTeam, err := liveGame.AwayTeam()
	if err != nil {
		return err
	}
	homeTeam, err := liveGame.HomeTeam()
	if err != nil {
		return err
	}

	// If this is a favorite team, we'll watch the scoreboard until the game is over
	isFavorite := (s.isFavorite(awayTeam.GetAbbreviation()) || s.isFavorite(homeTeam.GetAbbreviation()))

	timeWriter, err := s.getTimeWriter(canvas.Bounds())
	if err != nil {
		return err
	}

	scoreWriter, err := s.getScoreWriter(canvas.Bounds())
	if err != nil {
		return err
	}

	layers := rgbrender.NewLayerRenderer(60*time.Second, s.log)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		default:
		}

		quarter, err := liveGame.GetQuarter()
		if err != nil {
			return err
		}

		clock, err := liveGame.GetClock()
		if err != nil {
			return err
		}

		score, err := scoreStr(liveGame)
		if err != nil {
			return err
		}

		s.log.Debug("Live game",
			zap.String("score", score),
			zap.String("clock", clock),
		)

		layers.AddLayer(rgbrender.BackgroundPriority,
			rgbrender.NewLayer(
				func(ctx context.Context) (image.Image, error) {
					return s.RenderHomeLogo(ctx, canvas.Bounds(), homeTeam.GetAbbreviation())
				},
				func(canvas board.Canvas, img image.Image) error {
					draw.Draw(canvas, canvas.Bounds(), img, image.Point{}, draw.Over)
					return nil
				},
			),
		)

		layers.AddLayer(rgbrender.BackgroundPriority,
			rgbrender.NewLayer(
				func(ctx context.Context) (image.Image, error) {
					return s.RenderAwayLogo(ctx, canvas.Bounds(), awayTeam.GetAbbreviation())
				},
				func(canvas board.Canvas, img image.Image) error {
					draw.Draw(canvas, canvas.Bounds(), img, image.Point{}, draw.Over)
					return nil
				},
			),
		)

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		default:
		}

		if s.config.TimeColor == nil {
			s.config.TimeColor = color.White
		}

		_ = timeWriter.WriteAligned(
			rgbrender.CenterTop,
			canvas,
			canvas.Bounds(),
			[]string{
				quarter,
				clock,
			},
			s.config.TimeColor,
		)

		if s.config.ScoreColor == nil {
			s.config.ScoreColor = color.White
		}

		if s.config.HideFavoriteScore.Load() && isFavorite {
			s.log.Warn("hiding score for favorite team")
		} else {
			_ = scoreWriter.WriteAligned(
				rgbrender.CenterBottom,
				canvas,
				canvas.Bounds(),
				[]string{
					score,
				},
				s.config.ScoreColor,
			)
		}

		draw.Draw(canvas, canvas.Bounds(), counter, image.Point{}, draw.Over)

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		default:
		}

		s.log.Debug("rendering sportboard",
			zap.String("home", homeTeam.GetAbbreviation()),
			zap.String("away", awayTeam.GetAbbreviation()),
		)
		if err := canvas.Render(); err != nil {
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
	}
}

func (s *SportBoard) renderUpcomingGame(ctx context.Context, canvas board.Canvas, liveGame Game, counter image.Image) error {
	awayTeam, err := liveGame.AwayTeam()
	if err != nil {
		return err
	}
	homeTeam, err := liveGame.HomeTeam()
	if err != nil {
		return err
	}

	timeWriter, err := s.getTimeWriter(canvas.Bounds())
	if err != nil {
		return err
	}

	scoreWriter, err := s.getScoreWriter(canvas.Bounds())
	if err != nil {
		return err
	}

	gameTimeStr := ""

	if is, err := liveGame.IsPostponed(); err == nil && is {
		s.log.Debug("game was postponed", zap.Int("game ID", liveGame.GetID()))
		gameTimeStr = "PPD"
	} else {
		gameTime, err := liveGame.GetStartTime(ctx)
		if err != nil {
			return err
		}
		gameTimeStr = gameTime.Local().Format("3:04PM")
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		img, err := s.RenderHomeLogo(ctx, canvas.Bounds(), homeTeam.GetAbbreviation())
		if err != nil {
			select {
			case errs <- fmt.Errorf("failed to render home logo: %w", err):
			default:
			}
		}
		draw.Draw(canvas, canvas.Bounds(), img, image.Point{}, draw.Over)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		img, err := s.RenderAwayLogo(ctx, canvas.Bounds(), awayTeam.GetAbbreviation())
		if err != nil {
			select {
			case errs <- fmt.Errorf("failed to render home logo: %w", err):
			default:
			}
		}
		draw.Draw(canvas, canvas.Bounds(), img, image.Point{}, draw.Over)
	}()

	// Wait, but with a timeout
	doneWait := make(chan struct{})
	go func() {
		defer close(doneWait)
		wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
	case <-doneWait:
		s.log.Debug("logo renderers completed",
			zap.String("home", homeTeam.GetAbbreviation()),
			zap.String("away", awayTeam.GetAbbreviation()),
		)
	case <-time.After(60 * time.Second):
		return fmt.Errorf("timed out waiting for board to draw logos")
	}

	select {
	case err := <-errs:
		if err != nil {
			return err
		}
	default:
	}

	_ = timeWriter.WriteAligned(
		rgbrender.CenterTop,
		canvas,
		canvas.Bounds(),
		[]string{
			gameTimeStr,
		},
		s.config.TimeColor,
	)

	_ = scoreWriter.WriteAligned(
		rgbrender.CenterCenter,
		canvas,
		canvas.Bounds(),
		[]string{
			"VS",
		},
		s.config.ScoreColor,
	)

	draw.Draw(canvas, canvas.Bounds(), counter, image.Point{}, draw.Over)

	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
	default:
	}

	return canvas.Render()
}

func (s *SportBoard) renderCompleteGame(ctx context.Context, canvas board.Canvas, liveGame Game, counter image.Image) error {
	homeTeam, err := liveGame.HomeTeam()
	if err != nil {
		return err
	}
	awayTeam, err := liveGame.AwayTeam()
	if err != nil {
		return err
	}

	isFavorite := (s.isFavorite(awayTeam.GetAbbreviation()) || s.isFavorite(homeTeam.GetAbbreviation()))

	layers := rgbrender.NewLayerRenderer(60*time.Second, s.log)

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

	if err := layers.Render(ctx, canvas); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
	default:
	}

	return canvas.Render()
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
