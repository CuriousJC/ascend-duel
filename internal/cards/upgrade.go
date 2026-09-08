package cards

// Upgrades on a card face: **painting the left column from a picture instead of from a colour.**
//
// A card states its element in the left column — the tinted form mark and the cost ticks under it
// — and everywhere else on the card that decision is already made: the border is neutral, the
// surface is constant, and the only other hue on the face belongs to a ring. See CLAUDE.md, which
// records the swap.
//
// A *visible upgrade* is something the run has done to a card that the face has to say. The first
// one is the wildcard, whose whole subject is that the card no longer has one element — so it takes
// the column over and paints it from **the five element colours themselves**, in five bands. That
// is not a fourth thing on a card with three; it is the same fact the column always stated, stated
// about a card whose answer is now "all of them".
//
// # The three ways a mark and an ink can be combined
//
// They are here rather than chosen once and written in because which one reads best at 32 pixels
// on an off-white card is a question to answer by looking, not by arguing — `tools/upgradesheet`
// draws all three against all four form marks at card scale and enlarged. **The looking was done
// on 2026-09-07 and TintProject won**; the other two are kept because the answer turned out to
// depend on the ink rather than on the mode, so a future ink may well pick differently. See
// TintProject.
//
// **All three are premultiplied throughout**, for tintInk's reason: the form marks arrive
// downsampled, so their edge pixels are partly transparent, and brightness has to come off the
// *unpremultiplied* value or every soft edge comes out darker than the ink it belongs to.

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/systems"
)

// TintMode is how an upgrade's ink and a glyph's drawing are combined.
//
// **Not append-only and not serialized.** It is a review knob: nothing outside this package and
// `tools/upgradesheet` names one, and no file writes it down.
type TintMode int

const (
	// TintProject maps the mark's ink box onto the ink and takes the colour flat, discarding the
	// art's own shading entirely. **It is what the game draws** *(owner's call, 2026-09-07)*.
	//
	// **It won because of what the ink became.** While the ink was a broad diagonal gradient, the
	// question was "how do I keep this drawing readable through a wash of colour" and TintWeave
	// was the answer. The ink is now the five element colours in five bands, and that changes what
	// the mark is for: it is no longer "a rainbow means wild", it is "these five specific colours
	// are the five elements". Weave runs each band through a dark-to-light ramp, which pastels it
	// — ice all but vanishes — and at the 32 pixels the card actually draws, the *saturation* is
	// the message and the bevel is not legible anyway. So the loudest mode is the right one here,
	// which is the opposite of what was true a day earlier.
	//
	// **What it costs is the outline**, and that is a real loss rather than a free win: the rim is
	// why the form marks are drawn art rather than generated silhouettes. It is paid knowingly.
	//
	// **It is first in the enum so that it is the zero value.** `Spec.UpgradeTint` documents its
	// zero as "the mode the game uses", and a caller that never thinks about tint modes — which is
	// every caller but the sheet — must get the right picture. A DefaultTintMode that was not the
	// zero value would be a constant nothing reads.
	TintProject TintMode = iota

	// TintWeave takes the *hue* from the ink at the pixel's own position in the mark, and the
	// light and shade from the pixel's own brightness — the ramp between a dark and a light
	// version of that hue that tintInk already builds, with a different hue at every pixel.
	//
	// It is the mode that keeps the drawing: outline stays the darkest thing on the mark and the
	// specular stays the brightest, so a spear still reads as a spear rather than as a
	// spear-shaped hole in the ink. **It was the default until the ink became the element
	// colours** — see TintProject, which is where that argument is written down. It stays because
	// it is the right answer for any future ink whose bands are wide and whose colours are not
	// themselves the meaning.
	TintWeave

	// TintRamp takes the pixel's brightness and uses it to walk the ink's own diagonal: dark ink
	// lands at one end of the gradient, a specular at the other.
	//
	// It is the literal extension of tintInk — brightness picks a colour off a ramp — and it has
	// failed on every ink tried against it: a diagonal rainbow, vertical stripes, and the element
	// bands all collapse to one flat colour per mark. The reason is structural rather than bad
	// luck: an ink worth having varies in *hue*, a mark's own pixels cluster in a narrow band of
	// *brightness*, and this mode asks brightness to choose the hue. **It is kept only so the
	// sheet can show why it does not work**; nothing should reach for it.
	TintRamp
)

// DefaultTintMode is what the game draws with. See TintProject for why, and note that it is the
// zero value on purpose.
const DefaultTintMode = TintProject

func (m TintMode) String() string {
	switch m {
	case TintRamp:
		return "ramp"
	case TintWeave:
		return "weave"
	default:
		return "project"
	}
}

// TintModes is every mode in a fixed order, for the sheet that compares them. **The default
// leads**, so the page reads as "here is what the game draws, and here is what it declined".
func TintModes() []TintMode { return []TintMode{TintProject, TintWeave, TintRamp} }

// tintUpgrade recolours a glyph from an upgrade's ink rather than from one hue.
//
// It is tintInk's counterpart and takes the same contract: a fresh image rather than a write
// through the one it was handed, because `systems` caches what RenderGlyphAt returns and tinting
// in place would paint the first card drawn onto every card that asked afterwards.
//
// **Sampling is by the mark's *inked* bounds, not its canvas.** Every glyph is drawn on a square
// canvas and none of them fills it — the same fact placeInk exists for — so projecting the ink
// across the canvas would put the middle of the gradient wherever the drawing's margin happened
// to leave it, and two marks with different margins would take their colours from different parts
// of the picture.
func tintUpgrade(src *image.RGBA, ink *image.RGBA, mode TintMode) *image.RGBA {
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	if ink == nil {
		return out
	}

	box := inkBounds(src)
	if box.Empty() {
		return out
	}

	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := src.RGBAAt(b.Min.X+x, b.Min.Y+y)
			if c.A == 0 {
				continue
			}
			a := int(c.A)
			lum := brightness(c)

			var hue color.RGBA
			switch mode {
			case TintRamp:
				hue = sampleDiagonal(ink, lum)
			default:
				hue = sampleBox(ink, image.Pt(b.Min.X+x, b.Min.Y+y), box)
			}

			if mode == TintWeave {
				out.SetRGBA(x, y, rampToward(hue, lum, a, c.A))
				continue
			}
			// Ramp and project both take the ink's colour as it stands and only have to put it
			// back into premultiplied space against the glyph pixel's own alpha.
			out.SetRGBA(x, y, color.RGBA{
				R: uint8(int(hue.R) * a / 255),
				G: uint8(int(hue.G) * a / 255),
				B: uint8(int(hue.B) * a / 255),
				A: c.A,
			})
		}
	}
	return out
}

// brightness is a pixel's own light, unpremultiplied, 0..255.
//
// **The brightest channel rather than a luminance average**, which is tintInk's rule and is kept
// here so the two paths cannot disagree about how bright a pixel is: the art is nearly hueless
// already, and an average would take a near-white specular down to a mid grey.
func brightness(c color.RGBA) int {
	if c.A == 0 {
		return 0
	}
	lum := int(c.R)
	if int(c.G) > lum {
		lum = int(c.G)
	}
	if int(c.B) > lum {
		lum = int(c.B)
	}
	if lum = lum * 255 / int(c.A); lum > 255 {
		lum = 255
	}
	return lum
}

// rampToward is tintInk's body for one pixel: the hue at a third strength for the dark end, the
// hue most of the way to white for the light end, and the pixel's own brightness deciding where
// between them it lands.
func rampToward(hue color.RGBA, lum, a int, alpha uint8) color.RGBA {
	dark := systems.ColorAtStrength(hue, tintDarkPct)
	light := systems.ColorToward(hue, color.RGBA{R: 255, G: 255, B: 255, A: 255}, tintLightToward)

	mix := func(d, l uint8) uint8 {
		v := int(d) + (int(l)-int(d))*lum/255
		return uint8(v * a / 255)
	}
	return color.RGBA{
		R: mix(dark.R, light.R), G: mix(dark.G, light.G), B: mix(dark.B, light.B), A: alpha,
	}
}

// sampleDiagonal walks the ink's main diagonal by a 0..255 position. The diagonal rather than a
// row or a column because the wildcard ink's bands run corner to corner: a row crosses one band
// and reads as a single colour, where the diagonal crosses all of them.
func sampleDiagonal(ink *image.RGBA, at int) color.RGBA {
	b := ink.Bounds()
	last := b.Dx() - 1
	if b.Dy()-1 < last {
		last = b.Dy() - 1
	}
	if last < 0 {
		return color.RGBA{}
	}
	p := at * last / 255
	return unpremultiply(ink.RGBAAt(b.Min.X+p, b.Min.Y+p))
}

// sampleBox maps a point inside box onto the ink and returns the colour there.
func sampleBox(ink *image.RGBA, at image.Point, box image.Rectangle) color.RGBA {
	b := ink.Bounds()
	x := scaleInto(at.X-box.Min.X, box.Dx(), b.Dx())
	y := scaleInto(at.Y-box.Min.Y, box.Dy(), b.Dy())
	return unpremultiply(ink.RGBAAt(b.Min.X+x, b.Min.Y+y))
}

// scaleInto maps v in 0..span onto 0..into-1, clamped. Nearest rather than interpolated: the ink
// is a small authored picture and averaging its bands would mud the colours it exists to state.
func scaleInto(v, span, into int) int {
	if into <= 0 {
		return 0
	}
	if span <= 0 {
		return 0
	}
	out := v * into / span
	if out < 0 {
		return 0
	}
	if out >= into {
		return into - 1
	}
	return out
}

// unpremultiply takes a stored pixel back to straight colour, which is what every caller here
// wants: a hue to ramp, or a colour to re-multiply against a different alpha.
func unpremultiply(c color.RGBA) color.RGBA {
	if c.A == 0 || c.A == 255 {
		return c
	}
	up := func(v uint8) uint8 {
		n := int(v) * 255 / int(c.A)
		if n > 255 {
			n = 255
		}
		return uint8(n)
	}
	return color.RGBA{R: up(c.R), G: up(c.G), B: up(c.B), A: 255}
}

// drawUpgradedDashes is drawDashes painted from an upgrade's ink instead of from one colour.
//
// **The ink is projected across the whole stack, not across each tick.** A tick is four pixels
// tall and a gradient inside one is invisible; a gradient across the stack is what makes three
// ticks read as one rainbow column rather than as three arbitrary colours. It follows that a
// one-cost card shows a slice of the ink rather than all of it, which is correct — the column is
// as tall as the cost is.
//
// **State is applied per pixel**, through the same Spec.atState the flat path uses, so a selected
// or unaffordable wildcard fades with the rest of its card. That is the rule TestTheTicksAndTheBorderShareOneState
// holds, kept by going through one switch rather than by writing a second one.
func drawUpgradedDashes(dst *image.RGBA, s Spec, st Style, ink *image.RGBA) {
	stack, ok := dashStack(s, st)
	if !ok || ink == nil {
		return
	}
	for i := 0; i < s.Cost; i++ {
		y0 := st.DashTop + i*(st.DashHeight+st.DashGap)
		if y0+st.DashHeight > st.Height {
			return
		}
		for y := y0; y < y0+st.DashHeight; y++ {
			for x := st.DashLeft; x < st.DashLeft+st.DashWidth; x++ {
				c := sampleBox(ink, image.Pt(x, y), stack)
				dst.SetRGBA(x, y, s.atState(c))
			}
		}
	}
}

// dashStack is the rectangle the whole cost column occupies, which is what the ink is projected
// across. It reports false for a style or a card with no ticks at all.
//
// **The geometry is stated once here and read by both paths**, so the flat ticks and the
// upgraded ones cannot end up in different places.
func dashStack(s Spec, st Style) (image.Rectangle, bool) {
	if s.Cost <= 0 || st.DashWidth <= 0 || st.DashHeight <= 0 {
		return image.Rectangle{}, false
	}
	h := s.Cost*st.DashHeight + (s.Cost-1)*st.DashGap
	return image.Rect(st.DashLeft, st.DashTop, st.DashLeft+st.DashWidth, st.DashTop+h), true
}
