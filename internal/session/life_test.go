package session

import "testing"

// TestAWoundIsCarriedIntoTheNextRoom. Damage is a fact about the realm: the wound a fight leaves
// walks into the next room, less the share of max life a win gives back.
func TestAWoundIsCarriedIntoTheNextRoom(t *testing.T) {
	run := bare(t)

	run.WonFight(60, 100)

	want := 40 - PercentOfLife(100, winHealPercent)
	if run.Hurt() != want {
		t.Errorf("finished a fight on 60 of 100 and carried %d, want %d", run.Hurt(), want)
	}
	if got := run.LifeAtFightStart(100); got != 100-want {
		t.Errorf("walked into the next room on %d, want %d", got, 100-want)
	}
}

// TestABossWinKeepsTheWoundAndRaisesTheCeiling. A portal room heals what any win heals and no more:
// it raises the ceiling by a flat amount and the wound stays the size the win left it, so the life
// under the ceiling rises by exactly that amount.
func TestABossWinKeepsTheWoundAndRaisesTheCeiling(t *testing.T) {
	run := bare(t)

	// The outer and inner rooms of realm one, then the portal room, all survived badly.
	run.WonFight(10, 100)
	run.WonFight(10, 100)
	run.WonFight(10, 100)

	wound := 90 - PercentOfLife(100, winHealPercent)
	if run.Hurt() != wound {
		t.Errorf("a boss win left a wound of %d, want the %d any win leaves", run.Hurt(), wound)
	}
	if run.BossWins() != 1 {
		t.Errorf("beat one portal room and counted %d", run.BossWins())
	}
	ceiling := run.raiseLifeForBosses(100)
	if ceiling != 100+bossLifeBonus {
		t.Errorf("one boss raised a ceiling of 100 to %d, want %d", ceiling, 100+bossLifeBonus)
	}
	if got := run.LifeAtFightStart(ceiling); got != ceiling-wound {
		t.Errorf("walked into realm two on %d, want %d", got, ceiling-wound)
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
