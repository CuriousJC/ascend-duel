package cards

import (
	"fmt"
	"image"
	"image/color"
)

// The health bar: one widget, drawn the same way on whichever fighter is holding it.
//
// **The fraction is written across the bar rather than under it.** A bar and its own number are
// one reading — roughly how hurt, then exactly how hurt — and stacking them spent a whole row
// saying the second half of the first row's sentence. Overlaid, the pair is one object and the row
// it used to cost went to the block above it.
//
// **Its geometry is `HealthBarRect` and nothing computes it a second time.** Anything that wants to
// put something on the bar — a frame, a fill, a figure flying into it — asks for the rectangle
// rather than adding the inset to the width itself.
//
// **It draws the bar and nothing else.** No ground, no band, no panel behind it: it goes straight
// onto the card, which is a portrait on one fighter and the plain surface on the other.

// The health bar's two colors.
//
// **Red for what is left, not for what is lost.** The bar is a quantity the reader is tracking
// downward, so the saturated color has to be the part that shrinks — a bar where the red grows as
// the enemy weakens says the opposite of what it means. The empty part is a dim version of the
// same hue rather than a neutral gray, so the two read as one bar partly filled instead of as two
// bars.
var (
	HealthFull  = color.RGBA{R: 198, G: 46, B: 46, A: 255}
	HealthEmpty = color.RGBA{R: 92, G: 66, B: 66, A: 255}

	// HealthTextInk is the fraction written across the bar. It has to read against both halves —
	// the saturated red of what is left and the dim red of what is gone — so it is near-white
	// rather than either hue's own light end.
	HealthTextInk = color.RGBA{R: 248, G: 248, B: 250, A: 255}
)

// HealthBarRect is where the bar is drawn on a card in this style. It is empty for a style that
// has no health.
func HealthBarRect(st Style) image.Rectangle {
	if st.HealthBarHeight <= 0 {
		return image.Rectangle{}
	}
	return image.Rect(st.HealthBarInset, st.HealthBarTop,
		st.Width-st.HealthBarInset, st.HealthBarTop+st.HealthBarHeight)
}

// drawHealth draws the bar and writes the fraction across it.
//
// **The bar is square-cornered**, which matches the cost ticks and the card's own hard-edged
// corners.
//
// **The figure is near-white on both cards, whatever ink set the card around it is using.** It
// sits on the bar rather than on the card, and the bar is the same two reds wherever it is drawn —
// so what the fraction has to read against is the bar, not the face behind it.
//
// A zero or negative MaxLife draws the empty bar and no fraction rather than dividing by it.
func drawHealth(dst *image.RGBA, s Spec, st Style, f *Faces) error {
	bar := HealthBarRect(st)
	if bar.Empty() {
		return nil
	}

	fillRect(dst, bar.Min.X, bar.Min.Y, bar.Dx(), bar.Dy(), HealthEmpty)
	if s.MaxLife <= 0 {
		return nil
	}

	life := s.Life
	if life < 0 {
		life = 0
	}
	if life > s.MaxLife {
		life = s.MaxLife
	}
	fillRect(dst, bar.Min.X, bar.Min.Y, bar.Dx()*life/s.MaxLife, bar.Dy(), HealthFull)

	// **The exact number as well as the bar, deliberately.** A bar says roughly how hurt
	// something is, and a duel decided in whole points wants the figure.
	top, err := centeredTextTop(f, st.HealthTextSize, bar)
	if err != nil {
		return err
	}
	return drawTextHCentered(dst, f, st.HealthTextSize,
		fmt.Sprintf("%d/%d", life, s.MaxLife), st.Width, top, HealthTextInk)
}

// centeredTextTop is the y drawText wants for a line sitting in the middle of box.
//
// **Measured off the face rather than halved by eye**, so the figure stays centered when the bar's
// height or the type's size moves — the same rule the stat rule between the duelist's two groups
// follows.
func centeredTextTop(f *Faces, size float64, box image.Rectangle) (int, error) {
	face, err := f.at(size)
	if err != nil {
		return 0, err
	}
	m := face.Metrics()
	return box.Min.Y + (box.Dy()-m.Ascent.Ceil()-m.Descent.Ceil())/2, nil
}
