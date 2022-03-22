package rgbrender

import (
	"embed"
	"fmt"
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

// ColorChar is used to define text for writing in different colors
type ColorChar struct {
	Lines  []*ColorCharLine
	BoxClr color.Color
}

// ColorCharLine is a line in a multicolored text
type ColorCharLine struct {
	Chars []string
	Clrs  []color.Color
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

func (t *TextWriter) getDrawer(canvas draw.Image, clr color.Color) (*font.Drawer, error) {
	if t.font == nil {
		return nil, fmt.Errorf("font is not set")
	}
	return &font.Drawer{
		Dst: canvas,
		Src: image.NewUniform(clr),
		Face: truetype.NewFace(t.font,
			&truetype.Options{
				Size:    t.FontSize,
				Hinting: font.HintingFull,
			},
		),
	}, nil
}

// Write ...
func (t *TextWriter) Write(canvas draw.Image, bounds image.Rectangle, str []string, clr color.Color) error {
	drawer, err := t.getDrawer(canvas, clr)
	if err != nil {
		return err
	}
	startX := bounds.Min.X + t.XStartCorrection

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

// WriteAligned writes text aligned within a given bounds
func (t *TextWriter) WriteAligned(align Align, canvas draw.Image, bounds image.Rectangle, str []string, clr color.Color) error {
	drawer, err := t.getDrawer(canvas, clr)
	if err != nil {
		return err
	}

	var maxXWidth fixed.Int26_6
	for _, s := range str {
		if width := drawer.MeasureString(s); width > maxXWidth {
			maxXWidth = width
		}
	}

	yHeight := len(str) * int(math.Floor(t.FontSize+t.LineSpace))

	writeBox, err := AlignPosition(align, bounds, maxXWidth.Floor(), yHeight)
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

// MeasureStrings measures the pixel width of a list of strings
func (t *TextWriter) MeasureStrings(canvas draw.Image, str []string) ([]int, error) {
	lengths := make([]int, len(str))
	drawer, err := t.getDrawer(canvas, color.White)
	if err != nil {
		return nil, err
	}

	for i, s := range str {
		lengths[i] = drawer.MeasureString(s).Ceil()
	}

	return lengths, nil
}

// MaxChars returns the maximum number of characters that can fit a given pixel width
func (t *TextWriter) MaxChars(canvas draw.Image, pixWidth int) (int, error) {
	s := "E"
	num := 0
	for {
		l, err := t.MeasureStrings(canvas, []string{s})
		if err != nil {
			return 0, err
		}
		if len(l) < 1 {
			return 0, fmt.Errorf("unexpected MeaureStrings return")
		}
		if l[0] > pixWidth {
			return num, nil
		}
		num++
		s += "E"
	}
}

// WriteAlignedBoxed writes text aligned within a given bounds and draws a box sized to the text width
func (t *TextWriter) WriteAlignedBoxed(align Align, canvas draw.Image, bounds image.Rectangle, str []string, clr color.Color, boxColor color.Color) error {
	drawer, err := t.getDrawer(canvas, clr)
	if err != nil {
		return err
	}

	var maxXWidth fixed.Int26_6
	for _, s := range str {
		if width := drawer.MeasureString(s); width > maxXWidth {
			maxXWidth = width
		}
	}

	yHeight := len(str) * int(math.Floor(t.FontSize+t.LineSpace))

	boxAlign, err := AlignPosition(align, bounds, maxXWidth.Ceil(), yHeight)
	if err != nil {
		return err
	}
	draw.Draw(canvas, boxAlign, image.NewUniform(boxColor), image.Point{}, draw.Over)

	writeBox, err := AlignPosition(align, bounds, maxXWidth.Floor(), yHeight)
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

// WriteAlignedColorCodes writes text aligned within a given bounds and draws a box sized to the text width
func (t *TextWriter) WriteAlignedColorCodes(align Align, canvas draw.Image, bounds image.Rectangle, colorChars *ColorChar) error {
	if err := colorChars.validate(); err != nil {
		return err
	}

	drawer, err := t.getDrawer(canvas, color.White)
	if err != nil {
		return err
	}
	maxXWidth := t.maxWidth(colorChars, drawer)

	yHeight := len(colorChars.Lines) * int(math.Floor(t.FontSize+t.LineSpace))

	if colorChars.BoxClr != nil {
		boxAlign, err := AlignPosition(align, bounds, maxXWidth.Ceil(), yHeight)
		if err != nil {
			return err
		}
		draw.Draw(canvas, boxAlign, image.NewUniform(colorChars.BoxClr), image.Point{}, draw.Over)
	}

	writeBox, err := AlignPosition(align, bounds, maxXWidth.Floor(), yHeight)
	if err != nil {
		return err
	}

	// lineY represents how much space to add for the newline
	lineY := int(math.Floor(t.FontSize + t.LineSpace))

	startX := writeBox.Min.X + t.XStartCorrection
	y := int(math.Floor(t.FontSize)) + writeBox.Min.Y + t.YStartCorrection

	pt := fixed.P(startX, y)
	prev := []string{}
	for _, line := range colorChars.Lines {
		for i, char := range line.Chars {
			clr := line.Clrs[i]
			drawer, err := t.getDrawer(canvas, clr)
			if err != nil {
				return err
			}
			str := strings.Join(prev, "")
			prevWidth := drawer.MeasureString(str)
			drawer.Dot = pt
			drawer.Dot.X = prevWidth + drawer.Dot.X
			drawer.DrawString(char)
			prev = append(prev, char)
		}
		y += lineY + t.YStartCorrection
		pt = fixed.P(startX, y)
		prev = []string{}
	}

	return nil
}

// maxWidth finds the max width of a slice of ColoChar
func (t *TextWriter) maxWidth(clrChars *ColorChar, drawer *font.Drawer) fixed.Int26_6 {
	var m fixed.Int26_6
	for _, line := range clrChars.Lines {
		str := strings.Join(line.Chars, "")
		if width := drawer.MeasureString(str); width > m {
			m = width
		}
	}

	return m
}

func (c *ColorChar) validate() error {
	for _, line := range c.Lines {
		if len(line.Chars) != len(line.Clrs) {
			return fmt.Errorf("number of chars and colors must match")
		}
	}

	return nil
}

func (t *TextWriter) BreakText(canvas draw.Image, maxPixWidth int, text string) ([]string, error) {
	lines := [][]string{}
	lines = append(lines, []string{})
	words := strings.Fields(text)

	max, err := t.MaxChars(canvas, maxPixWidth)
	if err != nil {
		return []string{}, err
	}

	lineIndex := 0
	num := 0
	for _, s := range words {
		if num+len(s) > max {
			lines = append(lines, []string{})
			lines[lineIndex+1] = append(lines[lineIndex+1], s)
			lineIndex++
			num = 0
		} else {
			lines[lineIndex] = append(lines[lineIndex], s)
			num += len(s)
		}
	}

	retLines := []string{}

	for _, line := range lines {
		retLines = append(retLines, strings.Join(line, " "))
	}

	return retLines, nil
}
