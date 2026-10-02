package cards

// Upgrades on a card face: **authored art covering the face inside the border ring.**
//
// An upgrade is what the run has permanently made a card, and a card carries exactly one — see
// `systems.Upgrade`, which is where the vocabulary lives. This file is the drawing: the picture
// `data/upgrade_art.json` names for the upgrade, scaled to the card and clipped to the inside of its
// rounded border. It is the same placement the playing cards' own art takes, and the picture is
// committed at the same 200x280.
//
// **The left edge is the one that has to work.** The deck panel overlaps every card over the one
// before it, so a stacked row shows each card's left side and nothing else; the art there is what
// lets a gold card be picked out of a pile.
//
// **The art goes down before anything is drawn on the card**, so the figure, the form mark, the cost
// ticks, the badge and the name sit on top of it. The figure is lifted off its own ground to do so —
// see matte.go.
//
// **Inside the ring, not on it.** The border is the card's *state* — resting, selected,
// unaffordable, dragged — and an upgrade painted over it would be a second thing in the one place
// the card says the first.
//
// **It is the same picture at rest and selected.** Selecting a card lifts it out of the row, and
// that is the whole of what selection says; art that brightened on the click would be the card
// itself changing under the cursor. **Only an unaffordable card fades it**, by the distance the
// border walks toward SurfaceDisabled, so an unaffordable gold card reads as unaffordable first.

import (
	"fmt"
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// upgradeArt is the catalog, keyed by the upgrade each record draws for. UpgradeNone holds the
// default, which no card asks for by itself — UpgradeArtOf is the one door.
var upgradeArt = loadUpgradeArt()

// loadUpgradeArt reads data/upgrade_art.json and refuses a record naming no upgrade. **Here rather
// than in `data`**, because `data` sits under the vocabulary and cannot ask what an upgrade is
// called.
func loadUpgradeArt() map[systems.Upgrade]data.UpgradeArtData {
	out := map[systems.Upgrade]data.UpgradeArtData{}
	raw := data.LoadUpgradeArt()
	for _, k := range data.UpgradeArtOrder(raw) {
		if k == data.DefaultUpgradeArt {
			out[systems.UpgradeNone] = raw[k]
			continue
		}
		u, ok := systems.ParseUpgrade(k)
		if !ok {
			panic(fmt.Sprintf("upgrade_art.json: %q is not an upgrade", k))
		}
		out[u] = raw[k]
	}
	return out
}

// UpgradeArtOf is the picture an upgrade lays over a card, and false for UpgradeNone — a card the
// run has not altered draws none. An upgrade with no record of its own draws the default.
func UpgradeArtOf(u systems.Upgrade) (data.UpgradeArtData, bool) {
	if u == systems.UpgradeNone {
		return data.UpgradeArtData{}, false
	}
	if a, ok := upgradeArt[u]; ok {
		return a, true
	}
	return upgradeArt[systems.UpgradeNone], true
}

// drawUpgradeArt lays the card's upgrade over its face, inside the border ring.
//
// **The picture is fetched at the card's own size**, so the deck panel's half-size card is one
// averaging step down from the committed art rather than a second picture.
func drawUpgradeArt(dst *image.RGBA, s Spec, st Style) {
	a, ok := UpgradeArtOf(s.Upgrade)
	if !ok {
		return
	}
	art := systems.ArtMark(a.ArtKey(), st.Width, st.Height)
	if art == nil {
		return
	}
	inset := st.BorderWidth
	iw, ih := st.Width-2*inset, st.Height-2*inset
	if iw <= 0 || ih <= 0 {
		return
	}
	radius := clampRadius(iw, ih, st.CornerRadius-inset)

	fade, toward := 0, Surface
	if !s.Enabled {
		fade, toward = borderDisabledToward, SurfaceDisabled
	}
	b := art.Bounds()
	for y := 0; y < ih; y++ {
		for x := 0; x < iw; x++ {
			if !insideRounded(iw, ih, radius, x, y) {
				continue
			}
			c := art.RGBAAt(b.Min.X+x+inset, b.Min.Y+y+inset)
			if c.A == 0 {
				continue
			}
			ink := systems.ColorToward(unpremultiply(c), toward, fade)
			at := image.Pt(x+inset, y+inset)
			dst.SetRGBA(at.X, at.Y, blend(dst.RGBAAt(at.X, at.Y), ink, int(c.A)))
		}
	}
}

// unpremultiply takes a stored pixel back to straight color, which is what blending it at its own
// alpha wants.
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
