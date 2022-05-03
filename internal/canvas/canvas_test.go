package canvas

import (
	"context"
	"image/color"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/robbydyer/sports/internal/matrix"
)

func TestNewCanvas(t *testing.T) {
	canvas := NewCanvas(NewMatrixMock(64, 32))
	require.NotNil(t, canvas)
	require.Equal(t, 64, canvas.w)
	require.Equal(t, 32, canvas.h)
}

func TestRender(t *testing.T) {
	m := NewMatrixMock(10, 20)
	canvas := &Canvas{m: m}
	err := canvas.Render(context.Background())
	require.NoError(t, err)
}

func TestColorModel(t *testing.T) {
	canvas := &Canvas{}
	require.Equal(t, color.RGBAModel, canvas.ColorModel())
}

func TestBounds(t *testing.T) {
	canvas := &Canvas{w: 10, h: 20}

	b := canvas.Bounds()
	require.Equal(t, 0, b.Min.X)
	require.Equal(t, 0, b.Min.Y)
	require.Equal(t, 10, b.Max.X)
	require.Equal(t, 20, b.Max.Y)
}

func TestSet(t *testing.T) {
	m := NewMatrixMock(10, 20)
	canvas := &Canvas{w: 10, h: 20, m: m}
	canvas.Set(5, 15, color.White)

	require.Equal(t, color.White, m.colors[155])
}

func TestClear(t *testing.T) {
	m := NewMatrixMock(10, 20)

	canvas := &Canvas{w: 10, h: 20, m: m}
	err := canvas.Clear()
	require.NoError(t, err)

	for _, px := range m.colors {
		require.Equal(t, color.Black, px)
	}
}

func TestClose(t *testing.T) {
	m := NewMatrixMock(10, 20)
	canvas := &Canvas{w: 10, h: 20, m: m}
	err := canvas.Close()
	require.NoError(t, err)

	for _, px := range m.colors {
		require.Equal(t, color.Black, px)
	}
}

type MatrixMock struct {
	colors []color.Color
	w      int
	h      int
}

func NewMatrixMock(w int, h int) *MatrixMock {
	return &MatrixMock{
		colors: make([]color.Color, w*h),
		w:      w,
		h:      h,
	}
}

func (m *MatrixMock) Geometry() (width, height int) {
	return 64, 32
}

func (m *MatrixMock) Initialize() error {
	return nil
}

func (m *MatrixMock) At(x int, y int) color.Color {
	pos := position(x, y, m.w)
	if m.colors[pos] == nil {
		return color.Black
	}
	return m.colors[pos]
}

func (m *MatrixMock) Set(x int, y int, c color.Color) {
	pos := position(x, y, m.w)
	m.colors[pos] = c
}

func (m *MatrixMock) Render() error {
	return nil
}

func (m *MatrixMock) Close() error {
	return nil
}

func (m *MatrixMock) SetBrightness(brightness int) {
}

func (m *MatrixMock) PreLoad(points []matrix.MatrixPoint) {}

func (m *MatrixMock) Play(ctx context.Context, defInterval time.Duration, ch <-chan time.Duration) error {
	return nil
}
