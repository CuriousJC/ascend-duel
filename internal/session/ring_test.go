package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The catalogue, the worn set, and the three moments that fire out here rather than in a round.

// bare is a run wearing nothing, which is what most of these want: `New` opens wearing
// StartingRings, which is empty as shipped — so this is belt and braces against the day it is
// filled in for a look at something.
func bare(t *testing.T) *Session {
	t.Helper()

	run := New(testDeck())
	run.worn = nil
	return run
}

func wearing(t *testing.T, keys ...string) *Session {
	t.Helper()

	run := bare(t)
	for _, key := range keys {
		if !run.Wear(key) {
			t.Fatalf("%s would not go on", key)
		}
	}
	return run
}

func TestEveryRingInTheFileRegisters(t *testing.T) {
	// **The whole catalogue is parsed at package init and panics on a bad record**, so reaching this
	// test at all is most of the check. What it adds is the count: a record silently dropped would
	// otherwise look exactly like a ring nobody has authored yet.
	records := data.LoadRings()

	if len(records) == 0 {
		t.Fatal("rings.json holds no rings")
	}
	for key := range records {
		id, ok := RingID(key)
		if !ok {
			t.Errorf("%s is in the file and not in the registry", key)
			continue
		}
		if len(combat.RingOf(id).Rules) == 0 {
			t.Errorf("%s registered with no rules", key)
		}
	}
}

func TestARingIsWornOnceAndNoMoreThanFiveAreWornAtAll(t *testing.T) {
	run := bare(t)

	all := Rings()
	if len(all) < combat.MaxWornRings+1 {
		t.Skipf("only %d rings authored; this needs %d", len(all), combat.MaxWornRings+1)
	}

	for _, key := range all[:combat.MaxWornRings] {
		if !run.Wear(key) {
			t.Fatalf("%s would not go on", key)
		}
	}
	if run.Wear(all[0]) {
		t.Error("the same ring went on twice")
	}
	if run.Wear(all[combat.MaxWornRings]) {
		t.Errorf("a %dth ring went on, cap is %d", combat.MaxWornRings+1, combat.MaxWornRings)
	}
	if run.Wear("no-such-ring") {
		t.Error("a record the catalogue does not hold went on")
	}
}

func TestWornOrderIsTheOrderTheyWentOn(t *testing.T) {
	// **Worn order is a rule, not a presentation detail**: rings fire left to right and compound, so
	// the order has to be one the player can see. Sorting it here would quietly change what two
	// multiplicative rings come to.
	run := wearing(t, "thunder-ring", "fire-ring", "frozen-ring")

	want := []string{"thunder-ring", "fire-ring", "frozen-ring"}
	got := run.Worn()

	if len(got) != len(want) {
		t.Fatalf("wearing %v came out as %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("worn order is %v, want %v", got, want)
		}
	}
}

func TestAStatRingIsAddedAtFightStartAndNowhereElse(t *testing.T) {
	run := wearing(t, "might-ring", "bulwark-ring")

	base := combat.Duelist{DMG: 10, Actions: 5, MaxLife: 100, CurrentLife: 100}
	d := run.Equip(base)

	if got, want := d.DMG, base.DMG+10; got != want {
		t.Errorf("the might ring made DMG %d, want %d", got, want)
	}
	if got, want := d.MaxLife, base.MaxLife+25; got != want {
		t.Errorf("the bulwark ring made MaxLife %d, want %d", got, want)
	}
	if d.CurrentLife != d.MaxLife {
		t.Errorf("a full-health duelist equipped to %d/%d", d.CurrentLife, d.MaxLife)
	}
	if d.RingCount != 2 {
		t.Errorf("the duelist came out wearing %d rings, want 2", d.RingCount)
	}
}

func TestEquippingAWoundedDuelistRaisesTheCeilingWithoutHealingThem(t *testing.T) {
	// HP raises the ceiling and fills it, but a duelist arriving hurt keeps the wound. What must
	// never happen is CurrentLife climbing past MaxLife.
	run := wearing(t, "bulwark-ring")

	d := run.Equip(combat.Duelist{DMG: 10, Actions: 5, MaxLife: 100, CurrentLife: 40})

	if d.MaxLife != 125 {
		t.Errorf("MaxLife came out %d, want 125", d.MaxLife)
	}
	if d.CurrentLife != 65 {
		t.Errorf("a duelist on 40 came out on %d, want 65", d.CurrentLife)
	}
}

func TestAGrowingRingGainsOnEveryWin(t *testing.T) {
	// The accumulator is the first ring state that survives a fight, and it is keyed by record
	// because it is the first that will have to be serialized.
	run := wearing(t, "heart-ring")

	base := combat.Duelist{DMG: 10, Actions: 5, MaxLife: 100, CurrentLife: 100}
	if got := run.Equip(base).MaxLife; got != 105 {
		t.Errorf("fight one equipped to %d, want 105", got)
	}

	run.WonFight()
	run.WonFight()

	if got := run.Grown("heart-ring"); got != 10 {
		t.Errorf("two wins grew the ring by %d, want 10", got)
	}
	if got := run.Equip(base).MaxLife; got != 115 {
		t.Errorf("fight three equipped to %d, want 115 — +5 base plus +10 accumulated", got)
	}
}

func TestPropagationCountsFivesAndStopsAtFive(t *testing.T) {
	// +1 for every 5 held, capped at +5. Rounded down, like every other integer rule in the game.
	for _, tc := range []struct{ held, want int }{
		{0, 0}, {4, 0}, {5, 1}, {9, 1}, {10, 2}, {25, 5}, {60, 5},
	} {
		run := bare(t)
		run.vitae = tc.held
		run.WonFight()

		if got := run.Vitae() - tc.held; got != tc.want {
			t.Errorf("a purse of %d propagated %d, want %d", tc.held, got, tc.want)
		}
	}
}

func TestBankerScalesWhatTheCapProduced(t *testing.T) {
	// **The cap binds the base rate and the ring scales what the cap produced** (owner's call). At
	// 25 held that is +5 bare and +10 wearing Banker — an absolute cap would leave the ring doing
	// nothing past 25, which is a ring that stops working when a run can finally afford it.
	run := wearing(t, "banker-ring")
	run.vitae = 25
	run.WonFight()

	if got := run.Vitae() - 25; got != 10 {
		t.Errorf("Banker propagated %d on a purse of 25, want 10", got)
	}
}

func TestSoulTakerPaysFlatAndHungryAddsAPick(t *testing.T) {
	// The two `prizes-dealt` rings, and they are deliberately different objects: one changes a
	// value, the other changes how many choices there are.
	plain := bare(t)
	if got := plain.PrizeVitae(5); got != 5 {
		t.Errorf("the bare vitae card pays %d, want 5", got)
	}
	if got := plain.Picks(); got != 1 {
		t.Errorf("a bare run is offered %d picks, want 1", got)
	}

	rich := wearing(t, "soul-taker-ring", "hungry-ring")
	if got := rich.PrizeVitae(5); got != 10 {
		t.Errorf("Soul Taker's vitae card pays %d, want 10", got)
	}
	if got := rich.Picks(); got != 2 {
		t.Errorf("Hungry offers %d picks, want 2", got)
	}
}

func TestAFlipRecoloursWhatIsDealtAndNotWhatIsOwned(t *testing.T) {
	// The deck the fight is dealt from is a transform of the run's own, applied fresh every time.
	// That is what makes a flip non-composing by construction and what makes one bought mid-run
	// apply from the next fight without rewriting anything.
	run := wearing(t, "frozen-lightning-ring")
	run.deck = []combat.Card{
		{Concept: combat.Strike, Element: combat.Lightning},
		{Concept: combat.Strike, Element: combat.Fire},
	}

	dealt := run.FightDeck()
	if dealt[0].Element != combat.Ice {
		t.Errorf("a lightning card was dealt as %v, want ice", dealt[0].Element)
	}
	if dealt[1].Element != combat.Fire {
		t.Errorf("a fire card was dealt as %v, want fire", dealt[1].Element)
	}
	if owned := run.Deck(); owned[0].Element != combat.Lightning {
		t.Errorf("the flip wrote back into the run: %v", owned[0].Element)
	}
}

func TestADiscountRingPricesTheRunsOwnCards(t *testing.T) {
	// The post-battle screen draws deck cards with no duelist to ask, and a card whose price changed
	// when it reached the hand would be the game contradicting itself between two screens.
	run := wearing(t, "thrifty-ring")

	hot := combat.Card{Concept: combat.Strike, Element: combat.Fire}
	cold := combat.Card{Concept: combat.Strike, Element: combat.Ice}

	if got, want := run.CardCost(hot), hot.Cost()-1; got != want {
		t.Errorf("a fire Strike costs the run %d, want %d", got, want)
	}
	if got, want := run.CardCost(cold), cold.Cost(); got != want {
		t.Errorf("an ice Strike costs the run %d, want %d", got, want)
	}
}

func TestARunOpensBare(t *testing.T) {
	// **A run buys its rings** *(owner's call, 2026-08-21)*. StartingRings is the debug seat for
	// putting one on without playing to a shop, so this checks both: the shipped value is empty, and
	// whatever it holds is what a new run is wearing.
	run := New(testDeck())

	if len(StartingRings) != 0 {
		t.Errorf("StartingRings ships holding %v; empty is the shipped value", StartingRings)
	}
	if got, want := len(run.Worn()), len(StartingRings); got != want {
		t.Fatalf("a new run wears %d rings, want %d", got, want)
	}
	for i, key := range StartingRings {
		if run.Worn()[i] != key {
			t.Errorf("a new run wears %v, want %v", run.Worn(), StartingRings)
			break
		}
	}
}
