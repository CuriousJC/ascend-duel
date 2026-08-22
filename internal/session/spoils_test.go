package session

import "testing"

// TestAWinPaysInterestThenTheRoomAndTheLife. The three parts of a payout, and the order they are
// decided in: interest is on what the run walked *out of the fight* holding, never on what the win
// is about to pay it. That order is the whole reason WonFight freezes the figures.
func TestAWinPaysInterestThenTheRoomAndTheLife(t *testing.T) {
	run := bare(t)
	run.vitae = 20

	run.WonFight(63)

	got := run.Spoils()
	if got.Propagated != 4 {
		t.Errorf("a purse of 20 earned %d, want 4", got.Propagated)
	}
	if got.FromLife != 6 {
		t.Errorf("63 life left paid %d, want 6", got.FromLife)
	}
	if got.FromRoom != 3 {
		t.Errorf("floor 1's outer room paid %d, want 3", got.FromRoom)
	}
	if run.Vitae() != 20 {
		t.Errorf("the purse moved to %d before anything was claimed", run.Vitae())
	}
}

// TestTheRoomIsWorthThreeFourFive. Outer, inner, stairway — and flat for the whole climb, so a
// floor-8 boss pays what floor 1's does.
func TestTheRoomIsWorthThreeFourFive(t *testing.T) {
	for _, tc := range []struct{ fight, want int }{
		{0, 3}, {1, 4}, {2, 5}, {3, 3}, {4, 4}, {5, 5}, {21, 3}, {23, 5},
	} {
		run := bare(t)
		run.fight = tc.fight
		run.WonFight(0)

		if got := run.Spoils().FromRoom; got != tc.want {
			t.Errorf("fight %d paid %d, want %d", tc.fight, got, tc.want)
		}
	}
}

// TestSoulTakerPaysTheRoomFlat. The `prizes-dealt` ring was retargeted when the vitae card went;
// what it moves now is the room award, and it is still flat rather than a scaling.
func TestSoulTakerPaysTheRoomFlat(t *testing.T) {
	run := wearing(t, "soul-taker-ring")
	run.fight = 2
	run.WonFight(0)

	if got := run.Spoils().FromRoom; got != 10 {
		t.Errorf("a stairway paid %d wearing Soul Taker, want 10", got)
	}
}

// TestLifeLeftIsATenthRoundedDown. A win on nine life is worth nothing from this half of the
// payout, which is the honest reading of "a tenth" and the owner's call.
func TestLifeLeftIsATenthRoundedDown(t *testing.T) {
	for _, tc := range []struct{ life, want int }{{0, 0}, {9, 0}, {10, 1}, {65, 6}, {100, 10}} {
		run := bare(t)
		run.WonFight(tc.life)

		if got := run.Spoils().FromLife; got != tc.want {
			t.Errorf("%d life left paid %d, want %d", tc.life, got, tc.want)
		}
	}
}

// TestClaimingPaysOnce. Each part is cleared as it is claimed, so a screen re-entered mid-narration
// cannot pay a win twice.
func TestClaimingPaysOnce(t *testing.T) {
	run := bare(t)
	run.vitae = 20
	run.WonFight(63)
	want := run.Vitae() + run.Spoils().Total()

	run.ClaimPropagation()
	run.ClaimPropagation()
	run.ClaimFromLife()
	run.ClaimFromRoom()
	run.ClaimSpoils()

	if run.Vitae() != want {
		t.Errorf("the run ended on %d vitae, want %d", run.Vitae(), want)
	}
	if run.Spoils().Total() != 0 {
		t.Errorf("%d vitae is still owed after claiming everything", run.Spoils().Total())
	}
}

// TestAdvancePaysWhatWasNeverClaimed. The post-battle screen hands the parts over one at a time as
// it narrates them; this is what makes a win still pay in full when nothing narrates it.
func TestAdvancePaysWhatWasNeverClaimed(t *testing.T) {
	run := bare(t)
	run.vitae = 20
	run.WonFight(63)
	want := run.Vitae() + run.Spoils().Total()

	run.Advance() // fight -> reward
	run.Advance() // reward -> shop, where anything unnarrated is paid

	if run.Vitae() != want {
		t.Errorf("an unnarrated win left the run on %d vitae, want %d", run.Vitae(), want)
	}
}

// TestAdvancingIntoTheRewardKeepsTheSpoils. **The one that got away** *(2026-08-22)*: winning a
// fight advances the run from the duel to the reward screen, so a claim on every Advance emptied
// the payout a tick before the screen that narrates it was built — leaving it to read out a win
// worth nothing while the purse held the money.
func TestAdvancingIntoTheRewardKeepsTheSpoils(t *testing.T) {
	run := bare(t)
	run.vitae = 20
	run.WonFight(63)
	owed := run.Spoils().Total()

	run.Advance() // fight -> reward, the move a win makes

	if got := run.Spoils().Total(); got != owed {
		t.Errorf("arriving at the reward screen owed %d, want %d", got, owed)
	}
	if run.Phase() != PhaseReward {
		t.Fatalf("a win advanced to %s, want reward", run.Phase())
	}

	run.Advance() // reward -> shop, the move that pays whatever was not narrated
	if got := run.Spoils().Total(); got != 0 {
		t.Errorf("leaving the reward screen left %d unclaimed", got)
	}
}
