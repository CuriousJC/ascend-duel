package screens

// Spending one essence on several cards: the preview, the clock they all move on, and the row they
// land in.
//
// **An essence takes one card and a relic may make that two, or four** — see
// `combat.MomentEssenceSpent` and essence_targets.go. What that costs the screens is that every
// picture of an alteration is now a *row* of alterations: the reward screen and the shop's vial
// each had one before-card, one after-card and one morph, and each would have grown its own second
// copy of the arithmetic. This is that arithmetic, once.
//
// **The preview runs the real essence against a throwaway copy of the run**, which is the rule both
// screens were already under: a preview computed by its own arithmetic is a preview that can
// disagree with the thing it is previewing. What is new is the walk — **the deck is altered from
// the back** — because a removal shifts every position above it and leaves everything below it
// alone, so a descending walk is what makes several targets in one spend safe. `Session.ApplyToAll`
// walks the real deck the same way, and the two have to agree or the picture is of a deck the run
// never reached.

import (
	"image"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// essenceLanding is one card an essence came down on: where it set off from, what it was, what it
// became, and the change happening between the two faces.
//
// **What the essence *did* is not here**, because it is the same for every landing of one spend: an
// essence that eats eats all of them. The scene carries that as its own flag and this carries the
// card.
type essenceLanding struct {
	// from is the seat this card was drawn in when it was picked, and the seat it flies out of.
	// **The card that moves is the card the player was looking at.**
	from image.Rectangle

	before combat.Card
	after  combat.Card

	// change is the alteration happening: the old face coming apart and the new one coming through
	// it. Which of the three shapes it takes was decided when the spend was previewed.
	change ui.Morph
}

// previewEssence works out what one essence would do to every one of these deck positions, without
// touching the run.
//
// `at` is the deck positions, in the order the player is looking at them, and `from` is the seat
// each of them is drawn in — one per position, indexed alike. It reports false if any one of the
// cards cannot take the essence, which is the same question the lit state asked: a spend that is
// half legal is refused whole.
func previewEssence(gs *state.GlobalState, w session.Essence, at []int,
	from []image.Rectangle) ([]essenceLanding, bool) {

	if gs.Run == nil || len(at) == 0 || len(at) != len(from) {
		return nil, false
	}

	removes := w.Target == session.TargetRemove
	copied := w.Target == session.TargetDuplicate

	lands := make([]essenceLanding, len(at))
	for k, di := range at {
		before, ok := gs.Run.Card(di)
		if !ok || !gs.Run.CanApply(w, di) {
			return nil, false
		}
		lands[k] = essenceLanding{from: from[k], before: before}
	}

	// **Applied from the back of the deck forwards**, so a removal never moves a position still to
	// be aimed at. The order the player reads is the order of `at`; this is only the order the
	// trial deck is edited in.
	order := make([]int, len(at))
	for k := range order {
		order[k] = k
	}
	sort.Slice(order, func(i, j int) bool { return at[order[i]] > at[order[j]] })

	trial := session.New(gs.Run.Deck())
	for _, k := range order {
		di := at[k]
		if !trial.Apply(w, di) {
			return nil, false
		}
		switch {
		case copied:
			// The copy is appended, so the card that arrived is the last one — and it is the
			// *new* card that is the reward, even though it is identical to the one picked.
			lands[k].after, _ = trial.Card(trial.Size() - 1)
		case !removes:
			lands[k].after, _ = trial.Card(di)
		}
	}

	// **What the essence did decides which shape the change takes**, and the three cases are the
	// three things an essence can be: it recolored the card, it ate it, or it made a second one.
	// See cardmorph.go — the morph is handed two finished faces and works out the rest.
	for k := range lands {
		beforeSpec := ui.CardSpec(lands[k].before, ui.HeldByRun(gs, lands[k].before), true, false)
		afterSpec := ui.CardSpec(lands[k].after, ui.HeldByRun(gs, lands[k].after), true, false)
		switch {
		case removes:
			lands[k].change = ui.MorphAway(beforeSpec, cards.Hand)
		case copied:
			// **The original is not changed, so it does not morph.** It stands where it landed and
			// the copy arrives out of nothing beside it, which is the only honest picture of a
			// duplicate: there is no old face to come apart.
			lands[k].change = ui.MorphIn(afterSpec, cards.Hand)
		default:
			lands[k].change = ui.MorphInto(beforeSpec, afterSpec, cards.Hand)
		}
	}
	return lands, true
}

// tickLandings steps every change by one frame and reports whether they have all finished.
//
// **One beat for all of them** *(owner's call, the shield break's rule and the hand morph's)*. Four
// cards changing one after another is four pauses over a row the player is waiting to read, and
// what the beat says is one thing about the essence rather than one thing about each card.
func tickLandings(lands []essenceLanding) bool {
	done := true
	for i := range lands {
		if !lands[i].change.Done() {
			lands[i].change.Tick()
			done = false
		}
	}
	return done
}

// landingSeats is how many settled seats a row of landings takes: one each, or two each while a
// copy is being made — the original and the copy standing beside it.
func landingSeats(lands []essenceLanding, copied bool) int {
	if copied {
		return 2 * len(lands)
	}
	return len(lands)
}

// drawLandings puts the settled row up: every picked card flying to its own seat in the middle,
// changing there, and held while the player reads what it became.
//
// **The flight carries the old face and the morph carries the new one**, which is why nothing here
// asks what the essence did beyond the one flag it is handed: a removal ends on an empty seat, a
// duplicate ends on two cards, everything else ends on one.
//
// **An eaten card leaves nothing.** The morph is what draws the absence, by having no second face
// to hand over to — a seat outlined over the space a dissolve has just cleared puts the card's
// silhouette back on the table the frame after eating it.
func drawLandings(gs *state.GlobalState, screen *ebiten.Image, lands []essenceLanding,
	arrival ui.Travel, copied bool) {

	seats := settledSeats(gs, landingSeats(lands, copied))

	for i, l := range lands {
		seat := i
		if copied {
			seat = 2 * i
		}
		if seat >= len(seats) {
			return
		}

		at := ui.FlyingTo(l.from, seats[seat], arrival)
		if !copied {
			ui.DrawMorph(gs, screen, at, l.change)
			continue
		}

		// The card that was picked, untouched: while a copy is being made it is drawn plainly, and
		// the morph in the seat beside it is the whole of what is happening.
		ui.DrawCard(gs, screen, at, cards.Hand, l.before, ui.HeldByRun(gs, l.before), true, false)
		ui.DrawMorph(gs, screen, seats[seat+1].Min, l.change)
	}
}
