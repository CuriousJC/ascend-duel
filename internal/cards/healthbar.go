package cards

import (
	"fmt"
	"image"
	"image/color"

	xdraw "golang.org/x/image/draw"

	"github.com/curiousjc/ascend-duel/internal/systems"
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
//
// **The bar is authored art, three pieces of it** — see docs/art/health_bar_art_prompt.MD. A
// rounded trough, a rounded fill laid in it at the width life calls for, and a dashed marker laid
// across both at each line `Spec.LifeMarks` asks for. The trough and the fill are two end caps and
// a middle every column of which is identical, the shape the AP bar's cells are, so only the middle
// stretches and the ends stay round at any width. **Where a piece has not been filed the bar is
// drawn instead**, square-cornered, piece by piece, so a missing marker does not take the trough
// with it.

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

// The health bar's art: the asset keys, the size every piece is authored at, and how much of the
// trough's and the fill's width each end cap takes.
const (
	healthTroughArt = "health-bar-trough"
	healthFillArt   = "health-bar-fill"
	healthMarkArt   = "health-bar-third"

	healthArtW, healthArtH = 256, 64
	healthArtCap           = 32
	healthMarkArtW         = 32
)

// HealthMarkInk is the drawn marker's color where no marker art is filed: the AP bar's amber, so a
// line on the bar and a point of the turn's budget read as the same kind of mark.
var HealthMarkInk = color.RGBA{R: 196, G: 124, B: 12, A: 255}

// drawHealth draws the bar, its marks, and writes the fraction across it.
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

	if !drawStretched(dst, healthTroughArt, bar) {
		fillRect(dst, bar.Min.X, bar.Min.Y, bar.Dx(), bar.Dy(), HealthEmpty)
	}
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
	if w := bar.Dx() * life / s.MaxLife; w > 0 {
		fill := image.Rect(bar.Min.X, bar.Min.Y, bar.Min.X+w, bar.Max.Y)
		if !drawStretched(dst, healthFillArt, fill) {
			fillRect(dst, fill.Min.X, fill.Min.Y, fill.Dx(), fill.Dy(), HealthFull)
		}
	}
	drawHealthMarks(dst, s.LifeMarks, bar)

	// **The exact number as well as the bar, deliberately.** A bar says roughly how hurt
	// something is, and a duel decided in whole points wants the figure.
	top, err := centeredTextTop(f, st.HealthTextSize, bar)
	if err != nil {
		return err
	}
	return drawTextHCenteredAs(uiLetters, dst, f, st.HealthTextSize,
		fmt.Sprintf("%d/%d", life, s.MaxLife), st.Width, top, HealthTextInk)
}

// HealthMarkXs is where the marks fall on a bar: one at each line cutting it into that many equal
// parts, at the floor of each fraction. None for fewer than two parts.
func HealthMarkXs(bar image.Rectangle, parts int) []int {
	if parts < 2 || bar.Empty() {
		return nil
	}
	xs := make([]int, 0, parts-1)
	for k := 1; k < parts; k++ {
		xs = append(xs, bar.Min.X+bar.Dx()*k/parts)
	}
	return xs
}

// drawHealthMarks lays a marker across the bar at each line, centered on it and as tall as the
// bar, over the fill and the trough alike — **it marks the bar, not the life**, so it is in the
// same place however hurt the fighter is.
func drawHealthMarks(dst *image.RGBA, parts int, bar image.Rectangle) {
	h := bar.Dy()
	w := max(h*healthMarkArtW/healthArtH, 1)
	art := systems.ArtMark(healthMarkArt, healthMarkArtW, healthArtH)
	for _, x := range HealthMarkXs(bar, parts) {
		at := image.Rect(x-w/2, bar.Min.Y, x-w/2+w, bar.Max.Y)
		if art != nil {
			xdraw.CatmullRom.Scale(dst, at, art, art.Bounds(), xdraw.Over, nil)
			continue
		}
		// Three dashes down a two-pixel line, the gaps half a dash: the drawing the art replaces.
		dash := max(h*2/7, 1)
		gap := max(dash/2, 1)
		for i, y := 0, bar.Min.Y+(h-3*dash-2*gap)/2; i < 3; i, y = i+1, y+dash+gap {
			fillRect(dst, x-1, y, 2, dash, HealthMarkInk)
		}
	}
}

// drawStretched lays one of the bar's two stretchable pieces into r: each end cap scaled by the
// height and never by the width, and the middle stretched across whatever is left. A rectangle
// narrower than its two caps squeezes them rather than overlapping them, so a sliver of life is
// still a rounded sliver.
//
// False when the piece has not been filed, and the caller draws the bar instead.
func drawStretched(dst *image.RGBA, key string, r image.Rectangle) bool {
	src := systems.ArtMark(key, healthArtW, healthArtH)
	if src == nil {
		return false
	}
	capW := healthArtCap * r.Dy() / healthArtH
	if 2*capW > r.Dx() {
		capW = r.Dx() / 2
	}
	piece := func(sx0, sx1, dx0, dx1 int) {
		if dx1 <= dx0 {
			return
		}
		xdraw.CatmullRom.Scale(dst, image.Rect(dx0, r.Min.Y, dx1, r.Max.Y),
			src, image.Rect(sx0, 0, sx1, healthArtH), xdraw.Over, nil)
	}
	piece(0, healthArtCap, r.Min.X, r.Min.X+capW)
	piece(healthArtCap, healthArtW-healthArtCap, r.Min.X+capW, r.Max.X-capW)
	piece(healthArtW-healthArtCap, healthArtW, r.Max.X-capW, r.Max.X)
	return true
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
