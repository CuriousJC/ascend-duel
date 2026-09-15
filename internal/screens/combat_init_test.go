package screens

import "testing"

// The settings cog is live during a duel, so Init is entered two ways: a new fight, and a player
// coming back from a screen they stepped out to. It used to do the same thing either way, which
// silently restarted the fight — see CombatScene.Init.
//
// **This pins the key rather than the rebuild.** Driving Init itself needs fonts and a graphics
// context; what decides which of the two paths it takes is one predicate, and that is checkable.
//
// **What the key must not grow is a term about the duel being finished.** A settled duel has to
// survive a re-entry too, or stepping out after a defeat and coming back would hand the player a
// fresh fight and undo the death.
func TestSteppingOutOfADuelAndBackDoesNotStartANewOne(t *testing.T) {
	gs := saveState(t)

	s := &CombatScene{run: gs.Run, fightIndex: gs.Run.Fight()}
	if !s.showingDuel(gs) {
		t.Error("a scene already drawing this room's duel is treated as a new fight")
	}

	// A scene that has never built anything never matches, which is what makes the first Init a
	// new fight rather than a re-entry into nothing.
	if (&CombatScene{}).showingDuel(gs) {
		t.Error("a scene with no duel behind it matched the run's current room")
	}

	// The run walking into the next room is a new fight and must rebuild. `Init` is how the next
	// room starts, not only how the screen is entered — see nextFight.
	gs.Run.WonFight(40, 40)
	if s.showingDuel(gs) {
		t.Error("the next room reuses the last room's duel")
	}

	// A scene left over from an abandoned run does not match the run that replaced it, however far
	// along either of them happens to be.
	other := saveState(t)
	if s.fightIndex = other.Run.Fight(); s.showingDuel(other) {
		t.Error("a scene built for one run matched a different one")
	}

	// And neither does a scene with no run behind it — OpeningHand, tools/seeds and the flight
	// tests all drive this screen that way, and their fight index is 0 forever.
	other.Run = nil
	if s.showingDuel(other) {
		t.Error("a scene with no run matched")
	}
}
