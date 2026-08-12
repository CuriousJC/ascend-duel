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
	// The gap on either side of the row: it starts where the duelist card ends and stops
	// where the enemy card begins.
	//
	// **Both edges are read off the cards themselves** *(2026-08-12)*. The right edge was a
	// hardcoded 79%, chosen to clear an enemy card centred at 88% — a percentage standing in
	// for a position it could not see, and one that would have quietly overlapped the moment
	// either card moved. It moved the next day.
	ringPaneGap = 16

	// **The row sits ten pixels below the cards on either side of it** *(2026-08-12)*, where
	// it was flush with their tops for a day.
	//
	// Flush alignment and a backing panel do not both work: the backing's top edge would land
	// exactly on the two cards' top edges, and three things sharing one line reads as a single
	// wide object with two cards embedded in it — which is the "cards trapped in a panel"
	// failure the framed version was retired for, made worse by the frame now being wider than
	// the row. Dropping the row breaks that line, and the offset is what makes the backing
	// legible as a thing *behind* the rings rather than a border *around* everything.
	ringPaneTopDrop = 10

	// How far the rule sits below the cards, and how thick it is.
	ringRuleGap   = 12
	ringRuleWidth = 2

	// The cap fraction under the bottom-right corner, sized and spaced like the deck pile's
	// count, which is the same idea: a number saying how much of a fixed thing is in use.
	ringCountSize     = 22
	ringCountTopGap   = 6
	ringCountRightPad = 2

	// ringPaneBackPad is how far the backing extends past the row on every side. The pitch
	// puts the first card flush left and the last flush right, so with no padding the two end
	// cards would sit on the backing's edge and it would read as a border drawn around them
	// rather than as a surface they stand on.
	//
	// **It has to stay under ringPaneGap**, or the backing runs into the fighter card beside
	// it — 8 against 16 leaves half the gap still showing on each side.
	ringPaneBackPad = 8
)

// ringRuleColor is the line under the rings.
//
// **Neutral rather than the ring's pink** *(2026-08-11)*. The pane was a full pink box —
// filled, bordered and titled — for its first hour and the box was the loudest thing in the
// band, competing with the five saturated pink borders standing inside it. What is left is a
// rule: it says where the row ends and nothing else, so it takes a grey that is read as
// structure rather than as a colour meaning something.
var ringRuleColor = color.RGBA{R: 120, G: 122, B: 132, A: 255}

// ringPaneBackColor is the surface the rings stand on: a lighter grey than the screen's own
// {50,50,50}, and nothing else.
//
// **A fill, not a frame** *(2026-08-12)*, and the distinction is the whole history of this
// pane. The first version was a full pink box — filled, bordered and titled — and it was the
// loudest thing in the band, competing with five saturated pink borders standing inside it. It
// came out the same hour, leaving the cards to be the pane. What that lost is the thing being
// put back now: with the row spanning most of the screen's width and a fighter card at either
// end, nothing said where the middle *began*.
//
// So it is the quietest possible answer to that: one step lighter than the background, no
// border, no title, no hue. A colour that meant something would put it back in competition
// with the borders it sits behind.
var ringPaneBackColor = color.RGBA{R: 72, G: 74, B: 80, A: 255}

// ringPaneRect is the row's extent: the cards' own band, running between the two corner cards
// and dropped ringPaneTopDrop below them.
//
// **It is the middle of a three-part row** — duelist card, rings, enemy card — so it takes its
// edges from its neighbours rather than from percentages of the screen. Whichever card moves,
// the row follows, and the one thing that cannot happen is a ring drawn underneath one of them.
//
// **The rectangle is the cards and the rule, not the backing.** It is what the slots are cut
// out of and what the rule and the fraction hang off; the backing is derived from it — see
// ringPaneBackRect — so growing the padding cannot silently move a ring.
func (s *CombatScene) ringPaneRect(gs *state.GlobalState) image.Rectangle {
	duelist, enemy := s.duelistCardRect(gs), s.enemyCardRect(gs)

	left := duelist.Max.X + ringPaneGap
	top := duelist.Min.Y + ringPaneTopDrop
	right := enemy.Min.X - ringPaneGap
	bottom := top + cards.RingStyle.Height + ringRuleGap

	return image.Rect(left, top, right, bottom)
}

// ringPaneBackRect is the surface drawn behind the row: the row padded on every side, and
// **deep enough to hold the rule and the fraction under it**.
//
// The fraction hanging off the bottom edge onto the dark background would say the rule is the
// panel's floor and the number is loose underneath it, which is backwards — the count belongs
// to the row it counts.
func (s *CombatScene) ringPaneBackRect(gs *state.GlobalState) image.Rectangle {
	r := s.ringPaneRect(gs).Inset(-ringPaneBackPad)
	r.Max.Y = s.ringPaneRect(gs).Max.Y +
		ringRuleWidth + ringCountTopGap + ringCountSize + ringPaneBackPad
	return r
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

// drawRingPane draws the backing, the rings, a rule under them, and the cap as a fraction on
// its right end.
//
// **There is still no box** *(2026-08-12)*. The backing that arrived today is a fill and not a
// frame — one step lighter than the screen, no border, no title, no hue — which is a different
// thing from the pink panel this pane started as and was stripped of on 2026-08-11. What that
// stripping went too far on is legibility of the *edges*: with a fighter card at either end of
// a row spanning most of the screen, nothing said where the middle began. See
// ringPaneBackColor.
//
// **Empty slots are not drawn.** They were the first sketch and the fraction replaced them:
// five frames of which two are dashed outlines spends the loudest thing in the row on saying
// what you have *not* got, where `3/5` says the same thing on the end of the rule.
// MECHANICS.md's "the cap is never displayed — it surfaces when you try to buy a sixth" is the
// rule this softens, and softening it is the owner's call: the fraction is the deck pile's
// idea, where a count of a fixed total is read without being looked for.
func (s *CombatScene) drawRingPane(gs *state.GlobalState, screen *ebiten.Image) {
	r := s.ringPaneRect(gs)

	// The surface first, so everything else stands on it.
	back := s.ringPaneBackRect(gs)
	vector.DrawFilledRect(screen,
		float32(back.Min.X), float32(back.Min.Y), float32(back.Dx()), float32(back.Dy()),
		ringPaneBackColor, false)

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
