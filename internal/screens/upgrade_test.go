package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// These tests walk tables and compare constants. They create no `ebiten.Image` and need no window,
// which is the narrow exception `internal/screens` tests are allowed under — see CLAUDE.md.

// **Every rider draws.** A parasite the player spent on a card they cannot see they spent it on is
// the exact failure the whole upgrade mechanic was written to fix, and a rider added without a line
// in upgradeForRider would be silently invisible — the zero value is UpgradeNone, so it would draw
// as an ordinary card rather than fail anywhere.
func TestEveryRiderKindIsDrawn(t *testing.T) {
	for _, k := range combat.RiderKinds() {
		u, ok := upgradeForRider[k]
		if !ok {
			t.Errorf("rider %s has no upgrade — a card carrying it would look ordinary", k)
			continue
		}
		if u == systems.UpgradeNone {
			t.Errorf("rider %s is drawn as UpgradeNone, which is no drawing at all", k)
		}
	}

	// The other direction: a table entry for a rider the rules no longer have is a drawing nothing
	// can trigger, sitting here looking like coverage.
	known := map[combat.RiderKind]bool{}
	for _, k := range combat.RiderKinds() {
		known[k] = true
	}
	for k := range upgradeForRider {
		if !known[k] {
			t.Errorf("upgradeForRider names rider %d, which the rules do not have", k)
		}
	}
}

// **No two riders draw the same upgrade.** Two parasites that painted a card identically would be
// two purchases the player cannot tell apart afterwards, which is the same failure as a rider that
// draws nothing — one step further along.
func TestNoTwoRidersShareAnUpgrade(t *testing.T) {
	seen := map[systems.Upgrade]combat.RiderKind{}
	for _, k := range combat.RiderKinds() {
		u := upgradeForRider[k]
		if first, ok := seen[u]; ok {
			t.Errorf("riders %s and %s both draw as %s", first, k, u)
			continue
		}
		seen[u] = k
	}
}

// **An unridden card carries no upgrade**, which is what makes the common case the zero value and
// what stops every plain card in the deck being washed in something.
func TestAnUnriddenCardHasNoUpgrade(t *testing.T) {
	if got := upgradeOf(combat.Plain(combat.Strike)); got != systems.UpgradeNone {
		t.Errorf("a plain card draws as %s", got)
	}
}

// **The upgrade a card wears is the last one put on it.** The rules half is
// combat.TestACardCarriesOneUpgradeAndNoMore; this is the drawing agreeing with it, because a card
// whose face still said gold after a Leech would be the one place last-one-wins is invisible.
func TestTheFaceShowsTheLastUpgradePutOn(t *testing.T) {
	c := combat.Plain(combat.Strike).
		SetRider(combat.Rider{Kind: combat.RiderGolden, Amount: 5}).
		SetRider(combat.Rider{Kind: combat.RiderHealOnPlay, Amount: 10})
	if got := upgradeOf(c); got != systems.UpgradeHeal {
		t.Errorf("a gold card made into a leech draws as %s", got)
	}
}

// **Every rider a parasite can attach is one the catalogue actually grants**, and every rider the
// rules have is attachable. An upgrade nobody can acquire is a drawing with no way into the game,
// which is as invisible as a rider with no drawing — see tools/upgradesheet, which says so on the
// page for the same reason.
func TestEveryRiderKindIsGrantedBySomething(t *testing.T) {
	granted := map[combat.RiderKind]bool{}
	for _, p := range session.Parasites() {
		if p.Target == session.ParasiteRider {
			granted[p.Rider] = true
		}
	}
	for _, k := range combat.RiderKinds() {
		if !granted[k] {
			t.Errorf("nothing in data/parasites.json attaches rider %s, so the %s upgrade "+
				"cannot be acquired", k, upgradeForRider[k])
		}
	}
}

// **Every rider has a sentence.** parasiteRiderLine had one arm and a "does nothing" default, which
// was a lie about seven of the eight kinds that existed at the time.
func TestEveryRiderKindHasALine(t *testing.T) {
	for _, k := range combat.RiderKinds() {
		if line := parasiteRiderLine(k); line == "does nothing" {
			t.Errorf("rider %s falls through to the default line", k)
		}
	}
}
