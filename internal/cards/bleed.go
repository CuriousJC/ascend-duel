package cards

// Full-bleed art: a card whose picture is the whole face rather than a panel on it.
//
// **It is the second of two art paths and the two do not compose** *(owner's call,
// 2026-09-11)*. `drawArt` fits a picture into a box named by the style and leaves the off-white
// `Surface` around it; this covers the card with one, clips it to the rounded interior, and
// draws everything else on top. A style picks one by setting `ArtBleed`, and the art is authored
// against that choice: a fitted box wants a square picture and a bleeding card wants the card's
// own 200x280.
//
// **What it costs is the surface, and the surface was carrying the text.** Every ink in this
// package is near-black because it was written against `Surface`; over a dark picture none of
// them is legible. So a bleeding card paints a **scrim** — a dark band at partial alpha — over
// the art wherever type goes, and switches to the light ink set that band is for. The scrim is
// the honest answer rather than the cheap one: the alternative was to make the *prompt* keep the
// text zones quiet, which leaves every card one loud generation away from being unreadable and
// fails silently when it happens.
//
// **The band is derived, never authored.** It is computed from the same offsets the type is
// drawn at, so moving the text moves its ground with it — two numbers describing one band is how
// a scrim ends up half a line off the words it is under.

import (
	"image"
	"image/color"

	xdraw "golang.org/x/image/draw"

	"github.com/curiousjc/ascend-duel/internal/systems"
)

var (
	// ScrimSurface is the ground a bleeding card's type actually sits on: the colour the scrim
	// converges to where it is most opaque. It is the ink set's Surface, and it is what a
	// disabled bleeding card fades toward, for the reason SurfaceDisabled exists on the other
	// path — a state has to be expressed as distance to whatever the thing is drawn on.
	ScrimSurface = color.RGBA{R: 20, G: 21, B: 27, A: 255}

	// LabelInkOnScrim is the light counterpart of LabelInk, and it is the only one of the three
	// named inks a bleeding card can actually reach: neither bleeding style names itself, and
	// neither carries stat rows, so NameInk and NumberInk never arrive here. A translation for an
	// ink nothing passes would be a table entry no reader could check.
	//
	// **It is not the quiet one, unlike the ink it stands in for.** LabelInk is a stat row's
	// *word* on a card that also carries figures, where it is deliberately quieter than what it
	// labels. What arrives here as LabelInk is a worm's sentence — the whole content of its card —
	// because drawMarkedLine falls back to LabelInk for any run with no colour of its own. Set as
	// quietly as its name suggests, it read as a caption on a picture.
	LabelInkOnScrim = color.RGBA{R: 222, G: 224, B: 232, A: 255}
)

const (
	// scrimAlpha is how much of the picture a scrim keeps. High enough that a light figure
	// clears the loudest thing the art is allowed to put under it, low enough that the picture
	// is still visibly continuing behind the words rather than stopping at a bar.
	scrimAlpha = 205

	// scrimPad is the breathing room a scrim leaves above and below the type it is under. A band
	// stopping exactly on the ink reads as a highlighter rather than as a surface.
	scrimPad = 6

	// elementLiftToward is how far an authored element colour is carried toward white on a
	// bleeding card. The five element inks were chosen to read against off-white, and arcane and
	// earth in particular go dark; on a scrim they need lifting or the coloured run in a
	// sentence is the one part of it that cannot be read.
	elementLiftToward = 34
)

// scrimWhite is what elementLiftToward carries a colour toward. Not the glyph palette's pure
// white, which would wash the five hues into one another at this distance.
var scrimWhite = color.RGBA{R: 244, G: 246, B: 250, A: 255}

// drawArtBleed paints Spec.Art across the whole face, scaled to cover and clipped to the inside
// of the border.
//
// **Cover, not fit.** fitInto letterboxes, which on a card means a picture with the surface
// showing above and below it — the exact thing this path exists to remove. The art is authored
// at the card's own aspect, so the crop is normally nothing; covering is what keeps a picture
// that is a few pixels off from opening a seam.
//
// **Clipped to the border's inner curve**, which is why the radius is reduced along with the
// box: a picture clipped to the *outer* silhouette would be drawn underneath the border's own
// bevel and show through the corners where the two curves disagree.
func drawArtBleed(dst *image.RGBA, s Spec, st Style) {
	if s.Art == nil {
		return
	}
	inset := st.BorderWidth
	w, h := st.Width, st.Height
	iw, ih := w-2*inset, h-2*inset
	if iw <= 0 || ih <= 0 {
		return
	}

	tmp := image.NewRGBA(image.Rect(0, 0, w, h))
	coverInto(tmp, s.Art, image.Rect(0, 0, w, h))

	radius := clampRadius(iw, ih, st.CornerRadius-inset)
	for y := 0; y < ih; y++ {
		for x := 0; x < iw; x++ {
			if !insideRounded(iw, ih, radius, x, y) {
				continue
			}
			dst.SetRGBA(x+inset, y+inset, tmp.RGBAAt(x+inset, y+inset))
		}
	}
}

// coverInto scales src to cover box without distorting it and centres it there, cropping
// whichever axis is long. It is fitInto's opposite and the only difference is which of the two
// scales wins.
func coverInto(dst *image.RGBA, src image.Image, box image.Rectangle) {
	b := src.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 || box.Empty() {
		return
	}

	scale := float64(box.Dx()) / float64(b.Dx())
	if v := float64(box.Dy()) / float64(b.Dy()); v > scale {
		scale = v
	}
	w, h := int(float64(b.Dx())*scale), int(float64(b.Dy())*scale)
	if w <= 0 || h <= 0 {
		return
	}

	at := image.Rect(0, 0, w, h).
		Add(box.Min).
		Add(image.Pt((box.Dx()-w)/2, (box.Dy()-h)/2))

	xdraw.CatmullRom.Scale(dst, at, src, b, xdraw.Over, nil)
}

// drawScrims paints the dark bands a bleeding card's type is read against.
//
// Each band is computed from the offsets the type it covers is drawn at, so moving the text band
// moves its scrim with it and there is no second number to keep in step.
func drawScrims(dst *image.RGBA, st Style) {
	for _, band := range st.scrimBands() {
		scrimBand(dst, st, band)
	}
}

// scrimBands is every horizontal strip of a bleeding card that carries type.
//
// **Empty on a style that does not bleed**, so the whole feature is one predicate rather than a
// branch at each call site.
func (st Style) scrimBands() []image.Rectangle {
	if !st.ArtBleed {
		return nil
	}

	// **One band, because a bleeding card carries one piece of type.** Neither bleeding style
	// names itself — see ShowName on each — so the only thing needing a ground is the sentence,
	// and a style with nothing to say gets no scrim and shows the whole picture.
	var bands []image.Rectangle
	if st.TextBandBottom > st.TextBandTop {
		bands = append(bands, image.Rect(0, st.TextBandTop-scrimPad, st.Width, st.TextBandBottom+scrimPad))
	}
	return bands
}

// scrimBand darkens one strip of the face, clipped to the border's inner curve so a band running
// the full width cannot square off a corner.
func scrimBand(dst *image.RGBA, st Style, band image.Rectangle) {
	inset := st.BorderWidth
	w, h := st.Width, st.Height
	iw, ih := w-2*inset, h-2*inset
	if iw <= 0 || ih <= 0 {
		return
	}
	radius := clampRadius(iw, ih, st.CornerRadius-inset)

	top, bottom := max(band.Min.Y, inset), min(band.Max.Y, h-inset)
	for y := top; y < bottom; y++ {
		for x := inset; x < w-inset; x++ {
			if !insideRounded(iw, ih, radius, x-inset, y-inset) {
				continue
			}
			dst.SetRGBA(x, y, blend(dst.RGBAAt(x, y), ScrimSurface, scrimAlpha))
		}
	}
}

// blend mixes over into under by alpha out of 255. Plain Go arithmetic, like everything else
// this package rasterises: there is no graphics context here to ask for a blend mode.
func blend(under, over color.RGBA, alpha int) color.RGBA {
	m := func(u, o uint8) uint8 {
		return uint8((int(u)*(255-alpha) + int(o)*alpha) / 255)
	}
	return color.RGBA{R: m(under.R, over.R), G: m(under.G, over.G), B: m(under.B, over.B), A: 255}
}

// onScrim is the ink rule for a bleeding card: the card's own body ink is swapped for its light
// counterpart, an authored colour is lifted toward white, and the state transform runs after both
// against ScrimSurface rather than against the off-white one.
//
// **A named swap rather than a luminance test.** Flipping any dark colour would catch the element
// inks too and turn a fire run and an arcane run into the same near-white; naming the one ink is
// what keeps "this ink is type" and "this ink is an element" different questions.
func onScrim(s Spec) func(color.RGBA) color.RGBA {
	return func(c color.RGBA) color.RGBA {
		if c == LabelInk {
			c = LabelInkOnScrim
		} else {
			c = systems.ColorToward(c, scrimWhite, elementLiftToward)
		}
		if !s.Enabled {
			return systems.ColorToward(c, ScrimSurface, inkDisabledToward)
		}
		return c
	}
}
