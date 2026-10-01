package cards

// Upgrades on a card face: **a strip of authored art down each side of the face.**
//
// An upgrade is what the run has permanently made a card, and a card carries exactly one — see
// `systems.Upgrade`, which is where the vocabulary lives. This file is the drawing: the strip
// `data/edges.json` names for the upgrade, stretched to the face's height just inside the border
// ring, on the left edge — and, for a record whose `Sides` is `both`, mirrored on the right.
//
// **The left edge is the one that has to work.** The deck panel overlaps every card over the one
// before it, so a stacked row shows each card's left side and nothing else; a strip there is what
// lets a gold card be picked out of a pile. A right strip, where a record asks for one, is seen
// only on the top card of a stack and on a card standing alone, and is the same picture mirrored
// so the two edges are one object.
//
// **The strip goes down before anything is written on the card**, so the form mark, the cost ticks
// and the text sit on top of it. A strip wider than the margin left of the column is allowed and
// meant: what it covers is the card's ground, never what the card says.
//
// **Inside the ring, not on it**, at the default inset. The border is the card's *state* —
// resting, selected, unaffordable, dragged — and an upgrade painted over it would be a second thing
// in the one place the card says the first. A record may still set `Inset` to zero and take the
// ring, which is a draw property to try rather than a rule.
//
// **It fades out across its own width**, from FadeFrom to FadeTo, so a wide strip can reach into
// the face and dissolve rather than stop at a line.
//
// **It is the same picture at rest and selected.** Selecting a card lifts it out of the row, and
// that is the whole of what selection says; a strip that brightened on the click would be the card
// itself changing under the cursor. **Only an unaffordable card fades it**, by the distance the
// border walks toward SurfaceDisabled, so an unaffordable gold card reads as unaffordable first.

import (
	"fmt"
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// edges is the catalog, keyed by the upgrade each record draws for. UpgradeNone holds the default,
// which no card asks for by itself — EdgeOf is the one door.
var edges = loadEdges()

// loadEdges reads data/edges.json and refuses a record naming no upgrade. **Here rather than in
// `data`**, because `data` sits under the vocabulary and cannot ask what an upgrade is called.
func loadEdges() map[systems.Upgrade]data.EdgeData {
	out := map[systems.Upgrade]data.EdgeData{}
	raw := data.LoadEdges()
	for _, k := range data.EdgeOrder(raw) {
		if k == data.DefaultEdge {
			out[systems.UpgradeNone] = raw[k]
			continue
		}
		u, ok := systems.ParseUpgrade(k)
		if !ok {
			panic(fmt.Sprintf("edges.json: %q is not an upgrade", k))
		}
		out[u] = raw[k]
	}
	return out
}

// EdgeOf is the strip an upgrade draws, and false for UpgradeNone — a card the run has not altered
// draws none. An upgrade with no record of its own draws the default.
func EdgeOf(u systems.Upgrade) (data.EdgeData, bool) {
	if u == systems.UpgradeNone {
		return data.EdgeData{}, false
	}
	if e, ok := edges[u]; ok {
		return e, true
	}
	return edges[systems.UpgradeNone], true
}

// drawEdges paints the card's upgrade strip down both sides of the face.
//
// **The record's figures are measured on Hand and scaled to this style's width**, rounded to
// nearest, so the deck panel's half-size card draws a half-width strip at a half inset.
func drawEdges(dst *image.RGBA, s Spec, st Style) {
	e, ok := EdgeOf(s.Upgrade)
	if !ok {
		return
	}
	scale := func(v int) int { return (v*st.Width*2 + Hand.Width) / (Hand.Width * 2) }
	w, inset := max(scale(e.Width), 1), scale(e.Inset)
	fadeFrom, fadeTo := scale(e.FadeFrom), scale(e.FadeTo)
	iw, ih := st.Width-2*inset, st.Height-2*inset
	if iw <= 0 || ih <= 0 {
		return
	}
	strip := systems.ArtMark(e.ArtKey(), w, ih)
	if strip == nil {
		return
	}

	radius := clampRadius(iw, ih, st.CornerRadius-inset)
	fade, toward := 0, Surface
	if !s.Enabled {
		fade, toward = borderDisabledToward, SurfaceDisabled
	}
	b := strip.Bounds()
	for x := 0; x < w && x < iw; x++ {
		// How much of the strip's opacity this column keeps, out of 100.
		keep := 100
		switch {
		case x >= fadeTo:
			keep = 0
		case x >= fadeFrom:
			keep = 100 * (fadeTo - x) / (fadeTo - fadeFrom)
		}
		if keep == 0 {
			continue
		}
		for y := 0; y < ih; y++ {
			c := strip.RGBAAt(b.Min.X+x, b.Min.Y+y)
			if c.A == 0 {
				continue
			}
			ink := systems.ColorToward(unpremultiply(c), toward, fade)
			alpha := int(c.A) * e.Opacity * keep / 10000

			// The left strip as authored, the right one mirrored, so the outside of the picture
			// faces the outside of the card on both edges.
			for i, px := range [2]int{x, iw - 1 - x} {
				if i == 1 && e.Sides != data.EdgeBoth {
					break
				}
				if !insideRounded(iw, ih, radius, px, y) {
					continue
				}
				at := image.Pt(px+inset, y+inset)
				dst.SetRGBA(at.X, at.Y, blend(dst.RGBAAt(at.X, at.Y), ink, alpha))
			}
		}
	}
}

// unpremultiply takes a stored pixel back to straight color, which is what blending it at the
// strip's own alpha wants.
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
