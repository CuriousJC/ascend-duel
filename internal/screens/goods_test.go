package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The sealed good as a screen: the shop hands it over, the screen picks it up, and taking a card
// puts the player back where they bought it. Nothing here creates an ebiten.Image.

// **The shop pays and navigates; it no longer opens a dialog of its own** *(2026-09-19)*. The good
// travels as a record key on the state, because a screen cannot be handed an argument.
func TestBuyingASealedGoodGoesToItsScreen(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = state.ScreenWidth, state.ScreenHeight
	gs.Run.AddVitae(50)

	good := goodHolding(t, session.ContentsStones)

	var shop ShopScene
	shop.Init(gs)
	shop.openGood(gs, good.Record)

	if gs.ActiveScreen != state.Goods {
		t.Fatalf("buying a good left the player on %v, want Goods", gs.ActiveScreen)
	}
	if gs.PendingGood != good.Record {
		t.Errorf("the good travelled as %q, want %q", gs.PendingGood, good.Record)
	}

	var scene GoodsScene
	scene.Init(gs)
	if !scene.openNow() {
		t.Fatal("the screen opened nothing")
	}
	if gs.PendingGood != "" {
		t.Error("the screen left the good pending, so a second visit would open it again")
	}
	if scene.count() != len(scene.stones) || scene.count() == 0 {
		t.Errorf("the screen holds %d cards, want the bag's stones", scene.count())
	}
}

// **Taking a card is the only way out, and it goes back to the shop.** The good is already paid
// for, so nothing on this screen may abandon it — and it is not a station of the run, so what it
// returns to is the shop rather than the next phase.
func TestTakingFromASealedGoodReturnsToTheShop(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = state.ScreenWidth, state.ScreenHeight

	gs.PendingGood = goodHolding(t, session.ContentsStones).Record
	gs.ActiveScreen = state.Goods

	var scene GoodsScene
	scene.Init(gs)
	if !scene.openNow() {
		t.Fatal("the screen opened nothing")
	}

	// Still open: the frame is the good's while a card is standing there to be taken.
	if err := scene.Update(gs); err != nil {
		t.Fatalf("update: %v", err)
	}
	if gs.ActiveScreen != state.Goods {
		t.Fatalf("the screen left before a card was taken, to %v", gs.ActiveScreen)
	}

	scene.take(gs, 0)
	if scene.openNow() {
		t.Fatal("a stone was taken and the good is still open")
	}
	if err := scene.Update(gs); err != nil {
		t.Fatalf("update: %v", err)
	}
	if gs.ActiveScreen != state.Shop {
		t.Errorf("the good finished and left the player on %v, want Shop", gs.ActiveScreen)
	}
}

// **A visit is one deal, however many times the screen is entered** *(2026-09-19)*. Opening a
// sealed good is a screen now, so coming back re-enters the shop — and a second deal would restock
// the shelf and forget which goods had been opened.
func TestComingBackToTheShopDoesNotRedealIt(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = state.ScreenWidth, state.ScreenHeight
	gs.Run.AddVitae(50)

	var shop ShopScene
	shop.Init(gs)

	shelf := shelfKeys(shop.shelf)
	good := goodHolding(t, session.ContentsStones)
	shop.openGood(gs, good.Record)

	// Back from the good: Init runs again, on the same visit.
	shop.Init(gs)

	if got := shelfKeys(shop.shelf); !sameKeys(got, shelf) {
		t.Errorf("the shelf was re-dealt as %v, want the visit's own %v", got, shelf)
	}
	if !shop.goodTaken(good.Record) {
		t.Error("the shop forgot that the good had already been opened, so it is for sale again")
	}
}
