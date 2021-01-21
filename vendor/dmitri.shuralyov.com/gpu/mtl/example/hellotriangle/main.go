// +build darwin

// hellotriangle is an example Metal program that renders a single frame with a triangle.
// It writes the frame to a triangle.png file in current working directory.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"unsafe"

	"dmitri.shuralyov.com/gpu/mtl"
	"golang.org/x/image/math/f32"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: hellotriangle")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	device, err := mtl.CreateSystemDefaultDevice()
	if err != nil {
		return err
	}

	// Create a render pipeline state.
	const source = `#include <metal_stdlib>

using namespace metal;

struct Vertex {
	float4 position [[position]];
	float4 color;
};

vertex Vertex VertexShader(
	uint vertexID [[vertex_id]],
	device Vertex * vertices [[buffer(0)]]
) {
	return vertices[vertexID];
}

fragment float4 FragmentShader(Vertex in [[stage_in]]) {
	return in.color;
}
`
	lib, err := device.MakeLibrary(source, mtl.CompileOptions{})
	if err != nil {
		return err
	}
	vs, err := lib.MakeFunction("VertexShader")
	if err != nil {
		return err
	}
	fs, err := lib.MakeFunction("FragmentShader")
	if err != nil {
		return err
	}
	var rpld mtl.RenderPipelineDescriptor
	rpld.VertexFunction = vs
	rpld.FragmentFunction = fs
	rpld.ColorAttachments[0].PixelFormat = mtl.PixelFormatRGBA8UNorm
	rps, err := device.MakeRenderPipelineState(rpld)
	if err != nil {
		return err
	}

	// Create a vertex buffer.
	type Vertex struct {
		Position f32.Vec4
		Color    f32.Vec4
	}
	vertexData := [...]Vertex{
		{f32.Vec4{+0.00, +0.75, 0, 1}, f32.Vec4{1, 0, 0, 1}},
		{f32.Vec4{-0.75, -0.75, 0, 1}, f32.Vec4{0, 1, 0, 1}},
		{f32.Vec4{+0.75, -0.75, 0, 1}, f32.Vec4{0, 0, 1, 1}},
	}
	vertexBuffer := device.MakeBuffer(unsafe.Pointer(&vertexData[0]), unsafe.Sizeof(vertexData), mtl.ResourceStorageModeManaged)

	// Create an output texture to render into.
	td := mtl.TextureDescriptor{
		PixelFormat: mtl.PixelFormatRGBA8UNorm,
		Width:       512,
		Height:      512,
		StorageMode: mtl.StorageModeManaged,
	}
	texture := device.MakeTexture(td)

	cq := device.MakeCommandQueue()
	cb := cq.MakeCommandBuffer()

	// Encode all render commands.
	var rpd mtl.RenderPassDescriptor
	rpd.ColorAttachments[0].LoadAction = mtl.LoadActionClear
	rpd.ColorAttachments[0].StoreAction = mtl.StoreActionStore
	rpd.ColorAttachments[0].ClearColor = mtl.ClearColor{Red: 0.35, Green: 0.65, Blue: 0.85, Alpha: 1}
	rpd.ColorAttachments[0].Texture = texture
	rce := cb.MakeRenderCommandEncoder(rpd)
	rce.SetRenderPipelineState(rps)
	rce.SetVertexBuffer(vertexBuffer, 0, 0)
	rce.DrawPrimitives(mtl.PrimitiveTypeTriangle, 0, 3)
	rce.EndEncoding()

	// Encode all blit commands.
	bce := cb.MakeBlitCommandEncoder()
	bce.Synchronize(texture)
	bce.EndEncoding()

	cb.Commit()
	cb.WaitUntilCompleted()

	// Read pixels from output texture into an image.
	img := image.NewNRGBA(image.Rect(0, 0, texture.Width, texture.Height))
	bytesPerRow := 4 * texture.Width
	region := mtl.RegionMake2D(0, 0, texture.Width, texture.Height)
	texture.GetBytes(&img.Pix[0], uintptr(bytesPerRow), region, 0)

	// Write output image to a PNG file.
	err = writePNG("triangle.png", img)
	return err
}

// writePNG encodes the image m to a named file, in PNG format.
func writePNG(name string, m image.Image) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	err = png.Encode(f, m)
	return err
}
