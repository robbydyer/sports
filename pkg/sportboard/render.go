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

const (
	mls                 = "MLS"
	defaultTeamInfoArea = 22
	teamInfoPad         = 2
)

var red = color.RGBA{255, 0, 0, 255}

func (s *SportBoard) homeSide() side {
	if s.api.League() == mls {
		return left
	}
	return right
}

func (s *SportBoard) renderLoading(ctx context.Context, canvas board.Canvas) {
	writer, err := s.getTimeWriter(canvas.Bounds())
	if err != nil {
		s.log.Error("failed to get writer for loading screen",
			zap.Error(err),
		)
		return
	}

	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Second):
	}
	s.log.Info("rendering loading screen",
		zap.String("league", s.api.League()),
	)

LOOP:
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = writer.WriteAlignedBoxed(
			rgbrender.CenterCenter,
			canvas,
			rgbrender.ZeroedBounds(canvas.Bounds()),
			[]string{
				s.api.League(),
				"Loading...",
			},
			color.White,
			color.Black,
		)
		if err := canvas.Render(ctx); err != nil {
			return
		}
		if !canvas.Scrollable() {
			break LOOP
		}
	}
}

func (s *SportBoard) renderLiveGame(ctx context.Context, canvas board.Canvas, liveGame Game, counter image.Image) error {
	s.logCanvas(canvas, "render live canvas size")

	layers, err := rgbrender.NewLayerDrawer(60*time.Second, s.log)
	if err != nil {
		return err
	}

	if s.config.ShowRecord.Load() || s.config.GamblingSpread.Load() {
		var err error
		infos, err := s.teamInfoLayers(canvas, liveGame, canvas.Bounds())
		if err != nil {
			return err
		}
		for _, i := range infos {
			layers.AddTextLayer(rgbrender.ForegroundPriority, i)
		}
	}

	logos, err := s.logoLayers(liveGame, canvas.Bounds())
	if err != nil {
		return err
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
					writer, err := s.getScoreWriter(canvas.Bounds())
					if err != nil {
						return nil, nil, err
					}
					return writer, []string{}, nil
				},
				func(canvas board.Canvas, writer *rgbrender.TextWriter, text []string) error {
					isFavorite, err := s.isFavoriteGame(liveGame)
					if err != nil {
						return err
					}
					if s.config.HideFavoriteScore.Load() && isFavorite {
						s.log.Warn("hiding score for favorite team")
						return nil
					}
					a, err := liveGame.AwayTeam()
					if err != nil {
						return err
					}
					h, err := liveGame.HomeTeam()
					if err != nil {
						return err
					}
					aScore := a.Score()
					hScore := h.Score()
					s.log.Debug("writing scores",
						zap.Int("home", hScore),
						zap.Int("away", aScore),
					)
					prev := s.storeOrGetPreviousScore(liveGame.GetID(), a.Score(), h.Score())
					var chars []string
					var clrs []color.Color
					if s.homeSide() == left {
						chars = []string{
							fmt.Sprintf("%d", hScore),
							"-",
							fmt.Sprintf("%d", aScore),
						}
						if prev.home.hasScored(hScore) {
							clrs = append(clrs, red)
							s.log.Debug("home team scored")
						} else {
							clrs = append(clrs, color.White)
						}
						clrs = append(clrs, color.White)
						if prev.away.hasScored(aScore) {
							clrs = append(clrs, red)
							s.log.Debug("away team scored")
						} else {
							clrs = append(clrs, color.White)
						}
					} else {
						chars = []string{
							fmt.Sprintf("%d", aScore),
							"-",
							fmt.Sprintf("%d", hScore),
						}
						if prev.away.hasScored(aScore) {
							clrs = append(clrs, red)
							s.log.Debug("away team scored")
						} else {
							clrs = append(clrs, color.White)
						}
						clrs = append(clrs, color.White)
						if prev.home.hasScored(hScore) {
							clrs = append(clrs, red)
							s.log.Debug("home team scored")
						} else {
							clrs = append(clrs, color.White)
						}
					}
					clrCodes := &rgbrender.ColorChar{
						BoxClr: color.Black,
						Lines: []*rgbrender.ColorCharLine{
							{
								Chars: chars,
								Clrs:  clrs,
							},
						},
					}
					if err := writer.WriteAlignedColorCodes(
						rgbrender.CenterBottom,
						canvas,
						rgbrender.ZeroedBounds(canvas.Bounds()),
						clrCodes,
					); err != nil {
						s.log.Error("failed to write multicolored str", zap.Error(err))
						return err
					}

					return nil
				},
			),
		)

		if counter != nil {
			layers.AddLayer(rgbrender.ForegroundPriority, counterLayer(counter))
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

	if s.config.ShowRecord.Load() || s.config.GamblingSpread.Load() {
		infos, err := s.teamInfoLayers(canvas, liveGame, canvas.Bounds())
		if err != nil {
			return err
		}
		for _, i := range infos {
			layers.AddTextLayer(rgbrender.ForegroundPriority, i)
		}
	}

	logos, err := s.logoLayers(liveGame, canvas.Bounds())
	if err != nil {
		return err
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

	if s.config.ShowRecord.Load() || s.config.GamblingSpread.Load() {
		infos, err := s.teamInfoLayers(canvas, liveGame, canvas.Bounds())
		if err != nil {
			return err
		}
		for _, i := range infos {
			layers.AddTextLayer(rgbrender.ForegroundPriority, i)
		}
	}

	// Give the logoLayers the real bounds, as it already accounts for scroll mode itself
	logos, err := s.logoLayers(liveGame, canvas.Bounds())
	if err != nil {
		return err
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

func (s *SportBoard) teamInfoLayers(canvas draw.Image, liveGame Game, bounds image.Rectangle) ([]*rgbrender.TextLayer, error) {
	rightTeam, err := liveGame.HomeTeam()
	if err != nil {
		return nil, err
	}
	leftTeam, err := liveGame.AwayTeam()
	if err != nil {
		return nil, err
	}

	longestScore := numDigits(leftTeam.Score())
	if numDigits(rightTeam.Score()) > longestScore {
		longestScore = numDigits(rightTeam.Score())
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

	leftBounds := bounds
	rightBounds := bounds
	return []*rgbrender.TextLayer{
		rgbrender.NewTextLayer(
			func(ctx context.Context) (*rgbrender.TextWriter, []string, error) {
				rank := s.api.TeamRank(ctx, leftTeam, s.season())
				record := s.api.TeamRecord(ctx, leftTeam, s.season())

				writer, err := s.getTimeWriter(bounds)
				if err != nil {
					return nil, nil, err
				}

				/*
					if s.hasNoInfo(rank, record, oddStr, underDog, leftTeam.GetAbbreviation()) {
						return writer, []string{rank, record}, nil
					}
				*/

				widthStrs := []string{}
				if s.config.ShowRecord.Load() {
					widthStrs = append(widthStrs, rank, record)
				}
				if s.config.GamblingSpread.Load() {
					widthStrs = append(widthStrs, oddStr)
				}

				if !s.config.ScrollMode.Load() {
					w := 0
					if float32(z.Dx())/float32(z.Dy()) > 2.0 {
						var err error
						w, err = s.calculateTeamInfoWidth(canvas, writer, widthStrs)
						if err != nil {
							s.log.Error("failed to calculate team info width, using default",
								zap.Error(err),
							)
						}
					}
					s.log.Debug("set team info width",
						zap.Int("width", w),
						zap.String("league", s.api.League()),
						zap.String("team", leftTeam.GetAbbreviation()),
					)
					s.setTeamInfoWidth(s.api.League(), leftTeam.GetAbbreviation(), w)
					maxX := (leftBounds.Bounds().Dx() - s.textAreaWidth(leftBounds)) / 2
					leftBounds = image.Rect(leftBounds.Min.X, leftBounds.Min.Y, maxX, leftBounds.Max.Y)

					return writer, []string{rank, record}, nil
				}

				// Scroll mode
				infoWidth, err := s.getTeamInfoWidth(s.api.League(), leftTeam.GetAbbreviation())
				if err != nil || infoWidth == 0 {
					var err error
					infoWidth, err = s.calculateTeamInfoWidth(canvas, writer, widthStrs)
					if err != nil {
						s.log.Error("failed to calculate team info width, using default",
							zap.Error(err),
						)
					}
					infoWidth += teamInfoPad
					s.log.Debug("setting team info width",
						zap.String("league", s.api.League()),
						zap.String("team", leftTeam.GetAbbreviation()),
						zap.Int("width", infoWidth),
					)
					s.setTeamInfoWidth(s.api.League(), leftTeam.GetAbbreviation(), infoWidth)
				}
				endX := ((z.Dx() - textWidth) / 2) - teamInfoPad
				if longestScore == 2 {
					endX -= 5
				} else if longestScore == 3 {
					endX -= 9
				}
				leftBounds = image.Rect(endX-infoWidth, z.Min.Y, endX, z.Max.Y)

				return writer, []string{rank, record}, nil
			},
			func(canvas board.Canvas, writer *rgbrender.TextWriter, text []string) error {
				if len(text) != 2 {
					return fmt.Errorf("invalid rank/record input")
				}
				rank := text[0]
				record := text[1]

				if s.hasNoInfo(rank, record, oddStr, underDog, leftTeam.GetAbbreviation()) {
					return nil
				}

				if rank != "" && s.config.ShowRecord.Load() {
					_ = writer.WriteAlignedBoxed(
						rgbrender.RightTop,
						canvas,
						leftBounds,
						[]string{rank},
						color.White,
						color.Black,
					)
				}
				if record != "" && s.config.ShowRecord.Load() {
					_ = writer.WriteAlignedBoxed(
						rgbrender.RightBottom,
						canvas,
						leftBounds,
						[]string{record},
						color.White,
						color.Black,
					)
				}
				if s.config.GamblingSpread.Load() && oddStr != "" && strings.ToUpper(leftTeam.GetAbbreviation()) == underDog {
					_ = writer.WriteAlignedBoxed(
						rgbrender.RightCenter,
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
				rank := s.api.TeamRank(ctx, rightTeam, s.season())
				record := s.api.TeamRecord(ctx, rightTeam, s.season())

				writer, err := s.getTimeWriter(bounds)
				if err != nil {
					return nil, nil, err
				}

				/*
					if s.hasNoInfo(rank, record, oddStr, underDog, rightTeam.GetAbbreviation()) {
						return writer, []string{rank, record}, nil
					}
				*/

				widthStrs := []string{}
				if s.config.ShowRecord.Load() {
					widthStrs = append(widthStrs, rank, record)
				}
				if s.config.GamblingSpread.Load() {
					widthStrs = append(widthStrs, oddStr)
				}

				if !s.config.ScrollMode.Load() {
					w := 0
					if float32(z.Dx())/float32(z.Dy()) > 2.0 {
						var err error
						w, err = s.calculateTeamInfoWidth(canvas, writer, widthStrs)
						if err != nil {
							s.log.Error("failed to calculate team info width, using default",
								zap.Error(err),
							)
						}
					}
					s.log.Debug("set team info width",
						zap.Int("width", w),
						zap.String("league", s.api.League()),
						zap.String("team", rightTeam.GetAbbreviation()),
					)
					s.setTeamInfoWidth(s.api.League(), rightTeam.GetAbbreviation(), w)
					minX := ((rightBounds.Bounds().Dx() - s.textAreaWidth(rightBounds)) / 2) + s.textAreaWidth(rightBounds)
					rightBounds = image.Rect(minX, rightBounds.Min.Y, rightBounds.Max.X, rightBounds.Max.Y)

					return writer, []string{rank, record}, nil
				}

				// Scroll mode
				infoWidth, err := s.getTeamInfoWidth(s.api.League(), rightTeam.GetAbbreviation())
				if err != nil || infoWidth == 0 {
					var err error
					infoWidth, err = s.calculateTeamInfoWidth(canvas, writer, widthStrs)
					if err != nil {
						s.log.Error("failed to calculate team info width, using default",
							zap.Error(err),
						)
					}
					s.log.Debug("setting team info width",
						zap.String("league", s.api.League()),
						zap.String("team", rightTeam.GetAbbreviation()),
						zap.Int("width", infoWidth),
					)
					s.setTeamInfoWidth(s.api.League(), rightTeam.GetAbbreviation(), infoWidth)
				}
				logoWidth := (z.Dx() - textWidth) / 2
				startX := textWidth + logoWidth + teamInfoPad
				if longestScore == 2 {
					startX += 5
				} else if longestScore == 3 {
					startX += 9
				}
				rightBounds = image.Rect(startX, z.Min.Y, startX+infoWidth, z.Max.Y)

				return writer, []string{rank, record}, nil
			},
			func(canvas board.Canvas, writer *rgbrender.TextWriter, text []string) error {
				if len(text) != 2 {
					return fmt.Errorf("invalid rank/record input")
				}
				rank := text[0]
				record := text[1]

				if s.hasNoInfo(rank, record, oddStr, underDog, rightTeam.GetAbbreviation()) {
					return nil
				}

				if rank != "" && s.config.ShowRecord.Load() {
					_ = writer.WriteAlignedBoxed(
						rgbrender.LeftTop,
						canvas,
						rightBounds,
						[]string{rank},
						color.White,
						color.Black,
					)
				}
				if record != "" && s.config.ShowRecord.Load() {
					_ = writer.WriteAlignedBoxed(
						rgbrender.LeftBottom,
						canvas,
						rightBounds,
						[]string{record},
						color.White,
						color.Black,
					)
				}
				if s.config.GamblingSpread.Load() && oddStr != "" && strings.ToUpper(rightTeam.GetAbbreviation()) == underDog {
					_ = writer.WriteAlignedBoxed(
						rgbrender.LeftCenter,
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

func (s *SportBoard) hasNoInfo(rank string, record string, odds string, underDog string, thisTeam string) bool {
	if (s.config.ShowRecord.Load() && rank == "" && record == "") && (s.config.GamblingSpread.Load() && odds == "") {
		return true
	}
	if !s.config.ShowRecord.Load() && (s.config.GamblingSpread.Load() && odds == "") {
		return true
	}
	if !s.config.GamblingSpread.Load() && s.config.ShowRecord.Load() && rank == "" && record == "" {
		return true
	}
	if !s.config.ShowRecord.Load() && s.config.GamblingSpread.Load() && strings.EqualFold(underDog, thisTeam) {
		return true
	}

	return false
}

func (s *SportBoard) renderNoScheduled(ctx context.Context, canvas board.Canvas) error {
	s.log.Debug("no scheduled games", zap.String("league", s.api.League()))
	if !s.config.ShowNoScheduledLogo.Load() {
		return nil
	}

	writer, err := s.getTimeWriter(canvas.Bounds())
	if err != nil {
		return fmt.Errorf("failed to get writer for no scheduled game output: %w", err)
	}

	b := rgbrender.ZeroedBounds(canvas.Bounds())
	writeBox := image.Rect((b.Max.X/2)+2, 0, b.Max.X, b.Max.Y)
	logoBox := image.Rect(0, 0, (b.Max.X/2)-2, b.Max.Y)

	_ = writer.WriteAlignedBoxed(
		rgbrender.CenterCenter,
		canvas,
		logoBox,
		[]string{
			s.api.League(),
		},
		color.White,
		color.Black,
	)

	_ = writer.WriteAlignedBoxed(
		rgbrender.RightCenter,
		canvas,
		writeBox,
		[]string{
			"No",
			"Games",
			"Today",
		},
		color.White,
		color.Black,
	)

	if err := canvas.Render(ctx); err != nil {
		return err
	}

	if s.config.ScrollMode.Load() && canvas.Scrollable() {
		return nil
	}

	select {
	case <-ctx.Done():
		return context.Canceled
	case <-time.After(s.config.boardDelay / 2):
		return nil
	}
}
