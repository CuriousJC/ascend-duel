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
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// These are the narrow kind of screen test CLAUDE.md allows: they create no `ebiten.Image`, need
// no window, and they guard a cross-package invariant a compiler cannot see — that what the hand
// dialog writes on the screen is the arithmetic the resolver put on the event.
//
// **They test `hitScript`, which is the half of the box that has no geometry in it.** Where each
// figure flies *from* is a question about a row of cards on a screen; what the figures *say* is
// checkable, and it is the part that could quietly start lying about the round.

// handEvent builds a KindHand event by hand, standing in for one the resolver produced. Each hit is
// its card's figure times the multiplier.
func handEvent(hand string, amounts []int, multiplier, total int) combat.Event {
	id, ok := combat.HandIDForKey(hand)
	if !ok {
		panic("the catalog has no hand keyed " + hand)
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
		e.HitAmounts[i] = a * multiplier / 100
		e.HandCardCount++
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

// linesOf is every hit's line of an event, as the box writes them.
func linesOf(e combat.Event) []string {
	var out []string
	for i := 0; i < e.HandCardCount; i++ {
		out = append(out, scriptText(hitScript(e, i, i == 0)))
	}
	return out
}

func sameLines(t *testing.T, what string, got, want []string) {
	t.Helper()
	if strings.Join(got, " | ") != strings.Join(want, " | ") {
		t.Errorf("%s reads %q, want %q", what, got, want)
	}
}

// **Every hit is its own line, and every line but the first carries its plus on the left**
// *(owner's call)*. The hits add up across the table; the plus rides the line rather than taking a
// row of its own.
func TestEveryHitIsItsOwnLine(t *testing.T) {
	sameLines(t, "a Pair of Skewers", linesOf(handEvent("pair", []int{20, 20}, 150, 60)),
		[]string{"20 x 1.5 = 30", "+ 20 x 1.5 = 30"})
}

// A four-card hand is four lines, each multiplied on its own.
func TestAFourCardHandIsFourLines(t *testing.T) {
	sameLines(t, "a Four of a Kind",
		linesOf(handEvent("concept-four-of-a-kind", []int{20, 20, 20, 20}, 500, 400)),
		[]string{"20 x 5 = 100", "+ 20 x 5 = 100", "+ 20 x 5 = 100", "+ 20 x 5 = 100"})
}

// **An echo is three lines under one card**, each its own hit — the card seats the same index for
// every landing, and each landing is multiplied by the hand.
func TestAnEchoedCardIsALinePerLanding(t *testing.T) {
	e := handEvent("pair", []int{30, 20, 10, 30}, 150, 135)
	e.HandCards[1], e.HandCards[2], e.HandCards[3] = 0, 0, 1

	sameLines(t, "an echoed Pair", linesOf(e),
		[]string{"30 x 1.5 = 45", "+ 20 x 1.5 = 30", "+ 10 x 1.5 = 15", "+ 30 x 1.5 = 45"})
}

// **Every line reads the same shape, the identity multiplier included** *(owner's call)*: hands
// are going to be upgradable, so the No Hand's 1 is a number that will change.
func TestTheNoHandShowsItsMultiplier(t *testing.T) {
	sameLines(t, "a No Hand", linesOf(handEvent("no-hand", []int{20}, 100, 20)),
		[]string{"20 x 1 = 20"})
}

// **Every rung the engine names is shouted, the No Hand included.** What is withheld from it is the
// *lift*, not the word — see builtARung. An event naming no hand at all is still silent.
func TestEveryRungIsShoutedIncludingTheNoHand(t *testing.T) {
	if got := shoutFor(handEvent("pair", []int{20, 20}, 150, 60)); got != "PAIR!" {
		t.Errorf("a Pair shouts %q, want %q", got, "PAIR!")
	}
	if got := shoutFor(handEvent("no-hand", []int{20}, 100, 20)); got != "NO HAND!" {
		t.Errorf("a No Hand shouts %q, want %q", got, "NO HAND!")
	}
	if got := shoutFor(combat.Event{Kind: combat.KindHand, Hand: combat.HandNone}); got != "" {
		t.Errorf("an event naming no hand shouts %q, want silence", got)
	}
}

// **Every hand in the catalog can be shouted and none of them is empty.**
func TestEveryHandInTheCatalogHasAShout(t *testing.T) {
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

// **Each line flies its card's figure and the multiplier, and says which is which.**
// `placeFigures` seats every flying item by its mark rather than by counting, so a line missing a
// mark would seat a figure on nothing.
func TestEveryLineMarksItsCardAndItsMultiplier(t *testing.T) {
	e := handEvent("concept-three-of-a-kind", []int{10, 12, 14}, 200, 72)
	for i := 0; i < e.HandCardCount; i++ {
		cardTerms, mults, flies := 0, 0, 0
		for _, it := range hitScript(e, i, i == 0) {
			if it.fly {
				flies++
			}
			if it.cardTerm {
				cardTerms++
				if it.text != strconv.Itoa(e.HandAmounts[i]) {
					t.Errorf("hit %d's card figure reads %q, want %d", i, it.text, e.HandAmounts[i])
				}
			}
			if it.handMult {
				mults++
			}
		}
		if cardTerms != 1 || mults != 1 || flies != 2 {
			t.Errorf("hit %d marks %d card figures and %d multipliers over %d flights, want one each and two",
				i, cardTerms, mults, flies)
		}
	}
}

// **Each line ends with its hit's own figure, and the figure is the event's.** Nothing in the box
// may recompute a total: the figure shown and the figure landed have to be one number.
func TestEveryLineEndsWithItsHitsOwnFigure(t *testing.T) {
	e := handEvent("pair", []int{7, 7}, 150, 20)
	e.HitAmounts[1] = 11 // what the resolver's own rounding could leave, where 7 x 1.5 is 10

	for i := 0; i < e.HandCardCount; i++ {
		items := hitScript(e, i, i == 0)
		if last := items[len(items)-1]; last.text != strconv.Itoa(e.HitAmounts[i]) {
			t.Errorf("hit %d ends with %q, want the event's %d", i, last.text, e.HitAmounts[i])
		}
	}
}

// **A raise on the duelist is never a term in a line**: the cards kept back and the purse are
// inside each card's figure already, so a line is the card, the multiplier and the hit.
func TestARaiseOnTheDuelistIsNoTermInALine(t *testing.T) {
	e := handEvent("pair", []int{30, 30}, 100, 60)
	e.HeldDMG, e.HeldDMGSeats = 10, []bool{true}
	e.VitaeDMG, e.VitaeDMGSeats = 5, []bool{false, true}

	sameLines(t, "a Pair with two held cards and a purse relic", linesOf(e),
		[]string{"30 x 1 = 30", "+ 30 x 1 = 30"})
}

// --- where the lines sit --------------------------------------------------------------------

// **It needs a font, which is the one thing in this file that costs anything.** `LoadFontData`
// hands back bytes and `NewGoTextFaceSource` is pure Go parsing, so no `ebiten.Image` is created.
func mathTestState(t *testing.T) *state.GlobalState {
	t.Helper()

	src, err := text.NewGoTextFaceSource(bytes.NewReader(assets.LoadFontData()["kubasta"]))
	if err != nil {
		t.Fatalf("kubasta would not parse: %v", err)
	}
	return &state.GlobalState{
		ScreenWidth:  state.ScreenWidth,
		ScreenHeight: state.ScreenHeight,
		Fonts:        map[string]*text.GoTextFaceSource{"kubasta": src},
	}
}

// laidOut is a box of one line per hit, seated as the event says, measured.
func laidOut(t *testing.T, scene *CombatScene, e combat.Event) handMathBox {
	t.Helper()
	gs := mathTestState(t)

	box := handMathBox{side: combat.SideA}
	for i := 0; i < e.HandCardCount; i++ {
		box.columns = append(box.columns,
			mathColumn{hit: i, seat: e.HandCards[i], items: hitScript(e, i, i == 0)})
	}
	scene.layOutMath(gs, &box)
	return box
}

// **Every hit is a column of rows under its own card, and the answer is at the bottom.** Within a
// row each item rests to the right of the one before it; each row sits below the last; a row
// starts at every operator over the whole term; and each row is centered on the card that threw
// the hit.
func TestEveryHitIsStackedDownwardUnderItsCard(t *testing.T) {
	gs := mathTestState(t)
	var scene CombatScene
	e := handEvent("pair", []int{20, 20}, 150, 60)
	box := laidOut(t, &scene, e)

	for c, col := range box.columns {
		rows := mathRows(col.items)
		if len(rows) < 3 {
			t.Fatalf("hit %d is %d rows, want the term, the multiplier and the answer apart", c, len(rows))
		}
		card := scene.handCardCenter(gs, combat.SideA, col.seat).X
		for r, row := range rows {
			for i := 1; i < len(row); i++ {
				if row[i].at.X <= row[i-1].at.X {
					t.Fatalf("hit %d row %d: %q rests at x=%d, not right of %q at x=%d",
						c, r, row[i].text, row[i].at.X, row[i-1].text, row[i-1].at.X)
				}
				if row[i].at.Y != row[0].at.Y {
					t.Errorf("hit %d row %d: %q is off the row's line", c, r, row[i].text)
				}
			}
			if r > 0 && row[0].at.Y <= rows[r-1][0].at.Y {
				t.Errorf("hit %d row %d sits at y=%d, not below row %d", c, r, row[0].at.Y, r-1)
			}
			mid := (row[0].at.X + row[len(row)-1].at.X) / 2
			if diff := mid - card; diff < -cardWidth || diff > cardWidth {
				t.Errorf("hit %d row %d is centered near x=%d, want it under its card at x=%d", c, r, mid, card)
			}
		}
		last := rows[len(rows)-1]
		if end, total := last[len(last)-1], col.total(); end.text != total.text || end.at != total.at {
			t.Errorf("hit %d does not end on its answer", c)
		}
	}
	if box.columns[0].items[0].at.X >= box.columns[1].items[0].at.X {
		t.Error("the second card's hit does not sit to the right of the first's")
	}
}

// **A card that lands several times stacks its hits under itself**, each one starting below the
// last row of the one before, and another card's first hit starts back at the top.
func TestAnEchoedCardsHitsStackUnderIt(t *testing.T) {
	gs := mathTestState(t)
	var scene CombatScene
	e := handEvent("pair", []int{30, 20, 10, 30}, 150, 135)
	e.HandCards[1], e.HandCards[2], e.HandCards[3] = 0, 0, 1
	box := laidOut(t, &scene, e)

	bottom := scene.handCardCenter(gs, combat.SideA, 0).Y + cardHeight/2
	first := box.columns[0].items[0].at.Y
	if first <= bottom {
		t.Errorf("the first hit sits at y=%d, not under its card's bottom edge at %d", first, bottom)
	}
	for c := 1; c < 3; c++ {
		above := box.columns[c-1].total().at.Y
		if got := box.columns[c].items[0].at.Y; got <= above {
			t.Errorf("hit %d starts at y=%d, not below the answer of the hit above it at %d", c, got, above)
		}
	}
	if got := box.columns[3].items[0].at.Y; got != first {
		t.Errorf("the other card's hit starts at y=%d, want the top row at %d", got, first)
	}
}

// **An echoed card builds its hits up one after another** *(owner's call)*: its second hit does not
// begin until its first is totaled, while another card's hit runs alongside the first.
func TestAnEchoedCardsHitsBuildUpInTurn(t *testing.T) {
	var scene CombatScene
	e := handEvent("pair", []int{30, 20, 10, 30}, 150, 135)
	e.HandCards[1], e.HandCards[2], e.HandCards[3] = 0, 0, 1
	box := laidOut(t, &scene, e)
	box.active = true

	box.Tick()
	if !box.columns[0].begun || !box.columns[3].begun {
		t.Fatal("the first hit of each card has not begun")
	}
	for !box.columns[0].done() {
		if box.columns[1].begun || box.columns[2].begun {
			t.Fatal("the card's second or third hit began before its first was totaled")
		}
		box.Tick()
	}
	box.Tick()
	if !box.columns[1].begun || box.columns[2].begun {
		t.Errorf("once the first is totaled the second should run and the third wait: begun %v %v",
			box.columns[1].begun, box.columns[2].begun)
	}
}

// --- what became of each hit -----------------------------------------------------------------

// **A line finds its own hit's outcome by reading ahead**, and stops at the end of the hits: a
// hit that was never thrown has no outcome to find.
func TestEachLineFindsItsOwnOutcome(t *testing.T) {
	log := []combat.Event{
		{Kind: combat.KindHand},
		{Kind: combat.KindMissed, Hit: 0},
		{Kind: combat.KindDamage, Hit: 1},
		{Kind: combat.KindStatus, Hit: 1},
		{Kind: combat.KindDamage, Hit: 2},
		{Kind: combat.KindDefeated},
		{Kind: combat.KindRoundEnd},
		{Kind: combat.KindDamage, Hit: 3},
	}
	got := hitOutcomes(log, 0)
	for hit, want := range map[int]int{0: 1, 1: 2, 2: 4} {
		if got[hit] != want {
			t.Errorf("hit %d's outcome is at %d, want %d", hit, got[hit], want)
		}
	}
	if _, ok := got[3]; ok {
		t.Error("a hit past the end of this turn's hits was found")
	}
}

// --- the hand's name -------------------------------------------------------------------------

// **The longest hand name in the catalog fits the screen at the size it is shouted.** It is
// centered on the hand row and does not wrap, so a name too wide runs off *both* edges at once.
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

// **The name's second line is every hit's multiplier, said early** *(owner's call)*, so the two go
// through one formatting.
func TestTheHandNameCarriesTheMultiplierTheLinesWillShow(t *testing.T) {
	if got := handMultiplierLine(115); got != "1.15x" {
		t.Errorf("115%% reads %q, want %q", got, "1.15x")
	}
	if !systems.FigureCovers(handMultiplierLine(115)) {
		t.Error("the multiplier line is not all figure glyphs, so it would be drawn in the font")
	}

	for _, hand := range combat.Hands() {
		line := handMultiplierLine(hand.Multiplier)
		term := ui.HandMultiplierText(hand.Multiplier)
		if !strings.HasPrefix(line, term) {
			t.Errorf("%s is planned as %q and fires as %q", hand.Name, line, term)
		}
	}
}

// **The multiplier sets off at its own size and a card's figure grows into place.** The multiplier
// has been sitting under the hand's name since DUEL! and is simply traveling; a card's figure is
// appearing out of the card.
func TestTheMultiplierLeavesTheBannerAtItsOwnSize(t *testing.T) {
	for _, it := range hitScript(handEvent("pair", []int{20, 20}, 150, 60), 0, true) {
		switch {
		case it.handMult && it.fromScale != 1:
			t.Errorf("the multiplier sets off at %v, want 1", it.fromScale)
		case it.cardTerm && it.fromScale != 0:
			t.Errorf("a card's figure sets off at %v, want the flying default", it.fromScale)
		}
	}
}

// --- relics ----------------------------------------------------------------------------------

// **Every relic that fired says its own figure inside the hit it priced.**
func TestEveryRelicThatFiredIsAFactorInItsHit(t *testing.T) {
	e := handEvent("pair", []int{40, 44}, 150, 126)
	e.HandRelicScale[0] = []int{200, 100}
	e.HandRelicScale[1] = []int{200, 110}

	sameLines(t, "two relics on two hits", linesOf(e),
		[]string{"40 x 2 x 1 x 1.5 = 60", "+ 44 x 2 x 1.1 x 1.5 = 66"})
}

// **A relic that did not fire says nothing**, which is the only thing the zero means.
func TestARelicThatDidNotFireIsNotInTheLine(t *testing.T) {
	e := handEvent("pair", []int{20, 20}, 150, 60)
	e.HandRelicScale[0] = []int{}
	e.HandRelicScale[1] = []int{0}

	sameLines(t, "a relic that did not fire", linesOf(e),
		[]string{"20 x 1.5 = 30", "+ 20 x 1.5 = 30"})
}

// A relic's figure names the seat it sets off from, and the seats are the ones that fired.
func TestARelicFigureNamesTheSeatItFliesFrom(t *testing.T) {
	e := handEvent("pair", []int{20, 20}, 150, 60)
	e.HandRelicScale[0] = []int{0, 0, 250}

	var seats []int
	for _, it := range hitScript(e, 0, true) {
		if it.relicSeat > 0 {
			seats = append(seats, it.relicSeat-1)
		}
	}
	if len(seats) != 1 || seats[0] != 2 {
		t.Errorf("the relic figures fly from seats %v, want just seat 2", seats)
	}
}

// **The script names the relic; the table's seats are the screen's.** `hitScript` fills the relic a
// figure flies out of and leaves every played card's seat to `placeFigures`.
func TestTheScriptNamesRelicsAndLeavesCardSeatsToTheScreen(t *testing.T) {
	e := handEvent("pair", []int{20, 14}, 150, 51)
	e.HandRelicScale[1] = []int{0, 180}
	e.HandLanding[1] = []bool{true}

	var cards, relics []int
	for _, it := range hitScript(e, 1, false) {
		if it.cardSeat > 0 {
			cards = append(cards, it.cardSeat-1)
		}
		if it.relicSeat > 0 {
			relics = append(relics, it.relicSeat-1)
		}
	}
	if len(cards) != 0 {
		t.Errorf("hitScript filled card seats %v; that is placeFigures' job", cards)
	}
	if len(relics) != 1 || relics[0] != 1 {
		t.Errorf("the relic figures name seats %v, want just seat 1", relics)
	}
}

// --- the product inside a term ---------------------------------------------------------------

// splitEvent is a hand whose terms can be written as the product the game worked out: one DMG, and
// a percentage per card. `pcts` are the cards' own multipliers, 300 being a 3x card.
func splitEvent(hand string, dmg int, pcts []int, multiplier, total int) combat.Event {
	amounts := make([]int, len(pcts))
	for i, pct := range pcts {
		amounts[i] = dmg * pct / 100
	}
	e := handEvent(hand, amounts, multiplier, total)
	e.HandDMG, e.HandDMGBare = dmg, dmg
	for i, pct := range pcts {
		e.HandCardPct[i] = pct
		e.HandCardBase[i] = amounts[i]
	}
	return e
}

// **A term is the product, not its answer** *(owner's call)*.
func TestATermIsWrittenAsDMGTimesTheCardsMultiplier(t *testing.T) {
	sameLines(t, "a 3x card on 12 DMG", linesOf(splitEvent("no-hand", 12, []int{300}, 100, 36)),
		[]string{"12 x 3 x 1 = 36"})
}

// Every hit takes its own multiple of the same DMG, which is what each term row says.
func TestEveryHitSwingsAtTheSameDMG(t *testing.T) {
	sameLines(t, "a Pair", linesOf(splitEvent("pair", 12, []int{300, 100}, 100, 48)),
		[]string{"12 x 3 x 1 = 36", "+ 12 x 1 x 1 = 12"})
}

// **A relic is a factor in the term.**
func TestARelicIsAFactorInsideTheTerm(t *testing.T) {
	e := splitEvent("no-hand", 12, []int{300}, 100, 72)
	e.HandRelicScale[0] = []int{200}
	e.HitAmounts[0] = 72

	sameLines(t, "a doubling relic", linesOf(e), []string{"12 x 3 x 2 x 1 = 72"})
}

// **A term whose split does not come to the term is written flat** — see combat.Event.TermSplit.
func TestATermThatDoesNotSplitIsWrittenFlat(t *testing.T) {
	e := splitEvent("no-hand", 12, []int{300}, 100, 36)
	e.HandCardBase[0] = 35 // what an echo's rounding looks like from here

	sameLines(t, "a term that does not split", linesOf(e), []string{"36 x 1 = 36"})
}

// **The DMG figure flies out of the duelist and the card's multiplier out of the card.**
func TestTheDMGFigureBelongsToTheDuelist(t *testing.T) {
	e := splitEvent("pair", 12, []int{300, 100}, 100, 48)
	for i := 0; i < e.HandCardCount; i++ {
		duelist := 0
		for _, it := range hitScript(e, i, i == 0) {
			if it.fly && it.fromDuelist {
				duelist++
				if it.text != "12" {
					t.Errorf("hit %d: the duelist's figure reads %q, want the DMG", i, it.text)
				}
			}
		}
		if duelist != 1 {
			t.Errorf("hit %d: %d figures came off the duelist, want one", i, duelist)
		}
	}
}

// **A line has three levels and the air says which is which** *(owner's call)*: a product inside a
// term is set close, terms added to one another are set apart by the line's own gap, and the
// multiplier that applies to all of them is set further apart again.
func TestALineIsSetInThreeLevelsOfAir(t *testing.T) {
	e := splitEvent("pair", 12, []int{300, 200}, 100, 60)
	items := hitScript(e, 1, false)

	at := func(text string, nth int) int {
		for i, it := range items {
			if it.text == text {
				if nth == 0 {
					return i
				}
				nth--
			}
		}
		t.Fatalf("the line has no %q at that count: %q", text, scriptText(items))
		return 0
	}

	if got := gapBefore(items, at("2", 0)); got != mathTightGap {
		t.Errorf("a card's multiplier is set %v off the DMG, want the tight %v", got, mathTightGap)
	}

	last := 0
	for i, it := range items {
		if it.text == "x" {
			last = i
		}
	}
	if got := gapBefore(items, last); got != mathWideGap {
		t.Errorf("the hand's multiplier is set %v off the terms, want the wide %v", got, mathWideGap)
	}
	if got := gapBefore(items, at("=", 0)); got != mathWideGap {
		t.Errorf("the answer is set %v off the line, want the wide %v", got, mathWideGap)
	}

	if !(mathTightGap < mathItemGap && mathItemGap < mathWideGap) {
		t.Errorf("the three gaps are %v, %v, %v — they have to be three widths in that order",
			mathTightGap, mathItemGap, mathWideGap)
	}
}
