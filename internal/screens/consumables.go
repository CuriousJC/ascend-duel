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
// first time a third consumable arrives, so the cap is `session.MaxConsumables` — every kind the pane
// holds counted together, see internal/session/consumable.go — and a sack opened over a full pane
// holds its cards back until a carried one is sold or the sack is skipped.
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
	"strings"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// maxHeld is how many consumables can be carried at once, and it reads the run's rule rather than
// declaring a second two. A pane saying `held/2` while the run took a third is the drift that
// indirection prevents.
const maxHeld = session.MaxConsumables

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
// MaxConsumables on purpose so a fixture can plant a sack rather than buy one — see session.Start — and a
// pane pinned to the capacity drew two cards while the run carried fifty, which is the screen
// hiding the thing the fixture exists to show. The row tightens to fit them, exactly as the relic
// row does; the fraction on the corner still counts against the capacity, so an over-full sack
// reads as the `50/2` it is.
func consumableSeats(gs *state.GlobalState) int {
	if held := len(heldConsumables(gs)); held > heldSlots(gs) {
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
// It is two cards at the comfortable pitch, with relicSlotMaxGap between them — **not maxHeld of
// them**. A full sack of three packs inside it, overlapping, at the pitch relicSlotPitch works out;
// the cap moving does not widen the pane or take room from the relics.
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
	return consumablePaneCards*(cards.RelicStyle.Width+relicSlotMaxGap) - relicSlotMaxGap
}

// consumablePaneCards is how many cards the pane is wide enough to hold side by side.
const consumablePaneCards = 2

// topRowPanes divides the band between the duelist card and whatever ends the row into the two panes
// that stand in it: the consumables on the left, beside the duelist card, and the relics on the
// right.
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

	consumables = image.Rect(left, top, left+consumablePaneWidth(), bottom)
	relics = image.Rect(consumables.Max.X+topRowPaneGap, top, right, bottom)
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
	spendable func(session.Consumable) bool, skip func(int) bool, raise bool) {

	if skip == nil {
		skip = func(int) bool { return false }
	}

	if gs.Run == nil {
		return
	}

	back := consumablePaneBackRect(r)
	vector.FillRect(screen,
		float32(back.Min.X), float32(back.Min.Y), float32(back.Dx()), float32(back.Dy()),
		relicPaneBackColor, false)

	held := heldConsumables(gs)
	seats := consumableSeats(gs)
	raised := ui.RaisedSeat(gs, consumableRow{rect: r, held: len(held), seats: seats}, raise)
	for i := 0; i < seats; i++ {
		at := consumableSlotRect(r, i, seats)
		if i >= len(held) || i == raised || skip(i) {
			continue
		}
		// **A rune is lit exactly when clicking it would do something** *(2026-09-06)*, which
		// is what makes select-then-apply readable: the player never has to be told whether the
		// cards they have selected are the right ones, because the rune that wants them is the
		// one that is not dim. See consumableTarget.satisfiedBy.
		drawHeldCard(gs, screen, at.Min, i, held[i], canSpend(spendable, held[i]))
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
	// **And it keeps floating** *(owner's call)*, drawn exactly as the row draws it.
	if raised >= 0 && !skip(raised) {
		at := consumableSlotRect(r, raised, seats)
		drawHeldCard(gs, screen, at.Min, raised, held[raised], canSpend(spendable, held[raised]))
	}

	// **`held / cap`, the relic pane's figure in the relic pane's seat** — drawPaneCount, under the
	// backing's right edge. It names nothing: the pane is called consumables because of its shape, so
	// a new spendable standing here needs nothing changed.
	//
	// **The figure counts what weighs**, so a weightless copy is never what makes a pane read full.
	drawPaneCount(gs, screen, back, fmt.Sprintf("%d/%d", gs.Run.WeightedConsumables(), heldSlots(gs)))
}

// drawHeldCard puts one carried thing in its seat in the pane: floating like every card in the band,
// or — for a weightless copy — lifted, drifting and shimmering exactly as a weightless relic is.
// See ui.DrawWeightless.
func drawHeldCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point, seat int,
	c session.Consumable, enabled bool) {

	spec, st, ok := consumableFace(gs, c, enabled, false)
	if !ok {
		return
	}
	if c.Weightless {
		ui.DrawWeightless(screen, ui.CardImage(gs, spec, st), at, 1, float64(gs.Count), float64(seat)*2.1)
		return
	}
	ui.DrawFloatingCard(gs, screen, at, seat, spec, st)
}

// hoverConsumables explains whichever carried rune the cursor is resting on, and reports whether
// it found one.
//
// **Every screen that draws the pane gets it for free**, which is the lesson hoverBuildRelics records:
// the relic row was drawn on three screens and explained on one, and the row a player reads their
// build off went silent exactly where they were choosing what to do to it.
// **`essenceTargets` is how many cards a carried essence would take**, and it is the caller's
// because the answer depends on where the pane is standing: on the combat screen it is clamped to
// the hand the player is about to aim at, and on a between-fights screen there is no hand yet, so
// what the essence will do is all there is to say.
func hoverConsumables(gs *state.GlobalState, r image.Rectangle, at image.Point,
	tip *models.Tooltip, essenceTargets int) bool {

	held := heldConsumables(gs)
	seats := consumableSeats(gs)
	row := consumableRow{rect: r, held: len(held), seats: seats}

	// **The same seats and the same walk the raise uses** — see ui.HoveredSeat. A sack packs four
	// cards into two seats, so the card on top is the last drawn and a forward walk explains the one
	// behind it while the row lifts the one in front.
	i := ui.HoveredSeat(at, len(held), func(i int) image.Rectangle { return row.RowSlot(gs, i) })
	if i < 0 {
		return false
	}
	seat := row.RowSlot(gs, i)
	// **To the left of the card, never under it** *(owner's call)*: a panel under the top row
	// covers the pane's count and the tab a click hangs there. PointLeft flips right at the
	// screen's edge.
	lines := consumableTipLines(gs, held[i], essenceTargets)
	if held[i].Weightless {
		lines = append(lines, ui.WeightlessTipLine)
	}
	tip.PointLeft(seat, ui.TipLine(held[i].Name()), ui.TipLines(lines))
	return true
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
func canSpend(spendable func(session.Consumable) bool, c session.Consumable) bool {
	return spendable != nil && spendable(c)
}

// heldConsumables is everything the run is carrying, as the pane's row reads it.
func heldConsumables(gs *state.GlobalState) []session.Consumable {
	if gs.Run == nil {
		return nil
	}
	return gs.Run.Consumables()
}

// drawConsumableCard puts one carried thing in a seat, whichever kind it is.
//
// **This table and consumableTipLines are the whole of what a new consumable costs the screen** —
// a picture and a sentence. Everything else in the pane counts seats and knows nothing about what
// stands in them. A kind with no case draws nothing, which is the honest failure: a blank seat is
// a consumable nobody drew rather than a card lying about what it is.
func drawConsumableCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point,
	c session.Consumable, enabled, selected bool) {

	if spec, st, ok := consumableFace(gs, c, enabled, selected); ok {
		ui.BlitCard(gs, screen, at, spec, st)
	}
}

// consumableFace is a consumable's face and the style it is drawn at, for a caller that floats it
// rather than blitting it. False for a kind with no case.
func consumableFace(gs *state.GlobalState, c session.Consumable, enabled, selected bool) (cards.Spec, cards.Style, bool) {
	switch c.Kind {
	case session.ConsumableRune:
		return ui.RuneSpec(gs, c.Rune, enabled, selected), cards.EssenceStyle, true
	case session.ConsumableStone:
		// **A stone is never drawn selected**, because there is nothing to select it *for*: a rune
		// is aimed at cards and a stone names its own rung. The flag is taken anyway so every kind
		// answers one signature.
		return ui.StoneSpec(gs, c.Stone, enabled), cards.EssenceStyle, true
	case session.ConsumableEssence:
		return ui.EssenceSpec(gs, c.Essence, enabled), cards.EssenceStyle, true
	case session.ConsumableCantrip:
		return ui.CantripSpec(gs, c.Cantrip, enabled), cards.EssenceStyle, true
	}
	return cards.Spec{}, cards.Style{}, false
}

// consumableTipLines is what the pane says about one carried thing.
func consumableTipLines(gs *state.GlobalState, c session.Consumable, essenceTargets int) []string {
	switch c.Kind {
	case session.ConsumableStone:
		return stoneTipLines(gs, c.Stone)
	case session.ConsumableEssence:
		return essenceTipLines(c.Essence, essenceTargets)
	case session.ConsumableCantrip:
		return cantripTipLines(c.Cantrip)
	default:
		return runeTipLines(gs, c.Rune)
	}
}

// essenceTipLines is what the pane says about a carried essence: the line the catalog authored,
// and where it can be spent.
//
// **`ui.EssenceTip` is the one wording**, so a carried essence and an offered one say the same
// thing — a second sentence written here is how the pane comes to describe a mechanic the reward
// screen describes differently.
func essenceTipLines(w session.Essence, targets int) []string {
	_, lines := ui.EssenceTip(w, targets)
	return append(lines, "spent between the turns of a fight")
}

// cantripTipLines is what a cantrip says: its authored line, and when it can be cast.
//
// **The "when" carries the half the card cannot say** — that it lasts one fight. The line is the
// record's own and a break in it is the tooltip's own line, the treatment runeTipLines gives a rune.
func cantripTipLines(c session.Cantrip) []string {
	lines := strings.Split(c.Text, "\n")
	return append(lines, "cast between the turns of a fight,", "and it lasts until the fight ends")
}
