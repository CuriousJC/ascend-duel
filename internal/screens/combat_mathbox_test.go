package screens

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// These are the narrow kind of screen test CLAUDE.md allows: they create no `ebiten.Image`, need
// no window and no font, and they guard a cross-package invariant a compiler cannot see — that
// what the combo dialog writes on the screen is the arithmetic the resolver put on the event.
//
// **They test `mathScript`, which is the half of the box that has no geometry in it.** Where each
// figure flies *from* is a question about a row of cards on a screen, and there is no way to check
// that here; what the figures *say* is checkable, and it is the part that could quietly start
// lying about the round.

// comboEventFor builds a KindCombo event by hand, standing in for one the resolver produced.
func comboEvent(hand string, amounts []int, multiplier, total int) combat.Event {
	id, ok := combat.HandIDForKey(hand)
	if !ok {
		panic("the catalogue has no hand keyed " + hand)
	}

	e := combat.Event{
		Kind:       combat.KindCombo,
		Hand:       id,
		Multiplier: multiplier,
		Amount:     total,
	}
	for i, a := range amounts {
		e.ComboCards[i] = i
		e.ComboAmounts[i] = a
		e.ComboCardCount++
		e.Base += a
	}
	return e
}

// scriptText is the line the box would write, joined so a test can compare one string.
func scriptText(items []mathItem) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, it.text)
	}
	return strings.Join(parts, " ")
}

// **The sum is spelled out card by card.** This is the whole reason the dialog exists: the feed
// prints the hand's cards as one term, and what a player could not see was which card paid what.
func TestTheComboScriptSpellsOutEveryCard(t *testing.T) {
	got := scriptText(mathScript(comboEvent("pair", []int{20, 20}, 150, 60)))
	if want := "20 + 20 x 1.5 = 60"; got != want {
		t.Errorf("a Pair of Lunges reads %q, want %q", got, want)
	}
}

// A four-card hand keeps one plus between each pair of figures and gains nothing else.
func TestAFourCardHandReadsAsFourTerms(t *testing.T) {
	got := scriptText(mathScript(comboEvent("four-of-a-kind", []int{20, 20, 20, 20}, 500, 400)))
	if want := "20 + 20 + 20 + 20 x 5 = 400"; got != want {
		t.Errorf("a Four of a Kind reads %q, want %q", got, want)
	}
}

// **Every sum reads the same shape, the identity multiplier included** *(2026-08-19, owner's
// call)*. The High Card's `x 1` was dropped until then, on the argument that a sum times one says
// nothing — right about the arithmetic and wrong about the game: **hands are going to be
// upgradable**, so that 1 is a number that will change, and a term appearing only once it stops
// being 1 would make an upgrade read as a new rule rather than as a bigger figure. The log's line
// says it the same way.
func TestTheHighCardShowsItsMultiplier(t *testing.T) {
	got := scriptText(mathScript(comboEvent("high-card", []int{20}, 100, 20)))
	if want := "20 x 1 = 20"; got != want {
		t.Errorf("a High Card reads %q, want %q", got, want)
	}
}

// **Every hand the engine names is shouted, the High Card included** *(2026-08-19, owner's call)*.
// It was silent until then, on the argument that `HIGH CARD!` over a lone attack empties the word
// the way `COMBO!` over a single Strike does. What changed is that the name is now carried by the
// banner from DUEL! onward — so silence here would not withhold an announcement, it would take a
// word off the screen at the moment the blow lands.
//
// **An event naming no hand at all is still silent.** Nothing emits one, a turn with an attack in
// it always producing a blow, and a bare `!` at 124 points is what the check is worth.
func TestEveryNamedHandIsShouted(t *testing.T) {
	if got := shoutFor(comboEvent("pair", []int{20, 20}, 150, 60)); got != "PAIR!" {
		t.Errorf("a Pair shouts %q, want %q", got, "PAIR!")
	}
	if got := shoutFor(comboEvent("high-card", []int{20}, 100, 20)); got != "HIGH CARD!" {
		t.Errorf("a High Card shouts %q, want %q", got, "HIGH CARD!")
	}
	if got := shoutFor(combat.Event{Kind: combat.KindCombo, Hand: combat.HandNone}); got != "" {
		t.Errorf("an event naming no hand shouts %q, want silence", got)
	}
}

// **Every hand in the catalogue can be shouted and none of them is empty.** A hand added to
// `data/combos.json` with no name would put a bare `!` on the screen at 124 points.
//
// **The one-card hand is in the sweep now** rather than skipped: the High Card is shouted like any
// other since 2026-08-19.
func TestEveryHandInTheCatalogueHasAShout(t *testing.T) {
	for _, h := range combat.Hands() {
		e := combat.Event{Kind: combat.KindCombo, Hand: h.ID}
		got := shoutFor(e)
		if got == "" || got == "!" {
			t.Errorf("hand %q shouts %q", h.Key, got)
		}
		if got != strings.ToUpper(got) {
			t.Errorf("hand %q shouts %q, which is not upper case", h.Key, got)
		}
	}
}

// **Exactly the hand's own cards fly, plus the multiplier.** The flying items are the ones given a
// launch point in `startComboMath`, and it walks them expecting the cards first and the multiplier
// last — so a script that flew something else would seat a figure on the wrong card.
func TestTheFlyingItemsAreTheCardsThenTheMultiplier(t *testing.T) {
	for _, tc := range []struct {
		what  string
		e     combat.Event
		flies int
	}{
		{"a Pair", comboEvent("pair", []int{20, 20}, 150, 60), 3},
		{"a High Card", comboEvent("high-card", []int{20}, 100, 20), 2},
		{"trips", comboEvent("three-of-a-kind", []int{10, 10, 10}, 200, 60), 4},
	} {
		items := mathScript(tc.e)

		flies := 0
		for _, it := range items {
			if it.fly {
				flies++
			}
		}
		if flies != tc.flies {
			t.Errorf("%s flies %d items, want %d", tc.what, flies, tc.flies)
		}

		// The first ComboCardCount flying items have to be the cards, in order, or the launch
		// points in startComboMath are attached to the wrong figures.
		seen := 0
		for _, it := range items {
			if !it.fly || seen >= tc.e.ComboCardCount {
				continue
			}
			if want := tc.e.ComboAmounts[seen]; it.text != strconv.Itoa(want) {
				t.Errorf("%s: flying item %d reads %q, want the card's own %d", tc.what, seen, it.text, want)
			}
			seen++
		}
	}
}

// **The script ends with the answer, and the answer is the event's.** Nothing in the box may
// recompute a total: the figure shown and the figure landed have to be one number.
func TestTheScriptEndsWithTheEventsOwnTotal(t *testing.T) {
	e := comboEvent("pair", []int{7, 7}, 150, 21)
	items := mathScript(e)

	last := items[len(items)-1]
	if last.text != strconv.Itoa(e.Amount) {
		t.Errorf("the script ends with %q, want the event's %d", last.text, e.Amount)
	}
	// And it deliberately does not equal the sum of the terms: 7 + 7 is 14, and the total is what
	// the multiplier made of it. A box that added its own terms up would print 14 here.
	if last.text == strconv.Itoa(e.Base) {
		t.Errorf("the script ended with the base %d rather than the blow %d", e.Base, e.Amount)
	}
}

// --- the widest line the box can ever draw -------------------------------------------------

// **The line does not wrap and nothing shrinks to fit**, so the only thing keeping the sum inside
// its band is that the band is wider than the longest sum the rules can produce. That is a
// property worth pinning rather than eyeballing once: a bigger type size, a wider gap or a hand
// of six would all break it silently, and what the player would see is a figure half off the edge.
//
// **It needs a font, which is the one thing in this file that costs anything.** `LoadFontData`
// hands back bytes and `NewGoTextFaceSource` is pure Go parsing, so no `ebiten.Image` is created
// and nothing here needs a graphics context. The package already links Ebitengine directly, so
// this joins no group it was not in — see the note in CLAUDE.md about `xvfb-run` on Linux.
func mathTestState(t *testing.T) *state.GlobalState {
	t.Helper()

	src, err := text.NewGoTextFaceSource(bytes.NewReader(assets.LoadFontData()["kubasta"]))
	if err != nil {
		t.Fatalf("kubasta would not parse: %v", err)
	}
	return &state.GlobalState{
		ScreenWidth:  1280,
		ScreenHeight: 960,
		Fonts:        map[string]*text.GoTextFaceSource{"kubasta": src},
	}
}

// **The widest sum in the game fits its band**, with room to spare.
//
// **This is the test that found the box's width was wrong.** It first measured against `feedRect`'s
// band, which spans `handBand` and therefore *narrows as the hand empties*: a two-card hand gives
// about 330px against a widest sum of roughly 640, so the arithmetic would have run off both ends
// in exactly the rounds a duel is decided in. The box takes the table's width now — see
// `comboMathRect` — which is a function of the screen alone.
func TestTheWidestSumFitsItsBand(t *testing.T) {
	gs := mathTestState(t)

	var scene CombatScene
	band := scene.comboMathRect(gs)

	// Deliberately over the top: four terms of three digits, a three-digit multiplier written out,
	// and a five-digit total. Nothing the rules can produce is this wide, which is the point — the
	// margin is what a later type-size change is spending.
	e := comboEvent("four-of-a-kind", []int{999, 999, 999, 999}, 500, 19980)

	box := comboMathBox{items: mathScript(e)}
	scene.layOutMath(gs, &box)

	// **Measured to the ink, not to the centres.** Checking the resting *points* passes a line
	// half of which is off the screen, which is most of what this test is for — and it matters more
	// since the figures doubled on 2026-08-19.
	first, last := box.items[0], box.items[len(box.items)-1]
	firstW, _ := text.Measure(first.text, mathFace(gs, first.size), 0)
	lastW, _ := text.Measure(last.text, mathFace(gs, last.size), 0)

	if left := first.at.X - int(firstW/2); left <= band.Min.X {
		t.Errorf("the first figure starts at x=%d, outside the band's left edge at %d",
			left, band.Min.X)
	}
	if right := last.at.X + int(lastW/2); right >= band.Max.X {
		t.Errorf("the total ends at x=%d, outside the band's right edge at %d",
			right, band.Max.X)
	}
}

// **The line reads left to right and never doubles back.** Each item rests to the right of the one
// before it, which is the only thing making a sum read as a sum — and it is the property that
// breaks first if the widths and the gaps are ever measured against different faces.
func TestTheSumIsLaidOutLeftToRight(t *testing.T) {
	gs := mathTestState(t)

	var scene CombatScene
	box := comboMathBox{items: mathScript(comboEvent("four-of-a-kind", []int{20, 20, 20, 20}, 500, 400))}
	scene.layOutMath(gs, &box)

	for i := 1; i < len(box.items); i++ {
		if box.items[i].at.X <= box.items[i-1].at.X {
			t.Fatalf("item %d (%q) rests at x=%d, not right of item %d (%q) at x=%d",
				i, box.items[i].text, box.items[i].at.X,
				i-1, box.items[i-1].text, box.items[i-1].at.X)
		}
	}

	// And every item shares the band's vertical centre: the sum is one line, not a staircase.
	band := scene.comboMathRect(gs)
	cy := (band.Min.Y + band.Max.Y) / 2
	for i, it := range box.items {
		if it.at.Y != cy {
			t.Errorf("item %d (%q) sits at y=%d, want the band's centre %d", i, it.text, it.at.Y, cy)
		}
	}
}

// **The longest hand name in the catalogue fits the screen at the size it is shouted.** The shout
// doubled to 124 points on 2026-08-19, and a name is not a figure: `FOUR OF A KIND!` is fifteen
// characters against `19980`'s five, so the shout reaches its limit long before the sum does. It is
// centred on the hand row and does not wrap, so a name too wide runs off *both* edges at once.
//
// Measured against the whole screen rather than against a band, because that is what it is drawn
// on — the row it stands over is narrower than the name is allowed to be.
func TestTheWidestHandNameFitsTheScreen(t *testing.T) {
	gs := mathTestState(t)

	widest, name := 0.0, ""
	for _, hand := range combat.Hands() {
		w, _ := text.Measure(handShout(hand.Name), mathFace(gs, mathShoutSize), 0)
		if w > widest {
			widest, name = w, handShout(hand.Name)
		}
	}

	// The breath swells it, and the shout is faux-bold, so what has to fit is the widest it is
	// ever actually drawn rather than its resting width.
	widest = widest*(1+mathBreathAmount) + mathBoldStep(mathShoutSize)

	if widest > float64(gs.ScreenWidth) {
		t.Errorf("%q is %.0f wide at the shout's %d points, against a %d-wide screen",
			name, widest, mathShoutSize, gs.ScreenWidth)
	}
}
