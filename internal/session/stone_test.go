package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The catalogue's own promises. **A stone for every hand** is the mechanic as asked for, and it is
// the one thing about the file that nothing else would notice: a rung with no stone is a rung that
// can never be raised, and the bag simply never offers it.
func TestEveryRungHasAStone(t *testing.T) {
	byHand := map[string]string{}
	for _, s := range Stones() {
		if prev, dup := byHand[s.Hand]; dup {
			t.Errorf("%s and %s both raise %s", prev, s.Record, s.Hand)
		}
		byHand[s.Hand] = s.Record
	}

	for _, hand := range combat.HandKeys() {
		if _, ok := byHand[hand]; !ok {
			t.Errorf("hand %q has no stone", hand)
		}
	}
	if len(byHand) != len(combat.HandKeys()) {
		t.Errorf("%d stones against %d rungs", len(byHand), len(combat.HandKeys()))
	}
}

func TestEveryStoneNamesARungTheRulesHave(t *testing.T) {
	for _, s := range Stones() {
		if _, ok := combat.HandSlot(s.Hand); !ok {
			t.Errorf("%s raises %q, which is not a rung", s.Record, s.Hand)
		}
		if s.Name == "" || s.Text == "" {
			t.Errorf("%s has no name or no text, so its card says nothing", s.Record)
		}
	}
}

// A stone is worth a tenth of its own rung and never nothing. A rung so cheap that a tenth of it
// floored away would be a card promising `+0`.
func TestNoStoneIsWorthNothing(t *testing.T) {
	for _, s := range Stones() {
		if worth := StoneWorth(s.Hand); worth <= 0 {
			t.Errorf("%s is worth %d", s.Record, worth)
		}
	}
}

func TestUsingAStoneRaisesItsRungAndNothingElse(t *testing.T) {
	s := New(nil)

	stone, ok := StoneForHand("concept-pair")
	if !ok {
		t.Fatal("no stone raises concept-pair")
	}

	before := map[string]int{}
	for _, h := range combat.Hands() {
		before[h.Key], _ = s.HandMultiplier(h.Key)
	}

	if !s.UseStone(stone.Record) {
		t.Fatal("the stone was refused")
	}

	for _, h := range combat.Hands() {
		now, ok := s.HandMultiplier(h.Key)
		if !ok {
			t.Fatalf("%s has no multiplier", h.Key)
		}
		want := before[h.Key]
		if h.Key == "concept-pair" {
			want += StoneWorth(h.Key)
		}
		if now != want {
			t.Errorf("%s pays %d, want %d", h.Key, now, want)
		}
	}
}

// **Using it is the whole of owning it**, and two of the same rung stack additively — the owner's
// call the mechanic is priced off.
func TestTwoStonesOnOneRungAreWorthTwice(t *testing.T) {
	s := New(nil)
	stone, _ := StoneForHand("concept-pair")

	base, _ := s.HandMultiplier("concept-pair")
	s.UseStone(stone.Record)
	s.UseStone(stone.Record)

	want := base + 2*StoneWorth("concept-pair")
	if got, _ := s.HandMultiplier("concept-pair"); got != want {
		t.Errorf("two stones give %d, want %d", got, want)
	}
	if n := s.StonesOn("concept-pair"); n != 2 {
		t.Errorf("the run holds %d stones on that rung, want 2", n)
	}
}

// **The fighter is where a stone becomes a number.** A run that raised a rung and handed out a
// duelist that had not heard of it would be a mechanic that worked everywhere but in the fight.
func TestEquipCarriesTheRunsStonesOntoTheFighter(t *testing.T) {
	s := New(nil)
	stone, _ := StoneForHand("form-pair")
	s.UseStone(stone.Record)

	d := s.Equip(combat.Duelist{DMG: 10, Actions: 6, MaxLife: 60, CurrentLife: 60})
	if n := d.HandStoneCount("form-pair"); n != 1 {
		t.Fatalf("the fighter carries %d stones on form-pair, want 1", n)
	}

	want, _ := s.HandMultiplier("form-pair")
	for _, h := range d.HandTable() {
		if h.Key == "form-pair" && h.Multiplier != want {
			t.Errorf("the fighter plays form-pair at %d, the run says %d", h.Multiplier, want)
		}
	}
}

func TestAStoneTheCatalogueDoesNotHoldIsRefused(t *testing.T) {
	s := New(nil)
	if s.UseStone("not-a-stone") {
		t.Error("a stone nobody wrote was accepted")
	}
	if len(s.StoneCounts()) != 0 {
		t.Error("a refused stone still changed the run")
	}
}

// The two sealed goods take the vitae they say they take, and refuse when the purse is short.
func TestASealedGoodCostsWhatItSays(t *testing.T) {
	// **Spent down to nothing rather than topped up**, because a run opens with a purse of its own
	// and a test that assumed an empty one would be pinning `startingVitae` by accident.
	s := New(nil)
	s.SpendVitae(s.Vitae())
	s.AddVitae(BagPrice() + CanPrice())
	start := s.Vitae()

	if !s.BuyBag() {
		t.Fatal("the bag was refused with the purse full")
	}
	if got, want := s.Vitae(), start-BagPrice(); got != want {
		t.Errorf("the purse is %d after a bag, want %d", got, want)
	}
	if !s.BuyCan() {
		t.Fatal("the can was refused with the purse still covering it")
	}
	if got := s.Vitae(); got != 0 {
		t.Errorf("the purse is %d after both, want 0", got)
	}

	if s.BuyBag() || s.CanAffordBag() {
		t.Error("an empty purse bought a bag")
	}
}
