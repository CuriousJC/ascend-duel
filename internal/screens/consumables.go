package screens

// The consumables pane: the parasites a run is carrying, drawn beside the rings it is wearing.
//
// **The top row is two panes now** *(owner's call, 2026-09-06)*. It was one — the duelist card, a
// row of worn rings, and the opponent's card at the far end — and the rings pane has said `worn/5`
// on its corner since the count moved there. This puts a second pane on the same line saying
// `held/2`, because a parasite is the other thing a run carries into a fight and the only place it
// was visible was behind the `P` button, two clicks into a dialog that only exists mid-duel.
//
// **A cap is what makes a pane possible.** A row drawn as `n/2` has to be a rule or it is a lie the
// first time a third parasite arrives, so `session.MaxHeld` landed with this — see
// internal/session/parasite.go, and the shop's bucket seat, which goes dim rather than selling a
// parasite there is no room for.
//
// **It is the rings pane's twin and shares everything it can**: the same backing colour, the same
// eight pixels of padding, the same drop below the cards on either side, and the same count hung
// off the bottom-right corner. Two panes that were nearly alike would read as an inconsistency; two
// that are identical apart from their width read as one row divided.
//
// **The whole row packs at one pitch**, so a ring and a parasite sit the same distance apart and
// close up together when the span is tight. See topRowPitch.

import (
	"fmt"
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// maxHeld is how many parasites can be carried at once, and it reads the run's rule rather than
// declaring a second two — exactly as maxRings reads combat.MaxWornRings. A pane saying `held/2`
// while the bucket took a third is the drift that indirection prevents.
const maxHeld = session.MaxHeld

// topRowPaneGap is the bare ground between the two panes' backings.
//
// **Twice ringPaneGap**, so the gutter inside the row is visibly wider than the sixteen pixels
// separating the row from the cards at either end. Each pane's backing eats eight of it, so what is
// actually seen between them is sixteen — the same air the row keeps from the duelist card, which is
// what stops the split reading as one pane with a scratch down it.
const topRowPaneGap = 2 * ringPaneGap

// topRowPitch is the one pitch the whole top row packs at: the rings and the consumables alike.
//
// **One rhythm across both panes** *(owner's call, 2026-09-06)*. The first split gave the
// consumables a full pitch and let the rings close up to pay for it, which read as two rows at two
// spacings rather than one row divided. This solves for the pitch that makes both panes full at the
// same time, so a ring and a parasite sit the same distance apart on the same line.
//
// **The arithmetic.** The span holds five ring seats and two consumable seats, which is five pitches
// and two whole cards, plus the gutter between the panes:
//
//	span = (maxRings-1)*pitch + width + gap + (maxHeld-1)*pitch + width
//
// Solved for pitch, and then capped at the row's own maximum so a wide span spreads to a comfortable
// gap rather than to a sparse one — the same cap ringSlotPitch is under, and for the same reason: a
// row that spread to whatever it was given stopped reading as one build.
//
// **Overlap is the expected result on the combat screen and it is fine** *(owner's call)*. Between
// the two fighter cards the span is 1443, which gives a pitch of 201 against a 203-pixel card — the
// two-pixel overlap the row closes up by rather than shrinking a card that cannot be shrunk. On the
// shop and the reward screen there is no opponent card, the span is 1662, and the cap bites first,
// so nothing overlaps at all.
func topRowPitch(span int) int {
	w := cards.RingStyle.Width
	steps := (maxRings - 1) + (maxHeld - 1)

	pitch := (span - 2*w - topRowPaneGap) / steps
	if max := w + ringSlotMaxGap; pitch > max {
		return max
	}
	return pitch
}

// topRowPanes divides the band between the duelist card and whatever ends the row into the two panes
// that stand in it: the rings on the left, the consumables on the right.
//
// **One function for both screens and both panes**, which is the rule every row in this game is
// under: the combat screen and the build band ask the same question of different spans, and a second
// copy of this arithmetic is how the two come to disagree about where a pane ends.
//
// **Both panes are sized from the shared pitch**, so neither is the one that absorbs the slack. The
// consumables pane is two seats whether or not anything is in them — a promise the count on its
// corner is making — and the rings pane is five, and what is left over after both sits outside the
// row rather than inside either.
func topRowPanes(left, right, top int) (rings, consumables image.Rectangle) {
	bottom := top + cards.RingStyle.Height

	pitch := topRowPitch(right - left)
	held := (maxHeld-1)*pitch + cards.RingStyle.Width

	consumables = image.Rect(right-held, top, right, bottom)
	rings = image.Rect(left, top, consumables.Min.X-topRowPaneGap, bottom)
	return rings, consumables
}

// consumableSlotAt is where the i'th carried parasite's card sits.
//
// **The row's own pitch, over the seats it has rather than the cards in it** — ringSlotPitch, the
// function the rings use, asked for maxHeld every time. That is what makes the two panes share a
// rhythm: the pane was sized from the same pitch, so a full row lands exactly on its edges.
//
// **Left-aligned and never re-centred**, deliberately unlike the ring row. This row is two fixed
// seats with the empty one drawn, so a card that shifted as the bucket filled would move the one
// thing the player is being shown. The rings centre because their seats appear and disappear.
func consumableSlotAt(r image.Rectangle, i int) image.Point {
	return image.Pt(r.Min.X+i*ringSlotPitch(r, maxHeld), r.Min.Y)
}

// consumableSlotRect is one seat as a rectangle, for anything hit-testing the row. Same shape as
// ringSlotRect, and for the same drawn-here-clicked-there reason.
func consumableSlotRect(r image.Rectangle, i int) image.Rectangle {
	at := consumableSlotAt(r, i)
	return image.Rect(at.X, at.Y, at.X+cards.RingStyle.Width, at.Y+cards.RingStyle.Height)
}

// consumablePaneBackRect is the surface the cards stand on: the row padded, exactly as the ring
// pane's backing is derived from the ring row.
func consumablePaneBackRect(r image.Rectangle) image.Rectangle {
	return r.Inset(-ringPaneBackPad)
}

// drawConsumablePane puts the pane up: the backing, whatever is held, an empty seat for whatever is
// not, and the count on the corner.
//
// **Empty seats are drawn here where the ring row does not draw them**, and the difference is what
// the two rows are saying. Five ring seats mostly stand empty and drawing them would spend the
// loudest thing in the band on what you have not got — the fraction says it more quietly. Two seats
// is small enough that the pane would otherwise be a bare rectangle with nothing in it at all, which
// reads as something failing to draw rather than as room you have.
func drawConsumablePane(gs *state.GlobalState, screen *ebiten.Image, r image.Rectangle,
	spendable func(session.Parasite) bool) {

	if gs.Run == nil {
		return
	}

	back := consumablePaneBackRect(r)
	vector.DrawFilledRect(screen,
		float32(back.Min.X), float32(back.Min.Y), float32(back.Dx()), float32(back.Dy()),
		ringPaneBackColor, false)

	held := heldParasites(gs)
	for i := 0; i < maxHeld; i++ {
		at := consumableSlotRect(r, i)
		if i >= len(held) {
			drawEmptySeat(screen, at)
			continue
		}
		// **A parasite is lit exactly when clicking it would do something** *(2026-09-06)*, which
		// is what makes select-then-apply readable: the player never has to be told whether the
		// cards they have selected are the right ones, because the parasite that wants them is the
		// one that is not dim. See consumableTarget.satisfiedBy.
		drawSpecCard(gs, screen, at.Min, parasiteSpec(gs, held[i], canSpend(spendable, held[i]), false))
	}

	drawConsumableCount(gs, screen, back, len(held))
}

// drawConsumableCount writes `held / cap` on the pane's bottom-right corner — the ring pane's
// figure, in the ring pane's seat, at the ring pane's size.
//
// **It names parasites and not consumables**, which is what is actually in it. The pane is called
// consumables because that is the shape of the thing — a seat for something a run spends — and if a
// stone or another spendable ever stands here the label is what changes.
func drawConsumableCount(gs *state.GlobalState, screen *ebiten.Image, back image.Rectangle, held int) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(back.Max.X), float64(back.Max.Y+ringCountTopGap))
	op.PrimaryAlign = text.AlignEnd
	op.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, fmt.Sprintf("%d/%d parasites", held, maxHeld),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: ringCountSize}, op)
}

// hoverConsumables explains whichever carried parasite the cursor is resting on, and reports whether
// it found one.
//
// **Every screen that draws the pane gets it for free**, which is the lesson hoverBuildRings records:
// the ring row was drawn on three screens and explained on one, and the row a player reads their
// build off went silent exactly where they were choosing what to do to it.
func hoverConsumables(gs *state.GlobalState, r image.Rectangle, at image.Point,
	tip *models.Tooltip) bool {

	for i, p := range heldParasites(gs) {
		seat := consumableSlotRect(r, i)
		if !at.In(seat) {
			continue
		}
		tip.Point(seat, p.Name, parasiteTipLines(p))
		return true
	}
	return false
}

// canSpend asks the caller's predicate, treating a nil one as "nothing here can be spent".
//
// **The predicate is a parameter rather than a package hook**, because the pane is drawn on four
// screens and only one of them can spend anything. The shop and the reward screen draw the same two
// seats with the same cards and no gesture at all — a parasite is carried there, not used — so they
// pass nil and every card draws dim.
//
// **Dim is the honest state on those screens.** A lit card that did nothing when clicked would be
// worse than a dim one, and the tooltip still explains it wherever it is drawn.
func canSpend(spendable func(session.Parasite) bool, p session.Parasite) bool {
	return spendable != nil && spendable(p)
}

// consumableClicked reports which seat of the pane the cursor is over, or -1.
//
// **It answers for a seat rather than for a card**, so a click on an empty seat is a click on
// nothing rather than on whatever happens to be held at that index.
func consumableClicked(gs *state.GlobalState, r image.Rectangle, at image.Point) int {
	for i := range heldParasites(gs) {
		if at.In(consumableSlotRect(r, i)) {
			return i
		}
	}
	return -1
}
