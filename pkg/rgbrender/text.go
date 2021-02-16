package rgbrender

import (
	"embed"
	"image"
	"image/color"
	"image/draw"
	"math"
	"path/filepath"
	"strings"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

//go:embed assets/fonts
var fontDir embed.FS

const (
	defaultFont = "04b24.ttf"
)

// BuiltinFonts is a list of fonts names this pkg provides
var BuiltinFonts = []string{
	"04b24.ttf",
	"BlockStockRegular-A71p.ttf",
	"04B_03__.ttf",
	"score.ttf",
}

// TextWriter ...
type TextWriter struct {
	context          *freetype.Context
	font             *truetype.Font
	XStartCorrection int
	YStartCorrection int
	FontSize         float64
	LineSpace        float64
}

// DefaultTextWriter ...
func DefaultTextWriter() (*TextWriter, error) {
	fnt, err := GetFont(defaultFont)
	if err != nil {
		return nil, err
	}

	t := NewTextWriter(fnt, 8)
	t.YStartCorrection = -2

	return t, nil
}

// NewTextWriter ...
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

// GetFont gets a builtin font by the given name
func GetFont(name string) (*truetype.Font, error) {
	if !strings.HasPrefix("assets/fonts", name) {
		name = filepath.Join("assets/fonts", name)
	}
	f, err := fontDir.ReadFile(name)
	if err != nil {
		return nil, err
	}

	return freetype.ParseFont(f)
}

// Write ...
func (t *TextWriter) Write(canvas draw.Image, bounds image.Rectangle, str []string, clr color.Color) error {
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

// WriteCentered writes text in the center of the canvas, horizontally and vertically
func (t *TextWriter) WriteCentered(canvas draw.Image, bounds image.Rectangle, str []string, clr color.Color) error {
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

	var maxXWidth fixed.Int26_6
	for _, s := range str {
		if width := drawer.MeasureString(s); width > fixed.Int26_6(maxXWidth) {
			maxXWidth = width
		}
	}

	yHeight := len(str) * int(math.Floor(t.FontSize+t.LineSpace))

	writeBox, err := AlignPosition(CenterCenter, bounds, maxXWidth.Floor(), yHeight)
	if err != nil {
		return err
	}

	// lineY represents how much space to add for the newline
	lineY := int(math.Floor(t.FontSize + t.LineSpace))

	startX := writeBox.Min.X + t.XStartCorrection
	y := int(math.Floor(t.FontSize)) + writeBox.Min.Y + t.YStartCorrection
	drawer.Dot = fixed.P(startX, y)

	for _, s := range str {
		drawer.DrawString(s)
		y += lineY + t.YStartCorrection
		drawer.Dot = fixed.P(startX, y)
	}

	return nil
}
