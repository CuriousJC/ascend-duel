package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/seeds"
)

// lockedRelic is any relic behind an unlock, and its key. Read off the catalog rather than named,
// so the tests outlive whichever relic happens to be gated today.
func lockedRelic(t *testing.T) (string, string) {
	t.Helper()
	for _, key := range Relics() {
		if u := RelicUnlock(key); u != "" {
			return key, u
		}
	}
	t.Skip("no relic in the catalog is behind an unlock")
	return "", ""
}

func startedRun(t *testing.T) *Session {
	t.Helper()
	motifs, shape := rosters(t)
	seed, err := seeds.Parse(theSeed)
	if err != nil {
		t.Fatal(err)
	}
	return Start(motifs, shape, seed)
}

// TestALockedRelicIsOfferedOnlyToARunHoldingItsUnlock is the gate: the default pool leaves it out,
// and the unlock puts it back.
func TestALockedRelicIsOfferedOnlyToARunHoldingItsUnlock(t *testing.T) {
	key, unlock := lockedRelic(t)

	s := startedRun(t)
	s.TakeUnlocks(nil)
	if s.Offers(key) {
		t.Errorf("%s is behind %q and a run holding nothing was offered it", key, unlock)
	}
	for _, k := range s.OfferableRelics() {
		if k == key {
			t.Errorf("%s is in the offerable pool of a run that has not unlocked it", key)
		}
	}

	s.TakeUnlocks([]string{unlock})
	if !s.Offers(key) {
		t.Errorf("%s should be offered once %q is held", key, unlock)
	}
	if got, all := len(s.OfferableRelics()), len(Relics()); got > all {
		t.Errorf("offerable pool has %d relics, more than the catalog's %d", got, all)
	}
}

// TestEveryDefaultRelicIsOfferedToANewPlayer holds that a relic with no Unlock is on every shelf from
// the first run.
func TestEveryDefaultRelicIsOfferedToANewPlayer(t *testing.T) {
	s := startedRun(t)
	s.TakeUnlocks(nil)
	for _, key := range Relics() {
		if RelicUnlock(key) == "" && !s.Offers(key) {
			t.Errorf("%s is in the default pool and a new player was not offered it", key)
		}
	}
}

// TestARunKeepsTheUnlocksItStartedWith is the "next run" rule: the set is a copy, so the profile
// moving on after the run started does not move the run's pool.
func TestARunKeepsTheUnlocksItStartedWith(t *testing.T) {
	key, unlock := lockedRelic(t)
	held := []string{"not-" + unlock}
	s := startedRun(t)
	s.TakeUnlocks(held)
	held[0] = unlock // the profile's list is sorted in place as it grows
	if s.Offers(key) {
		t.Error("an unlock earned after the run started reached this run's shelf")
	}
}

// TestTheUnlocksSurviveASaveAndAnOldSaveHasNone is the resume half: the set comes back as it went
// out, and a snapshot written before runs carried one reads as not yet taken rather than as empty.
func TestTheUnlocksSurviveASaveAndAnOldSaveHasNone(t *testing.T) {
	key, unlock := lockedRelic(t)
	motifs, shape := rosters(t)
	seed, _ := seeds.Parse(theSeed)

	s := startedRun(t)
	s.TakeUnlocks([]string{unlock})
	back, _, err := Resume(motifs, shape, s.Snapshot(seed))
	if err != nil {
		t.Fatal(err)
	}
	if !back.UnlocksTaken() || !back.Offers(key) {
		t.Errorf("a resumed run lost its unlocks: %v", back.Unlocks())
	}

	none := startedRun(t)
	none.TakeUnlocks(nil)
	back, _, err = Resume(motifs, shape, none.Snapshot(seed))
	if err != nil {
		t.Fatal(err)
	}
	if !back.UnlocksTaken() {
		t.Error("a run started holding no unlocks must resume as having taken them, not as an old save")
	}

	old := s.Snapshot(seed)
	old.Unlocks = nil
	back, _, err = Resume(motifs, shape, old)
	if err != nil {
		t.Fatal(err)
	}
	if back.UnlocksTaken() {
		t.Error("a save with no unlocks field must resume as not yet taken, so the profile's are handed over")
	}
}

// TestEveryUnlockIsGrantedAndRead is the load check, asked again here so a failure names the key
// rather than arriving as a panic at init.
func TestEveryUnlockIsGrantedAndRead(t *testing.T) {
	granted := grantedUnlocks()
	read := map[string]bool{}
	for key, r := range data.LoadRelics() {
		if r.Unlock == "" {
			continue
		}
		read[r.Unlock] = true
		if !granted[r.Unlock] {
			t.Errorf("%s is behind %q, which no achievement grants", key, r.Unlock)
		}
	}
	for u := range granted {
		if !read[u] {
			t.Errorf("unlock %q is granted and opens nothing", u)
		}
	}
}
