package session

// **The journey won: the last realm's boss beaten.**
//
// The journey has a configured height — `Realms` in data/journey.json — and beating the boss of the
// realm at that height wins the run. **The run does not have to end there.** The player is asked
// whether to claim the victory, which ends the run as a win, or to walk on into the endless journey
// past it; either way the victory is the run's from the moment the boss fell, and it is counted
// then, so a player who walks on and falls later has still won.

import "github.com/curiousjc/ascend-duel/internal/journey"

// StatVictories is how many runs have won the journey — counted when the last boss falls, whether
// the run is claimed there or walked on into the endless journey.
const StatVictories = "victories"

// wonJourneyAt reports whether the fight at this index is the boss of the journey's last realm.
func (s *Session) wonJourneyAt(fight int) bool {
	if s.journey == nil || s.journey.Realms() < 1 {
		return false
	}
	return journey.RoomOf(fight) == journey.RoomPortal && journey.RealmOf(fight) == s.journey.Realms()
}

// Victorious reports whether this run has won the journey.
func (s *Session) Victorious() bool { return s.victorious }

// VictoryDue reports whether the run is standing straight after the fight that won the journey —
// which is the one moment the victory station is shown.
func (s *Session) VictoryDue() bool { return s.fight > 0 && s.wonJourneyAt(s.fight-1) }

// noteVictory marks the run as having won the journey and counts it, if the fight just won was the
// last boss. **Called from WonFight before the room counter moves**, so it reads the fight that was
// won rather than the one about to be met; once set it is saved with the run, and the realm it
// checks is met only once, so nothing counts it twice.
func (s *Session) noteVictory() {
	if s.victorious || !s.wonJourneyAt(s.fight) {
		return
	}
	s.victorious = true
	s.count(StatVictories, 1)
}
