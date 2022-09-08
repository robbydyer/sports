package scrollcanvas

import (
	"image"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/robbydyer/sports/internal/matrix"
)

func TestScrollCanvas(t *testing.T) {
	l := zaptest.NewLogger(t)
	m := matrix.NewConsoleMatrix(64, 32, io.Discard, l)
	c, err := NewScrollCanvas(m, l)
	require.NoError(t, err)

	defaultPad := 64 + int(float64(64)*0.25)

	require.Equal(t, image.Rect(defaultPad*-1, defaultPad*-1, 64+defaultPad, 32+defaultPad), c.Bounds())
	require.Equal(t, 64, c.w)
	require.Equal(t, 32, c.h)
}
