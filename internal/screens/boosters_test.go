package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The booster as a screen: the shop hands it over, the screen picks it up, and taking a card
// puts the player back where they bought it. Nothing here creates an ebiten.Image.

// **The shop pays and navigates; it no longer opens a dialog of its own** *(2026-09-19)*. The booster
// travels as a record key on the state, because a screen cannot be handed an argument.
func TestBuyingASealedBoosterGoesToItsScreen(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = state.ScreenWidth, state.ScreenHeight
	gs.Run.AddVitae(50)

	booster := boosterHolding(t, session.ContentsStones)

	var shop ShopScene
	shop.Init(gs)
	shop.openBooster(gs, booster.Record)

	if gs.ActiveScreen != state.Boosters {
		t.Fatalf("buying a booster left the player on %v, want Boosters", gs.ActiveScreen)
	}
	if gs.PendingBooster != booster.Record {
		t.Errorf("the booster travelled as %q, want %q", gs.PendingBooster, booster.Record)
	}

	var scene BoostersScene
	scene.Init(gs)
	if !scene.openNow() {
		t.Fatal("the screen opened nothing")
	}
	if gs.PendingBooster != "" {
		t.Error("the screen left the booster pending, so a second visit would open it again")
	}
	if scene.count() != len(scene.stones) || scene.count() == 0 {
		t.Errorf("the screen holds %d cards, want the bag's stones", scene.count())
	}
}

// **Taking a card is the only way out, and it goes back to the shop.** The booster is already paid
// for, so nothing on this screen may abandon it — and it is not a station of the run, so what it
// returns to is the shop rather than the next phase.
func TestTakingFromASealedBoosterReturnsToTheShop(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = state.ScreenWidth, state.ScreenHeight

	gs.PendingBooster = boosterHolding(t, session.ContentsStones).Record
	gs.ActiveScreen = state.Boosters

	var scene BoostersScene
	scene.Init(gs)
	if !scene.openNow() {
		t.Fatal("the screen opened nothing")
	}

	// Still open: the frame is the booster's while a card is standing there to be taken.
	if err := scene.Update(gs); err != nil {
		t.Fatalf("update: %v", err)
	}
	if gs.ActiveScreen != state.Boosters {
		t.Fatalf("the screen left before a card was taken, to %v", gs.ActiveScreen)
	}

	// A stone is used up in front of the player, and the screen leaves when that has played.
	scene.take(gs, 0)
	for i := 0; i < 1000 && gs.ActiveScreen == state.Boosters; i++ {
		if err := scene.Update(gs); err != nil {
			t.Fatalf("update: %v", err)
		}
	}
	if scene.openNow() {
		t.Fatal("a stone was taken and the booster is still open once its use has played")
	}
	if gs.ActiveScreen != state.Shop {
		t.Errorf("the booster finished and left the player on %v, want Shop", gs.ActiveScreen)
	}
}

// **A visit is one deal, however many times the screen is entered** *(2026-09-19)*. Opening a
// booster is a screen now, so coming back re-enters the shop — and a second deal would restock
// the shelf and forget which boosters had been opened.
func TestComingBackToTheShopDoesNotRedealIt(t *testing.T) {
	gs := testRun()
	gs.ScreenWidth, gs.ScreenHeight = state.ScreenWidth, state.ScreenHeight
	gs.Run.AddVitae(50)

	var shop ShopScene
	shop.Init(gs)

	shelf := shelfKeys(shop.shelf)
	booster := boosterHolding(t, session.ContentsStones)
	shop.openBooster(gs, booster.Record)

	// Back from the booster: Init runs again, on the same visit.
	shop.Init(gs)

	if got := shelfKeys(shop.shelf); !sameKeys(got, shelf) {
		t.Errorf("the shelf was re-dealt as %v, want the visit's own %v", got, shelf)
	}
	if !shop.boosterTaken(booster.Record) {
		t.Error("the shop forgot that the booster had already been opened, so it is for sale again")
	}
}
