package achieve

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"

	// **Imported for its side effect as much as for its cards.** Registering the enemy decks is
	// what puts a genuinely formless, colourless card in the registry, and a formless card is the
	// case every absence rule here turns on. `combat.Card{}` is not one — concept zero is a real
	// player card with a real form.
	"github.com/curiousjc/ascend-duel/internal/decks"
)

// drabCard is an enemy's card: no form, no colour. The absence, rather than a card that happens to
// be the zero value.
func drabCard(t *testing.T) combat.Card {
	t.Helper()
	records := decks.EnemyRecords()
	if len(records) == 0 {
		t.Fatal("no enemy decks registered")
	}
	cards := decks.EnemyCards(records[0])
	if len(cards) == 0 {
		t.Fatalf("%s has no cards", records[0])
	}
	c := cards[0]
	if c.Form() != combat.FormNone || c.Element != combat.Basic {
		t.Fatalf("an enemy card is meant to carry neither a form nor a colour, got %v/%v",
			c.Form(), c.Element)
	}
	return c
}

// card is a player card of a named concept and element, which is what a turn is made of.
func card(label string, e combat.Element) combat.Card {
	id, ok := combat.ConceptByKey(label)
	if !ok {
		panic("no player concept " + label)
	}
	return combat.Of(id, e)
}

// TestTheCatalogueLoads is the whole validation pass, run as a test rather than only at launch.
//
// **It is the one that matters most**, because every other check in this file is about a hand-built
// fixture and this one is about the file that ships. `load` panics on a bad record, so a failure
// here is the catalogue actually being wrong.
func TestTheCatalogueLoads(t *testing.T) {
	all := Loaded().All()
	if len(all) == 0 {
		t.Fatal("the catalogue is empty")
	}
	for _, a := range all {
		if a.Key == "" || a.Name == "" || a.How == "" {
			t.Errorf("%q is missing a key, a name or a how", a.Key)
		}
		if len(a.Said) == 0 {
			t.Errorf("%q says nothing when it lands", a.Key)
		}
	}
}

// TestEveryShippedAchievementIsReachable is the failure this whole grammar exists to prevent: an
// achievement nobody can earn looks exactly like one nobody has earned yet, and no amount of playing
// tells the two apart.
//
// **A moment must be one the game raises, a counter one something bumps, and a turn pattern one a
// turn could satisfy.** The first two are checked at load; this is the third, and it is checked by
// actually building a turn that satisfies each one.
func TestEveryShippedAchievementIsReachable(t *testing.T) {
	// The widest legal turn: five cards, which is combat.MaxActions and the most a turn can hold.
	// Every turn achievement in the file has to be satisfiable inside a turn of that size.
	//
	// **This checks the pattern, not the price.** A turn here is a set of cards with no budget in
	// front of it, so a pattern that is expressible but unaffordable passes — godslayer is five
	// 4 AP cards against a 6 AP budget and is deliberately out of reach today. What the test
	// refuses is the other failure: a pattern no turn could satisfy at any price, which is a row
	// that looks identical to one nobody has earned yet.
	//
	// Four attack forms are not available at once — there are three — so this is three attack
	// forms across five elements, plus a defence, which is deliberately the hardest single turn the
	// deck can build.
	widest := []combat.Card{
		card("Jab", combat.Fire),
		card("Cut", combat.Ice),
		card("Bash", combat.Lightning),
		card("Thrust", combat.Earth),
		card("Ward", combat.Arcane),
	}

	// A one-form five-element turn, for Prism.
	prism := []combat.Card{
		card("Jab", combat.Fire),
		card("Thrust", combat.Ice),
		card("Poke", combat.Lightning),
		card("Lunge", combat.Earth),
		card("Impale", combat.Arcane),
	}

	// Five of one concept at the bottom of the stab ladder, for tiny-but-fierce. `Poke` is the
	// 0 AP rung `duelist_cards.json` ships with no copies — the deck reaches it by demoting Jabs.
	free := []combat.Card{
		card("Poke", combat.Fire),
		card("Poke", combat.Ice),
		card("Poke", combat.Lightning),
		card("Poke", combat.Earth),
		card("Poke", combat.Arcane),
	}

	// The same turn at the top of that ladder, for godslayer. Twenty AP, which no run can pay for
	// today; see the note above about what this test does and does not check.
	heaviest := []combat.Card{
		card("Impale", combat.Fire),
		card("Impale", combat.Ice),
		card("Impale", combat.Lightning),
		card("Impale", combat.Earth),
		card("Impale", combat.Arcane),
	}

	reachable := map[string]bool{}
	for _, turn := range [][]combat.Card{widest, prism, free, heaviest} {
		for _, key := range Loaded().ByTurn(turn) {
			reachable[key] = true
		}
	}

	for _, a := range Loaded().All() {
		if a.trigger.kind != data.TriggerTurn {
			continue
		}
		if !reachable[a.Key] {
			t.Errorf("%q is a turn achievement no turn in this test can satisfy; either the "+
				"pattern is wrong or the deck can no longer build it", a.Key)
		}
	}
}

// TestSpectrumIsAtLeastFourElements holds the owner's call of 2026-09-06: a five-element turn earns
// the four-element achievement too, because the two are rungs of one ladder.
func TestSpectrumIsAtLeastFourElements(t *testing.T) {
	five := []combat.Card{
		card("Jab", combat.Fire),
		card("Cut", combat.Ice),
		card("Bash", combat.Lightning),
		card("Thrust", combat.Earth),
		card("Nick", combat.Arcane),
	}
	earned := map[string]bool{}
	for _, k := range Loaded().ByTurn(five) {
		earned[k] = true
	}
	if !earned["spectrum"] {
		t.Error("five elements at once must also be four; spectrum is at-least, not exactly")
	}
	if !earned["elementalist"] {
		t.Error("five elements at once is the elementalist")
	}
}

// TestFourElementsIsNotFive is the other half of the same rule, and the one that would fail
// silently: a threshold read as ">= 4" everywhere would light up the five-element row on four cards.
func TestFourElementsIsNotFive(t *testing.T) {
	four := []combat.Card{
		card("Jab", combat.Fire),
		card("Cut", combat.Ice),
		card("Bash", combat.Lightning),
		card("Thrust", combat.Earth),
	}
	for _, k := range Loaded().ByTurn(four) {
		if k == "elementalist" {
			t.Error("four elements is not the elementalist")
		}
	}
}

// TestArsenalNeedsTheDefenceBesideTheThreeForms is the achievement the clause-level filter exists
// for: three attack forms is Weaponmaster, and Arsenal is that plus something the attack filter
// cannot see.
func TestArsenalNeedsTheDefenceBesideTheThreeForms(t *testing.T) {
	threeForms := []combat.Card{
		card("Jab", combat.Fire),
		card("Cut", combat.Fire),
		card("Bash", combat.Fire),
	}
	got := map[string]bool{}
	for _, k := range Loaded().ByTurn(threeForms) {
		got[k] = true
	}
	if !got["weaponmaster"] {
		t.Error("stab, slash and crush together is the weaponmaster")
	}
	if got["arsenal"] {
		t.Error("three attack forms with no defence is not the arsenal")
	}

	withDefence := append(append([]combat.Card{}, threeForms...), card("Ward", combat.Fire))
	got = map[string]bool{}
	for _, k := range Loaded().ByTurn(withDefence) {
		got[k] = true
	}
	if !got["arsenal"] {
		t.Error("three attack forms and a defence is the arsenal")
	}
}

// TestADefenceIsNotAnAttackForm is what stops Weaponmaster being earned by two attacks and a Ward.
// **Defend is a fourth form** and joins hands like anything else, so the only thing keeping it out
// of an attack-form count is the clause's own category filter.
func TestADefenceIsNotAnAttackForm(t *testing.T) {
	turn := []combat.Card{
		card("Jab", combat.Fire),
		card("Cut", combat.Fire),
		card("Ward", combat.Fire),
	}
	for _, k := range Loaded().ByTurn(turn) {
		if k == "weaponmaster" {
			t.Error("a defence is not a third attack form")
		}
	}
}

// TestPrismWantsOneShapeInEveryColour, both ways round: a form five-of-a-kind and a card
// five-of-a-kind are one achievement, which is what the pattern alternation is for.
func TestPrismWantsOneShapeInEveryColour(t *testing.T) {
	oneForm := []combat.Card{
		card("Jab", combat.Fire),
		card("Thrust", combat.Ice),
		card("Poke", combat.Lightning),
		card("Lunge", combat.Earth),
		card("Impale", combat.Arcane),
	}
	oneCard := []combat.Card{
		card("Strike", combat.Fire),
		card("Strike", combat.Ice),
		card("Strike", combat.Lightning),
		card("Strike", combat.Earth),
		card("Strike", combat.Arcane),
	}
	mixed := []combat.Card{
		card("Jab", combat.Fire),
		card("Cut", combat.Ice),
		card("Bash", combat.Lightning),
		card("Thrust", combat.Earth),
		card("Nick", combat.Arcane),
	}

	for _, turn := range [][]combat.Card{oneForm, oneCard} {
		found := false
		for _, k := range Loaded().ByTurn(turn) {
			if k == "prism" {
				found = true
			}
		}
		if !found {
			t.Error("one form or one card in all five colours is the prism")
		}
	}
	for _, k := range Loaded().ByTurn(mixed) {
		if k == "prism" {
			t.Error("five elements across three forms is not the prism")
		}
	}
}

// TestFloorReachedIsAThreshold: a run that arrives on the sixth floor has reached the fifth.
func TestFloorReachedIsAThreshold(t *testing.T) {
	if got := Loaded().ByMoment(FloorReached(4)); len(got) != 0 {
		t.Errorf("floor four earns nothing yet, got %v", got)
	}
	for _, floor := range []int{5, 6, 30} {
		found := false
		for _, k := range Loaded().ByMoment(FloorReached(floor)) {
			if k == "fifth-floor" {
				found = true
			}
		}
		if !found {
			t.Errorf("reaching floor %d must earn the fifth floor", floor)
		}
	}
}

// TestCardAlteredMatchesOnTheResultingCard, not on the worm. Several worms can arrive at a Flinch
// and the achievement is about the card that came out.
func TestCardAlteredMatchesOnTheResultingCard(t *testing.T) {
	found := false
	for _, k := range Loaded().ByMoment(CardAltered("Flinch")) {
		if k == "flinch" {
			found = true
		}
	}
	if !found {
		t.Error("a card becoming a Flinch is the flinch achievement")
	}
	if got := Loaded().ByMoment(CardAltered("Ward")); len(got) != 0 {
		t.Errorf("a Ward is not a Flinch, got %v", got)
	}
}

// TestCountersNameBothAxes: a played turn adds to the form tally and the concept tally, because the
// two questions a player asks are different ones.
func TestCountersNameBothAxes(t *testing.T) {
	turn := []combat.Card{
		card("Slice", combat.Fire),
		card("Cut", combat.Ice),
		card("Ward", combat.Earth),
	}
	got := CountersFor(turn)

	if got["form:slash"] != 2 {
		t.Errorf("a Slice and a Cut are two slashing cards, got %d", got["form:slash"])
	}
	if got["concept:Slice"] != 1 {
		t.Errorf("one of them is the card called Slice, got %d", got["concept:Slice"])
	}
	if got["form:defend"] != 1 {
		t.Errorf("a Ward is a defending card, got %d", got["form:defend"])
	}
	if _, ok := got["form:none"]; ok {
		t.Error("a counter must never name an absence")
	}
}

// TestAnEnemyCardCountsTowardNoForm, which is the same rule the hand matcher holds: FormNone and
// Basic are absences rather than values.
func TestAnEnemyCardCountsTowardNoForm(t *testing.T) {
	got := CountersFor([]combat.Card{drabCard(t)})
	for name := range got {
		if name == "form:none" {
			t.Error("a formless card contributes no form tally")
		}
	}
}

// TestCountsAreAThresholdNotACrossing. ByCounts is asked once at the end of a duel against settled
// figures, so it must report everything at or over its mark rather than what moved last.
func TestCountsAreAThresholdNotACrossing(t *testing.T) {
	if got := Loaded().ByCounts(map[string]int{"form:slash": 299}); len(got) != 0 {
		t.Errorf("299 is not 300, got %v", got)
	}
	for _, n := range []int{300, 301, 5000} {
		found := false
		for _, k := range Loaded().ByCounts(map[string]int{"form:slash": n}) {
			if k == "slash-300" {
				found = true
			}
		}
		if !found {
			t.Errorf("%d slashing cards is past 300", n)
		}
	}
}

// TestProgressOnlyMeansSomethingForATally. A turn either happened or did not and a moment is not a
// fraction of anything, so a row that is either done or not must say nothing rather than "0 / 1".
func TestProgressOnlyMeansSomethingForATally(t *testing.T) {
	counts := map[string]int{"form:slash": 141}
	for _, a := range Loaded().All() {
		got := a.Progress(counts)
		switch a.trigger.kind {
		case data.TriggerCount:
			if got == "" {
				t.Errorf("%q is a tally and should show progress", a.Key)
			}
		default:
			if got != "" {
				t.Errorf("%q is not a tally and showed %q", a.Key, got)
			}
		}
	}
	if a, ok := Loaded().Find("slash-300"); ok {
		if got := a.Progress(counts); got != "141 / 300" {
			t.Errorf("progress should read %q, got %q", "141 / 300", got)
		}
	}
}

// TestAnEmptyTurnEarnsNothing. `same` over no cards is vacuously true, which would make an
// achievement earned by playing nothing.
func TestAnEmptyTurnEarnsNothing(t *testing.T) {
	if got := Loaded().ByTurn(nil); len(got) != 0 {
		t.Errorf("an empty turn earns nothing, got %v", got)
	}
}

// TestABadRecordIsRefused walks the shapes a hand-edited file can take. **Every one of these is a
// launch failure by design** — see the package doc for why an achievement that can never land is
// worse than a game that will not start.
func TestABadRecordIsRefused(t *testing.T) {
	bad := []struct {
		name string
		in   data.TriggerData
	}{
		{"no kind", data.TriggerData{}},
		{"unknown kind", data.TriggerData{Kind: "vibes"}},
		{"turn with no patterns", data.TriggerData{Kind: data.TriggerTurn}},
		{"count with no counter", data.TriggerData{Kind: data.TriggerCount, N: 5}},
		{"count with no n", data.TriggerData{Kind: data.TriggerCount, Counter: "form:slash"}},
		{"counter with no prefix", data.TriggerData{
			Kind: data.TriggerCount, Counter: "slash", N: 5}},
		{"counter naming no form", data.TriggerData{
			Kind: data.TriggerCount, Counter: "form:wibble", N: 5}},
		{"counter naming no card", data.TriggerData{
			Kind: data.TriggerCount, Counter: "concept:Wibble", N: 5}},
		{"moment naming nothing", data.TriggerData{Kind: data.TriggerMoment, Moment: "wibble"}},
		{"floor with no floor", data.TriggerData{
			Kind: data.TriggerMoment, Moment: MomentFloorReached}},
		{"card-altered with no card", data.TriggerData{
			Kind: data.TriggerMoment, Moment: MomentCardAltered}},
		{"a moment carrying a counter", data.TriggerData{
			Kind: data.TriggerMoment, Moment: MomentDuelWon, Counter: "form:slash"}},
		{"a clause with no mode", data.TriggerData{
			Kind:     data.TriggerTurn,
			Patterns: []data.PatternData{{Clauses: []data.ClauseData{{Axis: "element"}}}}}},
		{"a clause with no axis", data.TriggerData{
			Kind: data.TriggerTurn,
			Patterns: []data.PatternData{{Clauses: []data.ClauseData{
				{Mode: data.ModeDistinct, N: 3}}}}}},
		{"a clause naming no axis", data.TriggerData{
			Kind: data.TriggerTurn,
			Patterns: []data.PatternData{{Clauses: []data.ClauseData{
				{Axis: "wibble", Mode: data.ModeDistinct, N: 3}}}}}},
		{"a clause naming no category", data.TriggerData{
			Kind: data.TriggerTurn,
			Patterns: []data.PatternData{{Clauses: []data.ClauseData{
				{Of: "wibble", Axis: "element", Mode: data.ModeDistinct, N: 3}}}}}},
		{"a distinct clause wanting one value", data.TriggerData{
			Kind: data.TriggerTurn,
			Patterns: []data.PatternData{{Clauses: []data.ClauseData{
				{Axis: "element", Mode: data.ModeDistinct, N: 1}}}}}},
		{"a count clause on an axis", data.TriggerData{
			Kind: data.TriggerTurn,
			Patterns: []data.PatternData{{Clauses: []data.ClauseData{
				{Axis: "element", Mode: data.ModeCount, N: 2}}}}}},
	}

	for _, c := range bad {
		if _, err := parseTrigger(c.in); err == nil {
			t.Errorf("%s should be refused at load", c.name)
		}
	}
}

// TestAnAxisReadsTheSameWayTheHandMatcherDoes. This package keeps its own axisValue because
// combat's is unexported, and a silent disagreement would mean an achievement counting a hand the
// ladder does not. The rule that matters is that FormNone and Basic are absences.
func TestAnAxisReadsTheSameWayTheHandMatcherDoes(t *testing.T) {
	drab := drabCard(t)
	if _, ok := axisValue(drab, combat.AxisForm); ok {
		t.Error("FormNone is an absence, not a form")
	}
	if _, ok := axisValue(drab, combat.AxisElement); ok {
		t.Error("Basic is an absence, not an element")
	}

	fire := card("Jab", combat.Fire)
	if v, ok := axisValue(fire, combat.AxisElement); !ok || v != int(combat.Fire) {
		t.Error("a fire card counts as fire")
	}
	if v, ok := axisValue(fire, combat.AxisForm); !ok || v != int(combat.FormStab) {
		t.Error("a Jab counts as a stab")
	}
}

// **A cost clause reads the card's cost, not its concept's** *(2026-09-09)*. That is what makes
// godslayer true of five promoted Lunges and false of five Impales a Whetworm has made cheap — the
// achievement is about what the turn actually cost, and Card.Cost is where a worm's CostDelta lands.
func TestACostClauseReadsTheCardRatherThanTheConcept(t *testing.T) {
	turn := func(delta int) []combat.Card {
		out := make([]combat.Card, 0, 5)
		for _, e := range []combat.Element{
			combat.Fire, combat.Ice, combat.Lightning, combat.Earth, combat.Arcane,
		} {
			c := card("Impale", e)
			c.CostDelta = delta
			out = append(out, c)
		}
		return out
	}

	earned := func(cards []combat.Card) bool {
		for _, k := range Loaded().ByTurn(cards) {
			if k == "godslayer" {
				return true
			}
		}
		return false
	}

	if !earned(turn(0)) {
		t.Error("five plain Impales are five 4 AP attacks of one card and must earn godslayer")
	}
	if earned(turn(-1)) {
		t.Error("five Impales a worm made 3 AP are not five 4 AP attacks")
	}

	// And the other direction: a Lunge a worm made dearer *is* a 4 AP attack.
	lunges := make([]combat.Card, 0, 5)
	for _, e := range []combat.Element{
		combat.Fire, combat.Ice, combat.Lightning, combat.Earth, combat.Arcane,
	} {
		c := card("Lunge", e)
		c.CostDelta = 1
		lunges = append(lunges, c)
	}
	if !earned(lunges) {
		t.Error("five Lunges a worm made 4 AP are five 4 AP attacks of one card")
	}
}

// **Zero is a real cost filter and not an absent one**, which is the whole reason ClauseData.Cost is
// a pointer. `duelist_cards.json` ships a 0 AP rung on every attack ladder, so "the free ones" is a
// question the grammar has to be able to ask.
func TestAZeroCostClauseFiltersRatherThanMatchingEverything(t *testing.T) {
	free := []combat.Card{
		card("Poke", combat.Fire), card("Poke", combat.Ice), card("Poke", combat.Lightning),
		card("Poke", combat.Earth), card("Poke", combat.Arcane),
	}
	paid := []combat.Card{
		card("Jab", combat.Fire), card("Jab", combat.Ice), card("Jab", combat.Lightning),
		card("Jab", combat.Earth), card("Jab", combat.Arcane),
	}

	earned := func(cards []combat.Card) bool {
		for _, k := range Loaded().ByTurn(cards) {
			if k == "tiny-but-fierce" {
				return true
			}
		}
		return false
	}

	if !earned(free) {
		t.Error("five Pokes are five free attacks of one card")
	}
	if earned(paid) {
		t.Error("five 1 AP Jabs are not free; a zero cost filter must not match every card")
	}
}

// **shields-raised is a threshold, like floor-reached.** Standing behind eleven earns the row that
// asked for ten — a player who overshot should not be missing the step they went past.
func TestTheShieldsMomentIsAThreshold(t *testing.T) {
	earned := func(n int) bool {
		for _, k := range Loaded().ByMoment(ShieldsRaised(n)) {
			if k == "invulnerable" {
				return true
			}
		}
		return false
	}

	if earned(9) {
		t.Error("nine shields is not ten")
	}
	if !earned(10) {
		t.Error("ten shields is the achievement")
	}
	if !earned(11) {
		t.Error("eleven shields must earn the row asking for ten")
	}
}
