package achieve

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

func has(ms []Moment, name string) bool {
	for _, m := range ms {
		if m.Name == name {
			return true
		}
	}
	return false
}

func wild(c combat.Card, k combat.RiderKind) combat.Card { return c.SetRider(combat.Rider{Kind: k}) }

// TestADeckOfOneElementIsOneElement holds the element half of DeckShape: defend cards count, a
// wildcard fits, and a second element anywhere breaks it.
func TestADeckOfOneElementIsOneElement(t *testing.T) {
	fire := []combat.Card{card("Jab", combat.Fire), card("Bash", combat.Fire), card("Brace", combat.Fire)}
	if !has(DeckShape(fire), MomentDeckOneElement) {
		t.Error("an all-fire deck, defends included, is one element")
	}
	withWild := append(append([]combat.Card{}, fire...), wild(card("Cut", combat.Ice), combat.RiderWildElement))
	if !has(DeckShape(withWild), MomentDeckOneElement) {
		t.Error("a wild-element card fits whatever element the rest share")
	}
	mixed := append(append([]combat.Card{}, fire...), card("Block", combat.Ice))
	if has(DeckShape(mixed), MomentDeckOneElement) {
		t.Error("an ice defend card breaks a fire deck: defend cards count")
	}
	if has(DeckShape(nil), MomentDeckOneElement) {
		t.Error("an empty deck is no shape")
	}
}

// TestADeckOfOneFormAsksOnlyTheAttacks holds the form half: defend cards are not asked, a wild-form
// attack fits, and a second attack form breaks it.
func TestADeckOfOneFormAsksOnlyTheAttacks(t *testing.T) {
	stabs := []combat.Card{card("Jab", combat.Fire), card("Thrust", combat.Ice), card("Block", combat.Earth)}
	if !has(DeckShape(stabs), MomentDeckOneForm) {
		t.Error("stabs and a defend card are one form: defend is not asked")
	}
	withWild := append(append([]combat.Card{}, stabs...), wild(card("Bash", combat.Fire), combat.RiderWildForm))
	if !has(DeckShape(withWild), MomentDeckOneForm) {
		t.Error("a wild-form attack fits whatever form the rest share")
	}
	mixed := append(append([]combat.Card{}, stabs...), card("Cut", combat.Fire))
	if has(DeckShape(mixed), MomentDeckOneForm) {
		t.Error("a slash among stabs is two forms")
	}
	if has(DeckShape([]combat.Card{card("Brace", combat.Fire)}), MomentDeckOneForm) {
		t.Error("a deck with no attacks has no form to be one of")
	}
}

// TestTheHardestHitIsOneLandedHitOnTheCreature holds HardestHit to the creature's side and to a
// single hit rather than a sum.
func TestTheHardestHitIsOneLandedHitOnTheCreature(t *testing.T) {
	log := []combat.Event{
		{Kind: combat.KindDamage, Target: combat.SideB, Amount: 400},
		{Kind: combat.KindDamage, Target: combat.SideB, Amount: 700},
		{Kind: combat.KindDamage, Target: combat.SideA, Amount: 5000},
	}
	if got := HardestHit(log); got != 700 {
		t.Errorf("hardest hit = %d, want 700: the largest single hit on the creature", got)
	}
	if got := Loaded().ByMoment(HitLanded(999)); len(got) != 0 {
		t.Errorf("a 999 hit earned %v; the thresholds are thresholds", got)
	}
	if got := Loaded().ByMoment(HitLanded(1000)); len(got) == 0 {
		t.Error("a 1,000 hit earned nothing; it is the lowest threshold the catalog asks for")
	}
}

// TestCollectionReadsTheRoster walks Collection over the shipped roster: nothing beaten earns
// nothing, everything beaten earns all three.
func TestCollectionReadsTheRoster(t *testing.T) {
	motifs := data.LoadMotifs()
	if got := Collection(motifs, func(string, string) bool { return false }); len(got) != 0 {
		t.Errorf("nothing beaten earned %v", got)
	}
	all := Collection(motifs, func(string, string) bool { return true })
	for _, want := range []string{MomentBestiaryComplete, MomentRealmsComplete, MomentBossEveryElement} {
		if !has(all, want) {
			t.Errorf("everything beaten did not earn %s", want)
		}
	}
}

// TestEveryRealmHasABossToBeat holds the realms achievement earnable: a realm is beaten when its
// boss falls in its element, so a motif dealing an element with no boss in it is a realm nobody can
// ever beat — and an achievement that sits locked forever.
func TestEveryRealmHasABossToBeat(t *testing.T) {
	for key, m := range data.LoadMotifs() {
		for _, element := range data.AffinityElements {
			deals, boss := false, false
			for _, r := range m.Records {
				if r.HasAffinity(element) {
					deals = true
					boss = boss || r.Tier == data.TierBoss
				}
			}
			if deals && !boss {
				t.Errorf("%s deals %s but has no boss in it: that realm can never be beaten", key, element)
			}
		}
	}
}
