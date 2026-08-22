package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The shop's rules: the purse, the fifth finger, and what a sale costs.

// rich is a bare run with enough vitae to buy anything in the file, so a test about buying is not
// also a test about how much a fight pays.
func rich(t *testing.T) *Session {
	t.Helper()

	run := bare(t)
	run.vitae = 1000
	return run
}

func priceOf(t *testing.T, key string) int {
	t.Helper()

	price, ok := RingPrice(key)
	if !ok {
		t.Fatalf("%s has no price", key)
	}
	return price
}

func TestEveryRingIsPricedAndSellsForSomething(t *testing.T) {
	// **A price is registered rather than trusted** — a record with none panics at load, so
	// reaching this test is most of the check. What it adds is the floor on the sale: rounding a
	// A written per-tier figure is what stops the cheapest ring being worth nothing to take off.
	for _, key := range Rings() {
		price, ok := RingPrice(key)
		if !ok || price <= 0 {
			t.Errorf("%s is priced at %d", key, price)
			continue
		}
		if got := SellValue(key); got < 1 || got > price {
			t.Errorf("%s costs %d and sells for %d", key, price, got)
		}
	}
}

func TestEachTierPaysBackItsOwnFigure(t *testing.T) {
	// The three prices and the three rebates, written out, so a change to either is a change to
	// this table rather than something a run finds out at the shop.
	for _, tc := range []struct {
		rarity     data.Rarity
		price, buy int
	}{
		{data.Common, 3, 1}, {data.Uncommon, 5, 2}, {data.Rare, 7, 3},
	} {
		if got := tc.rarity.Price(); got != tc.price {
			t.Errorf("%s costs %d, want %d", tc.rarity, got, tc.price)
		}
		if got := tc.rarity.Sell(); got != tc.buy {
			t.Errorf("%s sells back for %d, want %d", tc.rarity, got, tc.buy)
		}
		// The round trip has to lose, or a shelf is free to try on.
		if tc.rarity.Sell() >= tc.rarity.Price() {
			t.Errorf("%s sells for %d and costs %d, which is not a loss",
				tc.rarity, tc.rarity.Sell(), tc.price)
		}
	}
}

func TestBuyingPaysAndPutsTheRingOn(t *testing.T) {
	run := rich(t)
	price := priceOf(t, "keen-ring")
	before := run.Vitae()

	if !run.Buy("keen-ring") {
		t.Fatal("the purchase was refused")
	}
	if got := run.Vitae(); got != before-price {
		t.Errorf("the purse came out %d, want %d", got, before-price)
	}
	if got := run.Worn(); len(got) != 1 || got[0] != "keen-ring" {
		t.Errorf("the run is wearing %v", got)
	}
}

func TestAnEmptyPurseBuysNothingAndChangesNothing(t *testing.T) {
	// **A run cannot go into debt**, and a refused purchase has to leave it exactly as it was —
	// the failure this guards is a ring going on before the purse is asked.
	run := bare(t)
	run.vitae = priceOf(t, "keen-ring") - 1

	if run.CanBuy("keen-ring") {
		t.Error("CanBuy said yes on a short purse")
	}
	if run.Buy("keen-ring") {
		t.Fatal("a ring was bought that could not be afforded")
	}
	if got := run.Vitae(); got != priceOf(t, "keen-ring")-1 {
		t.Errorf("the purse moved to %d on a refused purchase", got)
	}
	if len(run.Worn()) != 0 {
		t.Errorf("the run is wearing %v after a refused purchase", run.Worn())
	}
}

func TestTheSixthRingIsRefusedRatherThanSwapped(t *testing.T) {
	// **The cap surfaces when you try to buy a sixth** — MECHANICS.md — and selling is what frees a
	// finger. A purchase that quietly threw a ring away would be a ring lost to a misread click.
	run := rich(t)

	all := Rings()
	if len(all) < combat.MaxWornRings+1 {
		t.Skipf("only %d rings authored; this needs %d", len(all), combat.MaxWornRings+1)
	}
	for _, key := range all[:combat.MaxWornRings] {
		if !run.Buy(key) {
			t.Fatalf("%s would not go on", key)
		}
	}

	sixth := all[combat.MaxWornRings]
	held := run.Vitae()

	if run.CanBuy(sixth) || run.Buy(sixth) {
		t.Fatalf("a %dth ring went on", combat.MaxWornRings+1)
	}
	if run.Vitae() != held {
		t.Error("the purse moved on a refused sixth ring")
	}

	// Selling one is what makes room, and the sixth then goes on.
	if !run.Sell(all[0]) {
		t.Fatal("the first ring would not come off")
	}
	if !run.Buy(sixth) {
		t.Error("the sixth ring was still refused with a finger free")
	}
}

func TestABoughtRingIsNotOfferedAgain(t *testing.T) {
	run := rich(t)
	if !run.Buy("banker-ring") {
		t.Fatal("the purchase was refused")
	}
	held := run.Vitae()

	if run.CanBuy("banker-ring") || run.Buy("banker-ring") {
		t.Error("the same ring went on twice")
	}
	if run.Vitae() != held {
		t.Error("the purse paid for a ring already worn")
	}
}

func TestSellingTakesTheRingOffAndPaysBack(t *testing.T) {
	run := wearing(t, "fire-ring", "keen-ring", "banker-ring")
	run.vitae = 0

	if !run.Sell("keen-ring") {
		t.Fatal("the sale was refused")
	}
	if got := run.Vitae(); got != SellValue("keen-ring") {
		t.Errorf("the sale paid %d, want %d", got, SellValue("keen-ring"))
	}

	// **Worn order is the firing order**, so what is left has to stay in the order it went on.
	got := run.Worn()
	if len(got) != 2 || got[0] != "fire-ring" || got[1] != "banker-ring" {
		t.Errorf("the row came out %v, want the other two in worn order", got)
	}
}

func TestSellingSomethingYouAreNotWearingDoesNothing(t *testing.T) {
	run := wearing(t, "fire-ring")
	held := run.Vitae()

	if run.Sell("keen-ring") {
		t.Fatal("a ring that was not worn was sold")
	}
	if run.Vitae() != held {
		t.Error("the purse was paid for a ring nobody owned")
	}
}

func TestASoldRingLosesItsGrowth(t *testing.T) {
	// **The accumulator resets on removal** *(owner's call, 2026-08-21)*. `grown` is keyed by record
	// so a ring taken off and put back on is the same ring; the decision is that it is not the same
	// number. It is what stops a Heart Ring being parked in the shop between fights.
	run := rich(t)
	if !run.Buy("heart-ring") {
		t.Fatal("the purchase was refused")
	}
	run.WonFight()
	run.WonFight()

	if got := run.Grown("heart-ring"); got != 10 {
		t.Fatalf("two wins grew it by %d, want 10", got)
	}

	if !run.Sell("heart-ring") {
		t.Fatal("the sale was refused")
	}
	if got := run.Grown("heart-ring"); got != 0 {
		t.Errorf("a sold ring kept %d of its growth", got)
	}

	if !run.Buy("heart-ring") {
		t.Fatal("it would not go back on")
	}
	if got := run.Grown("heart-ring"); got != 0 {
		t.Errorf("re-buying it started at %d, want 0", got)
	}
}

func TestTheRoundTripCosts(t *testing.T) {
	// **A swap is meant to cost**, or the shelf is a free rerolling of the hand every visit. Buying
	// and immediately selling has to leave the run poorer for every ring in the file.
	run := rich(t)

	for _, key := range Rings() {
		before := run.Vitae()
		if !run.Buy(key) || !run.Sell(key) {
			t.Fatalf("%s would not go on and off", key)
		}
		if got := run.Vitae(); got >= before {
			t.Errorf("%s round-tripped from %d to %d", key, before, got)
		}
	}
}

func TestTheRarityInTheFileIsThePriceCharged(t *testing.T) {
	// The shop reads the run, the run reads the tier, and the tier is the only place a price is
	// written. A screen quoting one number while the purse pays another is the drift this rules out.
	records := data.LoadRings()

	for key, record := range records {
		if !record.Rarity.Valid() {
			t.Errorf("%s has rarity %q, which is not a tier", key, record.Rarity)
			continue
		}
		if got := priceOf(t, key); got != record.Rarity.Price() {
			t.Errorf("%s is %s and should cost %d, and the registry charges %d",
				key, record.Rarity, record.Rarity.Price(), got)
		}
		if got, want := RingWeight(key), record.Rarity.Weight(); got != want {
			t.Errorf("%s is %s and should be drawn at %d, and the shop draws it at %d",
				key, record.Rarity, want, got)
		}
	}
}
