package session

import "testing"

// TestAWoundIsCarriedIntoTheNextRoom. The whole point of the change: a duel used to start at full
// life whatever the last one cost, and damage is a fact about the realm now.
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

// TestABossWinKeepsTheWoundAndRaisesTheCeiling. Nothing between fights heals, the portal room
// included: it raises the ceiling by a flat amount and the wound stays the size it was, so the life
// under the ceiling rises by exactly that amount.
func TestABossWinKeepsTheWoundAndRaisesTheCeiling(t *testing.T) {
	run := bare(t)

	// The outer and inner rooms of realm one, then the portal room, all survived badly.
	run.WonFight(10, 100)
	run.WonFight(10, 100)
	run.WonFight(10, 100)

	if run.Hurt() != 90 {
		t.Errorf("a boss win left a wound of %d, want the 90 it was walked out with", run.Hurt())
	}
	if run.BossWins() != 1 {
		t.Errorf("beat one portal room and counted %d", run.BossWins())
	}
	ceiling := run.raiseLifeForBosses(100)
	if ceiling != 100+bossLifeBonus {
		t.Errorf("one boss raised a ceiling of 100 to %d, want %d", ceiling, 100+bossLifeBonus)
	}
	if got := run.LifeAtFightStart(ceiling); got != 10+bossLifeBonus {
		t.Errorf("walked into realm two on %d, want %d", got, 10+bossLifeBonus)
	}
}

// TestTheBossBonusIsFlat. Each portal room adds the same step to the body, so three bosses are
// three steps rather than a compounding share.
func TestTheBossBonusIsFlat(t *testing.T) {
	run := bare(t)
	run.bossWins = 3

	if got := run.raiseLifeForBosses(100); got != 100+3*bossLifeBonus {
		t.Errorf("three bosses raised a ceiling of 100 to %d, want %d", got, 100+3*bossLifeBonus)
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
