package screens

import (
	"image"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The shelf's arithmetic and the two rows' geometry, both of which need no window — the same narrow
// exception the other tests in this package take. Nothing here creates an ebiten.Image.

func TestTheShelfHoldsThreeRingsTheRunIsNotWearing(t *testing.T) {
	// A ring already on your hand offered back to you is a seat spent saying nothing, and `Buy`
	// would refuse the click anyway.
	gs := testRun()

	shelf := dealShelf(gs)
	if len(shelf) != shelfSize {
		t.Fatalf("the shelf holds %d, want %d", len(shelf), shelfSize)
	}

	worn := map[string]bool{}
	for _, key := range gs.Run.Worn() {
		worn[key] = true
	}

	seen := map[string]bool{}
	for _, item := range shelf {
		if worn[item.key] {
			t.Errorf("%s is for sale and already worn", item.key)
		}
		if seen[item.key] {
			t.Errorf("%s is on the shelf twice", item.key)
		}
		if _, ok := session.RingPrice(item.key); !ok {
			t.Errorf("%s is for sale and is in no record", item.key)
		}
		seen[item.key] = true
	}
}

func TestTheSameFightWalksIntoTheSameShop(t *testing.T) {
	// **Per fight, like the opponent and the worm offer**, so a defeat and a retry meet the same
	// shelf rather than rerolling it. The stream is what makes that true; this is what says so.
	first := shelfKeys(dealShelf(testRun()))
	again := shelfKeys(dealShelf(testRun()))

	if len(first) != len(again) {
		t.Fatalf("two deals of one fight offered %d and %d rings", len(first), len(again))
	}
	for i := range first {
		if first[i] != again[i] {
			t.Fatalf("one fight dealt %v and then %v", first, again)
		}
	}
}

func TestALaterFightIsADifferentShop(t *testing.T) {
	// Not a promise that no two fights ever coincide — three of seventeen will sometimes repeat —
	// but a shelf identical across a whole run would mean the fight index never reached the seed.
	gs := testRun()

	first := shelfKeys(dealShelf(gs))
	for i := 0; i < 6; i++ {
		gs.Run.WonFight(0)
		if !sameKeys(first, shelfKeys(dealShelf(gs))) {
			return
		}
	}
	t.Errorf("seven fights all offered %v", first)
}

func sameKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTheShelfIsCutToThreeAndNotPaddedToIt(t *testing.T) {
	// **The row is what is left, cut to three**, so a catalogue smaller than the shelf draws a short
	// row rather than one with holes in it. That case is unreachable today — seventeen rings against
	// a cap of five leaves at least twelve unworn — so what is checkable is the cut itself, and the
	// case that *is* reachable: a scene reached before main built a run.
	if got := len(dealShelf(testRun())); got != shelfSize {
		t.Errorf("a fresh run was offered %d rings, want %d", got, shelfSize)
	}
	if len(session.Rings())-combat.MaxWornRings < shelfSize {
		t.Errorf("the catalogue holds %d rings, which is no longer enough to fill a shelf against a "+
			"cap of %d — dealShelf's cut is now load-bearing and wants a test of its own",
			len(session.Rings()), combat.MaxWornRings)
	}

	if got := dealShelf(&state.GlobalState{}); got != nil {
		t.Errorf("a game with no run was offered %v", shelfKeys(got))
	}
}

func TestTheTwoRowsDoNotOverlapAndStayOnScreen(t *testing.T) {
	// Both rows are full-size cards at a fixed pitch, and neither can exceed five. This is what
	// says the fixed pitch actually fits the screen — the hand's compressing pitch exists because
	// eight cards do not.
	gs := &state.GlobalState{ScreenWidth: 1280, ScreenHeight: 960}

	for n := 1; n <= combat.MaxWornRings; n++ {
		left := rowSlot(gs, 0, n, 0)
		right := rowSlot(gs, n-1, n, 0)

		if left.Min.X < 0 || right.Max.X > gs.ScreenWidth {
			t.Errorf("a row of %d runs from %d to %d", n, left.Min.X, right.Max.X)
		}
		for i := 1; i < n; i++ {
			if rowSlot(gs, i, n, 0).Min.X <= rowSlot(gs, i-1, n, 0).Max.X {
				t.Errorf("a row of %d overlaps at seat %d", n, i)
			}
		}
	}

	// **The row is five seats now**, not three: the three rings plus the bag of rocks and the can
	// of worms. See ShopScene.rowWidth for why they share one row rather than taking a second.
	shelf := rowSlot(gs, 0, shelfSize+2, gs.PctY(shelfRowPct))
	if shelf.Max.Y+shopFigureGap+shopFigureSize >= gs.PctY(offerButtonsPct)-offerButtonHeight/2 {
		t.Error("the shelf's prices run into the Leave button")
	}
	if shelf.Min.Y-shopRowLabelGap <= shopHintTop {
		t.Errorf("the shelf's label at %d collides with the hint at %d",
			shelf.Min.Y-shopRowLabelGap, shopHintTop)
	}
}

// **The band's ring row is the shop's worn row now**, so what has to be checked is that the sell
// figure hung under a ring clears the narration that starts below the band — the collision the old
// two-row layout could not have, and the one the title and hint were silently losing to before
// this screen had a band at all.
func TestTheSellFiguresClearTheNarration(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: 1280, ScreenHeight: 960}

	var shop ShopScene
	for n := 1; n <= combat.MaxWornRings; n++ {
		seat := shop.wornSlot(gs, n-1, n)
		if seat.Max.X > gs.ScreenWidth {
			t.Errorf("a worn row of %d runs to %d", n, seat.Max.X)
		}
		if bottom := seat.Max.Y + shopFigureGap + shopFigureSize; bottom >= shopProseTop {
			t.Errorf("a sell figure ends at %d and the narration starts at %d",
				bottom, shopProseTop)
		}
	}
}

// The two narrated lines have to fit between the band and the hint under them. **The shopkeeper's
// wording is authored**, so a third sentence or a longer one is a layout change and this is what
// says so.
func TestTheShopkeeperFitsAboveTheShelf(t *testing.T) {
	lines := shopkeeperLines()
	bottom := shopProseTop + (len(lines)-1)*proseLineGap + proseLineGap/2
	if bottom >= shopHintTop {
		t.Errorf("%d lines of narration reach %d and the hint sits at %d",
			len(lines), bottom, shopHintTop)
	}
}

// **A click on a worn ring arms the tab; it does not sell.** The bug this exists for is a click
// aimed at a tooltip taking a ring off the player's hand, and it would come back silently — a
// sale looks exactly like a sale the player meant.
func TestClickingAWornRingOnlyArmsIt(t *testing.T) {
	gs := shopState(t)
	before := gs.Run.Worn()

	var shop ShopScene
	shop.arm(before[0])

	if got := gs.Run.Worn(); len(got) != len(before) {
		t.Errorf("arming sold a ring: wearing %v, was %v", got, before)
	}
	if shop.armed != before[0] {
		t.Errorf("armed %q, want %q", shop.armed, before[0])
	}

	// The same ring again puts the question away, rather than a second click confirming it.
	shop.arm(before[0])
	if shop.armed != "" {
		t.Errorf("a second click left %q armed", shop.armed)
	}
	if got := gs.Run.Worn(); len(got) != len(before) {
		t.Errorf("a second click sold a ring: wearing %v", got)
	}
}

// The tab hangs in the seat the sell figure was written in, so it has to clear the narration under
// the band exactly as that figure does.
func TestTheSellTabClearsTheNarration(t *testing.T) {
	gs := shopState(t)

	var shop ShopScene
	shop.armed = gs.Run.Worn()[0]

	tab := shop.sellTabRect(gs)
	if tab.Max.Y >= shopProseTop {
		t.Errorf("the tab ends at %d and the narration starts at %d", tab.Max.Y, shopProseTop)
	}

	seat, _ := shop.wornSeatOf(gs, shop.armed)
	if tab.Min.Y < seat.Max.Y {
		t.Errorf("the tab starts at %d, above the bottom of its ring at %d", tab.Min.Y, seat.Max.Y)
	}
	if wide := seat.Dx(); tab.Dx() > wide {
		t.Errorf("the tab is %d wide against a %d-wide ring", tab.Dx(), wide)
	}
}

// shopState is a run wearing a ring, on a 1280x960 screen.
func shopState(t *testing.T) *state.GlobalState {
	t.Helper()

	gs := &state.GlobalState{
		ScreenWidth: 1280, ScreenHeight: 960,
		Run: session.New(session.StartingDeck()),
	}
	for _, key := range session.Rings() {
		if gs.Run.Wear(key) {
			break
		}
	}
	if len(gs.Run.Worn()) == 0 {
		t.Fatal("the fixture could not put a single ring on")
	}
	return gs
}

// TestThePouchButtonClearsTheOtherTwoCornerToggles is the only thing about the pouch panel a test
// without a window can check: three square-ish buttons sharing one bottom line, and the third one
// added years after the layout was written.
//
// **A button drawn over another button is a click that goes to whichever was updated last**, which
// is invisible until somebody presses the wrong one.
func TestThePouchButtonClearsTheOtherTwoCornerToggles(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: 1280, ScreenHeight: 960}

	pouch := pouchCornerPlace(gs)
	hands := handsCornerPlace(gs)
	deck := cornerSlot(0)(gs)

	// Each is stored as its centre, so the edges are half a width either side.
	pouchLeft, pouchRight := pouch.X-logButtonSize/2, pouch.X+logButtonSize/2
	handsLeft, handsRight := hands.X-handsButtonWidth/2, hands.X+handsButtonWidth/2
	deckLeft := deck.X - logButtonSize/2

	if pouchRight >= handsLeft {
		t.Errorf("the pouch button ends at %d and the hands button starts at %d", pouchRight, handsLeft)
	}
	if handsRight >= deckLeft {
		t.Errorf("the hands button ends at %d and the deck button starts at %d", handsRight, deckLeft)
	}
	if pouchLeft < 0 {
		t.Errorf("the pouch button starts at %d, off the left of the screen", pouchLeft)
	}
	if pouch.Y != hands.Y || pouch.Y != deck.Y {
		t.Errorf("the three corner buttons sit at y %d, %d and %d", pouch.Y, hands.Y, deck.Y)
	}
}

// TestTheTabsUnderAnArmedStoneStayInsideThePanel holds the two confirm tabs against the frame they
// hang in. A tab drawn outside the modal panel is a control on a scrim.
func TestTheTabsUnderAnArmedStoneStayInsideThePanel(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: 1280, ScreenHeight: 960}

	run := session.New(combat.PlainCards(combat.Strike))
	for _, key := range []string{"agate", "jasper", "onyx"} {
		if !run.Carry(key) {
			t.Fatalf("the catalogue has no stone %q", key)
		}
	}
	gs.Run = run

	s := &ShopScene{}
	s.pouch.init()

	panel := modalPanelRect(gs)
	for i := range run.Carried() {
		s.pouch.armed = i
		use, sell := s.pouch.tabRects(gs)
		for _, tab := range []image.Rectangle{use, sell} {
			if !tab.In(panel) {
				t.Errorf("seat %d hangs a tab at %v, outside the panel %v", i, tab, panel)
			}
		}
		if use.Max.X >= sell.Min.X {
			t.Errorf("seat %d overlaps its own two tabs: %v and %v", i, use, sell)
		}
	}
}
