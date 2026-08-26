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
// what the hand dialog writes on the screen is the arithmetic the resolver put on the event.
//
// **They test `mathScript`, which is the half of the box that has no geometry in it.** Where each
// figure flies *from* is a question about a row of cards on a screen, and there is no way to check
// that here; what the figures *say* is checkable, and it is the part that could quietly start
// lying about the round.

// handEventFor builds a KindHand event by hand, standing in for one the resolver produced.
func handEvent(hand string, amounts []int, multiplier, total int) combat.Event {
	id, ok := combat.HandIDForKey(hand)
	if !ok {
		panic("the catalogue has no hand keyed " + hand)
	}

	e := combat.Event{
		Kind:       combat.KindHand,
		Hand:       id,
		Multiplier: multiplier,
		Amount:     total,
	}
	for i, a := range amounts {
		e.HandCards[i] = i
		e.HandAmounts[i] = a
		e.HandCardCount++
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
func TestTheHandScriptSpellsOutEveryCard(t *testing.T) {
	got := scriptText(mathScript(handEvent("concept-pair", []int{20, 20}, 150, 60)))
	if want := "20 + 20 x 1.5 = 60"; got != want {
		t.Errorf("a Pair of Lunges reads %q, want %q", got, want)
	}
}

// A four-card hand keeps one plus between each pair of figures and gains nothing else.
func TestAFourCardHandReadsAsFourTerms(t *testing.T) {
	got := scriptText(mathScript(handEvent("concept-four-of-a-kind", []int{20, 20, 20, 20}, 500, 400)))
	if want := "20 + 20 + 20 + 20 x 5 = 400"; got != want {
		t.Errorf("a Four of a Kind reads %q, want %q", got, want)
	}
}

// **An echo is spelled out as its own terms** *(2026-08-22)*. The whole reason Echo pays into the
// blow's sum rather than landing a second blow is that the player can watch the first card pay
// three times — so a hand event carrying echo terms has to read as extra figures, not as one
// bigger one.
func TestAnEchoedCardReadsAsExtraTerms(t *testing.T) {
	e := handEvent("concept-pair", []int{30, 30, 20, 10}, 150, 135)
	e.EchoTerms = 2

	if got, want := scriptText(mathScript(e)), "30 + 30 + 20 + 10 x 1.5 = 135"; got != want {
		t.Errorf("an echoed Pair reads %q, want %q", got, want)
	}
}

// **Every sum reads the same shape, the identity multiplier included** *(2026-08-19, owner's
// call)*. The High Card's `x 1` was dropped until then, on the argument that a sum times one says
// nothing — right about the arithmetic and wrong about the game: **hands are going to be
// upgradable**, so that 1 is a number that will change, and a term appearing only once it stops
// being 1 would make an upgrade read as a new rule rather than as a bigger figure. The log's line
// says it the same way.
func TestTheHighCardShowsItsMultiplier(t *testing.T) {
	got := scriptText(mathScript(handEvent("high-card", []int{20}, 100, 20)))
	if want := "20 x 1 = 20"; got != want {
		t.Errorf("a High Card reads %q, want %q", got, want)
	}
}

// **Every hand the engine names is shouted, the High Card included** *(2026-08-19, owner's call)*.
// It was silent until then, on the argument that `HIGH CARD!` over a lone attack empties the word
// the way `HAND!` over a single Strike does. What changed is that the name is now carried by the
// banner from DUEL! onward — so silence here would not withhold an announcement, it would take a
// word off the screen at the moment the blow lands.
//
// **An event naming no hand at all is still silent.** Nothing emits one, a turn with an attack in
// it always producing a blow, and a bare `!` at 124 points is what the check is worth.
func TestEveryNamedHandIsShouted(t *testing.T) {
	if got := shoutFor(handEvent("concept-pair", []int{20, 20}, 150, 60)); got != "CARD PAIR!" {
		t.Errorf("a Pair shouts %q, want %q", got, "CARD PAIR!")
	}
	if got := shoutFor(handEvent("high-card", []int{20}, 100, 20)); got != "HIGH CARD!" {
		t.Errorf("a High Card shouts %q, want %q", got, "HIGH CARD!")
	}
	if got := shoutFor(combat.Event{Kind: combat.KindHand, Hand: combat.HandNone}); got != "" {
		t.Errorf("an event naming no hand shouts %q, want silence", got)
	}
}

// **Every hand in the catalogue can be shouted and none of them is empty.** A hand added to
// `data/hands.json` with no name would put a bare `!` on the screen at 124 points.
//
// **The one-card hand is in the sweep now** rather than skipped: the High Card is shouted like any
// other since 2026-08-19.
func TestEveryHandInTheCatalogueHasAShout(t *testing.T) {
	for _, h := range combat.Hands() {
		e := combat.Event{Kind: combat.KindHand, Hand: h.ID}
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
// launch point in `startHandMath`, and it walks them expecting the cards first and the multiplier
// last — so a script that flew something else would seat a figure on the wrong card.
func TestTheFlyingItemsAreTheCardsThenTheMultiplier(t *testing.T) {
	for _, tc := range []struct {
		what  string
		e     combat.Event
		flies int
	}{
		{"a Pair", handEvent("concept-pair", []int{20, 20}, 150, 60), 3},
		{"a High Card", handEvent("high-card", []int{20}, 100, 20), 2},
		{"trips", handEvent("concept-three-of-a-kind", []int{10, 10, 10}, 200, 60), 4},
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

		// The first HandCardCount flying items have to be the cards, in order, or the launch
		// points in startHandMath are attached to the wrong figures.
		seen := 0
		for _, it := range items {
			if !it.fly || seen >= tc.e.HandCardCount {
				continue
			}
			if want := tc.e.HandAmounts[seen]; it.text != strconv.Itoa(want) {
				t.Errorf("%s: flying item %d reads %q, want the card's own %d", tc.what, seen, it.text, want)
			}
			seen++
		}
	}
}

// **The script ends with the answer, and the answer is the event's.** Nothing in the box may
// recompute a total: the figure shown and the figure landed have to be one number.
func TestTheScriptEndsWithTheEventsOwnTotal(t *testing.T) {
	e := handEvent("concept-pair", []int{7, 7}, 150, 21)
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
// `handMathRect` — which is a function of the screen alone.
func TestTheWidestSumFitsItsBand(t *testing.T) {
	gs := mathTestState(t)

	var scene CombatScene
	band := scene.handMathRect(gs)

	// Deliberately over the top: **seven** terms of three digits, a three-digit multiplier written
	// out, and a five-digit total. Seven is the widest a blow can read — five cards in a legal turn
	// plus the two extra landings an echo ring seats behind the first — and three digits each is
	// past anything the rules produce, which is the point: the margin is what a later type-size
	// change is spending.
	e := handEvent("concept-four-of-a-kind", []int{999, 999, 999, 999, 999, 999, 999}, 500, 19980)

	box := handMathBox{items: mathScript(e)}
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
	box := handMathBox{items: mathScript(handEvent("concept-four-of-a-kind", []int{20, 20, 20, 20}, 500, 400))}
	scene.layOutMath(gs, &box)

	for i := 1; i < len(box.items); i++ {
		if box.items[i].at.X <= box.items[i-1].at.X {
			t.Fatalf("item %d (%q) rests at x=%d, not right of item %d (%q) at x=%d",
				i, box.items[i].text, box.items[i].at.X,
				i-1, box.items[i-1].text, box.items[i-1].at.X)
		}
	}

	// And every item shares the band's vertical centre: the sum is one line, not a staircase.
	band := scene.handMathRect(gs)
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
		w, _ := text.Measure(handShout(hand.Name), mathFace(gs, mathNameSize), 0)
		if w > widest {
			widest, name = w, handShout(hand.Name)
		}
	}

	// The breath swells it, and the shout is faux-bold, so what has to fit is the widest it is
	// ever actually drawn rather than its resting width.
	widest = widest*(1+mathBreathAmount) + mathBoldStep(mathNameSize)

	if widest > float64(gs.ScreenWidth) {
		t.Errorf("%q is %.0f wide at the shout's %d points, against a %d-wide screen",
			name, widest, mathNameSize, gs.ScreenWidth)
	}
}

// **The name's second line is the sum's own multiplier, said early** *(2026-08-19, owner's call)*.
// The banner writes `1.15x DMG` under the hand's name from the moment the hand forms, and the same
// figure flies out of that word into the line when the hand fires — so the two go through one
// formatting. Two spellings of the same multiplier would read as two numbers, which is exactly the
// failure `handShout` exists to prevent for the name above it.
func TestTheHandNameCarriesTheMultiplierTheSumWillShow(t *testing.T) {
	if got := handMultiplierLine(115); got != "1.15x DMG" {
		t.Errorf("115%% reads %q, want %q", got, "1.15x DMG")
	}

	for _, hand := range combat.Hands() {
		line := handMultiplierLine(hand.Multiplier)
		term := handMultiplierText(hand.Multiplier)
		if !strings.HasPrefix(line, term) {
			t.Errorf("%s is planned as %q and fires as %q", hand.Name, line, term)
		}
	}
}

// **The multiplier sets off at its own size and a card's figure grows into place.** The two are
// different gestures for a reason and the difference is checkable without a window: a card's
// figure is appearing — it comes toward the reader out of the card that paid it — while the
// multiplier has been sitting under the hand's name since DUEL! and is simply travelling. A
// multiplier that grew on the way would read as a second copy of a figure already on screen.
func TestTheMultiplierLeavesTheBannerAtItsOwnSize(t *testing.T) {
	items := mathScript(handEvent("concept-pair", []int{20, 20}, 150, 60))

	mult := items[len(items)-3]
	if mult.text != "1.5" {
		t.Fatalf("the multiplier is item %q, want %q", mult.text, "1.5")
	}
	if mult.fromScale != 1 {
		t.Errorf("the multiplier sets off at %v, want 1", mult.fromScale)
	}
	if items[0].fromScale != 0 {
		t.Errorf("a card's figure sets off at %v, want the flying default", items[0].fromScale)
	}
}

// **Every ring that fired says its own figure beside the term it priced, not on the card.**
//
// The figure moves between the cards of one blow — the first fire card steps a growing ring and the
// second is counted bigger — so it is a fact about the term. And a card face carries no ring at all
// now, so the sum is the only place any of it can be seen.
func TestTheScriptAnnotatesEveryRingThatFired(t *testing.T) {
	e := handEvent("concept-pair", []int{40, 44}, 150, 126)
	e.HandRingScale[0] = [combat.MaxWornRings]int{200, 100}
	e.HandRingScale[1] = [combat.MaxWornRings]int{200, 110}

	got := scriptText(mathScript(e))
	if want := "40 2x 1x + 44 2x 1.1x x 1.5 = 126"; got != want {
		t.Errorf("the script reads %q, want %q", got, want)
	}
}

// **A ring that did not fire says nothing**, which is the only thing the zero means. A flat ring on
// a card its predicate does not match has no beat and no figure.
func TestARingThatDidNotFireIsNotInTheScript(t *testing.T) {
	e := handEvent("concept-pair", []int{20, 20}, 150, 60)
	e.HandRingScale[0] = [combat.MaxWornRings]int{}
	e.HandRingScale[1] = [combat.MaxWornRings]int{}

	got := scriptText(mathScript(e))
	if want := "20 + 20 x 1.5 = 60"; got != want {
		t.Errorf("the script reads %q, want %q", got, want)
	}
}

// **A ring firing at the identity still says so.** A fresh Enflamed is 1x and its card bounces on
// that beat; a bounce with no figure beside it would be a card jumping for no stated reason, and the
// climb off 1x is the thing the player is meant to watch.
func TestARingFiringAtTheIdentityStillSaysSo(t *testing.T) {
	e := handEvent("concept-pair", []int{20, 20}, 150, 60)
	e.HandRingScale[0] = [combat.MaxWornRings]int{100}
	e.HandRingScale[1] = [combat.MaxWornRings]int{100}

	got := scriptText(mathScript(e))
	if want := "20 1x + 20 1x x 1.5 = 60"; got != want {
		t.Errorf("the script reads %q, want %q", got, want)
	}
}

// **Every figure flies out of the thing that produced it**, the rings included *(owner's call,
// 2026-08-26)*. A multiplier appearing beside a term it had no visible part in was the one number on
// the line with no source.
//
// What this holds is the bookkeeping that makes that safe: `startHandMath` pairs the flying items
// with `HandCards` in order, so a ring's figure has to be tellable from a card's or every figure
// after the first ring sets off from the wrong card. `ringSeat` is that mark.
func TestEveryRingFigureFliesFromItsOwnRing(t *testing.T) {
	e := handEvent("concept-pair", []int{20, 22}, 150, 63)
	e.HandRingScale[0] = [combat.MaxWornRings]int{200, 110}
	e.HandRingScale[1] = [combat.MaxWornRings]int{200, 120}

	cards, rings := 0, 0
	for _, it := range mathScript(e) {
		if !it.fly {
			continue
		}
		if it.ringSeat > 0 {
			rings++
			continue
		}
		cards++
	}

	// Two card figures and the hand multiplier.
	if cards != 3 {
		t.Errorf("%d non-ring items fly, want 3 — the walk would pair figures with the wrong cards", cards)
	}
	// Two rings on each of two terms.
	if rings != 4 {
		t.Errorf("%d ring figures fly, want 4", rings)
	}
}

// A ring's figure has to name the seat it sets off from, and the seats have to be the ones that
// fired — a figure flying out of an empty finger is worse than one that simply appeared.
func TestARingFigureNamesTheSeatItFliesFrom(t *testing.T) {
	e := handEvent("concept-pair", []int{20, 20}, 150, 60)
	e.HandRingScale[0] = [combat.MaxWornRings]int{0, 0, 250}

	var seats []int
	for _, it := range mathScript(e) {
		if it.ringSeat > 0 {
			seats = append(seats, it.ringSeat-1)
		}
	}

	if len(seats) != 1 || seats[0] != 2 {
		t.Errorf("the ring figures fly from seats %v, want just seat 2", seats)
	}
}

// **The script is the sequencing.** The box runs its items strictly one at a time, so a card's
// figure landing before its rings' figures is a property of the order they are written in — the
// order the engine applied them.
func TestTheRingFiguresFollowTheirOwnTerm(t *testing.T) {
	e := handEvent("concept-pair", []int{20, 22}, 150, 63)
	e.HandRingScale[0] = [combat.MaxWornRings]int{200}
	e.HandRingScale[1] = [combat.MaxWornRings]int{210}

	got := scriptText(mathScript(e))
	if want := "20 2x + 22 2.1x x 1.5 = 63"; got != want {
		t.Errorf("the script reads %q, want %q", got, want)
	}
}

// **Everything in the sum is accompanied by a card shaking** *(owner's call, 2026-08-26)*: a card's
// damage shakes the card, a ring's multiplier shakes that ring, and an echo's extra term shakes the
// ring that bought the landing even though it has no figure on the line.
//
// This checks the script side of that — which item names what. The beat it happens on is the box's
// own item cursor, which no test without a window can reach.
func TestEachItemNamesWhatShakesWithIt(t *testing.T) {
	e := handEvent("concept-pair", []int{20, 14}, 150, 51)
	e.HandCards[0], e.HandCards[1] = 3, 3
	e.HandRingScale[1] = [combat.MaxWornRings]int{0, 180}
	e.HandLanding[1] = [combat.MaxWornRings]bool{true}

	var cards, rings []int
	for _, it := range mathScript(e) {
		if it.cardSeat > 0 {
			cards = append(cards, it.cardSeat-1)
		}
		if it.ringSeat > 0 {
			rings = append(rings, it.ringSeat-1)
		}
	}

	// mathScript fills neither: the seats are a fact about the table and the row, which is
	// startHandMath's half of the box. What it does fill is the ring the figure flies out of.
	if len(cards) != 0 {
		t.Errorf("mathScript filled card seats %v; that is startHandMath's job", cards)
	}
	if len(rings) != 1 || rings[0] != 1 {
		t.Errorf("the ring figures name seats %v, want just seat 1", rings)
	}
}
