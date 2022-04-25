package canvas

func position(x int, y int, canvasWidth int) int {
	return x + (y * canvasWidth)
}
