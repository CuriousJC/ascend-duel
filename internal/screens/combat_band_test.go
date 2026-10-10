package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// refitState is a run wearing one life relic, and a fight standing in the room equipped from it.
func refitState(t *testing.T, relic string) (*state.GlobalState, *CombatScene) {
	t.Helper()
	gs := &state.GlobalState{
		ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight,
		Run:      session.New(session.StartingDeck()),
		Duelists: data.LoadDuelists(),
		Decks:    data.LoadDecks(),
	}
	gs.Run.AddVitae(100)
	if !gs.Run.Wear(relic) {
		t.Fatalf("the run would not wear %s", relic)
	}
	if _, ok := gs.Duelists[ui.PlayerRecord]; !ok {
		t.Fatalf("no duelist record %q", ui.PlayerRecord)
	}
	s := &CombatScene{run: gs.Run, fighter: &entities.Combatant{}}
	s.fighter.Duelist = s.equippedFighter(gs)
	return gs, s
}

// **A relic sold mid-fight stops counting at once, and the fight's wound stays a wound** *(owner's
// call, 2026-10-04)*. Selling the life relic takes its life off the ceiling, and the duelist carries
// exactly the damage the fight had already done rather than being healed or hurt by the sale.
func TestARelicSoldMidFightStopsCountingAndTheWoundStays(t *testing.T) {
	gs, s := refitState(t, "hp-plus")
	before := s.fighter.Duelist
	s.fighter.CurrentLife -= 10

	s.sellRelic(gs, "hp-plus")

	after := s.fighter.Duelist
	if after.MaxLife >= before.MaxLife {
		t.Fatalf("selling a life relic left the ceiling at %d, was %d", after.MaxLife, before.MaxLife)
	}
	if wound := after.MaxLife - after.CurrentLife; wound != 10 {
		t.Errorf("the duelist carries a wound of %d after the sale, want the fight's 10", wound)
	}
	if len(after.Relics) != 0 {
		t.Errorf("the duelist still wears %v after the sale", after.Relics)
	}
}

// **A refit is a rebuild, never a second Equip.** Refitting a duelist with nothing changed leaves
// every figure the run pays where it was — the bug a mid-fight re-equip had, paying every flat
// bonus a second time.
func TestARefitWithNothingChangedChangesNothing(t *testing.T) {
	gs, s := refitState(t, "hp-plus")
	before := s.fighter.Duelist

	s.refit(gs)

	after := s.fighter.Duelist
	if after.MaxLife != before.MaxLife || after.DMG != before.DMG || after.CurrentLife != before.CurrentLife {
		t.Errorf("a refit moved the duelist: %d/%d life %d DMG, was %d/%d life %d DMG",
			after.CurrentLife, after.MaxLife, after.DMG, before.CurrentLife, before.MaxLife, before.DMG)
	}
}

// What the fight has done to the duelist comes across a refit: the shields standing, the surge
// banked, and the growth a remaining relic has made this fight.
func TestARefitCarriesTheFight(t *testing.T) {
	gs, s := refitState(t, "hp-plus")
	s.fighter.Shields[combat.Fire] = 2
	s.fighter.Surge = 1
	s.fighter.Relics[0].Grown = 7

	s.refit(gs)

	if got := s.fighter.Shields.Count(); got != 2 {
		t.Errorf("%d shields standing after a refit, want 2", got)
	}
	if s.fighter.Surge != 1 {
		t.Errorf("surge %d after a refit, want 1", s.fighter.Surge)
	}
	if s.fighter.Relics[0].Grown != 7 {
		t.Errorf("the relic's growth is %d after a refit, want the fight's 7", s.fighter.Relics[0].Grown)
	}
}

// **A cast is a refit with one more cantrip-relic on.** The relic is worn after the run's, weightless
// and ephemeral; the fight's wound stays a wound; what it added to the ceiling is tallied for the run
// to shed; and the run itself is never written to.
func TestACastCantripIsWornForTheFightAndKeepsTheWound(t *testing.T) {
	gs, s := refitState(t, "hp-plus")
	gs.Relics = data.LoadRelics()
	before := s.fighter.Duelist
	s.fighter.CurrentLife -= 10

	endurance, ok := session.CantripByKey("cantrip-endurance")
	if !ok {
		t.Fatal("the catalog has no cantrip-endurance")
	}
	s.cast = append(s.cast, endurance)
	s.refit(gs)

	after := s.fighter.Duelist
	if after.MaxLife != 2*before.MaxLife {
		t.Errorf("an Endurance made a %d ceiling %d, want %d", before.MaxLife, after.MaxLife, 2*before.MaxLife)
	}
	if wound := after.MaxLife - after.CurrentLife; wound != 10 {
		t.Errorf("the duelist carries a wound of %d after the cast, want the fight's 10", wound)
	}
	if s.cantripLife != after.MaxLife-before.MaxLife {
		t.Errorf("the cast is tallied as %d life, want %d", s.cantripLife, after.MaxLife-before.MaxLife)
	}

	worn := after.WornRelics()
	if len(worn) != 2 || worn[1].Relic != endurance.Relic || !worn[1].Weightless || !worn[1].Ephemeral {
		t.Errorf("the fighter wears %v, want the run's relic then a weightless, ephemeral Endurance", worn)
	}
	if got := len(s.paneRelics(gs)); got != 2 {
		t.Errorf("the row draws %d relics, want the run's one and the cast's", got)
	}
	if fresh := gs.Run.Equip(ui.DuelistFromRecord(gs, ui.PlayerRecord).Duelist); fresh.MaxLife != before.MaxLife {
		t.Errorf("after a cast the run equips a %d ceiling, want the unchanged %d", fresh.MaxLife, before.MaxLife)
	}
}

// **A cantrip-relic drags like any other relic**, and the order it is dragged into is the order the
// fighter wears it in — across the refit the next cast makes, too. The run never hears where a cast
// stands.
func TestACantripRelicDragsAndTheFighterWearsTheRowsOrder(t *testing.T) {
	gs, s := refitState(t, "hp-plus")
	gs.Relics = data.LoadRelics()
	might, _ := session.CantripByKey("cantrip-might")

	s.cast = append(s.cast, might)
	s.refit(gs)
	s.moveRelic(gs, 1, 0)

	if w := s.fighter.Duelist.WornRelics(); len(w) != 2 || w[0].Relic != might.Relic {
		t.Fatalf("after dragging the cast to the front the fighter wears %v, want Might first", w)
	}
	if worn := gs.Run.Worn(); len(worn) != 1 || worn[0] != "hp-plus" {
		t.Errorf("the run wears %v after a cast was dragged, want it untouched", worn)
	}

	s.cast = append(s.cast, might)
	s.refit(gs)
	w := s.fighter.Duelist.WornRelics()
	if len(w) != 3 || w[0].Relic != might.Relic || w[2].Relic != might.Relic {
		t.Errorf("a second cast left the fighter wearing %v, want Might, the run's relic, Might", w)
	}
	if got := s.paneRelics(gs); len(got) != 3 || got[1].RelicRecord != "hp-plus" {
		t.Errorf("the row draws %v, want the run's relic in the middle", got)
	}
}

// **A run relic dragged past a cantrip-relic moves in the run's own order by its place among the run's
// relics**, so the run keeps the order the player gave its relics.
func TestARunRelicDraggedPastACastMovesInTheRun(t *testing.T) {
	gs, s := refitState(t, "hp-plus")
	gs.Relics = data.LoadRelics()
	var second string
	for _, key := range session.Relics() {
		if key != "hp-plus" && gs.Run.Wear(key) {
			second = key
			break
		}
	}
	if second == "" {
		t.Fatal("the run would not wear a second relic")
	}
	might, _ := session.CantripByKey("cantrip-might")
	s.cast = append(s.cast, might)
	s.refit(gs)

	// hp-plus, second, Might — drag hp-plus to the end: second, Might, hp-plus.
	s.moveRelic(gs, 0, 2)

	if worn := gs.Run.Worn(); len(worn) != 2 || worn[0] != second || worn[1] != "hp-plus" {
		t.Errorf("the run wears %v, want %s then hp-plus", worn, second)
	}
	w := s.fighter.Duelist.WornRelics()
	if len(w) != 3 || w[1].Relic != might.Relic {
		t.Errorf("the fighter wears %v, want Might in the middle", w)
	}
	if _, ok := s.runRelicAt(gs, 1); ok {
		t.Error("the cast's seat offers itself for sale")
	}
	if key, ok := s.runRelicAt(gs, 2); !ok || key != "hp-plus" {
		t.Errorf("the last seat sells %q, want hp-plus", key)
	}
}
