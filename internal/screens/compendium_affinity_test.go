package screens

import (
	"slices"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// **Every moment and every verb of the relic grammar is sorted into an affinity, or named neutral.** A
// verb added to internal/combat fails here until compendium_affinity.go says what it is about — so a
// new kind of relic cannot land in SUPPORT without anybody having decided it belongs there.
func TestEveryMomentAndVerbIsSorted(t *testing.T) {
	moments := slices.Concat(attackMoments, defendMoments, apMoments, neutralMoments)
	for _, m := range combat.Moments() {
		if !slices.Contains(moments, m.String()) {
			t.Errorf("the moment %q is in no affinity list and not named neutral", m)
		}
	}
	verbs := slices.Concat(attackVerbs, defendVerbs, apVerbs, handVerbs, neutralVerbs)
	for _, v := range combat.RelicVerbs() {
		if !slices.Contains(verbs, v.String()) {
			t.Errorf("the verb %q is in no affinity list and not named neutral", v)
		}
	}
}

// **A flip is about both of its elements**: a relic turning lightning into ice is found under either
// chip, which is what "show me the arcane orbs" asks for.
func TestAFlipCarriesBothOfItsElements(t *testing.T) {
	r, ok := data.LoadRelics()["flip-lightning-to-ice"]
	if !ok {
		t.Skip("the catalog no longer holds the lightning-to-ice flip")
	}
	got := relicAffinities(r)
	for _, want := range []string{"lightning", "ice"} {
		if !slices.Contains(got, want) {
			t.Errorf("the lightning-to-ice flip carries %v, want %s among them", got, want)
		}
	}
}

// **Every relic stands under at least one affinity chip**, so the filter can reach any relic in the
// book, and every chip holds something. SUPPORT is the log's list to read when a new verb arrives:
// a relic landing there may be one the vocabulary should sort.
func TestEveryRelicHasAnAffinity(t *testing.T) {
	count := map[string]int{}
	var support []string
	for key, r := range data.LoadRelics() {
		as := relicAffinities(r)
		if len(as) == 0 {
			t.Errorf("%s carries no affinity", key)
		}
		if slices.Contains(as, affinitySupport) {
			support = append(support, key)
		}
		for _, a := range as {
			count[a]++
		}
	}
	for _, a := range affinityOrder() {
		if count[a] == 0 {
			t.Errorf("no relic carries the %s affinity", a)
		}
		t.Logf("%-9s %d", a, count[a])
	}
	slices.Sort(support)
	t.Logf("SUPPORT: %v", support)
}
