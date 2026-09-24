package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The catalog's own promises. **A stone for every shape** is the mechanic, and it is the one thing
// about the file that nothing else would notice: a shape with no stone is a set of rungs that can
// never be raised, and the bag simply never offers it.
func TestEveryShapeHasExactlyOneStone(t *testing.T) {
	byShape := map[string]string{}
	for _, s := range Stones() {
		if prev, dup := byShape[s.Shape]; dup {
			t.Errorf("%s and %s both raise %s", prev, s.Record, s.Shape)
		}
		byShape[s.Shape] = s.Record
	}

	for _, shape := range combat.HandShapes() {
		if _, ok := byShape[shape]; !ok {
			t.Errorf("shape %s has no stone", shape)
		}
	}
	for _, hand := range combat.HandKeys() {
		if _, ok := StoneForHand(hand); !ok {
			t.Errorf("hand %q has no stone", hand)
		}
	}
}

func TestEveryStoneNamesRungsTheRulesHave(t *testing.T) {
	for _, s := range Stones() {
		if len(s.Hands()) == 0 {
			t.Errorf("%s raises the shape %s, which no rung carries", s.Record, s.Shape)
		}
		if s.Name == "" || s.Text == "" {
			t.Errorf("%s has no name or no text, so its card says nothing", s.Record)
		}
	}
}

// A stone is worth a tenth of each of its rungs and never nothing. A rung so cheap that a tenth of
// it floored away would be a row promising `+0`.
func TestNoStoneIsWorthNothing(t *testing.T) {
	for _, s := range Stones() {
		for _, hand := range s.Hands() {
			if worth := StoneWorth(hand); worth <= 0 {
				t.Errorf("%s is worth %d on %s", s.Record, worth, hand)
			}
		}
	}
}

// **One stone raises every axis of its shape**, each by a tenth of that rung's own multiplier, and
// nothing outside the shape moves.
func TestUsingAStoneRaisesEveryRungOfItsShapeAndNothingElse(t *testing.T) {
	s := New(nil)

	stone, ok := StoneForHand("concept-three-of-a-kind")
	if !ok {
		t.Fatal("no stone raises concept-three-of-a-kind")
	}
	raised := map[string]bool{}
	for _, hand := range stone.Hands() {
		raised[hand] = true
	}
	for _, want := range []string{"concept-three-of-a-kind", "form-three-of-a-kind", "element-three-of-a-kind"} {
		if !raised[want] {
			t.Errorf("the three-of-a-kind stone does not raise %s", want)
		}
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
		if raised[h.Key] {
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
	stone, _ := StoneForHand("pair")

	base, _ := s.HandMultiplier("pair")
	s.UseStone(stone.Record)
	s.UseStone(stone.Record)

	want := base + 2*StoneWorth("pair")
	if got, _ := s.HandMultiplier("pair"); got != want {
		t.Errorf("two stones give %d, want %d", got, want)
	}
	if n := s.StonesOn("pair"); n != 2 {
		t.Errorf("the run holds %d stones on that rung, want 2", n)
	}
}

// **The fighter is where a stone becomes a number.** A run that raised a rung and handed out a
// duelist that had not heard of it would be a mechanic that worked everywhere but in the fight.
func TestEquipCarriesTheRunsStonesOntoTheFighter(t *testing.T) {
	s := New(nil)
	stone, _ := StoneForHand("pair")
	s.UseStone(stone.Record)

	d := s.Equip(combat.Duelist{DMG: 10, Actions: 6, MaxLife: 60, CurrentLife: 60})
	if n := d.HandStoneCount("pair"); n != 1 {
		t.Fatalf("the fighter carries %d stones on form-pair, want 1", n)
	}

	want, _ := s.HandMultiplier("pair")
	for _, h := range d.HandTable() {
		if h.Key == "pair" && h.Multiplier != want {
			t.Errorf("the fighter plays form-pair at %d, the run says %d", h.Multiplier, want)
		}
	}
}

func TestAStoneTheCatalogDoesNotHoldIsRefused(t *testing.T) {
	s := New(nil)
	if s.UseStone("not-a-stone") {
		t.Error("a stone nobody wrote was accepted")
	}
	if len(s.StoneCounts()) != 0 {
		t.Error("a refused stone still changed the run")
	}
}

// A sealed good takes the vitae it says it takes, and refuses when the purse is short.
func TestASealedGoodCostsWhatItSays(t *testing.T) {
	bag, ok := GoodHolding(ContentsStones)
	if !ok {
		t.Fatal("no good in goods.json holds stones, and the bag is the only way to a rock")
	}
	vial, ok := GoodHolding(ContentsEssences)
	if !ok {
		t.Fatal("no good in goods.json holds essences")
	}

	// **Spent down to nothing rather than topped up**, because a run opens with a purse of its own
	// and a test that assumed an empty one would be pinning `startingVitae` by accident.
	s := New(nil)
	s.SpendVitae(s.Vitae())
	s.AddVitae(bag.Price + vial.Price)
	start := s.Vitae()

	if !s.BuyGood(bag.Record) {
		t.Fatal("the bag was refused with the purse full")
	}
	if got, want := s.Vitae(), start-bag.Price; got != want {
		t.Errorf("the purse is %d after a bag, want %d", got, want)
	}
	if !s.BuyGood(vial.Record) {
		t.Fatal("the vial was refused with the purse still covering it")
	}
	if got := s.Vitae(); got != 0 {
		t.Errorf("the purse is %d after both, want 0", got)
	}

	if s.BuyGood(bag.Record) || s.CanAffordGood(bag.Record) {
		t.Error("an empty purse bought a bag")
	}

	// A key no record carries buys nothing and costs nothing, rather than spending a zero price.
	s.AddVitae(100)
	if s.BuyGood("no-such-good") {
		t.Error("a key no record carries was bought")
	}
	if got := s.Vitae(); got != 100 {
		t.Errorf("the purse is %d after buying nothing, want 100", got)
	}
}
