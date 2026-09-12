package screens

// The deck panel's column: what each control changes about the picture, and what none of them is
// allowed to change.
//
// **These need no window.** The grid is a layout function and the figures count a laid-out grid,
// which is the whole reason both were pulled out of the drawing in the first place — see
// pileGridLayout and countsOf.

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// panelRun is a run wearing the named relics, with a deck small enough to reason about by hand.
func panelRun(t *testing.T, deck []combat.Card, relics ...string) *session.Session {
	t.Helper()

	run := session.New(deck)
	for _, key := range relics {
		if !run.Wear(key) {
			t.Fatalf("could not wear %s", key)
		}
	}
	return run
}

// panelDeck is two lightning cards and a fire one — enough for a flip relic to have something to
// take and something to leave alone.
func panelDeck() []combat.Card {
	return []combat.Card{
		{Concept: combat.Bash, Element: combat.Lightning},
		{Concept: combat.Bash, Element: combat.Lightning},
		{Concept: combat.Bash, Element: combat.Fire},
	}
}

// laidOut is the grid a view produces, at the panel's real footprint.
func laidOut(d deckContents, v deckView) pileGridLayout {
	return d.grid(v, 640, 1177, 150)
}

func TestAlterationsAreOnByDefault(t *testing.T) {
	// **The default is the deck you will be dealt, not the deck you own** *(owner's call,
	// 2026-08-24)*. A run wearing a flip relic never draws a lightning card, so a panel opening on a
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
	// not be done anyway once two relics converge on one colour.
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

func TestTheFiguresCountWhatIsLit(t *testing.T) {
	// **The figures count the grid, not the piles** — so the two toggles reach them for free and
	// there is no second answer to "which cards is this panel about" to keep in step.
	run := panelRun(t, panelDeck())
	owned := run.Deck()

	d := deckContents{draw: owned[:2], spent: owned[2:], run: run, inFight: true}

	full := countsOf(laidOut(d, deckView{}).slots, d.holder, deckFilter{})
	played := countsOf(laidOut(d, deckView{played: true}).slots, d.holder, deckFilter{})

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

func TestTheFiguresFollowTheAlterationsToggle(t *testing.T) {
	// A figure that disagreed with the grid beside it would mean one of the two is lying, and the
	// grid is the one being looked at.
	run := panelRun(t, panelDeck(), "frozen-lightning-ring")
	d := deckContents{draw: run.Deck(), run: run}

	dealt := countsOf(laidOut(d, deckView{}).slots, d.holder, deckFilter{})
	asOwned := countsOf(laidOut(d, deckView{unaltered: true}).slots, d.holder, deckFilter{})

	if dealt.byElement[cards.Ice] != 2 || dealt.byElement[cards.Lightning] != 0 {
		t.Errorf("dealt: %d ice and %d lightning, want 2 and 0",
			dealt.byElement[cards.Ice], dealt.byElement[cards.Lightning])
	}
	if asOwned.byElement[cards.Lightning] != 2 || asOwned.byElement[cards.Ice] != 0 {
		t.Errorf("as owned: %d lightning and %d ice, want 2 and 0",
			asOwned.byElement[cards.Lightning], asOwned.byElement[cards.Ice])
	}
}

func TestEveryCardIsCountedOnceInEachBlock(t *testing.T) {
	// Three ways of counting one deck have to agree on how big it is, or one of the three has a
	// card falling between its buckets — a form with no mark, a cost past the walk's ceiling.
	run := session.New(session.StartingDeck())
	d := deckContents{draw: run.Deck(), run: run}

	c := countsOf(laidOut(d, deckView{}).slots, d.holder, deckFilter{})

	byForm, byElement, byCost := 0, 0, 0
	for _, n := range c.byForm {
		byForm += n
	}
	for _, n := range c.byElement {
		byElement += n
	}
	for _, n := range c.byCost {
		byCost += n
	}

	for name, got := range map[string]int{"form": byForm, "element": byElement, "AP": byCost} {
		if got != c.total {
			t.Errorf("the %s block counts %d of %d cards", name, got, c.total)
		}
	}
	if c.total != run.Size() {
		t.Errorf("the column counts %d cards and the run owns %d", c.total, run.Size())
	}

	// And every price a card carries gets a button, or the AP block would be a filter that cannot
	// reach part of the deck.
	priced := map[int]bool{}
	for _, cost := range c.costs {
		priced[cost] = true
	}
	for cost, n := range c.byCost {
		if n > 0 && !priced[cost] {
			t.Errorf("%d cards cost %d AP and there is no button for it", n, cost)
		}
	}
}

func TestEveryFormInTheColumnHasAMark(t *testing.T) {
	// A button with no drawing on it is a bare word beside a bare number. The column writes out its
	// own form order, so this is what catches a fifth form arriving in the rules and not here.
	for _, f := range combat.Forms() {
		if f == combat.FormNone {
			continue
		}
		if _, ok := form(f).Glyph(); !ok {
			t.Errorf("%v has no mark, so its button would be a bare number", f)
		}
	}

	listed := map[combat.Form]bool{}
	for _, f := range deckFilterForms() {
		listed[f] = true
	}
	for _, f := range combat.Forms() {
		if f != combat.FormNone && !listed[f] {
			t.Errorf("%v is a form the rules have and the column cannot filter on", f)
		}
	}
}

func TestTheFilterColumnFitsThePanel(t *testing.T) {
	// **Height is the panel's dimension with no give**, which is already true of the grid — see
	// deckRowGap. The column is now the taller of the two things inside the panel, so this is the
	// arithmetic that says it still ends above the bottom margin.
	//
	// The internal resolution, which Layout fixes. Written out rather than imported because game
	// imports screens and not the reverse.
	bottom := state.ScreenHeight*modalPanelBottomPct/100 - modalBodyBottom

	// Three toggles is what a fight shows: ALTERATIONS, FULL and SHOW ALL. Four prices is the
	// shipping deck's spread with room for one more.
	if got := deckColumnBottom(3, 4); got > bottom {
		t.Errorf("the column ends at %d and the panel's margin is at %d", got, bottom)
	}
}

func TestTheFilterColumnAndTheGridDoNotOverlap(t *testing.T) {
	// The column is drawn from the panel's left edge and the grid from what is left of it, and both
	// derive that boundary from deckGridSpan — this is what says the pair actually fits.
	left := state.ScreenWidth * modalPanelLeftPct / 100
	right := state.ScreenWidth * modalPanelRightPct / 100

	gridLeft, gridRight := deckGridSpan(left, right)
	if columnRight := left + deckColumnInset + deckColumnWidth; gridLeft <= columnRight {
		t.Errorf("the grid starts at %d and the column ends at %d", gridLeft, columnRight)
	}
	if gridRight <= gridLeft {
		t.Errorf("the column leaves the grid %d pixels", gridRight-gridLeft)
	}
}

// The filter itself: what a set of pressed buttons means, and what it is never allowed to touch.

func TestAnEmptyFilterPicksNothing(t *testing.T) {
	// **The panel opens pointing at nothing**, or every card in it would be marked and the mark
	// would say nothing at all.
	run := session.New(session.StartingDeck())
	d := deckContents{draw: run.Deck(), run: run}

	for _, s := range laidOut(d, deckView{}).slots {
		if s.picked {
			t.Fatal("a card is marked on a panel with no filter set")
		}
	}
}

func TestAxesAreAndedAndValuesWithinOneAreOred(t *testing.T) {
	// **The combining rule** *(owner's call, 2026-09-11)*: crush with fire is the cards that are
	// both, crush with slash is either. It is the only rule under which every button both adds and
	// removes something.
	var f deckFilter
	f.toggleForm(combat.FormCrush)
	f.toggleElement(cards.Fire)

	cases := []struct {
		form    combat.Form
		element cards.Element
		want    bool
	}{
		{combat.FormCrush, cards.Fire, true},
		{combat.FormCrush, cards.Ice, false},
		{combat.FormSlash, cards.Fire, false},
	}
	for _, c := range cases {
		if got := f.matches(c.form, 1, c.element, axisNone); got != c.want {
			t.Errorf("%v %v matched %v, want %v", c.form, c.element, got, c.want)
		}
	}

	// A second value on an axis widens that axis and narrows nothing.
	f.toggleForm(combat.FormSlash)
	if !f.matches(combat.FormSlash, 1, cards.Fire, axisNone) {
		t.Error("picking slash beside crush did not widen the form axis")
	}
	if f.matches(combat.FormCrush, 1, cards.Ice, axisNone) {
		t.Error("a second form loosened the element axis")
	}
}

func TestPressingAValueTwiceTakesItBackOff(t *testing.T) {
	// Every one of these is a toggle, so the way out of a selection is the button that made it.
	var f deckFilter
	f.toggleCost(2)
	if !f.onCost(2) {
		t.Fatal("the first press did not pick 2 AP")
	}
	f.toggleCost(2)
	if f.onCost(2) {
		t.Error("the second press did not take 2 AP back off")
	}
	if !f.empty() {
		t.Error("an axis emptied by a toggle still reads as a filter")
	}
}

func TestClearDropsEveryAxis(t *testing.T) {
	var f deckFilter
	f.toggleForm(combat.FormStab)
	f.toggleCost(1)
	f.toggleElement(cards.Earth)

	f.clear()
	if !f.empty() {
		t.Error("SHOW ALL left something picked")
	}
}

func TestAButtonsFigureIgnoresItsOwnAxis(t *testing.T) {
	// **This is what stops a number lying.** Counted under the whole filter, every unpicked value of
	// a picked axis reads zero — slash would say 0 beside a press that adds fourteen cards. The
	// count is taken with the button's own axis set aside, so it says what pressing it is worth.
	run := session.New(session.StartingDeck())
	d := deckContents{draw: run.Deck(), run: run}
	slots := laidOut(d, deckView{}).slots

	whole := countsOf(slots, d.holder, deckFilter{})

	var f deckFilter
	f.toggleForm(combat.FormCrush)
	picked := countsOf(slots, d.holder, f)

	if picked.byForm[combat.FormSlash] != whole.byForm[combat.FormSlash] {
		t.Errorf("with crush picked, slash reads %d and the deck holds %d",
			picked.byForm[combat.FormSlash], whole.byForm[combat.FormSlash])
	}
	if picked.total != whole.byForm[combat.FormCrush] {
		t.Errorf("the heading counts %d cards and the deck holds %d crush",
			picked.total, whole.byForm[combat.FormCrush])
	}

	// The other axes do narrow — that is the recount the column is for.
	narrowed := 0
	for _, n := range picked.byElement {
		narrowed += n
	}
	if narrowed != picked.total {
		t.Errorf("the element block counts %d against a selection of %d", narrowed, picked.total)
	}
	if narrowed >= whole.total {
		t.Errorf("the element block counts %d of %d cards — it did not narrow", narrowed, whole.total)
	}
}

func TestTheGridMarksExactlyTheCardsTheHeadingCounts(t *testing.T) {
	// The figure and the marked cards are two readings of one answer, and the panel is unusable if
	// they disagree: the number would send the player looking for cards that are not lit.
	run := session.New(session.StartingDeck())
	d := deckContents{draw: run.Deck(), run: run}

	v := deckView{}
	v.filter.toggleForm(combat.FormStab)
	v.filter.toggleElement(cards.Fire)

	slots := laidOut(d, v).slots
	marked := 0
	for _, s := range slots {
		if s.picked {
			marked++
		}
	}

	if c := countsOf(slots, d.holder, v.filter); marked != c.total {
		t.Errorf("%d cards are marked and the heading says %d", marked, c.total)
	}
	if marked == 0 {
		t.Fatal("fire stabs picked nothing, so this test is checking two zeroes")
	}
}

func TestTheFilterMovesNoCardAndDimsNothing(t *testing.T) {
	// **The panel's governing idea**: a card does not move when something changes about the view, it
	// only changes how it is drawn. The filter is a mark, on a separate channel from the dimming —
	// see cards.MarkPicked — so neither a seat nor a lit flag may differ under one.
	owned := session.New(session.StartingDeck())
	deck := owned.Deck()
	d := deckContents{draw: deck[:20], spent: deck[20:], run: owned, inFight: true}

	plain := laidOut(d, deckView{}).slots

	v := deckView{}
	v.filter.toggleForm(combat.FormCrush)
	filtered := laidOut(d, v).slots

	if len(plain) != len(filtered) {
		t.Fatalf("the filter changed the grid from %d cards to %d", len(plain), len(filtered))
	}
	for i := range plain {
		if plain[i].at != filtered[i].at {
			t.Fatalf("card %d sits at %v unfiltered and %v filtered", i, plain[i].at, filtered[i].at)
		}
		if plain[i].lit != filtered[i].lit {
			t.Fatalf("card %d is lit %v unfiltered and %v filtered", i, plain[i].lit, filtered[i].lit)
		}
	}
}
