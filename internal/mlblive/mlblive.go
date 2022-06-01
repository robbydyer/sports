package mlblive

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/logo"
	"github.com/robbydyer/sports/internal/rgbrender"
)

var fillColor = color.RGBA{255, 255, 0, 100}

const (
	maxCanvasWidth = 64
)

type MlbLive struct {
	Logger *zap.Logger
	writer *rgbrender.TextWriter
}

type Runners struct {
	First  bool
	Second bool
	Third  bool
}

type InningState struct {
	Number   string
	IsTop    bool
	IsMiddle bool
	Outs     int
}

type Game interface {
	GetRunners(ctx context.Context) (*Runners, error)
	GetCount(ctx context.Context) (string, error)
	GetInningState(ctx context.Context) (*InningState, error)
	GetHomeScore(ctx context.Context) (int, error)
	GetAwayScore(ctx context.Context) (int, error)
	HomeAbbrev() string
	AwayAbbrev() string
	HomeColor() (*color.RGBA, *color.RGBA, error)
	AwayColor() (*color.RGBA, *color.RGBA, error)
}

func getCanvasWidth(canvas board.Canvas) int {
	min := rgbrender.ZeroedBounds(canvas.Bounds()).Dx()
	if min > maxCanvasWidth {
		min = maxCanvasWidth
	}
	return min
}

func (m *MlbLive) RenderLive(ctx context.Context, canvas board.Canvas, game Game, homeLogo *logo.Logo, awayLogo *logo.Logo) error {
	zeroed := rgbrender.ZeroedBounds(canvas.Bounds())
	midX := zeroed.Max.X / 2

	canvasWidth := getCanvasWidth(canvas)

	quarterW := canvasWidth / 4

	awayLogoBounds := image.Rect(midX-(canvasWidth/2), zeroed.Min.Y, midX-(quarterW), zeroed.Max.Y/2)
	awayScoreBounds := image.Rect(awayLogoBounds.Max.X, zeroed.Min.Y+1, midX, (zeroed.Max.Y / 2))

	homeLogoBounds := image.Rect(midX-(canvasWidth/2), zeroed.Max.Y/2, midX-(quarterW), zeroed.Max.Y)
	homeScoreBounds := image.Rect(homeLogoBounds.Max.X, (zeroed.Max.Y / 2), midX, zeroed.Max.Y-1)

	runnerBounds := image.Rect(midX, zeroed.Min.Y, midX+(canvasWidth/2), (zeroed.Max.Y/4)*3)

	awayLogoImg, err := awayLogo.GetThumbnail(ctx, awayLogoBounds.Bounds())
	if err != nil {
		return err
	}
	homeLogoImg, err := homeLogo.GetThumbnail(ctx, homeLogoBounds.Bounds())
	if err != nil {
		return err
	}

	writer, err := m.getWriter(image.Rect(0, 0, canvasWidth, 32))
	if err != nil {
		return err
	}

	origX := writer.XStartCorrection
	origY := writer.YStartCorrection

	defer func() {
		writer.XStartCorrection = origX
		writer.YStartCorrection = origY
	}()

	homeScore, err := game.GetHomeScore(ctx)
	if err != nil {
		return err
	}

	awayScore, err := game.GetAwayScore(ctx)
	if err != nil {
		return err
	}

	draw.Draw(canvas, awayLogoBounds, awayLogoImg, image.Point{}, draw.Over)
	draw.Draw(canvas, homeLogoBounds, homeLogoImg, image.Point{}, draw.Over)

	var homeClr color.Color
	var awayClr color.Color
	homeClr, _, err = game.HomeColor()
	if err != nil {
		homeClr = color.White
	}
	awayClr, _, err = game.AwayColor()
	if err != nil {
		awayClr = color.White
	}

	img := rgbrender.GradientXRectangle(awayScoreBounds, 0.0, awayClr, m.Logger)
	draw.Draw(canvas, img.Bounds(), img, image.Pt(img.Bounds().Min.X, img.Bounds().Min.Y), draw.Over)
	img = rgbrender.GradientXRectangle(homeScoreBounds, 0.0, homeClr, m.Logger)
	draw.Draw(canvas, img.Bounds(), img, image.Pt(img.Bounds().Min.X, img.Bounds().Min.Y), draw.Over)

	writer.XStartCorrection = 2

	m.writeScore(canvas, awayScoreBounds, writer, game.AwayAbbrev(), awayScore, color.White)
	m.writeScore(canvas, homeScoreBounds, writer, game.HomeAbbrev(), homeScore, color.White)

	runners, err := game.GetRunners(ctx)
	if err != nil {
		m.Logger.Error("could not get live baseball runner info",
			zap.Error(err),
		)
		runners = &Runners{}
	}

	diamondW := runnerBounds.Dx() / 4
	diamondStartShift := runnerBounds.Min.X + (runnerBounds.Dx() - ((diamondW + 1) * 3))

	m.Logger.Debug("diamond start shift",
		zap.Int("shift", diamondStartShift-runnerBounds.Min.X),
		zap.Int("width", diamondW),
		zap.Int("runner width", runnerBounds.Dx()),
	)

	var thirdFill color.Color
	thirdFill = color.Black
	if runners.Third {
		thirdFill = fillColor
	}
	// Third base
	rgbrender.DrawDiamond(
		canvas,
		image.Pt(diamondStartShift, runnerBounds.Max.Y/2),
		diamondW,
		diamondW,
		color.White,
		thirdFill,
	)

	var secondFill color.Color
	secondFill = color.Black
	if runners.Second {
		secondFill = fillColor
	}
	// 2nd base
	rgbrender.DrawDiamond(
		canvas,
		image.Pt(diamondStartShift+(diamondW/2)+2, runnerBounds.Max.Y/4),
		diamondW,
		diamondW,
		color.White,
		secondFill,
	)

	var firstFill color.Color
	firstFill = color.Black
	if runners.First {
		firstFill = fillColor
	}
	// 1st base
	rgbrender.DrawDiamond(
		canvas,
		image.Pt(diamondStartShift+(((diamondW/2)+2)*2), runnerBounds.Max.Y/2),
		diamondW,
		diamondW,
		color.White,
		firstFill,
	)

	inning, err := game.GetInningState(ctx)
	if err != nil {
		return err
	}

	inningPointBounds := image.Rect(midX, zeroed.Max.Y-(zeroed.Dy()/2), midX+(canvasWidth/8), zeroed.Max.Y)
	inningNumBounds := image.Rect(inningPointBounds.Max.X, zeroed.Max.Y-(zeroed.Dy()/2), inningPointBounds.Max.X+(canvasWidth/8), zeroed.Max.Y)

	if inning.IsTop {
		rgbrender.DrawUpTriangle(
			canvas,
			image.Pt(inningPointBounds.Min.X+1, inningPointBounds.Min.Y+(inningPointBounds.Dy()/2)),
			inningPointBounds.Dx()/2,
			inningPointBounds.Dx()/4,
			color.White,
			color.White,
		)
	} else {
		rgbrender.DrawDownTriangle(
			canvas,
			image.Pt(inningPointBounds.Min.X+1, inningPointBounds.Min.Y+(inningPointBounds.Dy()/2)),
			inningPointBounds.Dx()/2,
			inningPointBounds.Dx()/4,
			color.White,
			color.White,
		)
	}

	inning.Number = strings.ToLower(inning.Number)
	writer.XStartCorrection = 0
	writer.YStartCorrection = 3
	_ = writer.Write(
		canvas,
		inningNumBounds,
		[]string{
			strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(inning.Number, "th", ""), "nd", ""), "st", ""), "rd", ""),
		},
		color.White,
	)

	cnt, err := game.GetCount(ctx)
	if err != nil {
		cnt = ""
	}

	countBounds := image.Rect(midX+(canvasWidth/4), zeroed.Max.Y-(zeroed.Dy()/2), midX+canvasWidth, runnerBounds.Max.Y+zeroed.Dy()/4)
	outBounds := image.Rect(countBounds.Min.X, zeroed.Max.Y-(zeroed.Dy()/4), countBounds.Min.X+(canvasWidth/4), zeroed.Max.Y)

	writer.YStartCorrection = 0
	_ = writer.Write(
		canvas,
		countBounds,
		[]string{
			cnt,
		},
		color.White,
	)

	var out1Clr color.Color
	var out2Clr color.Color
	out1Clr = color.Black
	out2Clr = color.Black
	if inning.Outs == 1 {
		out1Clr = fillColor
	}
	if inning.Outs == 2 {
		out1Clr = fillColor
		out2Clr = fillColor
	}

	outBox := 3
	// Out 1
	rgbrender.DrawSquare(
		canvas,
		image.Pt(outBounds.Min.X, outBounds.Min.Y+2),
		outBox,
		color.White,
		out1Clr,
	)
	// Out 2
	rgbrender.DrawSquare(
		canvas,
		image.Pt(outBounds.Min.X+outBox+2, outBounds.Min.Y+2),
		outBox,
		color.White,
		out2Clr,
	)

	return nil
}

func (m *MlbLive) getWriter(canvasBounds image.Rectangle) (*rgbrender.TextWriter, error) {
	if m.writer != nil {
		return m.writer, nil
	}

	w, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return nil, err
	}

	if canvasBounds.Dy() <= 256 {
		w.FontSize = 8.0
	} else {
		w.FontSize = 0.25 * float64(canvasBounds.Dy())
	}

	if canvasBounds.Dy() <= 256 {
		w.YStartCorrection = -2
	} else {
		w.YStartCorrection = -1 * ((canvasBounds.Dy() / 32) + 1)
	}

	m.writer = w

	return w, nil
}

func (m *MlbLive) writeScore(canvas draw.Image, bounds image.Rectangle, writer *rgbrender.TextWriter, abbrev string, score int, teamClr color.Color) {
	clrs := make([]color.Color, len(abbrev))
	for x := 0; x < len(clrs); x++ {
		clrs[x] = teamClr
	}
	scrClrs := make([]color.Color, len(fmt.Sprintf("%d", score)))
	for x := 0; x < len(scrClrs); x++ {
		scrClrs[x] = color.White
	}

	_ = writer.WriteColorCodes(
		canvas,
		bounds,
		&rgbrender.ColorChar{
			Lines: []*rgbrender.ColorCharLine{
				{
					Chars: strings.Split(abbrev, ""),
					Clrs:  clrs,
				},
				{
					Chars: strings.Split(fmt.Sprintf("%d", score), ""),
					Clrs:  scrClrs,
				},
			},
		},
	)
}
