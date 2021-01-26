// +build darwin

// movingtriangle is an example Metal program that displays a moving triangle in a window.
// It opens a window and renders a triangle that follows the mouse cursor.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"
	"unsafe"

	"dmitri.shuralyov.com/gpu/mtl"
	"dmitri.shuralyov.com/gpu/mtl/example/movingtriangle/internal/appkit"
	"dmitri.shuralyov.com/gpu/mtl/example/movingtriangle/internal/coreanim"
	"github.com/go-gl/glfw/v3.3/glfw"
	"golang.org/x/image/math/f32"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: movingtriangle")
		flag.PrintDefaults()
	}
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
	fmt.Println("Metal device:", device.Name)

	err = glfw.Init()
	if err != nil {
		return err
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(640, 480, "Metal Example", nil, nil)
	if err != nil {
		return err
	}
	defer window.Destroy()

	ml := coreanim.MakeMetalLayer()
	ml.SetDevice(device)
	ml.SetPixelFormat(mtl.PixelFormatBGRA8UNorm)
	ml.SetDrawableSize(window.GetFramebufferSize())
	ml.SetMaximumDrawableCount(3)
	ml.SetDisplaySyncEnabled(true)
	cv := appkit.NewWindow(window.GetCocoaWindow()).ContentView()
	cv.SetLayer(ml)
	cv.SetWantsLayer(true)

	// Set callbacks.
	window.SetFramebufferSizeCallback(func(_ *glfw.Window, width, height int) {
		ml.SetDrawableSize(width, height)
	})
	var windowSize = [2]int32{640, 480}
	window.SetSizeCallback(func(_ *glfw.Window, width, height int) {
		windowSize[0], windowSize[1] = int32(width), int32(height)
	})
	var pos [2]float32
	window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		pos[0], pos[1] = float32(x), float32(y)
	})

	// Create a render pipeline state.
	const source = `#include <metal_stdlib>

using namespace metal;

struct Vertex {
	float4 position [[position]];
	float4 color;
};

vertex Vertex VertexShader(
	uint vertexID [[vertex_id]],
	device Vertex * vertices [[buffer(0)]],
	constant int2 * windowSize [[buffer(1)]],
	constant float2 * pos [[buffer(2)]]
) {
	Vertex out = vertices[vertexID];
	out.position.xy += *pos;
	float2 viewportSize = float2(*windowSize);
	out.position.xy = float2(-1 + out.position.x / (0.5 * viewportSize.x),
	                          1 - out.position.y / (0.5 * viewportSize.y));
	return out;
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
	rpld.ColorAttachments[0].PixelFormat = ml.PixelFormat()
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
		{f32.Vec4{0, 0, 0, 1}, f32.Vec4{1, 0, 0, 1}},
		{f32.Vec4{300, 100, 0, 1}, f32.Vec4{0, 1, 0, 1}},
		{f32.Vec4{0, 100, 0, 1}, f32.Vec4{0, 0, 1, 1}},
	}
	vertexBuffer := device.MakeBuffer(unsafe.Pointer(&vertexData[0]), unsafe.Sizeof(vertexData), mtl.ResourceStorageModeManaged)

	cq := device.MakeCommandQueue()

	frame := startFPSCounter()

	for !window.ShouldClose() {
		glfw.PollEvents()

		// Create a drawable to render into.
		drawable, err := ml.NextDrawable()
		if err != nil {
			return err
		}

		cb := cq.MakeCommandBuffer()

		// Encode all render commands.
		var rpd mtl.RenderPassDescriptor
		rpd.ColorAttachments[0].LoadAction = mtl.LoadActionClear
		rpd.ColorAttachments[0].StoreAction = mtl.StoreActionStore
		rpd.ColorAttachments[0].ClearColor = mtl.ClearColor{Red: 0.35, Green: 0.65, Blue: 0.85, Alpha: 1}
		rpd.ColorAttachments[0].Texture = drawable.Texture()
		rce := cb.MakeRenderCommandEncoder(rpd)
		rce.SetRenderPipelineState(rps)
		rce.SetVertexBuffer(vertexBuffer, 0, 0)
		rce.SetVertexBytes(unsafe.Pointer(&windowSize[0]), unsafe.Sizeof(windowSize), 1)
		rce.SetVertexBytes(unsafe.Pointer(&pos[0]), unsafe.Sizeof(pos), 2)
		rce.DrawPrimitives(mtl.PrimitiveTypeTriangle, 0, 3)
		rce.EndEncoding()

		cb.PresentDrawable(drawable)
		cb.Commit()

		frame <- struct{}{}
	}

	return nil
}

func startFPSCounter() chan struct{} {
	frame := make(chan struct{}, 4)
	go func() {
		second := time.Tick(time.Second)
		frames := 0
		for {
			select {
			case <-second:
				fmt.Println("fps:", frames)
				frames = 0
			case <-frame:
				frames++
			}
		}
	}()
	return frame
}
