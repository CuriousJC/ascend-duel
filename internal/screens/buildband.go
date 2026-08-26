package screens

// **The band that says what you are: your duelist card and the rings on your fingers.**
//
// The combat screen has drawn this since the character block became a card — a duelist card in the
// top-left corner and the worn rings in a row beside it. **The reward screen wants the same thing**
// *(owner's call, 2026-08-22)*: the payout it narrates lands on the purse written on that card, and
// choosing a worm is a choice about a deck you can only judge against the build you are holding.
//
// **The shop draws it too** *(2026-08-22)*, and it is what took the shop's second ring row away:
// with the band up, a separate "worn" row was the same five rings drawn twice. The shop calls the
// two halves separately — `drawBuildCard`, then its own ring row over `buildRingRect` — because a
// ring there is a thing you can sell.
//
// **It is a free function over the run rather than a method on a scene**, which is what lets a
// second screen draw it. What it deliberately does *not* do is move: the combat screen's own band
// is still its own — it draws a live fighter, mid-fight life, banked AP and an opponent's card at
// the far end, none of which exist here. This is the between-fights view of the same thing.
//
// **Rings are laid out by the same functions the combat screen uses** — `ringSlotAt`, `wornRings` —
// so the row cannot drift between the two screens. The pane rectangle is the only thing computed
// here, because there is no enemy card to end it at.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// buildBandRightPct is where the ring row stops. **It mirrors duelistCardLeftPct**, exactly as the
// combat screen's enemy card does — there is no opponent on this screen, so the row simply runs to
// the far margin.
const buildBandRightPct = 99

// buildCardRect is where the duelist card sits: the same corner it occupies in a fight, so the
// player's card does not move between the duel and the screen that follows it.
func buildCardRect(gs *state.GlobalState) image.Rectangle {
	left, top := gs.PctX(duelistCardLeftPct), gs.PctY(topRowTopPct)
	return image.Rect(left, top, left+cards.DuelistStyle.Width, top+cards.DuelistStyle.Height)
}

// buildRingRect is the row's extent, taken off the duelist card beside it for the reason the
// combat screen's is: whichever card moves, the row follows.
func buildRingRect(gs *state.GlobalState) image.Rectangle {
	card := buildCardRect(gs)
	top := card.Min.Y + ringPaneTopDrop
	return image.Rect(card.Max.X+ringPaneGap, top,
		gs.PctX(buildBandRightPct), top+cards.RingStyle.Height)
}

// buildBandBottom is where the band ends, so a screen below it knows what it has left.
func buildBandBottom(gs *state.GlobalState) int {
	return buildCardRect(gs).Max.Y
}

// drawBuildBand puts the whole thing up: the duelist as they came out of the fight, and the rings
// they are wearing.
//
// **Life is the run's `LifeLeft`, not a full bar.** The fight is over and the card still says what
// it cost — a win on nine life reads as one, and it is also the figure the payout was a tenth of.
//
// The AP figure is the duelist's own budget with no bank on it, because a bank is a thing that
// exists inside a round.
func drawBuildBand(gs *state.GlobalState, screen *ebiten.Image, vitae int, drag *cardDrag) {
	drawBuildCard(gs, screen, vitae)
	drawBuildRings(gs, screen, drag)
}

// drawBuildCard is the duelist half of the band on its own.
//
// **It is split out for the shop** *(2026-08-22)*, which draws the ring row itself: a ring there is
// a thing you can sell, so it carries a price under it and it moves when the row re-centres. The
// row's *geometry* is still `buildRingRect` and `ringSlotAt`, so the two screens cannot disagree
// about where a finger is — only about what is drawn on it.
func drawBuildCard(gs *state.GlobalState, screen *ebiten.Image, vitae int) {
	if gs.Run == nil {
		return
	}

	if fighter := buildFighter(gs); fighter != nil {
		name := fighter.Name
		if name == "" {
			name = duelistName
		}
		spec := duelistSpec(fighter, name, vitae, gs.Run.LifeLeft(), fighter.ActionPoints())
		if img := cardImage(gs, spec, cards.DuelistStyle); img != nil {
			r := buildCardRect(gs)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
			screen.DrawImage(img, op)
		}
	}
}

// drawBuildRings is the row of worn rings: what is on your fingers, in firing order.
//
// **`drag` is the press in progress over it**, and it may be nil for a caller that does not let the
// row be reordered. Nothing passes nil today; the parameter is there so that a screen putting the
// band up to be *read* does not have to invent a controller nobody drives.
func drawBuildRings(gs *state.GlobalState, screen *ebiten.Image, drag *cardDrag) {
	if gs.Run == nil {
		return
	}

	row := buildRingRect(gs)
	worn := wornRings(gs)
	counters := runCounters(gs)
	for i, record := range worn {
		// The seat a dragged ring left stays empty; see the combat screen's row.
		if drag != nil && drag.dragging() && i == drag.origin() {
			continue
		}
		at := ringSlotAt(row, i, len(worn))
		drawRingCard(gs, screen, at, record, counters[record.RingRecord], true, false)
	}

	if drag != nil {
		drawDraggedRing(gs, screen, drag, counters)
	}
}

// buildRingRow is the band's row, addressed by the shared drag: the reward screen's and the shop's.
//
// **There is no live duelist on either screen**, so the run is the only copy of the row and
// moveWornRing is the whole move. The combat screen's is the one with more to do — see
// CombatScene.moveRing.
func buildRingRow(gs *state.GlobalState, click func(i int)) ringRow {
	return ringRow{
		rect:  buildRingRect(gs),
		worn:  len(wornRings(gs)),
		click: click,
		move:  func(from, to int) { moveWornRing(gs, from, to) },
	}
}

// hoverBuildRings explains whichever worn ring the cursor is resting on, and reports whether it
// found one.
//
// **The band draws rings on three screens and only one of them explained them** *(2026-08-22)*.
// The combat screen has `hoverRings`, the shop grew its own loop, and the reward screen had
// neither — so the row a player reads their build off went silent on exactly the screen where they
// are choosing what to do to that build. It is one function now, over the same geometry the row is
// drawn with, so a fourth screen putting the band up cannot forget again.
//
// **The tooltip says where a ring sits in the firing order**, which is what `ringTip` is for: worn
// order is firing order, and the row is the only place that fact is visible.
func hoverBuildRings(gs *state.GlobalState, at image.Point, tip *models.Tooltip) bool {
	if gs.Run == nil {
		return false
	}

	row := buildRingRect(gs)
	worn := gs.Run.Worn()
	for i, key := range worn {
		seat := ringSlotRect(row, i, len(worn))
		if !at.In(seat) {
			continue
		}
		record, ok := gs.Rings[key]
		if !ok {
			return false
		}
		title, lines := ringTip(record, i, len(worn))
		tip.Point(seat, title, lines)
		return true
	}
	return false
}

// ringSlotRect is one finger as a rectangle. `ringSlotAt` answers with the point a card is drawn
// from; **anything hit-testing the row needs the same seat as an area**, and deriving it twice is
// the drawn-here-clicked-there bug every other row in this game is shaped to avoid.
func ringSlotRect(row image.Rectangle, i, worn int) image.Rectangle {
	at := ringSlotAt(row, i, worn)
	return image.Rect(at.X, at.Y, at.X+cards.RingStyle.Width, at.Y+cards.RingStyle.Height)
}

// buildFighter is the player as a combatant, equipped with what they are wearing.
//
// **It is rebuilt rather than kept.** The fighter that fought is the combat screen's and dies with
// it; what survives is the run, and the run is enough to say what the player *is*. The one thing it
// cannot say is mid-fight life, which is why `LifeLeft` is stored.
func buildFighter(gs *state.GlobalState) *entities.Combatant {
	c := duelistFromRecord(gs, playerRecord)
	if c == nil {
		return nil
	}
	c.Duelist = gs.Run.Equip(c.Duelist)
	return c
}
