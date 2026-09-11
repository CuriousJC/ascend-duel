package screens

// **Where an achievement key meets the player's profile.**
//
// `internal/achieve` decides what a turn, a tally or a moment has earned and knows nothing about a
// profile; `save.go` writes files and knows nothing about a card. This is the seam: four raisers
// that ask the catalogue a question and hand whatever comes back to `award`.
//
// **Everything is idempotent and nothing is filtered upstream.** The catalogue reports every
// achievement a moment satisfies, including ones the player got months ago; `profile.Award` reports
// whether anything actually changed, and that boolean is what stops the toast firing twice. So a
// raiser can be called on every win, every turn and every altered card without a caller having to
// remember what has already happened.
//
// **Counters are held in memory and settled when a duel ends** *(owner's call, 2026-09-06)*. A card
// played is a `Bump`, which touches no file; `settleCounters` is the one place the profile is
// written for them, and it is called where a fight finishes. What that costs is stated rather than
// discovered: a crash mid-duel loses that duel's tallies and nothing else. The alternative was a
// disk write per card played.

import (
	"github.com/curiousjc/ascend-duel/internal/achieve"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// earnMoment records everything a named moment has earned.
func earnMoment(gs *state.GlobalState, m achieve.Moment) {
	earn(gs, achieve.Loaded().ByMoment(m))
}

// earnTurn records everything the cards played in one turn have earned.
//
// **The resolved turn, read once, off the queue the engine was handed** — not off the playback. A
// turn achievement counted as the animation reached each card would be a record the player could
// change by leaving the screen, and presentation may never change an outcome. Same rule
// `recordHandsPlayed` and `payHeldVitae` are written under.
func earnTurn(gs *state.GlobalState, turn []combat.Card) {
	earn(gs, achieve.Loaded().ByTurn(turn))
}

// bumpCounters adds a played turn to the lifetime tallies, in memory.
//
// **It writes nothing.** See settleCounters, which is where the figures reach the disk and where
// the count achievements are asked.
func bumpCounters(gs *state.GlobalState, turn []combat.Card) {
	if gs == nil || gs.Profile == nil {
		return
	}
	for name, n := range achieve.CountersFor(turn) {
		gs.Profile.Bump(name, n)
	}
}

// settleCounters asks the count achievements against the tallies as they now stand, and saves.
//
// **Called when a duel ends, however it ended.** A run given up on halfway through a fight has
// still played those cards, so this is not gated on winning — what a loss costs is the fight, not
// the record of what was swung.
//
// **The save is here rather than in bumpCounters** and is the whole point of the split: this runs
// once a duel where that runs once a turn.
func settleCounters(gs *state.GlobalState) {
	if gs == nil || gs.Profile == nil {
		return
	}
	earn(gs, achieve.Loaded().ByCounts(gs.Profile.Counters))
	saveProfile(gs)
}

// earn awards a list of keys, and queues a toast for each one that was actually new.
//
// **An unlock is a second key on a second list** *(owner's call, 2026-09-06)*. `profile.go` draws
// the line — an achievement is a record and changes nothing, an unlock is an input to the rules —
// so a relic behind an achievement reads the unlock and never the award. Nothing carries one yet;
// the bridge is here so that the day one does, it is a field in the JSON rather than a change here.
func earn(gs *state.GlobalState, keys []string) {
	if gs == nil || gs.Profile == nil || len(keys) == 0 {
		return
	}
	changed := false
	for _, key := range keys {
		if !gs.Profile.Award(key) {
			continue
		}
		changed = true
		gs.EarnedThisSession = append(gs.EarnedThisSession, key)

		if a, ok := achieve.Loaded().Find(key); ok {
			for _, u := range a.Unlocks {
				gs.Profile.Unlock(u)
			}
		}
	}
	if changed {
		saveProfile(gs)
	}
}
