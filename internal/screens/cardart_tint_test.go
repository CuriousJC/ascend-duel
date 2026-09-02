package screens

import (
	"image"
	"image/color"
	"testing"
)

// A tinted pip keeps its own brightness: white stays the ink, mid-grey comes out about half of it,
// and transparent stays transparent. **The failure this catches is a whole row of black pips**,
// which is what one shift too many produced.
func TestTintedPipKeepsItsBrightness(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	src.SetRGBA(1, 0, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	src.SetRGBA(2, 0, color.RGBA{})

	ink := color.RGBA{R: 235, G: 120, B: 45, A: 255}
	out := tintedPip(src, ink).(*image.RGBA)

	if got := out.RGBAAt(0, 0); got.R != ink.R || got.G != ink.G || got.B != ink.B {
		t.Errorf("a white pixel came out %v, want the ink %v", got, ink)
	}
	if got := out.RGBAAt(1, 0); got.R < 100 || got.R > 130 {
		t.Errorf("a mid-grey pixel came out %v, want about half the ink", got)
	}
	if got := out.RGBAAt(2, 0); got.A != 0 {
		t.Errorf("a transparent pixel came out %v", got)
	}
}
