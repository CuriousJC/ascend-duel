package session

import (
	"sort"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/journey"
	"github.com/curiousjc/ascend-duel/internal/seeds"
)

// tonicRun is a run standing in a shop of the given realm, with a purse deep enough for anything and
// the catalog walked in the order given.
func tonicRun(t *testing.T, order ...string) *Session {
	t.Helper()
	s := New(testDeck())
	s.AddVitae(500)
	s.tonicOrder = order
	s.fight = 1 // the shop after realm one's outer room
	return s
}

// inRealm moves the run to the shop after the outer room of a realm.
func inRealm(s *Session, realm int) { s.fight = journey.FirstFightInRealm(realm) + 1 }

// The order is a permutation of the catalog and a function of the run code alone.
func TestTheTonicOrderIsTheWholeCatalogShuffledBySeed(t *testing.T) {
	seed, _ := seeds.Parse(theSeed)
	a, b := newTonicOrder(seed), newTonicOrder(seed)
	if len(a) != len(Tonics()) {
		t.Fatalf("the order holds %d tonics, the catalog %d", len(a), len(Tonics()))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("one run code shuffled two orders: %v and %v", a, b)
		}
	}
	got := append([]string(nil), a...)
	sort.Strings(got)
	var want []string
	for _, tn := range Tonics() {
		want = append(want, tn.Record)
	}
	sort.Strings(want)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the order is not the catalog: %v", a)
		}
	}
}

// A realm offers one tonic at every shop in it, drinking it empties the seat for the rest of the
// realm, and the next realm offers the next one.
func TestARealmOffersOneTonicAndDrinkingItEmptiesTheSeat(t *testing.T) {
	s := tonicRun(t, "pressure-tonic", "jewellers-tonic", "jugglers-tonic")

	got, ok := s.OfferTonic()
	if !ok || got.Record != "pressure-tonic" {
		t.Fatalf("realm one offered %q, wanted the first in the order", got.Record)
	}
	s.fight = 2 // the realm's inner-room shop
	if again, _ := s.OfferTonic(); again.Record != "pressure-tonic" {
		t.Fatalf("a second shop in the realm offered %q", again.Record)
	}
	if !s.DrinkTonic("pressure-tonic") {
		t.Fatal("could not drink the tonic on offer")
	}
	s.fight = 3 // the portal room's shop, still realm one
	if _, ok := s.OfferTonic(); ok {
		t.Fatal("the seat refilled inside the realm it was drunk in")
	}
	if s.DrinkTonic("pressure-tonic") {
		t.Fatal("a tonic was drunk twice")
	}

	inRealm(s, 2)
	if next, _ := s.OfferTonic(); next.Record != "jewellers-tonic" {
		t.Fatalf("realm two offered %q, wanted the next in the order", next.Record)
	}
}

// A tonic passed over is gone for good.
func TestATonicNotBoughtIsNeverOfferedAgain(t *testing.T) {
	s := tonicRun(t, "pressure-tonic", "jewellers-tonic")
	s.OfferTonic()
	inRealm(s, 2)
	if got, _ := s.OfferTonic(); got.Record != "jewellers-tonic" {
		t.Fatalf("realm two offered %q after realm one's was passed", got.Record)
	}
	inRealm(s, 3)
	if got, ok := s.OfferTonic(); ok {
		t.Fatalf("realm three offered %q with the order exhausted", got.Record)
	}
}

// A tonic whose requirement has not been drunk is skipped, not spent — and comes back once it is
// allowed.
func TestATonicNotYetAllowedIsSkippedAndComesBack(t *testing.T) {
	s := tonicRun(t, "olympians-tonic", "athletes-tonic", "jewellers-tonic")

	if got, _ := s.OfferTonic(); got.Record != "athletes-tonic" {
		t.Fatalf("realm one offered %q; the Olympian's needs the Athlete's first", got.Record)
	}
	s.DrinkTonic("athletes-tonic")

	inRealm(s, 2)
	if got, _ := s.OfferTonic(); got.Record != "olympians-tonic" {
		t.Fatalf("realm two offered %q; the skipped Olympian's is allowed now", got.Record)
	}
}

// Ten percent off, rounded up so it is always at least one vitae, and never below one.
func TestTheJewellersCutIsAtLeastOneAndNeverFree(t *testing.T) {
	s := tonicRun(t, "jewellers-tonic")
	s.OfferTonic()
	s.DrinkTonic("jewellers-tonic")

	for base, want := range map[int]int{1: 1, 2: 1, 3: 2, 5: 4, 10: 9, 11: 9, 20: 18, 0: 0} {
		if got := s.Price(base); got != want {
			t.Errorf("a price of %d came to %d, wanted %d", base, got, want)
		}
	}
}

// The cut comes off every card's price after the relics, and never takes one below one.
func TestTheCostCutStopsAtOne(t *testing.T) {
	for _, c := range []struct{ cost, cut, want int }{
		{3, 1, 2}, {3, 2, 1}, {2, 2, 1}, {1, 1, 1}, {0, 2, 0},
	} {
		if got := cutCost(c.cost, c.cut); got != c.want {
			t.Errorf("a %d AP card under a cut of %d costs %d, wanted %d", c.cost, c.cut, got, c.want)
		}
	}
}

// cutCost takes a 1 AP player card, moves it to the given cost the way an essence would, and asks
// what it costs under a cut.
func cutCost(cost, cut int) int {
	for _, c := range StartingDeck() {
		if c.Cost() == 1 {
			c.CostDelta = cost - 1
			return combat.CostWith(nil, cut, c)
		}
	}
	panic("the starting deck has no 1 AP card")
}

// Athlete's and Olympian's stack, and reach the fighter through Equip.
func TestTheCostCutsStackOnTheFighter(t *testing.T) {
	s := tonicRun(t, "athletes-tonic", "olympians-tonic")
	s.OfferTonic()
	s.DrinkTonic("athletes-tonic")
	inRealm(s, 2)
	s.OfferTonic()
	s.DrinkTonic("olympians-tonic")

	if got := s.Equip(combat.Duelist{}).CostCut; got != 2 {
		t.Errorf("the fighter carries a cut of %d, wanted 2", got)
	}
}

// Pressure takes a round off every fight and triples everything the run earns.
func TestPressureShortensTheFightAndTriplesTheEarnings(t *testing.T) {
	s := tonicRun(t, "pressure-tonic")
	s.OfferTonic()
	s.DrinkTonic("pressure-tonic")

	if got := s.RoundLimit(); got != combat.DefaultRoundLimit-1 {
		t.Errorf("the clock reads %d rounds, wanted %d", got, combat.DefaultRoundLimit-1)
	}
	d := s.Equip(combat.Duelist{})
	if d.RoundLimit != combat.DefaultRoundLimit-1 || d.VitaeFactor != 3 {
		t.Errorf("the fighter carries %d rounds at x%d", d.RoundLimit, d.VitaeFactor)
	}
	if got := d.Earned(4); got != 12 {
		t.Errorf("4 vitae earned in a fight came to %d, wanted 12", got)
	}

	plain := New(testDeck())
	plain.fight, s.fight = 1, 1
	if got, want := s.spoilsFor(60), plain.spoilsFor(60); got.FromLife != 3*want.FromLife ||
		got.FromRoom != 3*want.FromRoom {
		t.Errorf("a win paid %+v under pressure against %+v without", got, want)
	}
}

// The three container tonics each move their figure by one.
func TestTheContainerTonicsWidenTheRun(t *testing.T) {
	s := tonicRun(t, "jugglers-tonic", "sifters-tonic", "collectors-tonic")
	for realm, key := range []string{"jugglers-tonic", "sifters-tonic", "collectors-tonic"} {
		inRealm(s, realm+1)
		s.OfferTonic()
		if !s.DrinkTonic(key) {
			t.Fatalf("could not drink %s", key)
		}
	}
	if s.HandSize(8) != 9 || s.Discards(4) != 5 || s.RelicSlots() != combat.DefaultRelicSlots+1 {
		t.Errorf("hand %d, discards %d, relics %d", s.HandSize(8), s.Discards(4), s.RelicSlots())
	}
}

// What was drunk and what each realm offered come back from a save, and a sixth relic worn on a
// Collector's finger goes back on.
func TestTonicsSurviveAResume(t *testing.T) {
	motifs, shape := rosters(t)
	seed, _ := seeds.Parse(theSeed)
	s := Start(motifs, shape, seed)
	s.AddVitae(500)
	s.tonicOrder = []string{"collectors-tonic", "pressure-tonic"}
	s.WonFight(40, 40)
	s.SetPhase(PhaseShop)
	s.OfferTonic()
	s.DrinkTonic("collectors-tonic")
	for _, key := range Relics()[:combat.DefaultRelicSlots+1] {
		if !s.Wear(key) {
			t.Fatalf("could not wear %s on %d fingers", key, s.RelicSlots())
		}
	}

	back, _, err := Resume(motifs, shape, s.Snapshot(seed))
	if err != nil {
		t.Fatalf("a snapshot with tonics must resume: %v", err)
	}
	if !back.Drank("collectors-tonic") || len(back.Worn()) != len(s.Worn()) {
		t.Errorf("drank %v wearing %d, wanted the collector's and %d", back.DrunkTonics(),
			len(back.Worn()), len(s.Worn()))
	}
	if _, ok := back.OfferTonic(); ok {
		t.Error("the realm's drunk seat refilled across a resume")
	}
	if back.RelicSlots() != s.RelicSlots() {
		t.Errorf("fingers came back as %d, wanted %d", back.RelicSlots(), s.RelicSlots())
	}
}
