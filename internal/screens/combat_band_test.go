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
