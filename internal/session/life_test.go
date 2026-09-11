package session

import "testing"

// TestAWoundIsCarriedIntoTheNextRoom. The whole point of the change: a duel used to start at full
// life whatever the last one cost, and damage is a fact about the floor now.
func TestAWoundIsCarriedIntoTheNextRoom(t *testing.T) {
	run := bare(t)

	run.WonFight(60, 100)

	if run.Hurt() != 40 {
		t.Errorf("finished a fight on 60 of 100 and carried %d, want 40", run.Hurt())
	}
	if got := run.LifeAtFightStart(100); got != 60 {
		t.Errorf("walked into the next room on %d, want 60", got)
	}
}

// TestABossWinHealsToFullAndRaisesTheCeiling. The stairway is the only thing that takes a wound
// away, and it pays a third more body on top.
func TestABossWinHealsToFullAndRaisesTheCeiling(t *testing.T) {
	run := bare(t)

	// The outer and inner rooms of floor one, both survived badly.
	run.WonFight(10, 100)
	run.WonFight(10, 100)
	if run.Hurt() != 90 {
		t.Fatalf("two hard rooms left a wound of %d, want 90", run.Hurt())
	}
	if run.BossWins() != 0 {
		t.Fatalf("an ordinary room counted as a stairway")
	}

	// The stairway.
	run.WonFight(10, 100)

	if run.Hurt() != 0 {
		t.Errorf("a boss win left a wound of %d, want none", run.Hurt())
	}
	if run.BossWins() != 1 {
		t.Errorf("beat one stairway and counted %d", run.BossWins())
	}
	if got := run.scaleLifeForBosses(100); got != 133 {
		t.Errorf("one boss raised a ceiling of 100 to %d, want 133", got)
	}
}

// TestTheBossBonusCompounds. Owner's call, 2026-09-06: each stairway is a third more than the body
// the run already had, not a third of the body it started with.
//
// **The rounding is down at every step**, which is why three bosses land on 234 rather than the 235
// the arithmetic in the round would give: 100 to 133 to 176 to 234. A ceiling is a whole number of
// hit points and a run is never handed a fraction of one.
func TestTheBossBonusCompounds(t *testing.T) {
	run := bare(t)
	run.bossWins = 3

	if got := run.scaleLifeForBosses(100); got != 234 {
		t.Errorf("three bosses raised a ceiling of 100 to %d, want 234", got)
	}
}

// TestAWoundNeverStartsAFightDead. A wound outliving the ceiling that was holding it is reachable
// by selling the relic that raised the ceiling, and a run that cannot start a fight is worse than
// one that starts it on a sliver.
func TestAWoundNeverStartsAFightDead(t *testing.T) {
	run := bare(t)
	run.hurt = 500

	if got := run.LifeAtFightStart(100); got != 1 {
		t.Errorf("a wound deeper than the ceiling started the fight on %d, want 1", got)
	}
}

// TestAFullHealthWinCarriesNothing. The wound is *set* by each win rather than accumulated, so a
// room walked out of untouched clears whatever the room before it cost.
func TestAFullHealthWinCarriesNothing(t *testing.T) {
	run := bare(t)

	run.WonFight(40, 100)
	run.WonFight(100, 100)

	if run.Hurt() != 0 {
		t.Errorf("walked out whole and still carried %d", run.Hurt())
	}
}
