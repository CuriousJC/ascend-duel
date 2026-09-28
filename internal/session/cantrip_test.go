package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/seeds"
)

// anyCantrip is a cantrip from the shipped catalog with this effect, whichever one it is.
func anyCantrip(t *testing.T, e CantripEffect) Cantrip {
	t.Helper()
	for _, c := range Cantrips() {
		if c.Effect == e {
			return c
		}
	}
	t.Fatalf("the catalog has no %s cantrip", e)
	return Cantrip{}
}

// An Endurance raises the ceiling and the life under it together, so a wounded duelist keeps the same
// share of their body: 30 of 60 becomes 60 of 120, not 30 of 120.
func TestEnduranceScalesTheLifeWithTheCeiling(t *testing.T) {
	c := anyCantrip(t, CantripScaleLife)
	d := c.Cast(combat.Duelist{MaxLife: 60, CurrentLife: 30})

	wantMax := 60 * c.Amount / 100
	wantLife := 30 * c.Amount / 100
	if d.MaxLife != wantMax || d.CurrentLife != wantLife {
		t.Errorf("%s turned 30/60 into %d/%d, want %d/%d",
			c.Record, d.CurrentLife, d.MaxLife, wantLife, wantMax)
	}
}

// A Might adds to whatever DMG is standing, and ten of them are ten times the amount — each cast is
// contained by itself and reads only the duelist in front of it.
func TestMightStacksByAddition(t *testing.T) {
	c := anyCantrip(t, CantripAddDMG)
	d := combat.Duelist{DMG: 10}
	for i := 0; i < 10; i++ {
		d = c.Cast(d)
	}
	if want := 10 + 10*c.Amount; d.DMG != want {
		t.Errorf("ten casts of %s left DMG at %d, want %d", c.Record, d.DMG, want)
	}
}

// What a cantrip added to the ceiling is shed when the run takes the life back, and a life above the
// run's own ceiling is that ceiling — the extra was a heal, never a debt.
func TestShedCantripsClampsToTheRunsCeiling(t *testing.T) {
	cases := []struct {
		life, max, added, wantLife, wantMax int
	}{
		{life: 90, max: 120, added: 60, wantLife: 60, wantMax: 60}, // above the ceiling: healed to it
		{life: 40, max: 120, added: 60, wantLife: 40, wantMax: 60}, // under it: the wound stands
		{life: 40, max: 60, added: 0, wantLife: 40, wantMax: 60},   // no cantrip: nothing moves
	}
	for _, c := range cases {
		life, ceiling := ShedCantrips(c.life, c.max, c.added)
		if life != c.wantLife || ceiling != c.wantMax {
			t.Errorf("ShedCantrips(%d, %d, %d) = %d/%d, want %d/%d",
				c.life, c.max, c.added, life, ceiling, c.wantLife, c.wantMax)
		}
	}
}

// The cap is the pane's, not a kind's: a rune and a cantrip share the two seats, and a full pane
// refuses either.
func TestTheConsumablesCapCountsEveryKind(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash))
	r := anyWithTarget(t, RuneRemove).Record
	scroll := anyCantrip(t, CantripAddDMG).Record

	if !run.Hold(r) {
		t.Fatal("an empty pane refused a rune")
	}
	for run.ConsumableCount() < MaxConsumables {
		if !run.HoldCantrip(scroll) {
			t.Fatalf("a pane holding %d of %d refused a cantrip", run.ConsumableCount(), MaxConsumables)
		}
	}
	if run.HoldCantrip(scroll) {
		t.Errorf("a full pane took a cantrip")
	}
	if run.Hold(r) {
		t.Errorf("a full pane took a rune")
	}

	kinds := map[ConsumableKind]int{}
	for _, c := range run.Consumables() {
		kinds[c.Kind]++
	}
	if kinds[ConsumableRune] != 1 || kinds[ConsumableCantrip] != MaxConsumables-1 {
		t.Errorf("the pane reads %v, want one rune and the rest cantrips", kinds)
	}

	if !run.DropScroll(0) || run.ConsumablesFull() {
		t.Errorf("casting a cantrip did not make room")
	}
}

// A carried cantrip comes back with the run, in order.
func TestCarriedCantripsSurviveAResume(t *testing.T) {
	motifs, shape := rosters(t)
	seed, _ := seeds.Parse(theSeed)

	s := Start(motifs, shape, seed)
	for _, c := range Cantrips() {
		s.holdCantrip(c.Record)
	}
	want := s.Scrolls()

	back, _, err := Resume(motifs, shape, s.Snapshot(seed))
	if err != nil {
		t.Fatal(err)
	}
	got := back.Scrolls()
	if len(got) != len(want) {
		t.Fatalf("the scroll case came back holding %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the scroll case came back holding %v, want %v", got, want)
		}
	}
}

// Every bundle of scrolls holds cantrips, and the catalog it draws from is not empty.
func TestEveryBundleOfScrollsHasSomethingToHold(t *testing.T) {
	found := false
	for _, g := range Goods() {
		if g.Contains == ContentsCantrips {
			found = true
		}
	}
	if !found {
		t.Fatal("goods.json has no bundle of scrolls")
	}
	if len(Cantrips()) == 0 {
		t.Fatal("cantrips.json is empty and a bundle has nothing to hold")
	}
}
