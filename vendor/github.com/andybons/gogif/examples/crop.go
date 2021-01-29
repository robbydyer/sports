package main

// This demonstrates a problem with the encoder where pixels are only written
// to half of the cropped image.  I believe that it is related to images where
// the bounds do not start at (0, 0).
//
// Usage something like this:
// go run examples/crop.go && open blob.out.gif shapes.out.gif

import (
	"image"
	"image/draw"
	"image/gif"
	"log"
	"os"

	"github.com/andybons/gogif"
)

func main() {
	crop("shapes")
	crop("blob")
}

func crop(filename string) {

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
	imgBounds := firstFrame.Bounds()
	img := image.NewRGBA(imgBounds)

	for index, frame := range im.Image {
		bounds := frame.Bounds()
		draw.Draw(img, bounds, frame, bounds.Min, draw.Src)
		im.Image[index] = ImageToPaletted(img.SubImage(image.Rect(
			imgBounds.Min.X,
			imgBounds.Min.Y+imgBounds.Dy()/4,
			imgBounds.Max.X,
			imgBounds.Max.Y-imgBounds.Dy()/4)))
	}

	out, err := os.Create(filename + ".out.gif")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer out.Close()

	gogif.EncodeAll(out, im)
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
