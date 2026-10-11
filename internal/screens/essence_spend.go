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

	"github.com/curiousjc/ascend-duel/internal/achieve"
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
	// from is the seat this card was drawn in when it was picked, and the seat it changes in.
	// **The card that changes is the card the player was looking at, where they were looking at it.**
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

// essenceMotes is what an essence's change travels as: the new element's own sparks for an essence
// that recolors, and the rune motes for everything else, which changes no color.
func essenceMotes(w session.Essence) ui.Emitter {
	if w.Target == session.TargetElement {
		return ui.Emitter{Element: ui.ArtFor(w.Element), Set: ui.ParticleSparks, Rate: 1.2, Size: 22}
	}
	return ui.Emitter{Element: cards.Arcane, Rate: 1.2, Size: 22,
		Sprites: ui.FirstDrawn(ui.RuneMoteSprites, []string{"spark-lightning-1", "spark-lightning-2"})}
}

// essenceInPlace is an essence used where it stands — `seat`, its own card's seat — sending its
// change to every card it was aimed at, which changes where it stands too. **Nothing flies to the
// middle**: the essence comes apart in light and the deck's cards change in their own row.
//
// A copy has no old face to change, so it arrives out of nothing just above the card it copies.
func essenceInPlace(gs *state.GlobalState, w session.Essence, seat image.Rectangle,
	lands []essenceLanding, copied bool) ui.Consume {

	to := make([]image.Rectangle, len(lands))
	changes := make([]ui.Morph, len(lands))
	for i, l := range lands {
		r := image.Rect(l.from.Min.X, l.from.Min.Y, l.from.Min.X+cardWidth, l.from.Min.Y+cardHeight)
		if copied {
			r = r.Add(image.Pt(cardWidth/3, -cardHeight/3))
		}
		to[i], changes[i] = r, l.change
	}
	return ui.NewConsume(ui.Consume{
		Exit:    ui.ExitAbsorb,
		Face:    ui.EssenceSpec(gs, w, true),
		Style:   cards.EssenceStyle,
		From:    seat,
		Stage:   seat,
		To:      to,
		Changes: changes,
		Motes:   essenceMotes(w),
		Seed:    cards.MarkSeed(w.Name),
	})
}

// earnLadderWraps raises the ladder-wrapped moment for every landing that walked a card off one end
// of its ladder and onto the other. See achieve.LadderWrapped.
func earnLadderWraps(gs *state.GlobalState, lands []essenceLanding) {
	for _, l := range lands {
		if m, ok := achieve.LadderWrapped(l.before, l.after); ok {
			earnMoment(gs, m)
		}
	}
}
