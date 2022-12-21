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

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/logo"
	"github.com/robbydyer/sports/internal/rgbrender"
)

const (
	defaultTeamInfoArea = 22
	teamInfoPad         = 3
)

var (
	red                   = color.RGBA{255, 0, 0, 255}
	green                 = color.RGBA{0, 255, 0, 255}
	infoLayerPriority     = rgbrender.BackgroundPriority + 2
	counterLayerPriority  = rgbrender.ForegroundPriority
	scoreLayerPriority    = rgbrender.BackgroundPriority + 3
	logoLayerPriority     = rgbrender.BackgroundPriority
	gradientLayerPriority = rgbrender.BackgroundPriority + 1
)

func (s *SportBoard) homeSide() side {
	if s.api.HomeSideSwap() {
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
			layers.AddTextLayer(infoLayerPriority, i)
		}
	}

	logos, err := s.logoLayers(liveGame, canvas.Bounds())
	if err != nil {
		return err
	}

	var gradientLayers []*rgbrender.Layer
	if s.config.UseGradient.Load() {
		score, err := scoreStr(liveGame, s.homeSide())
		if err != nil {
			return err
		}
		gradientLayers = s.gradientLayer(rgbrender.ZeroedBounds(canvas.Bounds()), len(strings.ReplaceAll(score, " ", "")))
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
	default:
	}

	for _, l := range gradientLayers {
		layers.AddLayer(gradientLayerPriority, l)
	}
	for _, l := range logos {
		layers.AddLayer(logoLayerPriority, l)
	}

	layers.AddTextLayer(scoreLayerPriority,
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
					s.writeBoxColor(),
				)
			},
		),
	)

	layers.AddTextLayer(scoreLayerPriority,
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
		layers.AddLayer(counterLayerPriority, counterLayer(counter))
	}

	if err := layers.Draw(ctx, canvas); err != nil {
		return err
	}

	return nil
}

func (s *SportBoard) renderUpcomingGame(ctx context.Context, canvas board.Canvas, liveGame Game, counter image.Image) error {
	layers, err := rgbrender.NewLayerDrawer(60*time.Second, s.log)
	if err != nil {
		return err
	}

	if counter != nil {
		layers.AddLayer(counterLayerPriority, counterLayer(counter))
	}

	if s.config.ShowRecord.Load() || s.config.GamblingSpread.Load() {
		infos, err := s.teamInfoLayers(canvas, liveGame, canvas.Bounds())
		if err != nil {
			return err
		}
		for _, i := range infos {
			layers.AddTextLayer(infoLayerPriority, i)
		}
	}

	logos, err := s.logoLayers(liveGame, canvas.Bounds())
	if err != nil {
		return err
	}

	if s.config.UseGradient.Load() {
		for _, l := range s.gradientLayer(rgbrender.ZeroedBounds(canvas.Bounds()), 0) {
			s.log.Debug("Adding gradient layer")
			layers.AddLayer(gradientLayerPriority, l)
		}
	}

	for _, l := range logos {
		layers.AddLayer(logoLayerPriority, l)
	}

	layers.AddTextLayer(scoreLayerPriority,
		rgbrender.NewTextLayer(
			func(ctx context.Context) (*rgbrender.TextWriter, []string, error) {
				timeWriter, err := s.getTimeWriter(canvas.Bounds())
				if err != nil {
					return nil, nil, err
				}
				gameTimeStr := ""
				dateStr := ""
				if is, err := liveGame.IsPostponed(); err == nil && is {
					s.log.Debug("game was postponed", zap.Int("game ID", liveGame.GetID()))
					gameTimeStr = "PPD"
				} else {
					gameTime, err := liveGame.GetStartTime(ctx)
					if err != nil {
						return nil, nil, err
					}
					gameTimeStr = gameTime.Local().Format("3:04PM")
					if gameTime.Local().Format("01/02/2006") != time.Now().Local().Format("01/02/2006") {
						dateStr = gameTime.Local().Format("01/02")
					}

					if s.config.Enable24Hour.Load() {
						gameTimeStr = gameTime.Local().Format("15:04")
					}

					s.log.Debug("game time",
						zap.String("time", gameTimeStr),
						zap.String("day", dateStr),
					)
				}
				return timeWriter, []string{gameTimeStr, dateStr}, nil
			},
			func(canvas board.Canvas, writer *rgbrender.TextWriter, text []string) error {
				return writer.WriteAlignedBoxed(
					rgbrender.CenterTop,
					canvas,
					rgbrender.ZeroedBounds(canvas.Bounds()),
					text,
					s.config.TimeColor,
					s.writeBoxColor(),
				)
			},
		),
	)
	layers.AddTextLayer(scoreLayerPriority,
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
					rgbrender.CenterBottom,
					canvas,
					rgbrender.ZeroedBounds(canvas.Bounds()),
					text,
					s.config.ScoreColor,
					s.writeBoxColor(),
				)
			},
		),
	)

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
			layers.AddTextLayer(infoLayerPriority, i)
		}
	}

	if s.config.UseGradient.Load() {
		score, err := scoreStr(liveGame, s.homeSide())
		if err != nil {
			return err
		}
		for _, l := range s.gradientLayer(rgbrender.ZeroedBounds(canvas.Bounds()), len(strings.ReplaceAll(score, " ", ""))) {
			layers.AddLayer(gradientLayerPriority, l)
		}
	}

	// Give the logoLayers the real bounds, as it already accounts for scroll mode itself
	logos, err := s.logoLayers(liveGame, canvas.Bounds())
	if err != nil {
		return err
	}

	for _, l := range logos {
		layers.AddLayer(logoLayerPriority, l)
	}

	layers.AddTextLayer(scoreLayerPriority,
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
					s.writeBoxColor(),
				)
			},
		),
	)

	layers.AddTextLayer(scoreLayerPriority,
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
					s.writeBoxColor(),
				)
			},
		),
	)

	if counter != nil {
		layers.AddLayer(counterLayerPriority, counterLayer(counter))
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

	if s.api.HomeSideSwap() {
		// MLS does Home team on left
		leftTeam, rightTeam = rightTeam, leftTeam
	}

	return []*rgbrender.Layer{
		rgbrender.NewLayer(
			func(ctx context.Context) (image.Image, error) {
				l, err := s.RenderLeftLogo(ctx, bounds, leftTeam.GetID())
				if err != nil {
					s.log.Error("failed to render left logo",
						zap.Error(err),
					)
				}

				return l, nil
			},
			func(canvas board.Canvas, img image.Image) error {
				if img == nil {
					writer, err := s.getTimeWriter(bounds)
					if err != nil {
						return err
					}
					_ = writer.WriteAligned(
						rgbrender.LeftCenter,
						canvas,
						rgbrender.ZeroedBounds(bounds),
						[]string{leftTeam.GetAbbreviation()},
						color.White,
					)
					return nil
				}
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
				l, err := s.RenderRightLogo(ctx, bounds, rightTeam.GetID())
				if err != nil {
					s.log.Error("failed to render right logo",
						zap.Error(err),
					)
				}

				return l, nil
			},
			func(canvas board.Canvas, img image.Image) error {
				if img == nil {
					writer, err := s.getTimeWriter(bounds)
					if err != nil {
						return err
					}
					_ = writer.WriteAligned(
						rgbrender.RightCenter,
						canvas,
						rgbrender.ZeroedBounds(bounds),
						[]string{rightTeam.GetAbbreviation()},
						color.White,
					)
					return nil
				}
				pt := image.Pt(img.Bounds().Min.X, img.Bounds().Min.Y)
				draw.Draw(canvas, img.Bounds(), img, pt, draw.Over)
				return nil
			},
		),
	}, nil
}

func (s *SportBoard) gradientLayer(bounds image.Rectangle, scoreLen int) []*rgbrender.Layer {
	txtArea := s.textAreaWidth(bounds)

	var width int
	if scoreLen == 5 {
		width = int(float64(txtArea) * 4.2)
	} else if scoreLen > 5 {
		width = int(float64(txtArea) * 4.7)
	} else {
		width = int(float64(txtArea) * 3.7)
	}

	gradientBounds := image.Rect(
		bounds.Min.X+((bounds.Max.X-width)/2),
		bounds.Min.Y,
		bounds.Min.X+((bounds.Max.X-width)/2)+width,
		bounds.Max.Y,
	)
	fillPct := float64(txtArea) / float64(width)

	s.log.Debug("gradient",
		zap.String("league", s.api.League()),
		zap.Int("start X", gradientBounds.Min.X),
		zap.Int("start Y", gradientBounds.Min.Y),
		zap.Int("end X", gradientBounds.Max.X),
		zap.Int("end Y", gradientBounds.Max.Y),
		zap.Int("width", width),
		zap.Int("fill width", int((float64(gradientBounds.Dx())*fillPct))),
	)
	return []*rgbrender.Layer{
		rgbrender.NewLayer(
			func(ctx context.Context) (image.Image, error) {
				gradient := rgbrender.GradientXRectangle(gradientBounds, fillPct, color.Black, s.log)
				return gradient, nil
			},
			func(canvas board.Canvas, img image.Image) error {
				pt := image.Pt(img.Bounds().Min.X, img.Bounds().Min.Y)
				draw.Draw(canvas, img.Bounds(), img, pt, draw.Over)

				return nil
			},
		),
	}
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
	rightScore := numDigits(rightTeam.Score())
	if rightScore > longestScore {
		longestScore = rightScore
	}

	if s.api.HomeSideSwap() {
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

	shiftMyRank := rankShift(canvas.Bounds())
	s.log.Debug("rank shift",
		zap.Int("shift", shiftMyRank),
		zap.Int("canvas X", canvas.Bounds().Dx()),
	)

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
					s.setTeamInfoWidth(s.api.League(), leftTeam.GetID(), w)
					maxX := (leftBounds.Bounds().Dx() - s.textAreaWidth(leftBounds)) / 2
					maxX -= teamInfoPad
					leftBounds = image.Rect(leftBounds.Min.X, leftBounds.Min.Y, maxX, leftBounds.Max.Y)

					return writer, []string{rank, record}, nil
				}

				// Scroll mode
				infoWidth, err := s.getTeamInfoWidth(s.api.League(), leftTeam.GetID())
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
					s.setTeamInfoWidth(s.api.League(), leftTeam.GetID(), infoWidth)
				}
				endX := ((z.Dx() - textWidth) / 2) - teamInfoPad
				switch longestScore {
				case 2:
					endX -= 5
				case 3:
					endX -= 9
				default:
					endX -= 2
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

				if rank != "" && s.config.ShowRecord.Load() {
					rankBounds := image.Rect(leftBounds.Min.X, leftBounds.Min.Y, leftBounds.Max.X-shiftMyRank, leftBounds.Max.Y)
					_ = writer.WriteAlignedBoxed(
						rgbrender.RightTop,
						canvas,
						rankBounds,
						[]string{rank},
						green,
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
					s.setTeamInfoWidth(s.api.League(), rightTeam.GetID(), w)
					minX := ((rightBounds.Bounds().Dx() - s.textAreaWidth(rightBounds)) / 2) + s.textAreaWidth(rightBounds)
					minX += teamInfoPad
					rightBounds = image.Rect(minX, rightBounds.Min.Y, rightBounds.Max.X, rightBounds.Max.Y)

					return writer, []string{rank, record}, nil
				}

				// Scroll mode
				infoWidth, err := s.getTeamInfoWidth(s.api.League(), rightTeam.GetID())
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
					s.setTeamInfoWidth(s.api.League(), rightTeam.GetID(), infoWidth)
				}
				logoWidth := (z.Dx() - textWidth) / 2
				startX := textWidth + logoWidth + teamInfoPad
				switch longestScore {
				case 2:
					startX += 5
				case 3:
					startX += 9
				default:
					startX += 2
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

				if rank != "" && s.config.ShowRecord.Load() {
					rankBounds := image.Rect(rightBounds.Min.X+shiftMyRank, rightBounds.Min.Y, rightBounds.Max.X, rightBounds.Max.Y)
					_ = writer.WriteAlignedBoxed(
						rgbrender.LeftTop,
						canvas,
						rankBounds,
						[]string{rank},
						green,
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

func (s *SportBoard) renderLeagueLogo(ctx context.Context, canvas board.Canvas) error {
	s.logoLock.Lock()
	defer s.logoLock.Unlock()

	l, ok := s.logos["league"]
	if !ok {
		l = logo.New(
			s.api.League(),
			s.leagueLogoGetter,
			"/tmp/sportsmatrix/leaguelogos",
			canvas.Bounds(),
			&logo.Config{
				FitImage: true,
				Abbrev:   s.api.League(),
				Pt: &logo.Pt{
					X:    0,
					Y:    0,
					Zoom: 1,
				},
			},
		)
		s.logos["league"] = l
	}

	zeroed := rgbrender.ZeroedBounds(canvas.Bounds())
	img, err := l.GetThumbnail(ctx, zeroed)
	if err != nil {
		return err
	}

	if img == nil {
		return fmt.Errorf("failed to get league logo thumbnail")
	}

	draw.Draw(canvas, zeroed, img, image.Point{}, draw.Over)

	return nil
}
