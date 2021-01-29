package main

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"time"

	"github.com/markbates/pkger"
	rgb "github.com/robbydyer/sports/pkg/rgbmatrix-rpi"
	"github.com/robbydyer/sports/pkg/rgbrender"
)

type testBoard struct {
	canvas *rgb.Canvas
}

func (t *testBoard) Name() string {
	return "Test Board"
}

func (t *testBoard) Render(ctx context.Context, matrix rgb.Matrix) error {
	delay := 10 * time.Second
	fmt.Println("Rendering testBoard!")

	f, err := pkger.Open("/assets/images/goal_light.png")
	if err != nil {
		return fmt.Errorf("failed to open packed image: %w", err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	if t.canvas == nil {
		t.canvas = rgb.NewCanvas(matrix)
	}

	if err := rgbrender.DrawImage(t.canvas, t.canvas.Bounds(), img); err != nil {
		return fmt.Errorf("failed to draw test image: %w", err)
	}

	if err := t.canvas.Render(); err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(delay / 2):
	}

	textWriter, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return fmt.Errorf("failed to get TextWriter: %w", err)
	}

	center, err := rgbrender.AlignPosition(rgbrender.CenterCenter, t.canvas.Bounds(), 64, 32)
	if err != nil {
		return err
	}

	fmt.Printf("Writing text in rect: %d, %d to %d, %d\n",
		center.Min.X,
		center.Min.Y,
		center.Max.X,
		center.Max.Y,
	)

	if err := textWriter.Write(t.canvas, center, []string{"Hello World", "How are you?"}, image.White); err != nil {
		return fmt.Errorf("failed to write text: %w", err)
	}

	if err := t.canvas.Render(); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
	case <-time.After(delay / 2):
	}

	return nil
}

func (y *testBoard) Enabled() bool {
	return true
}

func (t *testBoard) Cleanup() {
	if t.canvas != nil {
		_ = t.canvas.Clear()
	}
}

func (t *testBoard) HasPriority() bool {
	return false
}
