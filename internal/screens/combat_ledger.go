package screens

// **How a duel gets into the run's account.**
//
// The screen is the only place that has both halves: the resolved events, and the words for them.
// So it words a round the moment the round is over and hands the lines to `internal/session`,
// which keeps them for the length of the run. See session/ledger.go for why the run stores
// sentences rather than the events they were written from.
//
// **Three call sites and no more**, on the terms save.go is under: a fight opening, a round
// finishing, and a duel settling. A recording spread thinly through the screen would be a record
// that is wrong in exactly the rounds nobody was watching.

import (
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// openLedgerFight starts a record for the duel this screen is about to run.
//
// **Called from Init, which runs again on a retry** — and a retry is a different duel with a
// different shuffle, so it is a different record. An entry left unfought is dropped rather than
// kept; see session.BeginFight.
func (s *CombatScene) openLedgerFight(gs *state.GlobalState) {
	if gs.Run == nil || s.enemy == nil {
		return
	}
	s.ledgerWritten, s.ledgerClosed = false, false
	// **`Floor()` is already the floor, counting from one.** It was `+1` for a few hours and made
	// every heading name the floor above the one the fight was on.
	gs.Run.BeginFight(gs.Run.Floor(), s.enemy.Name)
}

// recordRound writes the round that has just been watched into the account, once.
//
// **Called on the frame playback reaches the end of the log**, so the round the player has just
// played is in the ledger while they plan the next one. Waiting for the next round to replace it
// left it out for the whole planning phase, which is when the panel is most likely to be opened.
//
// **It words the round through the same walk the screen has always used**, so a line read back
// three fights later is the line that was on screen while it happened — which is what makes the
// account impossible to disagree with the round it reports. See prose.go.
func (s *CombatScene) recordRound() {
	if s.ledgerWritten || len(s.log) == 0 || s.run == nil {
		return
	}
	s.ledgerWritten = true
	s.run.RecordRound(s.ledgerLines(s.log), dealtBy(combat.SideA, s.log))
}

// closeLedgerFight records the last round and says what became of the duel.
//
// **The round is recorded here as a backstop**, and is almost always already written: playback
// finishing is what records a round, and a duel settles on the same frame that finishes for the
// last one. recordRound is guarded, so saying it twice costs nothing and forgetting it once would
// cost the killing blow.
func (s *CombatScene) closeLedgerFight() {
	if s.run == nil {
		return
	}
	s.recordRound()

	outcome := session.OutcomeWon
	if !s.fighter.Alive() {
		outcome = session.OutcomeLost
	}
	s.run.EndFight(outcome)
}

// dealtBy is what one side's blows came to in a round: the damage that actually landed on the
// other duelist, after everything the defence took off it.
//
// **The landed figure rather than the hand's**, because the summary line it feeds is answering
// "how did this fight go" — and a fight lost to shields is a fight where the hands were fine and
// nothing arrived. See combat.Event.Base, which is the other figure and the wrong one here.
func dealtBy(side combat.Side, events []combat.Event) int {
	total := 0
	for _, e := range events {
		if e.Kind == combat.KindDamage && e.Target != side {
			total += e.Amount
		}
	}
	return total
}
