package main

import (
	"image"
	"image/draw"
	"image/gif"
	"log"
	"os"

	"github.com/andybons/gogif"
	"github.com/nfnt/resize"
)

func main() {
	process("shapes")
	process("blob")
}

func process(filename string) {

	f, err := os.Open("testdata/" + filename + ".gif")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer f.Close()

	im, err := gif.DecodeAll(f)
	if err != nil {
		log.Fatal(err.Error())
	}

	firstFrame := im.Image[0]
	img := image.NewRGBA(firstFrame.Bounds())

	// Frames in an animated gif aren't necessarily the same size. Subsequent
	// frames are overlayed on previous frames. Resizing the frames individually
	// causes problems due to aliasing of transparent pixels. In theory we can get
	// around this by building up the resultant frames from all previous frames.
	// This results in slower processing times and a bigger end image, but so far
	// I haven't thought of an alternative method.

	for index, frame := range im.Image {
		bounds := frame.Bounds()
		draw.Draw(img, bounds, frame, bounds.Min, draw.Src)
		im.Image[index] = ImageToPaletted(ProcessImage(img))
	}

	out, err := os.Create(filename + ".out.gif")
	if err != nil {
		log.Fatal(err.Error())
	}

	defer out.Close()

	gogif.EncodeAll(out, im)
}

func ProcessImage(img image.Image) image.Image {
	return resize.Resize(150, 0, img, resize.Bilinear)
}

// Converts an image to an image.Paletted with 256 colors.
func ImageToPaletted(img image.Image) *image.Paletted {
	pm, ok := img.(*image.Paletted)
	if !ok {
		b := img.Bounds()
		pm = image.NewPaletted(b, nil)
		q := &gogif.MedianCutQuantizer{NumColor: 256}
		q.Quantize(pm, b, img, image.ZP)
	}
	return pm
}
