package screens

import (
	"sort"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

// **The lesson makes the player buy three things, so the three have to be affordable.**
//
// The shop steps gate on `relics-worn` and `dmg-bought` and lock the screen to the shelf and to the
// Draught's seat. A player who cannot afford what the step demands has no legal click that
// satisfies it and no way past — the tutorial simply stops, on the last screen before the climb
// starts, which is the worst place in the game for it to happen.
//
// Nothing enforces it: a relic's rarity in `relics.json` decides its price, the shelf is a weighted
// draw off the run seed, and the potion's price is its own record. Any of the three can move for
// its own reasons and leave the lesson demanding 14 vitae from a 13-vitae purse.
//
// **The purse figure is measured, not derived**, and that is this test's one weakness said out
// loud: the taught run reaches the shop with 13, which is the fight's vitae plus the prize card,
// and modelling that faithfully here would mean replaying the duel. If the payout changes this
// test will not notice — but a price or a rarity moving is the far likelier break, and that it
// does catch. **If it goes red, the fix is a cheaper shelf or a different seed, not a bigger
// number here.**
func TestTheTutorialsShopCanAffordWhatTheLessonDemands(t *testing.T) {
	// What the taught run walks into the shop holding, observed in play on 2026-09-06.
	const purse = 13

	// The lesson buys two relics and one Draught.
	const relicsToBuy = 2

	script := tutorial.Load()
	runSeed, err := seeds.Parse(script.Seed)
	if err != nil {
		t.Fatalf("tutorial.json seed %q: %v", script.Seed, err)
	}

	gs := &state.GlobalState{
		ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight, RunSeed: runSeed,
	}
	gs.Run = session.New(nil)
	gs.Run.Teach(script)

	// **The shop the lesson actually reaches**, which is the one after the taught fight — the
	// shelf is drawn per fight, so reading it at fight zero is reading a different shop.
	gs.Run.WonFight(48, 60)

	shelf := dealShelf(gs, shopRNG(gs, seeds.ShopStock))
	if len(shelf) < relicsToBuy {
		t.Fatalf("the shelf stands %d relics and the lesson tells the player to buy %d",
			len(shelf), relicsToBuy)
	}

	prices := make([]int, 0, len(shelf))
	for _, item := range shelf {
		p, ok := session.RelicPrice(item.key)
		if !ok {
			t.Fatalf("shelf relic %q has no price", item.key)
		}
		prices = append(prices, p)
	}
	sort.Ints(prices)

	relics := 0
	for _, p := range prices[:relicsToBuy] {
		relics += p
	}

	draught := 0
	for _, p := range shopPotions() {
		if p.Effect == session.PotionDMG {
			draught = p.Price
		}
	}
	if draught == 0 {
		t.Fatal("no potion in the catalogue adds DMG, and the lesson tells the player to drink one")
	}

	t.Logf("shelf %v, cheapest %d relics cost %d, draught %d, purse %d",
		prices, relicsToBuy, relics, draught, purse)

	if total := relics + draught; total > purse {
		t.Errorf("the lesson demands %d relics and a Draught for %d vitae against a purse of %d: "+
			"the shop steps cannot be satisfied and the tutorial stops there", relicsToBuy, total, purse)
	}
}
