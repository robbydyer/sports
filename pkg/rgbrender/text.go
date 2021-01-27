package rgbrender

import (
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"math"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"

	"github.com/markbates/pkger"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type TextWriter struct {
	context          *freetype.Context
	font             *truetype.Font
	XStartCorrection int
	YStartCorrection int
	FontSize         float64
	LineSpace        float64
}

func DefaultTextWriter() (*TextWriter, error) {
	fnt, err := DefaultFont()
	if err != nil {
		return nil, err
	}

	t := NewTextWriter(fnt, 8)
	t.YStartCorrection = -2

	return t, nil
}

func NewTextWriter(font *truetype.Font, fontSize float64) *TextWriter {
	cntx := freetype.NewContext()
	cntx.SetFont(font)
	cntx.SetFontSize(fontSize)

	return &TextWriter{
		context:   cntx,
		font:      font,
		FontSize:  fontSize,
		LineSpace: 0.5,
	}
}

func DefaultFont() (*truetype.Font, error) {
	return FontFromAsset("github.com/robbydyer/sports:/assets/fonts/04b24.ttf")
}

func FontFromAsset(asset string) (*truetype.Font, error) {
	f, err := pkger.Open(asset)
	if err != nil {
		return nil, fmt.Errorf("failed to open asset %s with pkger: %w", asset, err)
	}
	defer f.Close()

	fBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read font bytes: %w", err)
	}

	fnt, err := freetype.ParseFont(fBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font: %w", err)
	}

	return fnt, nil
}

func (t *TextWriter) SetFont(fnt *truetype.Font) {
	t.font = fnt
	if t.context == nil {
		t.context = freetype.NewContext()
	}
	t.context.SetFont(fnt)
}

func (t *TextWriter) Write(canvas *rgb.Canvas, bounds image.Rectangle, str []string, clr color.Color) error {
	if t.context == nil {
		return fmt.Errorf("invalid TextWriter, must initialize with NewTextWriter()")
	}
	t.context.SetFontSize(t.FontSize)

	textColor := image.NewUniform(clr)
	t.context.SetClip(bounds)
	t.context.SetDst(canvas)
	t.context.SetSrc(textColor)
	t.context.SetHinting(font.HintingFull)

	point := freetype.Pt(bounds.Min.X, int(t.context.PointToFixed(t.FontSize)>>6))
	for _, c := range str {
		_, err := t.context.DrawString(c, point)
		if err != nil {
			return err
		}
		point.Y += t.context.PointToFixed(t.FontSize * t.LineSpace)
	}

	return nil
}

func (t *TextWriter) Write2(canvas *rgb.Canvas, bounds image.Rectangle, str []string, clr color.Color) error {
	startX := bounds.Min.X + t.XStartCorrection
	drawer := &font.Drawer{
		Dst: canvas,
		Src: image.NewUniform(clr),
		Face: truetype.NewFace(t.font,
			&truetype.Options{
				Size:    t.FontSize,
				Hinting: font.HintingFull,
			},
		),
	}

	// lineY represents how much space to add for the newline
	lineY := int(math.Floor(t.FontSize + t.LineSpace))

	y := int(math.Floor(t.FontSize)) + bounds.Min.Y + t.YStartCorrection
	drawer.Dot = fixed.P(startX, y)

	for _, s := range str {
		drawer.DrawString(s)
		y += lineY + t.YStartCorrection
		drawer.Dot = fixed.P(startX, y)
	}

	return nil
}
