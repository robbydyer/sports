package rgbmatrix

import (
	"image"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestScrollCanvas(t *testing.T) {
	l := zaptest.NewLogger(t)
	m := NewConsoleMatrix(64, 32, ioutil.Discard, l)
	c, err := NewScrollCanvas(m, l, WithRightToLeft())
	require.NoError(t, err)

	defaultPad := 64 + int(float64(64)*0.25)

	require.Equal(t, image.Rect(defaultPad*-1, defaultPad*-1, 64+defaultPad, 32+defaultPad), c.Bounds())
	require.Equal(t, 64, c.w)
	require.Equal(t, 32, c.h)
}
