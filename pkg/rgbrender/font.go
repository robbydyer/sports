package rgbrender

import (
	"image"
	"image/color"
	"io/ioutil"

	"github.com/golang/freetype"
	"github.com/markbates/pkger"
	rgb "github.com/robbydyer/rgbmatrix-rpi"
)

type TextWriter struct {
	fontCtx *freetype.Context
}

func DefaultTextWriter() (*freetype.Context, error) {
	f, err := pkger.Open("/assets/fonts/04b24.otf")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	fnt, err := freetype.ParseFont(fBytes)
	if err != nil {
		return nil, err
	}

	cntx := freetype.NewContext()
	cntx.SetFont(fnt)
	cntx.SetFontSize(8)

	return cntx, nil
}

func (t *TextWriter) Write(canvas *rgb.Canvas, bounds image.Rectangle, str string, clr color.Color) error {
	return nil
}
