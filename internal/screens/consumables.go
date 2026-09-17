package screens

// The consumables pane: the runes a run is carrying, drawn beside the relics it is wearing.
//
// **The top row is two panes now** *(owner's call, 2026-09-06)*. It was one — the duelist card, a
// row of worn relics, and the opponent's card at the far end — and the relics pane has said `worn/5`
// on its corner since the count moved there. This puts a second pane on the same line saying
// `held/2`, because a rune is the other thing a run carries into a fight and the only place it
// was visible was behind the `P` button, two clicks into a dialog that only exists mid-duel.
//
// **A cap is what makes a pane possible.** A row drawn as `n/2` has to be a rule or it is a lie the
// first time a third rune arrives, so `session.MaxHeld` landed with this — see
// internal/session/rune.go, and the shop's sack seat, which goes dim rather than selling a
// rune there is no room for.
//
// **It is the relics pane's twin and shares everything it can**: the same backing color, the same
// eight pixels of padding, the same drop below the cards on either side, and the same count hung
// off the bottom-right corner. Two panes that were nearly alike would read as an inconsistency; two
// that are identical apart from their width read as one row divided.
//
// **The whole row packs at one pitch**, so a relic and a rune sit the same distance apart and
// close up together when the pane is tight. See relicSlotPitch, and consumablePaneWidth
// for why each pane packs on its own.

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

// maxHeld is how many runes can be carried at once, and it reads the run's rule rather than
// declaring a second two. A pane saying `held/2` while the sack took a third is the drift that
// indirection prevents.
const maxHeld = session.MaxHeld

// heldSlots is the sack's capacity — what the fraction on the pane's corner counts against, and
// the consumables counterpart of relicSlots.
//
// **A function rather than the constant at the call sites** *(2026-09-17)*, for the reason
// relicSlots is one: the moment anything grows a sack, the seats the row draws and the seats the
// run actually has must not be two numbers. It reads the constant today because nothing grows one
// yet.
func heldSlots(gs *state.GlobalState) int {
	_ = gs
	return maxHeld
}

// consumableSeats is how many seats the pane actually draws, which is the capacity **or whatever is
// being carried, if that is more** *(owner's call, 2026-09-17)*.
//
// **The sack can be over-full and the pane has to be able to say so.** `Session.hold` goes past
// MaxHeld on purpose so a fixture can plant a sack rather than buy one — see session.Start — and a
// pane pinned to the capacity drew two cards while the run carried fifty, which is the screen
// hiding the thing the fixture exists to show. The row tightens to fit them, exactly as the relic
// row does; the fraction on the corner still counts against the capacity, so an over-full sack
// reads as the `50/2` it is.
func consumableSeats(gs *state.GlobalState) int {
	if held := len(heldRunes(gs)); held > heldSlots(gs) {
		return held
	}
	return heldSlots(gs)
}

// topRowPaneGap is the bare ground between the two panes' backings.
//
// **Twice relicPaneGap**, so the gutter inside the row is visibly wider than the sixteen pixels
// separating the row from the cards at either end. Each pane's backing eats eight of it, so what is
// actually seen between them is sixteen — the same air the row keeps from the duelist card, which is
// what stops the split reading as one pane with a scratch down it.
const topRowPaneGap = 2 * relicPaneGap

// consumablePaneWidth is how wide the consumables pane is, and it is **a fixed size nothing else in
// the row can move** *(owner's call, 2026-09-17)*.
//
// It is the sack's own seats at the comfortable pitch — two cards with relicSlotMaxGap between them
// — so the pane is exactly as big as a full sack drawn properly, and never any bigger.
//
// **This replaced a pitch solved across both panes at once** *(reversing the 2026-09-06 call)*. That
// version divided the whole span so both panes filled together, which gave the row one rhythm and
// one fatal property: the number of seats in either pane set the spacing in *both*. Eight relics
// squeezed the two runes; fifty runes squeezed the relics into a corner of slivers and handed most
// of the table to a pane of things the player is carrying rather than wearing. Locking this pane and
// giving the relics everything else is what makes each pane answerable for its own contents.
//
// **What is given up is the shared rhythm**, which was a real thing: a relic and a rune now sit at
// whatever spacing their own pane works out, so the two can differ. That is the accepted cost of
// each pane keeping its own size.
func consumablePaneWidth() int {
	return (maxHeld-1)*(cards.RelicStyle.Width+relicSlotMaxGap) + cards.RelicStyle.Width
}

// topRowPanes divides the band between the duelist card and whatever ends the row into the two panes
// that stand in it: the relics on the left, the consumables on the right.
//
// **One function for both screens and both panes**, which is the rule every row in this game is
// under: the combat screen and the build band ask the same question of different spans, and a second
// copy of this arithmetic is how the two come to disagree about where a pane ends.
//
// **The consumables pane is a fixed width and the relics take everything that is left.** Neither
// pane's size depends on how much is in either — what a pane holds decides how tightly that pane
// packs, and nothing else. See consumablePaneWidth.
func topRowPanes(left, right, top int) (relics, consumables image.Rectangle) {
	bottom := top + cards.RelicStyle.Height

	consumables = image.Rect(right-consumablePaneWidth(), top, right, bottom)
	relics = image.Rect(left, top, consumables.Min.X-topRowPaneGap, bottom)
	return relics, consumables
}

// consumableSlotAt is where the i'th carried rune's card sits.
//
// **The row's own pitch, over the seats it has rather than the cards in it** — relicSlotPitch, the
// function the relics use, asked for maxHeld every time. That is what makes the two panes share a
// rhythm: the pane was sized from the same pitch, so a full row lands exactly on its edges.
//
// **Left-aligned and never re-centered**, deliberately unlike the relic row. This row is two fixed
// seats with the empty one drawn, so a card that shifted as the sack filled would move the one
// thing the player is being shown. The relics center because their seats appear and disappear.
func consumableSlotAt(r image.Rectangle, i, seats int) image.Point {
	return image.Pt(r.Min.X+i*relicSlotPitch(r, seats), r.Min.Y)
}

// consumableSlotRect is one seat as a rectangle, for anything hit-testing the row. Same shape as
// relicSlotRect, and for the same drawn-here-clicked-there reason.
func consumableSlotRect(r image.Rectangle, i, seats int) image.Rectangle {
	at := consumableSlotAt(r, i, seats)
	return image.Rect(at.X, at.Y, at.X+cards.RelicStyle.Width, at.Y+cards.RelicStyle.Height)
}

// consumablePaneBackRect is the surface the cards stand on: the row padded, exactly as the relic
// pane's backing is derived from the relic row.
func consumablePaneBackRect(r image.Rectangle) image.Rectangle {
	return r.Inset(-relicPaneBackPad)
}

// drawConsumablePane puts the pane up: the backing, whatever is held, an empty seat for whatever is
// not, and the count on the corner.
//
// **An empty seat draws nothing at all** *(owner's call, 2026-09-07)*. It was outlined for a day
// on the argument that two bare seats would read as something failing to draw — and on screen the
// outline was the loudest thing in the top row, two heavy black rectangles beside the relic pane
// saying only that the run is carrying nothing. **The pane's own surface is the hole.** The count
// on the corner is what says how much room is left. Nothing anywhere outlines an absent card now —
// the reward screen was the last place doing it, and gave it up on 2026-09-08.
func drawConsumablePane(gs *state.GlobalState, screen *ebiten.Image, r image.Rectangle,
	spendable func(session.Rune) bool, skip func(int) bool, raise bool) {

	if skip == nil {
		skip = func(int) bool { return false }
	}

	if gs.Run == nil {
		return
	}

	back := consumablePaneBackRect(r)
	vector.DrawFilledRect(screen,
		float32(back.Min.X), float32(back.Min.Y), float32(back.Dx()), float32(back.Dy()),
		relicPaneBackColor, false)

	held := heldRunes(gs)
	seats := consumableSeats(gs)
	raised := raisedSeat(gs, runeRow{rect: r, held: len(held), seats: seats}, raise)
	for i := 0; i < seats; i++ {
		at := consumableSlotRect(r, i, seats)
		if i >= len(held) || i == raised || skip(i) {
			continue
		}
		// **A rune is lit exactly when clicking it would do something** *(2026-09-06)*, which
		// is what makes select-then-apply readable: the player never has to be told whether the
		// cards they have selected are the right ones, because the rune that wants them is the
		// one that is not dim. See consumableTarget.satisfiedBy.
		drawRuneCard(gs, screen, at.Min, held[i], canSpend(spendable, held[i]), false)
	}

	// **The card under the cursor is drawn last, so it is drawn whole** *(owner's call,
	// 2026-09-17)*. A sack packs its seats into whatever width it has, and past a handful the cards
	// are slivers of each other — so the one being pointed at is unreadable exactly when the player
	// is asking what it is. Raising it is the picture half of the tooltip: the type says what the
	// rune does and this says which card that is.
	//
	// **Raised rather than moved.** It stays in its seat and simply stops being covered, so the row
	// does not rearrange itself under a cursor that is about to click.
	//
	// **It lands on the tooltip's own tick** *(owner's call, 2026-09-17)*, which the caller asks for
	// — see models.Tooltip.Showing. Raising on arrival made the row flinch at a cursor crossing it,
	// and raising halfway split one gesture into two answers a beat apart.
	if raised >= 0 && !skip(raised) {
		at := consumableSlotRect(r, raised, seats)
		drawRuneCard(gs, screen, at.Min, held[raised], canSpend(spendable, held[raised]), false)
	}

	drawConsumableCount(gs, screen, back, len(held))
}

// drawConsumableCount writes `held / cap` on the pane's bottom-right corner — the relic pane's
// figure, in the relic pane's seat, at the relic pane's size.
//
// **It is the bare fraction and names nothing** *(owner's call, 2026-09-11)*. It read
// `0/2 runes` until then, and the noun was the figure repeating what the cards standing on the
// pane already say. The relic pane's corner lost its word in the same call and the two have to stay
// twins — see drawRelicCount.
//
// The pane is called consumables because that is the shape of the thing — a seat for something a
// run spends — so a stone or another spendable standing here needs nothing changed here at all,
// which is what the dropped noun buys.
func drawConsumableCount(gs *state.GlobalState, screen *ebiten.Image, back image.Rectangle, held int) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(back.Max.X), float64(back.Max.Y+relicCountTopGap))
	op.PrimaryAlign = text.AlignEnd
	op.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, fmt.Sprintf("%d/%d", held, heldSlots(gs)),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: relicCountSize}, op)
}

// hoverConsumables explains whichever carried rune the cursor is resting on, and reports whether
// it found one.
//
// **Every screen that draws the pane gets it for free**, which is the lesson hoverBuildRelics records:
// the relic row was drawn on three screens and explained on one, and the row a player reads their
// build off went silent exactly where they were choosing what to do to it.
func hoverConsumables(gs *state.GlobalState, r image.Rectangle, at image.Point,
	tip *models.Tooltip) bool {

	seats := consumableSeats(gs)
	for i, p := range heldRunes(gs) {
		seat := consumableSlotRect(r, i, seats)
		if !at.In(seat) {
			continue
		}
		tip.Point(seat, tipLine(p.Name), tipLines(runeTipLines(gs, p)))
		return true
	}
	return false
}

// canSpend asks the caller's predicate, treating a nil one as "nothing here can be spent".
//
// **The predicate is a parameter rather than a package hook**, because the pane is drawn on four
// screens and only one of them can spend anything. The shop and the reward screen draw the same two
// seats with the same cards and no gesture at all — a rune is carried there, not used — so they
// pass nil and every card draws dim.
//
// **Dim is the honest state on those screens.** A lit card that did nothing when clicked would be
// worse than a dim one, and the tooltip still explains it wherever it is drawn.
func canSpend(spendable func(session.Rune) bool, p session.Rune) bool {
	return spendable != nil && spendable(p)
}

// consumableClicked reports which seat of the pane the cursor is over, or -1.
//
// **It answers for a seat rather than for a card**, so a click on an empty seat is a click on
// nothing rather than on whatever happens to be held at that index.
func consumableClicked(gs *state.GlobalState, r image.Rectangle, at image.Point) int {
	seats := consumableSeats(gs)
	for i := len(heldRunes(gs)) - 1; i >= 0; i-- {
		if at.In(consumableSlotRect(r, i, seats)) {
			return i
		}
	}
	return -1
}
