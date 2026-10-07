package session

// **Which relics a run may be offered, read off the player's unlocks.**
//
// A relic record may carry an `Unlock` key, and a run started for a player whose profile does not
// hold that key never sees it on a shelf or inside a sealed good. The profile owns the unlocks and
// this package does not import it, so the run is handed the set when it starts — `TakeUnlocks` —
// and keeps it for its whole life.
//
// **A run's pool is fixed when it starts** *(owner's call, 2026-10-03)*. An unlock earned partway
// through a run opens the relic on the *next* run, so a retried fight walks into the same shop and
// a resumed run deals what it would have dealt without the quit. The set is saved with the run for
// that reason.
//
// **A chosen seed reads the profile like any other run** *(owner's call, 2026-10-03)*. A code is
// "this journey, given your unlocks" rather than one shop for everybody; it still earns nothing.
//
// **Two checks at load, one each way.** A relic's `Unlock` that no achievement grants is a relic no
// shelf can ever offer; an achievement granting a key no relic reads is a reward that opens
// nothing. Both are failures nothing else would notice, so both panic.

import (
	"fmt"
	"sort"

	"github.com/curiousjc/ascend-duel/data"
)

// relicUnlocks is each locked relic's unlock key, by record. A relic absent from it is in the
// default pool.
var relicUnlocks = loadRelicUnlocks()

func loadRelicUnlocks() map[string]string {
	granted := grantedUnlocks()
	out := map[string]string{}
	read := map[string]bool{}
	for key, r := range data.LoadRelics() {
		if r.Unlock == "" {
			continue
		}
		if !granted[r.Unlock] {
			panic(fmt.Sprintf("relics.json: %s is behind unlock %q, which no achievement grants",
				key, r.Unlock))
		}
		out[key] = r.Unlock
		read[r.Unlock] = true
	}
	for _, a := range data.LoadAchievements() {
		for _, u := range a.Unlocks {
			if !read[u] {
				panic(fmt.Sprintf("achievements.json: %s grants unlock %q, which nothing reads",
					a.APIName, u))
			}
		}
	}
	return out
}

// grantedUnlocks is every unlock key some achievement grants.
func grantedUnlocks() map[string]bool {
	out := map[string]bool{}
	for _, a := range data.LoadAchievements() {
		for _, u := range a.Unlocks {
			out[u] = true
		}
	}
	return out
}

// RelicUnlock is the unlock key a relic is behind, or "" for a relic in the default pool.
func RelicUnlock(key string) string { return relicUnlocks[key] }

// RelicsBehind is every relic behind one unlock key, sorted.
func RelicsBehind(unlock string) []string {
	var out []string
	for key, u := range relicUnlocks {
		if u == unlock {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

// checkUnlock refuses an archived record's Unlock that no achievement grants, so a relic moved back
// out of the archive still loads.
func checkUnlock(r data.RelicData) error {
	if r.Unlock != "" && !grantedUnlocks()[r.Unlock] {
		return fmt.Errorf("%s is behind unlock %q, which no achievement grants", r.RelicRecord, r.Unlock)
	}
	return nil
}

// TakeUnlocks hands the run the player's unlocks, once, as it starts or as an old save without
// them resumes. **A copy**, so the profile moving on does not move the run's pool.
func (s *Session) TakeUnlocks(keys []string) {
	s.unlocks = append([]string{}, keys...)
	sort.Strings(s.unlocks)
}

// UnlocksTaken reports whether the run has been handed its unlocks. A save written before runs
// carried them has not, and is handed the profile's as it resumes.
func (s *Session) UnlocksTaken() bool { return s.unlocks != nil }

// Unlocks is the set the run was started with, sorted.
func (s *Session) Unlocks() []string { return append([]string{}, s.unlocks...) }

// Offers reports whether this run may be offered a relic: it is in the default pool, or the run
// was started holding its unlock.
//
// **The offer, never the wearing.** A relic already on the run — put there by a scenario or by
// StartingRelics — keeps firing whatever this says.
func (s *Session) Offers(key string) bool {
	u := relicUnlocks[key]
	if u == "" {
		return true
	}
	i := sort.SearchStrings(s.unlocks, u)
	return i < len(s.unlocks) && s.unlocks[i] == u
}

// OfferableRelics is every relic this run may be offered, sorted — Relics with the locked ones it
// was not started holding left out. **Every shelf and every sealed good draws from this**, never
// from Relics.
func (s *Session) OfferableRelics() []string {
	var out []string
	for _, key := range Relics() {
		if s.Offers(key) {
			out = append(out, key)
		}
	}
	return out
}
