package screens

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// **This is the first test in internal/screens, and it is a deliberate narrow exception
// to "the combat package is the one that can be tested without a window".**
//
// The rule is really about windowlessness, and these two assertions happen to be
// windowless: they compare constants and walk a switch statement. Nothing here creates an
// ebiten.Image, calls RunGame, or touches a GlobalState, so no graphics driver is ever
// initialised and the test runs on a headless CI box.
//
// They earn the exception because both guard a duplication that the compiler cannot see.
// If either becomes awkward, delete it — do not start reaching for a window to keep it
// alive, and do not read this as licence to test the rest of the screen.

func TestCardFootprintMatchesTheRenderer(t *testing.T) {
	// cardWidth and cardHeight lay out the hand — the pitch, the band, the drop
	// indicator, every hit rectangle. cards.Hand draws the card that sits in those
	// rectangles. They are two copies of one number because one is a const and the other
	// a var field, and nothing but this test stops them drifting.
	//
	// Drift would not crash anything. It would put the cards a few pixels out of their
	// own hit boxes, which reads as "clicking the edge of a card sometimes does nothing"
	// and is miserable to track down.
	if cardWidth != cards.Hand.Width {
		t.Errorf("cardWidth is %d but cards.Hand.Width is %d — the hand would lay out cards at the wrong pitch",
			cardWidth, cards.Hand.Width)
	}
	if cardHeight != cards.Hand.Height {
		t.Errorf("cardHeight is %d but cards.Hand.Height is %d — hit rectangles would not match the art",
			cardHeight, cards.Hand.Height)
	}
}

func TestEveryElementHasItsOwnArt(t *testing.T) {
	// combat.Element and cards.Element are separate enums on purpose: the rules say what an
	// element *does* and the drawing package says what colour it is, and neither wants the
	// other's vocabulary. The cost is a hand-written switch, which the compiler cannot check
	// for completeness.
	//
	// A missing case falls through to Basic, so the failure mode is two elements sharing
	// a border colour — a fire card that looks plain. Distinctness is what is asserted.
	//
	// It walks combat.AllElements rather than a list written out here, so an element
	// appended to the rules fails this test until it has been given a colour.
	seen := map[cards.Element]combat.Element{}
	for _, e := range combat.AllElements {
		got := artFor(e)
		if prev, dup := seen[got]; dup {
			t.Errorf("%s and %s both map to cards.%v — one of them is missing from the switch in artFor()",
				prev, e, got)
		}
		seen[got] = e
	}

	if len(seen) != len(cards.Elements()) {
		t.Errorf("the screen maps %d distinct elements but internal/cards knows %d",
			len(seen), len(cards.Elements()))
	}
}

func TestElementNamesAgreeAcrossThePackages(t *testing.T) {
	// The two enums also carry names, and the deck reads element names out of
	// cards.json. If the drawing package spells one differently, a sheet labelled
	// "lightning" could be showing the colour the game calls something else.
	for _, e := range combat.AllElements {
		if got, want := artFor(e).String(), e.String(); got != want {
			t.Errorf("screen calls it %q, internal/cards calls it %q", want, got)
		}
	}
}

func TestEveryFamilyHasItsOwnMark(t *testing.T) {
	// The same hand-written-switch hazard as the elements, one type over. A family
	// falling through to FamilyNone draws no mark at all, which on a card whose
	// category *word* has been deleted means the card says nothing about what it is.
	seen := map[cards.Family]combat.Family{}
	for _, f := range combat.Families() {
		got := family(f)
		if got == cards.FamilyNone {
			t.Errorf("%v maps to FamilyNone — it would draw no mark and the card would not say its family", f)
		}
		if prev, dup := seen[got]; dup {
			t.Errorf("%v and %v both map to cards.%v", prev, f, got)
		}
		seen[got] = f

		if got.String() != f.String() {
			t.Errorf("rules call it %q, internal/cards calls it %q", f.String(), got.String())
		}
	}

	// **FamilyNone has to survive the crossing too.** The opponent's cards carry it, and one that
	// mapped onto a real family would draw a letter claiming membership of a deck the player
	// cannot combo with.
	if got := family(combat.FamilyNone); got != cards.FamilyNone {
		t.Errorf("the rules' FamilyNone maps to cards.%v, want FamilyNone", got)
	}
}

func TestEveryConceptHasEffectText(t *testing.T) {
	// A card with no text draws a name, a cost, a corner mark and nothing that says what it does.
	//
	// **It walks the whole registry, not the player's twelve** *(2026-08-16)*. Every enemy carries
	// its own cards and the table lays an enemy's queue out as cards, so a verb the generator does
	// not cover is four hundred blank faces rather than one.
	for _, a := range combat.AllConcepts() {
		if cardEffect(a) == "" {
			t.Errorf("%v has no effect text — its card would say nothing about what it does",
				combat.ConceptOf(a).Key)
		}
	}
}

func TestEveryCardTextFitsItsBand(t *testing.T) {
	// **The wording is here and the band is in internal/cards, so neither package can check
	// this alone.** Render draws every line it wraps to rather than clamping, so an overlong
	// string runs off the bottom of the card — this is what fails first.
	//
	// It needs the real font because wrapping is measured, which is also why it is worth
	// having: "Negate 1 attack, deal 0.5x damage back" fits in three lines or four depending on
	// a comma, and nobody can tell by looking at the string.
	ttf := assets.LoadFontData()["kubasta"]
	if len(ttf) == 0 {
		t.Fatal("no kubasta font data embedded")
	}
	f, err := cards.NewFaces(ttf)
	if err != nil {
		t.Fatal(err)
	}

	st := cards.Hand
	width := st.Width - st.TextColumnLeft - st.TextInset

	for _, a := range combat.AllConcepts() {
		lines, err := cards.WrapText(f, st.TextSize, cardEffect(a), width)
		if err != nil {
			t.Fatalf("%v: %v", a, err)
		}
		if len(lines) > st.TextLines() {
			t.Errorf("%v's text wraps to %d lines and the band holds %d: %q",
				combat.ConceptOf(a).Key, len(lines), st.TextLines(), cardEffect(a))
		}
	}
}

func TestNoEffectTextWordIsWiderThanItsColumn(t *testing.T) {
	// **Wrapping breaks on spaces only**, so a single word wider than the column is not
	// wrapped, it overruns — silently, and only on the one card that has it. The column is
	// ~100px at 18pt, which is around a dozen characters, so this is a real constraint on the
	// wording rather than a theoretical one.
	ttf := assets.LoadFontData()["kubasta"]
	if len(ttf) == 0 {
		t.Fatal("no kubasta font data embedded")
	}
	f, err := cards.NewFaces(ttf)
	if err != nil {
		t.Fatal(err)
	}

	st := cards.Hand
	width := st.Width - st.TextColumnLeft - st.TextInset

	for _, a := range combat.AllConcepts() {
		for _, word := range strings.Fields(cardEffect(a)) {
			w, err := cards.TextWidth(f, st.TextSize, word)
			if err != nil {
				t.Fatalf("%v: %v", a, err)
			}
			if w > width {
				t.Errorf("%v: %q is %dpx, wider than the %dpx column — it will run off the card",
					a, word, w, width)
			}
		}
	}
}

func TestDeckPitchMatchesTheCard(t *testing.T) {
	// The overlay lays cards out at deckStackPitch and internal/cards sizes its layout
	// against the strip that leaves visible. The two live in different packages — screens
	// imports cards and never the reverse — so nothing but this stops them drifting.
	//
	// Drift is silent and ugly: tighten the pitch for a longer row and the name simply
	// stops being visible, with no error anywhere.
	if deckStackPitch > cards.Mini.Width {
		t.Errorf("pitch %d exceeds the card width %d, so the row would have gaps in it",
			deckStackPitch, cards.Mini.Width)
	}

	// The internal resolution, which Layout fixes. Written here rather than imported
	// because game imports screens and not the reverse; if it ever changes, this test is
	// the thing that should be updated to match.
	const screenW, screenH = 1280, 960
	pctX := func(p int) int { return screenW * p / 100 }
	pctY := func(p int) int { return screenH * p / 100 }

	// A full row plus its label gutter has to fit the panel it is drawn in. The cap is what a
	// row may hold, so it is what has to fit — the plan row is the one that reaches it.
	row := (deckMaxPerRow-1)*deckStackPitch + cards.Mini.Width + deckRowLabelWidth
	if panel := pctX(deckPanelRightPct) - pctX(deckPanelLeftPct); row > panel-24 {
		t.Errorf("a full row is %dpx wide against a %dpx panel", row, panel)
	}

	// And every row has to fit between the legend above and the closing hint below.
	// Derived from the panel constants rather than written down, because the last time it
	// was a hardcoded number it went stale the moment deckGridTop moved.
	top := pctY(deckPanelTopPct) + deckGridTop
	bottom := pctY(deckPanelBottomPct) - deckHintUp
	rows := deckRowCount*(cards.Mini.Height+deckRowGap) - deckRowGap
	if budget := bottom - top; rows > budget {
		t.Errorf("%d rows of %d is %dpx tall against a %dpx budget (y=%d..%d)",
			deckRowCount, cards.Mini.Height, rows, budget, top, bottom)
	}

	// The grid must also start below the legend it sits under, which is what the six-row
	// squeeze most easily breaks.
	if deckGridTop <= deckLegendTop {
		t.Errorf("the grid starts at y=%d, at or above the legend at y=%d", deckGridTop, deckLegendTop)
	}
}

func TestEveryCardLandsInExactlyOneDeckRow(t *testing.T) {
	// **The panel's whole claim is that it shows the deck**, so a card with nowhere to go, or a
	// row over the cap, is the panel quietly lying. The plan row is the one at the cap — three
	// concepts at four copies — and the element rows hold nine each.
	counts := make([]int, deckRowCount)
	for _, e := range startingDeck {
		row := deckRowFor(e.card)
		if row < 0 || row >= deckRowCount {
			t.Fatalf("%v maps to row %d, which does not exist", e.card, row)
		}
		counts[row] += e.count
	}

	for row, n := range counts {
		name, _ := deckRowLabel(row)
		if n == 0 {
			t.Errorf("the %q row is empty", name)
		}
		if n > deckMaxPerRow {
			t.Errorf("the %q row holds %d cards against a cap of %d — %d would be hidden",
				name, n, deckMaxPerRow, n-deckMaxPerRow)
		}
	}

	// And the plans are in the plan row rather than in basic, which is the whole point of the
	// sixth row: they are all basic, and leaving them there overflowed it.
	for _, e := range startingDeck {
		if e.card.Category() == combat.CategoryPlan && deckRowFor(e.card) != deckPlanRow {
			t.Errorf("%v is a plan and sits in row %d", e.card, deckRowFor(e.card))
		}
	}
}

func TestTheCardHoldsAsManyEffectsAsThereAreStatuses(t *testing.T) {
	// `cards.MaxEffects` is a layout number in a package that cannot see the rules, and the
	// rules decide how many statuses one duelist can carry at once: one per element, since a
	// status does not stack and the array is indexed by element. This is the join.
	//
	// A fifth element would silently drop a badge off the enemy card — the row would draw four
	// of five and look like a rendering glitch rather than a missing status. Failing here is
	// what makes adding an element a layout change too, exactly as MaxStatLines does for a
	// fourth stat row.
	statuses := combat.ElementCount - 1 // Basic carries none
	if cards.MaxEffects < statuses {
		t.Errorf("a card shows %d status badges against %d statuses a duelist can carry — %d would be dropped",
			cards.MaxEffects, statuses, statuses-cards.MaxEffects)
	}
}

func TestEveryStatusElementHasABadge(t *testing.T) {
	// A status with no artwork falls back to the default badge, which is a shape nobody has
	// learned — fine as a backstop, wrong as the thing a shipped element draws. This walks the
	// elements the rules can actually put on a duelist and asks for a picture of its own.
	for _, e := range combat.AllElements {
		if e == combat.Basic {
			continue
		}
		key, ok := effectKeys[e]
		if !ok {
			t.Errorf("%v has no status badge and would draw the default", e)
			continue
		}
		if _, ok := assets.LoadImageData()[key]; !ok {
			t.Errorf("%v's badge is %q, which is not an embedded image", e, key)
		}
	}
}
