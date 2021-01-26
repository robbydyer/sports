package rgbrender

import (
	"fmt"
	"image"
	"image/color"

	"github.com/gobuffalo/packr/v2"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"

	//"github.com/markbates/pkger"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"golang.org/x/image/font"
)

type TextWriter struct {
	context  *freetype.Context
	font     *truetype.Font
	FontSize float64
}

func DefaultTextWriter() (*TextWriter, error) {
	fnt, err := DefaultFont()
	if err != nil {
		return nil, err
	}

	return NewTextWriter(fnt, 8), nil
}

func NewTextWriter(font *truetype.Font, fontSize float64) *TextWriter {
	cntx := freetype.NewContext()
	cntx.SetFont(font)
	cntx.SetFontSize(fontSize)

	return &TextWriter{
		context:  cntx,
		font:     font,
		FontSize: fontSize,
	}
}

/*
func DefaultFont() (*truetype.Font, error) {
	box := packr.NewBox("../assets")
	dat, err := box.Find("fonts/04b24.ttf")
	if err != nil {
		return nil, err
	}

	fnt, err := freetype.ParseFont(dat)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font: %w", err)
	}

	return fnt, nil
}
*/

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
		point.Y += t.context.PointToFixed(t.FontSize * 0.75)
	}

	return nil
}
