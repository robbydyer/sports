package imgcanvas

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"sync"
)

type ImgCanvas struct {
	width      int
	height     int
	pixels     []uint32
	lastPng    *bytes.Buffer
	lastRender image.Image
	sync.Mutex
}

func New(width int, height int) *ImgCanvas {
	i := &ImgCanvas{
		width:  width,
		height: height,
		pixels: make([]uint32, (width * height)),
	}
	i.Clear()

	return i
}

func (i *ImgCanvas) Clear() error {
	for x := range i.pixels {
		i.pixels[x] = colorToUint32(color.Black)
	}
	return nil
}

func (i *ImgCanvas) LastRender() image.Image {
	i.Lock()
	defer i.Unlock()
	return i.lastRender
}

func (i *ImgCanvas) LastPng() io.Reader {
	i.Lock()
	defer i.Unlock()
	return i.lastPng
}

func (i *ImgCanvas) Render() error {
	buf := &bytes.Buffer{}

	if err := png.Encode(buf, i); err != nil {
		return err
	}

	i.Lock()
	defer i.Unlock()
	i.lastPng = buf

	i.Clear()

	return nil
}

// ColorModel returns the canvas' color model, always color.RGBAModel
func (i *ImgCanvas) ColorModel() color.Model {
	return color.RGBAModel
}

// Bounds return the topology of the Canvas
func (i *ImgCanvas) Bounds() image.Rectangle {
	return image.Rect(0, 0, i.width, i.height)
}

// At returns the color of the pixel at (x, y)
func (i *ImgCanvas) At(x, y int) color.Color {
	pos := i.position(x, y)
	if pos > len(i.pixels)-1 || pos < 0 {
		return color.Black
	}
	return uint32ToColor(i.pixels[pos])
}

// Set set LED at position x,y to the provided 24-bit color value
func (i *ImgCanvas) Set(x, y int, clr color.Color) {
	pos := i.position(x, y)
	if pos > len(i.pixels)-1 || pos < 0 {
		return
	}
	i.pixels[pos] = colorToUint32(clr)
}

func (i *ImgCanvas) position(x, y int) int {
	return x + (y * i.width)
}

func uint32ToColor(u uint32) color.Color {
	return color.RGBA{
		uint8(u>>16) & 255,
		uint8(u>>8) & 255,
		uint8(u>>0) & 255,
		255,
	}
}

func colorToUint32(c color.Color) uint32 {
	if c == nil {
		return 0
	}

	// A color's RGBA method returns values in the range [0, 65535]
	red, green, blue, _ := c.RGBA()
	return (red>>8)<<16 | (green>>8)<<8 | blue>>8
}
