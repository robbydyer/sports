package main

import (
	"context"
	"fmt"
	"image"
	"time"

	"github.com/markbates/pkger"
	rgb "github.com/robbydyer/rgbmatrix-rpi"
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

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	if t.canvas == nil {
		t.canvas = rgb.NewCanvas(matrix)
	}

	if err := rgbrender.DrawImage(t.canvas, t.canvas.Bounds(), img); err != nil {
		return fmt.Errorf("failed to draw test image: %w", err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(delay / 2):
	}

	textWriter, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return fmt.Errorf("failed to get TextWriter: %w", err)
	}

	if err := textWriter.Write(t.canvas, t.canvas.Bounds(), []string{"Hello Test"}, image.Black); err != nil {
		return fmt.Errorf("failed to write text: %w", err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(delay / 2):
	}

	return nil
}

func (t *testBoard) Cleanup() {
	if t.canvas != nil {
		_ = t.canvas.Clear()
	}
}

func (t *testBoard) HasPriority() bool {
	return false
}
