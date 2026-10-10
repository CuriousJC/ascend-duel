package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/seeds"
)

// cantrip is one record from the shipped catalog, by key.
func cantrip(t *testing.T, key string) Cantrip {
	t.Helper()
	c, ok := CantripByKey(key)
	if !ok {
		t.Fatalf("the catalog has no %s", key)
	}
	return c
}

// casts is n casts of one cantrip, as the screen hands a fight's casts to EquipWearing.
func casts(c Cantrip, n int) []combat.WornRelic {
	out := make([]Cantrip, n)
	for i := range out {
		out[i] = c
	}
	return CantripRelics(out)
}

// Two Endurances are two 2x life relics, and they compound: each scales what the one before it
// left, the way any two scaling relics do.
func TestEndurancesCompound(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash))
	base := combat.Duelist{MaxLife: 60, CurrentLife: 60}

	once := run.EquipWearing(base, casts(cantrip(t, "cantrip-endurance"), 1))
	twice := run.EquipWearing(base, casts(cantrip(t, "cantrip-endurance"), 2))
	if once.MaxLife != 120 || twice.MaxLife != 240 {
		t.Errorf("one Endurance made a 60 body %d and two made it %d, want 120 and 240",
			once.MaxLife, twice.MaxLife)
	}
}

// Ten Mights are ten +10 DMG relics.
func TestMightsStackByAddition(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash))
	d := run.EquipWearing(combat.Duelist{DMG: 10}, casts(cantrip(t, "cantrip-might"), 10))
	if d.DMG != 110 {
		t.Errorf("ten Mights left DMG at %d, want 110", d.DMG)
	}
}

// Chill doubles an ice card and leaves every other color alone.
func TestChillDoublesIceCards(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash))
	bare := run.Equip(combat.Duelist{DMG: 10})
	chilled := run.EquipWearing(combat.Duelist{DMG: 10}, casts(cantrip(t, "cantrip-chill"), 1))

	ice := combat.Card{Concept: combat.Bash, Element: combat.Ice}
	fire := combat.Card{Concept: combat.Bash, Element: combat.Fire}
	if got, want := chilled.CardDamage(ice), 2*bare.CardDamage(ice); got != want {
		t.Errorf("a chilled ice Bash deals %d, want %d", got, want)
	}
	if got, want := chilled.CardDamage(fire), bare.CardDamage(fire); got != want {
		t.Errorf("a chilled fire Bash deals %d, want the unchilled %d", got, want)
	}
}

// A cantrip-relic takes no slot: a hand full of the run's relics still wears every cast.
func TestACantripRelicIsWornOverAFullHand(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash))
	run.SetRelicSlots(1)
	for _, key := range Relics() {
		if run.Wear(key) {
			break
		}
	}
	if len(run.Worn()) != 1 {
		t.Fatalf("the run wears %v, want one relic filling its one slot", run.Worn())
	}

	d := run.EquipWearing(combat.Duelist{DMG: 10}, casts(cantrip(t, "cantrip-might"), 2))
	if n := len(d.WornRelics()); n != 3 {
		t.Errorf("a full hand plus two casts wears %d relics, want 3", n)
	}
}

// A cantrip-relic is never the run's: equipping without the casts is the duelist the run would have
// had, which is what the next fight starts from.
func TestACantripNeverReachesTheRun(t *testing.T) {
	run := runWith(combat.Plain(combat.Bash))
	run.EquipWearing(combat.Duelist{DMG: 10}, casts(cantrip(t, "cantrip-might"), 1))
	if d := run.Equip(combat.Duelist{DMG: 10}); d.DMG != 10 || len(d.Relics) != 0 {
		t.Errorf("after a cast the run equips a duelist with DMG %d wearing %v", d.DMG, d.Relics)
	}
}

// Every cantrip casts a relic the row can name and the tooltip can say, and the screen can find the
// cantrip again from the relic it is holding.
func TestEveryCantripCastsARelic(t *testing.T) {
	for _, c := range Cantrips() {
		if c.RelicName == "" || c.RelicText == "" {
			t.Errorf("%s casts a relic with no name or no line", c.Record)
		}
		if back, ok := CantripByRelic(c.Relic); !ok || back.Record != c.Record {
			t.Errorf("%s's relic does not lead back to it", c.Record)
		}
	}
}

// A cantrip-relic waking at a moment only the run answers would load and never fire, so it is
// refused.
func TestACantripRelicMayNotWakeOutsideTheFight(t *testing.T) {
	for _, when := range []string{"card-drawn", "fight-won", "prizes-dealt", "essence-spent"} {
		rec := data.CantripData{
			CantripRecord: "cantrip-test",
			Name:          "Test",
			Relic: data.CantripRelicData{
				Name: "Test",
				Rules: []data.RelicRuleData{{When: when,
					Then: []data.RelicEffectData{{Do: "adjust-picks", Amount: 1}}}},
			},
		}
		if _, err := checkCantripRecord(rec); err == nil {
			t.Errorf("a cantrip-relic waking at %s was accepted", when)
		}
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
	scroll := cantrip(t, "cantrip-might").Record

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
