package screens

// The deck panel's two toggles: what each one changes about the picture, and what neither of them
// is allowed to change.
//
// **These need no window.** The grid is a layout function and the tallies count a laid-out grid,
// which is the whole reason both were pulled out of the drawing in the first place — see
// pileGridLayout and tallyOf.

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// panelRun is a run wearing the named rings, with a deck small enough to reason about by hand.
func panelRun(t *testing.T, deck []combat.Card, rings ...string) *session.Session {
	t.Helper()

	run := session.New(deck)
	for _, key := range rings {
		if !run.Wear(key) {
			t.Fatalf("could not wear %s", key)
		}
	}
	return run
}

// panelDeck is two lightning cards and a fire one — enough for a flip ring to have something to
// take and something to leave alone.
func panelDeck() []combat.Card {
	return []combat.Card{
		{Concept: combat.Strike, Element: combat.Lightning},
		{Concept: combat.Strike, Element: combat.Lightning},
		{Concept: combat.Strike, Element: combat.Fire},
	}
}

// laidOut is the grid a view produces, at the panel's real footprint.
func laidOut(d deckContents, v deckView) pileGridLayout {
	return d.grid(v, 640, 1177, 150)
}

func TestAlterationsAreOnByDefault(t *testing.T) {
	// **The default is the deck you will be dealt, not the deck you own** *(owner's call,
	// 2026-08-24)*. A run wearing a flip ring never draws a lightning card, so a panel opening on a
	// list of lightning cards is showing a deck that does not exist for the length of that run.
	//
	// The zero deckView is what every caller starts from, so this pins the field's sense as well as
	// the value: `unaltered` inverts, rather than `altered` having to be switched on.
	var v deckView
	if v.unaltered {
		t.Error("the panel opens showing the cards as owned; alterations are the default")
	}
	if v.played {
		t.Error("the panel opens on the whole deck")
	}

	run := panelRun(t, panelDeck(), "frozen-lightning-ring")
	d := deckContents{draw: run.Deck(), run: run}

	ice := 0
	for _, s := range laidOut(d, v).slots {
		if s.card.Element == combat.Ice {
			ice++
		}
	}
	if ice != 2 {
		t.Errorf("the default view shows %d ice cards, want 2 — Frozen Lightning is not being applied", ice)
	}
}

func TestTheAlterationsToggleShowsBothFacesOfOneCard(t *testing.T) {
	// **A card in the discard has been through a draw and holds only what it became.** Showing it
	// as the run owns it is a lookup by ID and nothing else — no inversion of the flip, which could
	// not be done anyway once two rings converge on one colour.
	run := panelRun(t, panelDeck(), "frozen-lightning-ring")

	owned := run.Deck()
	// The first card, drawn: this is what the combat screen puts in the hand.
	drawn := run.DrawnAs(owned[0])
	if drawn.Element != combat.Ice {
		t.Fatalf("a lightning card was drawn as %v, want ice", drawn.Element)
	}

	d := deckContents{
		draw:    owned[1:],
		spent:   []combat.Card{drawn},
		run:     run,
		inFight: true,
	}

	for _, tc := range []struct {
		name      string
		unaltered bool
		want      combat.Element
	}{
		{"as dealt", false, combat.Ice},
		{"as owned", true, combat.Lightning},
	} {
		found := false
		for _, s := range laidOut(d, deckView{unaltered: tc.unaltered}).slots {
			if s.available {
				continue
			}
			found = true
			if s.card.Element != tc.want {
				t.Errorf("%s: the spent card is drawn as %v, want %v",
					tc.name, s.card.Element, tc.want)
			}
		}
		if !found {
			t.Errorf("%s: the spent card is not on the panel at all", tc.name)
		}
	}
}

func TestFullAndPlayedInvertTheLitHalfAndMoveNothing(t *testing.T) {
	// **The panel's governing idea, applied to the new toggle**: a card does not move, it only
	// dims. FULL lights what is still to draw and PLAYED lights what has been drawn — the same
	// grid, the same order, the other half lit.
	run := panelRun(t, panelDeck())
	owned := run.Deck()

	d := deckContents{
		draw:    owned[:2],
		spent:   owned[2:],
		run:     run,
		inFight: true,
	}

	full := laidOut(d, deckView{})
	played := laidOut(d, deckView{played: true})

	if len(full.slots) != len(played.slots) {
		t.Fatalf("FULL lays out %d cards and PLAYED %d — the toggle is dropping cards",
			len(full.slots), len(played.slots))
	}

	litFull, litPlayed := 0, 0
	for i := range full.slots {
		if full.slots[i].at != played.slots[i].at {
			t.Fatalf("card %d sits at %v under FULL and %v under PLAYED — the grid moved",
				i, full.slots[i].at, played.slots[i].at)
		}
		if full.slots[i].card != played.slots[i].card {
			t.Fatalf("position %d holds a different card under each toggle", i)
		}
		if full.slots[i].lit == played.slots[i].lit {
			t.Errorf("card %d is lit the same way under both toggles; the halves should invert", i)
		}
		if full.slots[i].lit {
			litFull++
		}
		if played.slots[i].lit {
			litPlayed++
		}
	}

	if litFull != 2 || litPlayed != 1 {
		t.Errorf("FULL lights %d and PLAYED lights %d, want 2 and 1", litFull, litPlayed)
	}
}

func TestNothingIsEverPlayedBetweenFights(t *testing.T) {
	// The button is not drawn on a screen with one pile, and the grid must not act on it either —
	// a state no control can reach is a state that quietly goes wrong.
	run := panelRun(t, panelDeck())
	d := deckContents{draw: run.Deck(), run: run}

	for _, s := range laidOut(d, deckView{played: true}).slots {
		if !s.lit {
			t.Fatal("a card is dimmed on a panel where nothing has been played")
		}
	}
}

func TestTheTalliesCountWhatIsLit(t *testing.T) {
	// **The tallies count the grid, not the piles** — so the two toggles reach them for free and
	// there is no second answer to "which cards is this panel about" to keep in step.
	run := panelRun(t, panelDeck())
	owned := run.Deck()

	d := deckContents{draw: owned[:2], spent: owned[2:], run: run, inFight: true}

	full := tallyOf(laidOut(d, deckView{}).slots, d.holder)
	played := tallyOf(laidOut(d, deckView{played: true}).slots, d.holder)

	if full.total != 2 {
		t.Errorf("FULL counts %d cards, want the 2 still to draw", full.total)
	}
	if played.total != 1 {
		t.Errorf("PLAYED counts %d cards, want the 1 spent", played.total)
	}
	if full.byElement[cards.Lightning] != 2 {
		t.Errorf("FULL counts %d lightning, want 2", full.byElement[cards.Lightning])
	}
	if played.byElement[cards.Fire] != 1 {
		t.Errorf("PLAYED counts %d fire, want 1", played.byElement[cards.Fire])
	}
}

func TestTheTalliesFollowTheAlterationsToggle(t *testing.T) {
	// A tally that disagreed with the grid above it would mean one of the two is lying, and the
	// grid is the one being looked at.
	run := panelRun(t, panelDeck(), "frozen-lightning-ring")
	d := deckContents{draw: run.Deck(), run: run}

	dealt := tallyOf(laidOut(d, deckView{}).slots, d.holder)
	asOwned := tallyOf(laidOut(d, deckView{unaltered: true}).slots, d.holder)

	if dealt.byElement[cards.Ice] != 2 || dealt.byElement[cards.Lightning] != 0 {
		t.Errorf("dealt: %d ice and %d lightning, want 2 and 0",
			dealt.byElement[cards.Ice], dealt.byElement[cards.Lightning])
	}
	if asOwned.byElement[cards.Lightning] != 2 || asOwned.byElement[cards.Ice] != 0 {
		t.Errorf("as owned: %d lightning and %d ice, want 2 and 0",
			asOwned.byElement[cards.Lightning], asOwned.byElement[cards.Ice])
	}
}

func TestEveryCardIsCountedOnceInEachTally(t *testing.T) {
	// Three ways of counting one deck have to agree on how big it is, or one of the three has a
	// card falling between its buckets — a form with no mark, a cost past the walk's ceiling.
	run := session.New(session.StartingDeck())
	d := deckContents{draw: run.Deck(), run: run}

	t2 := tallyOf(laidOut(d, deckView{}).slots, d.holder)

	byForm, byElement, byCost := 0, 0, 0
	for _, n := range t2.byForm {
		byForm += n
	}
	for _, n := range t2.byElement {
		byElement += n
	}
	for _, row := range t2.byFormCost {
		for _, n := range row {
			byCost += n
		}
	}

	for name, got := range map[string]int{"form": byForm, "element": byElement, "form and AP": byCost} {
		if got != t2.total {
			t.Errorf("the %s tally counts %d of %d cards", name, got, t2.total)
		}
	}
	if t2.total != run.Size() {
		t.Errorf("the tallies count %d cards and the run owns %d", t2.total, run.Size())
	}
}

func TestEveryFormInATallyHasAMark(t *testing.T) {
	// A row with no drawing at the head of it is a count of nothing anyone can name. The tally
	// writes out its own form order, so this is what catches a fifth form arriving in the rules and
	// not here.
	for _, f := range combat.Forms() {
		if f == combat.FormNone {
			continue
		}
		if _, ok := form(f).Glyph(); !ok {
			t.Errorf("%v has no mark, so its tally row would be a bare number", f)
		}
	}

	listed := map[combat.Form]bool{}
	for _, f := range tallyForms() {
		listed[f] = true
	}
	for _, f := range combat.Forms() {
		if f != combat.FormNone && !listed[f] {
			t.Errorf("%v is a form the rules have and the tally does not count", f)
		}
	}
}

func TestTheTallyBandFitsBetweenTheGridAndTheButtons(t *testing.T) {
	// **Height is the panel's dimension with no give**, which is already true of the grid — see
	// deckRowGap. The band underneath is four rows plus a heading, and the buttons stand under
	// that, so this is the arithmetic that says the three still fit.
	//
	// The internal resolution, which Layout fixes. Written out rather than imported because game
	// imports screens and not the reverse.
	const screenH = state.ScreenHeight
	pctY := func(p int) int { return screenH * p / 100 }

	top := pctY(modalPanelTopPct)
	bottom := pctY(modalPanelBottomPct)

	gridBottom := top + modalBareBodyTop + deckRowCount*(cards.Mini.Height+deckRowGap)
	bandTop := gridBottom + tallyTop
	bandBottom := bandTop + tallyHeadDrop + len(tallyForms())*tallyRowHeight

	buttonTop := bottom - deckViewButtonBottom - deckViewButtonHeight/2

	if bandBottom > buttonTop {
		t.Errorf("the tally band ends at %d and the buttons start at %d — they overlap",
			bandBottom, buttonTop)
	}
	if buttonTop+deckViewButtonHeight/2 > bottom-modalBodyBottom {
		t.Errorf("the buttons run past the panel's bottom margin")
	}
}

func TestTheTwoToggleButtonsDoNotOverlap(t *testing.T) {
	// They are placed as a pair from the panel's centre, so this is the arithmetic that says the
	// pair fits the panel and that neither sits on the other.
	const screenW = state.ScreenWidth
	pctX := func(p int) int { return screenW * p / 100 }

	span := 2*deckViewButtonWidth + deckViewButtonGap
	panel := pctX(modalPanelRightPct) - pctX(modalPanelLeftPct)

	if span > panel {
		t.Errorf("the two buttons need %d pixels and the panel is %d wide", span, panel)
	}
	if deckViewButtonGap <= 0 {
		t.Error("the buttons touch, so they read as one widget")
	}
}
