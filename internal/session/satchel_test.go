package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// anyWithEssenceTarget is an essence from the shipped catalog that does this, whichever one it is —
// the posture anyWithTarget takes for a rune, and for the same reason: a test's subject is the
// grammar rather than the record, so a rename must not break a test that was not about that record.
func anyWithEssenceTarget(t *testing.T, target EssenceTarget) (string, Essence) {
	t.Helper()
	for _, key := range essenceOrder {
		if w := essences[key]; w.Target == target {
			return key, w
		}
	}
	t.Fatalf("the catalog holds no %s essence, so nothing exercises it", target)
	return "", Essence{}
}

// TestTheSatchelCarriesEssencesIntoAFight. An essence taken at a reward is spent there; one stowed
// is carried, and the consumables pane is where it is spent from.
func TestTheSatchelCarriesEssencesIntoAFight(t *testing.T) {
	key, w := anyWithEssenceTarget(t, TargetPromote)
	run := runWith(combat.Plain(combat.Jab))

	if run.Stow("no-such-essence") {
		t.Error("an essence the catalog has not got was stowed")
	}
	if !run.Stow(key) || !run.Stow(key) {
		t.Fatal("a catalog essence was refused the satchel")
	}
	if got := run.StowCount(); got != 2 {
		t.Errorf("the satchel holds %d, want 2", got)
	}

	// **Two of the same are two cards to spend**, which is why the satchel is a list rather than a
	// count per key — the sack's rule and the pouch's.
	if got := run.Stowed(); len(got) != 2 || got[0] != key || got[1] != key {
		t.Errorf("the satchel reads %v", got)
	}

	// It stands in the merged row as its own kind, between the runes and the stones.
	var seen int
	for _, c := range run.Consumables() {
		if c.Kind == ConsumableEssence {
			seen++
			if c.Essence.Name != w.Name {
				t.Errorf("a stowed essence draws as %q", c.Essence.Name)
			}
			if c.Name() != w.Name {
				t.Errorf("a stowed essence names itself %q", c.Name())
			}
		}
	}
	if seen != 2 {
		t.Errorf("the pane offers %d essences, want 2", seen)
	}

	if !run.DropStowed(0) {
		t.Error("a stowed essence could not be dropped")
	}
	if run.DropStowed(9) {
		t.Error("a satchel position that is not there was dropped")
	}
	if got := run.StowCount(); got != 1 {
		t.Errorf("after one spend the satchel holds %d, want 1", got)
	}
}

// TestAnEssenceIsAimedByIdentityMidFight. A fight holds copies of the run's cards across three
// piles, so a card in the hand is found by its identity and never by a deck position.
func TestAnEssenceIsAimedByIdentityMidFight(t *testing.T) {
	_, promote := anyWithEssenceTarget(t, TargetPromote)
	run := runWith(combat.Plain(combat.Poke), combat.Plain(combat.Jab))

	second := ids(run)[1]
	if !run.CanApplyTo(promote, second) {
		t.Fatal("a Jab could not be promoted by identity")
	}
	if run.CanApplyTo(promote, 0) {
		t.Error("an identity the run does not hold was offered")
	}
	if run.ApplyTo(promote, 0) {
		t.Error("an identity the run does not hold claimed to work")
	}

	up, _ := combat.Neighbor(combat.Jab, 1)
	if !run.ApplyTo(promote, second) {
		t.Fatal("promoting by identity was refused")
	}

	// **The card that changed is the one that was named**, and it keeps the handle it had.
	got, ok := run.CardByID(second)
	if !ok {
		t.Fatal("the promoted card lost its identity")
	}
	if got.Concept != up {
		t.Errorf("the named card became %v, want %v", got.Concept, up)
	}
	if first, _ := run.Card(0); first.Concept != combat.Poke {
		t.Errorf("the card beside it became %v", first.Concept)
	}
}

// TestAStowedDuplicateReachesTheHand. An essence spent between fights only has to put the copy in
// the deck; one spent in the middle of a duel hands it over so the screen can seat it, exactly as a
// duplicate rune's copy is handed over.
func TestAStowedDuplicateReachesTheHand(t *testing.T) {
	_, dupe := anyWithEssenceTarget(t, TargetDuplicate)
	run := runWith(combat.Plain(combat.Bash))
	id := ids(run)[0]

	if !run.ApplyTo(dupe, id) {
		t.Fatal("duplicating by identity was refused")
	}
	minted := run.Duplicated()
	if len(minted) != 1 {
		t.Fatalf("a duplicate handed over %d cards, want 1", len(minted))
	}
	if minted[0].Concept != combat.Bash {
		t.Errorf("the copy is a %v", minted[0].Concept)
	}
	if minted[0].ID == id {
		t.Error("the copy carries the identity it was copied from")
	}

	// **Cleared by the next one rather than by the reader**, so it only ever says what the last
	// consumable did.
	_, promote := anyWithEssenceTarget(t, TargetPromote)
	if !run.ApplyTo(promote, id) {
		t.Fatal("promoting after a duplicate was refused")
	}
	if got := run.Duplicated(); len(got) != 0 {
		t.Errorf("an essence that minted nothing handed over %d cards", len(got))
	}
}

// TestTheSatchelSurvivesASaveAndAResume. A carried essence is a thing the player owns, so it is
// written down with the sack and the pouch.
func TestTheSatchelSurvivesASaveAndAResume(t *testing.T) {
	key, _ := anyWithEssenceTarget(t, TargetDemote)
	run := runWith(combat.Plain(combat.Jab))
	if !run.Stow(key) {
		t.Fatal("a catalog essence was refused the satchel")
	}

	snap := run.Snapshot(1)
	if len(snap.Satchel) != 1 || snap.Satchel[0] != key {
		t.Fatalf("the snapshot writes the satchel as %v", snap.Satchel)
	}

	back, _, err := Resume(nil, nil, snap)
	if err != nil {
		t.Fatalf("a run carrying an essence would not resume: %v", err)
	}
	if got := back.Stowed(); len(got) != 1 || got[0] != key {
		t.Errorf("the resumed satchel reads %v", got)
	}
}
