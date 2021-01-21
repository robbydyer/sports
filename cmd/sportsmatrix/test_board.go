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

func (t *testBoard) Render(ctx context.Context, matrix rgb.Matrix, rotationDelay time.Duration) error {
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

	if err := rgbrender.ShowImage(t.canvas, img); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
	case <-time.After(rotationDelay):
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
