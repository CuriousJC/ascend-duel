package screens

// The ring pane: what the player is wearing, drawn as cards.
//
// **It draws what the run is wearing** *(2026-08-17)*. The worn set lives on `session.Session` —
// which is what makes it survive a fight — and every rule a ring has is in `data/rings.json` in the
// `When` / `If` / `Then` grammar. This file is the row; it no longer decides anything. **The loop
// around it landed on 2026-08-21** — see shop.go, which draws the same ring cards on a shelf and is
// the only thing that puts one on or takes one off.
//
// **It claims the band the full-height panes vacated.** Action Flow is built and not drawn,
// and Resolution left for the three-line feed above the hand on 2026-08-11, so 12–46% was
// empty. That is what paid for full-size ring cards; the alternative was a row of chips at deck
// -stack size, which would have made the ring a different object from every other card in the
// game.

import (
	"fmt"
	"image"
	"log"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"image/color"
)

// maxRings is how many rings can be worn at once.
//
// **Five, and it is a rule rather than a number the layout chose** — brands are what expand it, and
// nothing else may. It reads `combat.MaxWornRings` rather than declaring a second five: the rule
// moved into the engine with the grammar on 2026-08-17, exactly as `maxSelected` did before it, and
// the row saying `worn/5` while the duelist held a sixth is the drift this prevents.
const maxRings = combat.MaxWornRings

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

// ringPaneBackColor is the surface the rings stand on: one step off `screenGround`, and
// nothing else.
//
// **A fill, not a frame** *(2026-08-12)*, and the distinction is the whole history of this
// pane. The first version was a full pink box — filled, bordered and titled — and it was the
// loudest thing in the band, competing with five saturated pink borders standing inside it. It
// came out the same hour, leaving the cards to be the pane. What that lost is the thing being
// put back now: with the row spanning most of the screen's width and a fighter card at either
// end, nothing said where the middle *began*.
//
// So it is the quietest possible answer to that: one step off the background, no border, no
// title, no hue. A colour that meant something would put it back in competition with the
// borders it sits behind.
//
// **The step goes down now that the ground is cream** *(2026-08-14)*. It was one step
// *lighter* than {50,50,50} for as long as the screen was dark, and kept that way it would be
// a near-white slab — the loudest thing in the band, which is the exact failure this pane was
// cut back to a fill to avoid. Which direction "one step" means is a function of the ground.
var ringPaneBackColor = color.RGBA{R: 207, G: 189, B: 156, A: 255}

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
// The fraction hanging off the bottom edge onto the bare ground would say the rule is the
// panel's floor and the number is loose underneath it, which is backwards — the count belongs
// to the row it counts.
func (s *CombatScene) ringPaneBackRect(gs *state.GlobalState) image.Rectangle {
	r := s.ringPaneRect(gs).Inset(-ringPaneBackPad)
	r.Max.Y = s.ringPaneRect(gs).Max.Y +
		ringRuleWidth + ringCountTopGap + ringCountSize + ringPaneBackPad
	return r
}

// ringSlotMaxGap is the most bare table ever left between two ring cards.
//
// **A row's pitch is capped and then the row is centred** *(2026-08-24)*. Without the cap the row
// spread to whatever pane it was handed, so a run wearing two rings put one against the duelist
// card and the other in the far corner of an empty screen — two rings reading as two unrelated
// things rather than as one build. The cap is what makes a growing row *grow*: rings sit at a
// fixed pitch and the row widens outwards from the middle as one is added, up to the point where
// five of them fill the pane and the pitch has to close up again.
//
// 22 is the gap five rings leave in the combat screen's pane, so the fullest row on the screen the
// pitch was originally derived from is drawn exactly where it always was.
const ringSlotMaxGap = 22

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
// **The spread is capped at ringSlotMaxGap and the row is centred in the pane by ringSlotAt**, so
// a pane wider than the rings in it leaves its slack at both ends rather than between the cards.
func ringSlotPitch(r image.Rectangle, worn int) int {
	if worn < 2 {
		return 0
	}
	pitch := (r.Dx() - cards.RingStyle.Width) / (worn - 1)
	if max := cards.RingStyle.Width + ringSlotMaxGap; pitch > max {
		return max
	}
	return pitch
}

// ringSlotRowWidth is how much of the pane the row actually occupies: every pitch but the last,
// plus the card that sits on the final one.
func ringSlotRowWidth(r image.Rectangle, worn int) int {
	if worn < 1 {
		return 0
	}
	return (worn-1)*ringSlotPitch(r, worn) + cards.RingStyle.Width
}

// ringSlotAt is where the i'th ring card's top-left corner sits. **Flush with the pane's top**,
// which is the top of the character block beside it — the two are aligned directly rather than
// each being inset inside a frame of its own.
//
// **Horizontally the row is centred on the pane**, which is only visible once the pitch is capped:
// a full row still starts where it always did, because there is no slack left to share out.
func ringSlotAt(r image.Rectangle, i, worn int) image.Point {
	left := r.Min.X + (r.Dx()-ringSlotRowWidth(r, worn))/2
	return image.Pt(left+i*ringSlotPitch(r, worn), r.Min.Y)
}

// wornRings is what the player is wearing, as records, in worn order.
//
// **The run is the authority and this is the lookup** *(2026-08-17)*. `session.Session` holds the
// worn keys — in worn order, which is a rule, since rings fire left to right — and `gs.Rings` holds
// the record each key names, for its art and its name. The screen decides nothing.
//
// **A worn key with no record is reported, not ignored.** It is the failure `ParseElement` refuses to
// fall back on: a ring that quietly does not draw looks exactly like a ring that was never bought.
func wornRings(gs *state.GlobalState) []data.RingData {
	if gs.Run == nil {
		return nil
	}

	out := make([]data.RingData, 0, maxRings)
	for _, key := range gs.Run.Worn() {
		if len(out) == maxRings {
			break
		}
		record, ok := gs.Rings[key]
		if !ok {
			log.Printf("the run is wearing %q, which is in no record", key)
			continue
		}
		out = append(out, record)
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

	// **Flat, where the deck panel and the fight log are bevelled** *(owner's call, 2026-08-24)*.
	// It was sunken for an afternoon, on the argument that the ring cards stand *in* it. What that
	// misses is what is standing there: five bevelled cards on a bevelled tray on a bevelled
	// screen is three depths in one corner, and the cards are the thing meant to be read. The two
	// panels that keep their bevel are overlays — they cover the game, so a lit edge is what says
	// they are in front of it. This backing covers nothing.
	vector.DrawFilledRect(screen,
		float32(back.Min.X), float32(back.Min.Y), float32(back.Dx()), float32(back.Dy()),
		ringPaneBackColor, false)

	worn := wornRings(gs)
	for i, ring := range worn {
		drawRingCard(gs, screen, ringSlotAt(r, i, len(worn)), ring, true)
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
	op.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, fmt.Sprintf("%d/%d", len(worn), maxRings),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: ringCountSize}, op)
}
