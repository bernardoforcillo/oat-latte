package oat

import "github.com/gdamore/tcell/v2"

// RenderComponentToScreen is a testing bridge that renders c onto screen using
// a freshly created Buffer covering the full screen dimensions.
//
// This function exists solely to allow the oattest package (and other external
// test helpers) to drive a real Render pass without needing access to the
// unexported Buffer type.  It must not be used in production code.
//
// Typical usage (from oattest.RenderToString):
//
//	screen := tcell.NewSimulationScreen("")
//	_ = screen.Init()
//	screen.SetSize(width, height)
//	oat.RenderComponentToScreen(c, screen)
//	screen.Show()
//	// now inspect screen.GetContent(x, y) for each cell
func RenderComponentToScreen(c Component, screen tcell.Screen) {
	w, h := screen.Size()
	if w == 0 || h == 0 {
		return
	}
	buf := newBuffer(screen)
	region := Region{X: 0, Y: 0, Width: w, Height: h}
	c.Render(buf, region)
}
