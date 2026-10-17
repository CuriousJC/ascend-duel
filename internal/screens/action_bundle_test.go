package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// TestEveryActionBundleDealsItsSizeWithoutRepeats. A bundle is a choice between different cards, so
// two of one card in it is a seat spent saying nothing — the bag's rule.
func TestEveryActionBundleDealsItsSizeWithoutRepeats(t *testing.T) {
	gs := testRun()
	found := 0
	for _, g := range session.Goods() {
		if g.Contains != session.ContentsCards {
			continue
		}
		found++
		dealt := dealActionCards(gs, g.Record, g.Size)
		if len(dealt) != g.Size {
			t.Errorf("%s dealt %d cards, want %d", g.Record, len(dealt), g.Size)
		}
		seen := map[[2]int]bool{}
		for _, c := range dealt {
			k := [2]int{int(c.Concept), int(c.Element)}
			if seen[k] {
				t.Errorf("%s dealt %s %s twice", g.Record, c.Element, c.Label())
			}
			seen[k] = true
		}
	}
	if found == 0 {
		t.Fatal("goods.json holds no action bundle")
	}
}

// TestAnActionBundleIsTheSameBundleTwice. The deal and its traits are functions of the run, the fight
// and the bundle, so opening it again — a resumed shop — shows what it showed.
func TestAnActionBundleIsTheSameBundleTwice(t *testing.T) {
	gs := testRun()
	g := goodHolding(t, session.ContentsCards)
	a, b := dealActionCards(gs, g.Record, g.Size), dealActionCards(gs, g.Record, g.Size)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("seat %d dealt %v then %v", i, a[i], b[i])
		}
	}
}

// TestTheActionBundleDrawsFromItsOwnStream. Dealing a bundle consumes nothing the shelf reads.
func TestTheActionBundleDrawsFromItsOwnStream(t *testing.T) {
	gs := testRun()
	before := shelfKeys(dealShelf(gs, shopRNG(gs, seeds.ShopStock), nil, nil))
	g := goodHolding(t, session.ContentsCards)
	dealActionCards(gs, g.Record, g.Size)
	if after := shelfKeys(dealShelf(gs, shopRNG(gs, seeds.ShopStock), nil, nil)); !sameKeys(before, after) {
		t.Fatalf("the shelf changed after a bundle was dealt: %v then %v", before, after)
	}
}

// TestBundledTraitsTurnUpAtTheirOdds. Over many fights, a card trait priced above zero turns up on
// some bundled cards and not on all of them.
func TestBundledTraitsTurnUpAtTheirOdds(t *testing.T) {
	g := goodHolding(t, session.ContentsCards)
	for _, tr := range session.CardTraitCatalog() {
		if tr.Percent <= 0 || tr.Percent >= 100 {
			continue
		}
		hits, dealt := 0, 0
		for seed := int64(0); seed < 400; seed++ {
			gs := testRun()
			gs.RunSeed = seed
			for _, c := range dealActionCards(gs, g.Record, g.Size) {
				dealt++
				if c.HasTrait(tr.Trait) {
					hits++
				}
			}
		}
		if hits == 0 || hits == dealt {
			t.Errorf("%s came up on %d of %d bundled cards at %d%%", tr.Trait, hits, dealt, tr.Percent)
		}
	}
}
