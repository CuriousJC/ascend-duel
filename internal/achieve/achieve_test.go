package achieve

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"

	// **Imported for its side effect as much as for its cards.** Registering the enemy decks is
	// what puts a genuinely formless, colorless card in the registry, and a formless card is the
	// case every absence rule here turns on. `combat.Card{}` is not one — concept zero is a real
	// player card with a real form.
	"github.com/curiousjc/ascend-duel/internal/decks"
)

// drabCard is an enemy's card: no form, no color. The absence, rather than a card that happens to
// be the zero value.
func drabCard(t *testing.T) combat.Card {
	t.Helper()
	records := decks.EnemyRecords()
	if len(records) == 0 {
		t.Fatal("no enemy decks registered")
	}
	cards := decks.EnemyCards(records[0], "")
	if len(cards) == 0 {
		t.Fatalf("%s has no cards", records[0])
	}
	c := cards[0]
	if c.Form() != combat.FormNone || c.Element != combat.Basic {
		t.Fatalf("an enemy card is meant to carry neither a form nor a color, got %v/%v",
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

// TestTheCatalogLoads is the whole validation pass, run as a test rather than only at launch.
//
// **It is the one that matters most**, because every other check in this file is about a hand-built
// fixture and this one is about the file that ships. `load` panics on a bad record, so a failure
// here is the catalog actually being wrong.
func TestTheCatalogLoads(t *testing.T) {
	all := Loaded().All()
	if len(all) == 0 {
		t.Fatal("the catalog is empty")
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
	// front of it, so a pattern that is expressible but unaffordable passes. What the test refuses
	// is the other failure: a pattern no turn could satisfy at any price, which is a row
	// that looks identical to one nobody has earned yet.
	//
	// Four attack forms are not available at once — there are three — so this is three attack
	// forms across five elements, plus a defense, which is deliberately the hardest single turn the
	// deck can build.
	widest := []combat.Card{
		card("Jab", combat.Fire),
		card("Cut", combat.Ice),
		card("Thump", combat.Lightning),
		card("Thrust", combat.Earth),
		card("Brace", combat.Arcane),
	}

	// A one-form five-element turn, for Prism.
	prism := []combat.Card{
		card("Jab", combat.Fire),
		card("Thrust", combat.Ice),
		card("Jab", combat.Lightning),
		card("Skewer", combat.Earth),
		card("Thrust", combat.Arcane),
	}

	reachable := map[string]bool{}
	for _, turn := range [][]combat.Card{widest, prism} {
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
		card("Thump", combat.Lightning),
		card("Thrust", combat.Earth),
		card("Slice", combat.Arcane),
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
		card("Thump", combat.Lightning),
		card("Thrust", combat.Earth),
	}
	for _, k := range Loaded().ByTurn(four) {
		if k == "elementalist" {
			t.Error("four elements is not the elementalist")
		}
	}
}

// TestArsenalNeedsTheDefenseBesideTheThreeForms is the achievement the clause-level filter exists
// for: three attack forms is Weaponmaster, and Arsenal is that plus something the attack filter
// cannot see.
func TestArsenalNeedsTheDefenseBesideTheThreeForms(t *testing.T) {
	threeForms := []combat.Card{
		card("Jab", combat.Fire),
		card("Cut", combat.Fire),
		card("Thump", combat.Fire),
	}
	got := map[string]bool{}
	for _, k := range Loaded().ByTurn(threeForms) {
		got[k] = true
	}
	if !got["weaponmaster"] {
		t.Error("stab, slash and crush together is the weaponmaster")
	}
	if got["arsenal"] {
		t.Error("three attack forms with no defense is not the arsenal")
	}

	withDefense := append(append([]combat.Card{}, threeForms...), card("Brace", combat.Fire))
	got = map[string]bool{}
	for _, k := range Loaded().ByTurn(withDefense) {
		got[k] = true
	}
	if !got["arsenal"] {
		t.Error("three attack forms and a defense is the arsenal")
	}
}

// TestADefenseIsNotAnAttackForm is what stops Weaponmaster being earned by two attacks and a Brace.
// **Defend is a fourth form** and joins hands like anything else, so the only thing keeping it out
// of an attack-form count is the clause's own category filter.
func TestADefenseIsNotAnAttackForm(t *testing.T) {
	turn := []combat.Card{
		card("Jab", combat.Fire),
		card("Cut", combat.Fire),
		card("Brace", combat.Fire),
	}
	for _, k := range Loaded().ByTurn(turn) {
		if k == "weaponmaster" {
			t.Error("a defense is not a third attack form")
		}
	}
}

// TestPrismWantsOneShapeInEveryColor, both ways round: a form five-of-a-kind and a card
// five-of-a-kind are one achievement, which is what the pattern alternation is for.
func TestPrismWantsOneShapeInEveryColor(t *testing.T) {
	oneForm := []combat.Card{
		card("Jab", combat.Fire),
		card("Thrust", combat.Ice),
		card("Jab", combat.Lightning),
		card("Skewer", combat.Earth),
		card("Thrust", combat.Arcane),
	}
	oneCard := []combat.Card{
		card("Bash", combat.Fire),
		card("Bash", combat.Ice),
		card("Bash", combat.Lightning),
		card("Bash", combat.Earth),
		card("Bash", combat.Arcane),
	}
	mixed := []combat.Card{
		card("Jab", combat.Fire),
		card("Cut", combat.Ice),
		card("Thump", combat.Lightning),
		card("Thrust", combat.Earth),
		card("Slice", combat.Arcane),
	}

	for _, turn := range [][]combat.Card{oneForm, oneCard} {
		found := false
		for _, k := range Loaded().ByTurn(turn) {
			if k == "prism" {
				found = true
			}
		}
		if !found {
			t.Error("one form or one card in all five colors is the prism")
		}
	}
	for _, k := range Loaded().ByTurn(mixed) {
		if k == "prism" {
			t.Error("five elements across three forms is not the prism")
		}
	}
}

// TestRealmReachedIsAThreshold: a run that arrives on the sixth realm has reached the fifth.
func TestRealmReachedIsAThreshold(t *testing.T) {
	if got := Loaded().ByMoment(RealmReached(4)); len(got) != 0 {
		t.Errorf("realm four earns nothing yet, got %v", got)
	}
	for _, realm := range []int{5, 6, 30} {
		found := false
		for _, k := range Loaded().ByMoment(RealmReached(realm)) {
			if k == "fifth-realm" {
				found = true
			}
		}
		if !found {
			t.Errorf("reaching realm %d must earn the fifth realm", realm)
		}
	}
}

// TestHandFormedMatchesOnTheRungItNamed: a hand-formed record lands on its own rung and no other,
// read off the catalog rather than naming a record, so it holds for every hand achievement authored.
func TestHandFormedMatchesOnTheRungItNamed(t *testing.T) {
	for _, a := range Loaded().list {
		if a.trigger.kind != data.TriggerMoment || a.trigger.moment != MomentHandFormed {
			continue
		}
		if !contains(Loaded().ByMoment(HandFormed(a.trigger.value)), a.Key) {
			t.Errorf("forming %s must earn %s", a.trigger.value, a.Key)
		}
		for _, h := range combat.Hands() {
			if h.Key != a.trigger.value && contains(Loaded().ByMoment(HandFormed(h.Key)), a.Key) {
				t.Errorf("forming %s earned %s, which asks for %s", h.Key, a.Key, a.trigger.value)
			}
		}
	}
}

func contains(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

// TestCountersNameBothAxes: a played turn adds to the form tally and the concept tally, because the
// two questions a player asks are different ones.
func TestCountersNameBothAxes(t *testing.T) {
	turn := []combat.Card{
		card("Slice", combat.Fire),
		card("Cut", combat.Ice),
		card("Brace", combat.Earth),
	}
	got := CountersFor(turn)

	if got["form:slash"] != 2 {
		t.Errorf("a Slice and a Cut are two slashing cards, got %d", got["form:slash"])
	}
	if got["concept:Slice"] != 1 {
		t.Errorf("one of them is the card called Slice, got %d", got["concept:Slice"])
	}
	if got["form:defend"] != 1 {
		t.Errorf("a Brace is a defending card, got %d", got["form:defend"])
	}
	if _, ok := got["form:none"]; ok {
		t.Error("a counter must never name an absence")
	}
	if got["element:fire"] != 1 || got["element:ice"] != 1 || got["element:earth"] != 1 {
		t.Errorf("each card counts once in its own element, got %v", got)
	}
}

// TestAWildcardCountsInEveryElement. A wildcard is played as all five colors, so it is a use of
// each; a basic card is no color and is a use of none.
func TestAWildcardCountsInEveryElement(t *testing.T) {
	wild := card("Jab", combat.Fire).SetRider(combat.Rider{Kind: combat.RiderWildElement})
	got := CountersFor([]combat.Card{wild, card("Jab", combat.Basic)})
	for _, e := range combat.AllElements {
		name := "element:" + e.String()
		want := 1
		if e == combat.Basic {
			want = 0
		}
		if got[name] != want {
			t.Errorf("%s counted %d, want %d", name, got[name], want)
		}
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
	for _, k := range Loaded().ByCounts(map[string]int{"form:slash": 299}) {
		if k == "slash-300" {
			t.Error("299 is not 300")
		}
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
		{"counter naming no element", data.TriggerData{
			Kind: data.TriggerCount, Counter: "element:wibble", N: 5}},
		{"counter naming the absence of an element", data.TriggerData{
			Kind: data.TriggerCount, Counter: "element:basic", N: 5}},
		{"moment naming nothing", data.TriggerData{Kind: data.TriggerMoment, Moment: "wibble"}},
		{"realm with no realm", data.TriggerData{
			Kind: data.TriggerMoment, Moment: MomentRealmReached}},
		{"ladder-wrapped with no direction", data.TriggerData{
			Kind: data.TriggerMoment, Moment: MomentLadderWrapped}},
		{"ladder-wrapped naming no direction", data.TriggerData{
			Kind: data.TriggerMoment, Moment: MomentLadderWrapped, Value: "sideways"}},
		{"hand-formed with no rung", data.TriggerData{
			Kind: data.TriggerMoment, Moment: MomentHandFormed}},
		{"hand-formed naming no rung", data.TriggerData{
			Kind: data.TriggerMoment, Moment: MomentHandFormed, Value: "wibble-of-a-kind"}},
		{"a moment carrying a value it never sets", data.TriggerData{
			Kind: data.TriggerMoment, Moment: MomentDuelWon, Value: "pair"}},
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

// **shields-raised is a threshold, like realm-reached.** Standing behind eleven earns the row that
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

// TestALadderWrapIsTheTwoEndsOfAThreeRungLadder. A Jab demoted to a Skewer and a Skewer promoted to
// a Jab are the two wraps; a step along the ladder, a defense walking its two rungs and a card that
// was removed are not.
func TestALadderWrapIsTheTwoEndsOfAThreeRungLadder(t *testing.T) {
	jab, thrust, skewer := card("Jab", combat.Fire), card("Thrust", combat.Fire), card("Skewer", combat.Fire)

	if m, ok := LadderWrapped(jab, skewer); !ok || m.Value != WrapDown {
		t.Errorf("a Jab demoted to a Skewer raised %+v, %v; want %q", m, ok, WrapDown)
	}
	if m, ok := LadderWrapped(skewer, jab); !ok || m.Value != WrapUp {
		t.Errorf("a Skewer promoted to a Jab raised %+v, %v; want %q", m, ok, WrapUp)
	}
	for _, c := range [][2]combat.Card{
		{jab, thrust}, {thrust, skewer}, {skewer, thrust},
		{card("Brace", combat.Fire), card("Block", combat.Fire)},
		{card("Block", combat.Fire), card("Brace", combat.Fire)},
		{jab, {}},
	} {
		if m, ok := LadderWrapped(c[0], c[1]); ok {
			t.Errorf("%v to %v raised %+v", c[0].Label(), c[1].Label(), m)
		}
	}

	// Each direction is the achievement it names, and only that one.
	down, _ := LadderWrapped(jab, skewer)
	up, _ := LadderWrapped(skewer, jab)
	if got := Loaded().ByMoment(down); len(got) != 1 || got[0] != "mighty-mouse" {
		t.Errorf("a wrap down earned %v, want mighty-mouse", got)
	}
	if got := Loaded().ByMoment(up); len(got) != 1 || got[0] != "quittin-roids" {
		t.Errorf("a wrap up earned %v, want quittin-roids", got)
	}
}
