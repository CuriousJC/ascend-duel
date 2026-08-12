package screens

import (
	"testing"

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
	// **combat.Poison is exempt, deliberately.** Poison lost its border colour when the
	// sheet dropped to four primaries plus basic; `cards.json` contains no poison card,
	// so nothing can be dealt one. The enum member survives because MECHANICS.md lists
	// poison as a secondary element that may get cards later — and if it does, it needs
	// a colour in internal/cards and an arm in artFor(), which is what this exemption is
	// here to make someone notice.
	dealable := []combat.Element{combat.Basic, combat.Fire, combat.Ice, combat.Lightning, combat.Earth}

	seen := map[cards.Element]combat.Element{}
	for _, e := range dealable {
		got := artFor(e)
		if prev, dup := seen[got]; dup {
			t.Errorf("%s and %s both map to cards.%v — one of them is missing from the switch in artFor()",
				prev, e, got)
		}
		seen[got] = e
	}

	if len(seen) != len(cards.Elements()) {
		t.Errorf("the screen maps %d distinct dealable elements but internal/cards knows %d",
			len(seen), len(cards.Elements()))
	}

	// Poison must land somewhere real rather than panicking or drawing nothing.
	if got := artFor(combat.Poison); got != cards.Basic {
		t.Errorf("poison maps to cards.%v, want the Basic fallback", got)
	}
}

func TestElementNamesAgreeAcrossThePackages(t *testing.T) {
	// The two enums also carry names, and the deck reads element names out of
	// cards.json. If the drawing package spells one differently, a sheet labelled
	// "lightning" could be showing the colour the game calls something else.
	//
	// Poison is excluded for the reason above: it has no art of its own to agree with.
	for _, e := range []combat.Element{combat.Basic, combat.Fire, combat.Ice, combat.Lightning, combat.Earth} {
		if got, want := artFor(e).String(), e.String(); got != want {
			t.Errorf("screen calls it %q, internal/cards calls it %q", want, got)
		}
	}
}

func TestEveryCategoryHasAGlyph(t *testing.T) {
	// The same hand-written-switch hazard as the elements, one type over. A category
	// falling through to CategoryNone draws no glyph at all, which on a card whose
	// category *word* has been deleted means the phase becomes unstated.
	all := []combat.Category{combat.CategoryPrepare, combat.CategoryAttack, combat.CategoryDefend}

	seen := map[cards.Category]combat.Category{}
	for _, c := range all {
		got := category(c)
		if got == cards.CategoryNone {
			t.Errorf("%v maps to CategoryNone — it would draw no glyph and the card would not say its phase", c)
		}
		if prev, dup := seen[got]; dup {
			t.Errorf("%v and %v both map to cards.%v", prev, c, got)
		}
		seen[got] = c

		if got.String() != c.String() {
			t.Errorf("rules call it %q, internal/cards calls it %q", c.String(), got.String())
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

	// A full row plus its label gutter has to fit the panel it is drawn in.
	const conceptsPerElement = 12
	row := (conceptsPerElement-1)*deckStackPitch + cards.Mini.Width + deckRowLabelWidth
	if panel := pctX(deckPanelRightPct) - pctX(deckPanelLeftPct); row > panel-24 {
		t.Errorf("a full row is %dpx wide against a %dpx panel", row, panel)
	}

	// And every row has to fit between the legend above and the closing hint below.
	// Derived from the panel constants rather than written down, because the last time it
	// was a hardcoded number it went stale the moment deckGridTop moved.
	top := pctY(deckPanelTopPct) + deckGridTop
	bottom := pctY(deckPanelBottomPct) - deckHintUp
	rows := len(cards.Elements())*(cards.Mini.Height+deckRowGap) - deckRowGap
	if budget := bottom - top; rows > budget {
		t.Errorf("%d rows of %d is %dpx tall against a %dpx budget (y=%d..%d)",
			len(cards.Elements()), cards.Mini.Height, rows, budget, top, bottom)
	}
}
