package screens

import (
	"math/rand"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/achieve"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// anUnlockingAchievement is any achievement that grants an unlock, the key it grants, and a relic
// behind that key — read off the catalogs so the test outlives today's records.
func anUnlockingAchievement(t *testing.T) (string, string, string) {
	t.Helper()
	for _, key := range session.Relics() {
		u := session.RelicUnlock(key)
		if u == "" {
			continue
		}
		for _, a := range achieve.Loaded().All() {
			for _, granted := range a.Unlocks {
				if granted == u {
					return a.APIName, u, key
				}
			}
		}
	}
	t.Skip("no achievement grants an unlock a relic reads")
	return "", "", ""
}

// TestEarningAnAchievementOpensItsRelicOnTheNextRun is the whole loop: the award lands the unlock,
// the run already under way keeps its pool, and the next run is offered the relic.
func TestEarningAnAchievementOpensItsRelicOnTheNextRun(t *testing.T) {
	award, unlock, relic := anUnlockingAchievement(t)
	gs := saveState(t)
	seed, _ := seeds.Parse("00H602")

	NewRunOn(gs, seed, false)
	if gs.Run.Offers(relic) {
		t.Fatalf("a new player's run was offered %s, which is behind %q", relic, unlock)
	}

	earn(gs, []string{award})
	if !gs.Profile.Unlocked(unlock) {
		t.Fatalf("earning %s must unlock %q", award, unlock)
	}
	if gs.Run.Offers(relic) {
		t.Error("an unlock earned mid-run reached the run already under way")
	}

	NewRunOn(gs, seed, true)
	if !gs.Run.Offers(relic) {
		t.Errorf("the next run must be offered %s, a chosen seed's included", relic)
	}
}

// TestReconcileGrantsWhatAnEarlierAwardOpens is the backfill: a profile holding the award but not
// the unlock — earned before the achievement carried one — is handed it at launch.
func TestReconcileGrantsWhatAnEarlierAwardOpens(t *testing.T) {
	award, unlock, _ := anUnlockingAchievement(t)
	gs := saveState(t)
	gs.Run = nil
	gs.Profile.Achievements = append(gs.Profile.Achievements, award)

	ReconcileUnlocks(gs)
	if !gs.Profile.Unlocked(unlock) {
		t.Errorf("a profile holding %s must be given %q at launch", award, unlock)
	}
}

// TestTheShelfNeverDealsALockedRelic deals a new player's shelf for two hundred fights and fails on a
// relic the run was not started holding.
func TestTheShelfNeverDealsALockedRelic(t *testing.T) {
	gs := saveState(t)
	gs.Run.TakeUnlocks(nil)
	for fight := 0; fight < 200; fight++ {
		rng := rand.New(rand.NewSource(seeds.ForFight(gs.RunSeed, seeds.ShopStock, fight)))
		for _, item := range dealShelf(gs, rng, nil) {
			if !gs.Run.Offers(item.key) {
				t.Fatalf("the shelf dealt %s, which this run has not unlocked", item.key)
			}
		}
	}
}
