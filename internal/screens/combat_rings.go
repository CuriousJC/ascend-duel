package screens

// The ring pane: what the player is wearing, drawn as cards.
//
// **A layout sketch, added 2026-08-11.** Nothing here is a mechanic — no ring is bought,
// equipped, unequipped or read by any rule, and `internal/combat` still cannot see an element
// so neither the discount nor the flip MECHANICS.md describes can exist yet. What this does is
// put five ring-sized slots on the screen and fill the first few from `data/rings.json`, so the
// space rings will want is spoken for while the rules are still being written.
//
// **It claims the band the full-height panes vacated.** Action Flow is built and not drawn,
// and Resolution left for the three-line feed above the hand on 2026-08-11, so 12–46% was
// empty. That is what paid for full-size ring cards; the alternative was a row of chips at deck
// -stack size, which would have made the ring a different object from every other card in the
// game.

import (
	"fmt"
	"image"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"image/color"
)

// maxRings is how many rings can be worn at once.
//
// **Five, and it is a rule from MECHANICS.md rather than a number the layout chose** — brands
// are what expand it, and nothing else may. It lives here because the pane is the only thing
// that reads it today; when equipping becomes real it moves to wherever the rule does, exactly
// as `maxSelected` moved off this screen and into `internal/combat`.
const maxRings = 5

const (
	// The pane starts where the character block ends, and runs to a right edge that clears the
	// enemy card. **79% is not arbitrary**: the enemy card is centred at 88% and is a card
	// wide, so its left edge is around 1045 on a 1280 screen and anything past ~81% is drawn
	// under it. The margin grew when the cards shrank on 2026-08-11 and 79% was left alone.
	ringPaneGap      = 16
	ringPaneRightPct = 79

	// How far the rule sits below the cards, and how thick it is.
	ringRuleGap   = 12
	ringRuleWidth = 2

	// The cap fraction under the bottom-right corner, sized and spaced like the deck pile's
	// count, which is the same idea: a number saying how much of a fixed thing is in use.
	ringCountSize     = 22
	ringCountTopGap   = 6
	ringCountRightPad = 2
)

// ringRuleColor is the line under the rings.
//
// **Neutral rather than the ring's pink** *(2026-08-11)*. The pane was a full pink box —
// filled, bordered and titled — for its first hour and the box was the loudest thing in the
// band, competing with the five saturated pink borders standing inside it. What is left is a
// rule: it says where the row ends and nothing else, so it takes a grey that is read as
// structure rather than as a colour meaning something.
var ringRuleColor = color.RGBA{R: 120, G: 122, B: 132, A: 255}

// ringPaneRect is the row's extent: the cards' own band, from the character block's right edge
// to the enemy card's left, aligned with the top of the block.
//
// **It is not a box any more.** Nothing is filled or framed — the bottom edge is where the rule
// is drawn and where the fraction hangs, and the rectangle exists so those two and the slots
// all read one geometry.
func (s *CombatScene) ringPaneRect(gs *state.GlobalState) image.Rectangle {
	block := s.fighterBlockRect(gs)

	left := block.Max.X + ringPaneGap
	top := block.Min.Y
	right := gs.PctX(ringPaneRightPct)
	bottom := top + cards.RingStyle.Height + ringRuleGap

	return image.Rect(left, top, right, bottom)
}

// ringSlotPitch is how far apart two ring cards start, **for the number actually worn**.
//
// **The row spreads to fill the pane and closes up as it fills** *(2026-08-11)*. Three rings
// stand well apart and fully visible; five sit shoulder to shoulder with a few pixels between
// them, since 5 x 162 is 810 against roughly 825 of pane. **It overlapped by 26 pixels each
// until the cards came down a tenth in width later the same day**, and it would again the
// moment the pane narrowed or a sixth slot was ever allowed — the pitch is derived, so the row
// closes up by itself rather than by anyone redoing this arithmetic. Overlapping is the
// accepted failure mode, not shrinking: a card cannot be scaled, a smaller ring is a
// *different drawing*, and there is no ring style below this one.
//
// Pitch is always first-card-flush-left to last-card-flush-right, so the row's edges are fixed
// and only the spacing inside them moves. One ring sits at the left edge, since with a single
// card the two anchors are the same one and the left is where a growing row starts.
func ringSlotPitch(r image.Rectangle, worn int) int {
	if worn < 2 {
		return 0
	}
	return (r.Dx() - cards.RingStyle.Width) / (worn - 1)
}

// ringSlotAt is where the i'th ring card's top-left corner sits. **Flush with the pane's top**,
// which is the top of the character block beside it — the two are aligned directly rather than
// each being inset inside a frame of its own.
func ringSlotAt(r image.Rectangle, i, worn int) image.Point {
	return image.Pt(r.Min.X+i*ringSlotPitch(r, worn), r.Min.Y)
}

// equippedRings is what the player is wearing.
//
// **Everything defined, up to the cap** — this is the sketch's whole rule, and it is the reason
// the pane shows something at all before equipping exists. It walks `data.RingOrder` rather
// than the map, per the determinism rules: map order would deal a different row of rings every
// launch and it would look like a bug in the layout rather than one in the iteration.
func equippedRings(gs *state.GlobalState) []data.RingData {
	order := data.RingOrder(gs.Rings)
	if len(order) > maxRings {
		order = order[:maxRings]
	}

	out := make([]data.RingData, 0, len(order))
	for _, key := range order {
		out = append(out, gs.Rings[key])
	}
	return out
}

// drawRingPane draws the rings, a rule under them, and the cap as a fraction on its right end.
//
// **There is no box** *(2026-08-11)*. The first version framed the row the way the character
// block is framed — dim pink fill, pink border, a "Rings" title — and it read as a panel with
// cards trapped in it rather than as the cards themselves. What is left is the row, aligned
// directly with the top of the block beside it, and a line saying where it ends. **The cards
// are the pane.**
//
// **Empty slots are not drawn.** They were the first sketch and the fraction replaced them:
// five frames of which two are dashed outlines spends the loudest thing in the row on saying
// what you have *not* got, where `3/5` says the same thing on the end of the rule.
// MECHANICS.md's "the cap is never displayed — it surfaces when you try to buy a sixth" is the
// rule this softens, and softening it is the owner's call: the fraction is the deck pile's
// idea, where a count of a fixed total is read without being looked for.
func (s *CombatScene) drawRingPane(gs *state.GlobalState, screen *ebiten.Image) {
	r := s.ringPaneRect(gs)

	worn := equippedRings(gs)
	for i, ring := range worn {
		img := cardImage(gs, ringSpec(gs, ring), cards.RingStyle)
		if img == nil {
			continue // a missing font: drawCard does nothing for the same reason
		}
		at := ringSlotAt(r, i, len(worn))

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(at.X), float64(at.Y))
		screen.DrawImage(img, op)
	}

	// The rule: the row's whole width, whatever is standing on it. **It is drawn even with no
	// rings equipped**, which is deliberate — an empty band with a line under it says the row
	// exists and is empty, where nothing at all says the screen forgot to draw something.
	vector.DrawFilledRect(screen,
		float32(r.Min.X), float32(r.Max.Y), float32(r.Dx()), ringRuleWidth,
		ringRuleColor, false)

	// **`worn / cap`, exactly like the pile's `left / owned`.** The numerator is what moves and
	// the denominator deliberately never does, so the figure is read as "three of your five
	// fingers are spoken for" rather than as two unrelated numbers.
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(r.Max.X-ringCountRightPad),
		float64(r.Max.Y+ringRuleWidth+ringCountTopGap))
	op.PrimaryAlign = text.AlignEnd
	text.Draw(screen, fmt.Sprintf("%d/%d", len(worn), maxRings),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: ringCountSize}, op)
}
