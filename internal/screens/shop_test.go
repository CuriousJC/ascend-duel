package screens

import (
	"image"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The shelf's arithmetic and the two rows' geometry, both of which need no window — the same narrow
// exception the other tests in this package take. Nothing here creates an ebiten.Image.

func TestTheShelfHoldsThreeRingsTheRunIsNotWearing(t *testing.T) {
	// A ring already on your hand offered back to you is a seat spent saying nothing, and `Buy`
	// would refuse the click anyway.
	gs := testRun()

	shelf := dealShelf(gs, shopRNG(gs, seeds.ShopStock))
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
	first := shelfKeys(dealShelfFor(testRun()))
	again := shelfKeys(dealShelfFor(testRun()))

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

	first := shelfKeys(dealShelf(gs, shopRNG(gs, seeds.ShopStock)))
	for i := 0; i < 6; i++ {
		gs.Run.WonFight(0, 0)
		if !sameKeys(first, shelfKeys(dealShelf(gs, shopRNG(gs, seeds.ShopStock)))) {
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
	if got := len(dealShelfFor(testRun())); got != shelfSize {
		t.Errorf("a fresh run was offered %d rings, want %d", got, shelfSize)
	}
	if len(session.Rings())-combat.MaxWornRings < shelfSize {
		t.Errorf("the catalogue holds %d rings, which is no longer enough to fill a shelf against a "+
			"cap of %d — dealShelf's cut is now load-bearing and wants a test of its own",
			len(session.Rings()), combat.MaxWornRings)
	}

	if got := dealShelf(&state.GlobalState{}, nil); got != nil {
		t.Errorf("a game with no run was offered %v", shelfKeys(got))
	}
}

// The shelf is four panes on one line, and the row is solved rather than laid out by eye — so what
// has to hold is that every pane fits between its neighbours, that the whole row stays on screen,
// and that what hangs under it clears the button at the bottom. **A row that overlaps inside a pane
// is expected**, which is why the check is on the panes rather than on the cards: nine full-size
// cards cannot sit apart on a 1920-wide screen and shopPitch is what admits it.
func TestTheFourPanesFitOnOneRow(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}

	panes := shopPaneOrder()
	first := shopPaneBackRect(gs, panes[0])
	last := shopPaneBackRect(gs, panes[len(panes)-1])

	if first.Min.X < 0 || last.Max.X > gs.ScreenWidth {
		t.Errorf("the row runs from %d to %d on a screen %d wide",
			first.Min.X, last.Max.X, gs.ScreenWidth)
	}

	for i := 1; i < len(panes); i++ {
		prev := shopPaneBackRect(gs, panes[i-1])
		this := shopPaneBackRect(gs, panes[i])
		if this.Min.X < prev.Max.X {
			t.Errorf("pane %d starts at %d and pane %d ends at %d",
				i, this.Min.X, i-1, prev.Max.X)
		}
		if this.Min.Y != prev.Min.Y {
			t.Errorf("pane %d sits at %d and pane %d at %d, so the shelf is not one row",
				i, this.Min.Y, i-1, prev.Min.Y)
		}
	}

	// Every seat is inside the pane it belongs to, however tight the pitch gets.
	for _, p := range panes {
		back := shopPaneBackRect(gs, p)
		for i := 0; i < shopPaneSeats(p); i++ {
			seat := shopSeatRect(gs, p, i)
			if seat.Min.X < back.Min.X || seat.Max.X > back.Max.X {
				t.Errorf("seat %d of pane %d runs from %d to %d outside %v",
					i, p, seat.Min.X, seat.Max.X, back)
			}
		}
	}

	if top := shopPaneRect(gs, shopPaneRings).Min.Y; top-shopFigureSize <= shopHintTop {
		t.Errorf("the shelf starts at %d and the hint sits at %d", top, shopHintTop)
	}

	// The prices hang under the panes and the two reroll buttons hang under those, so it is the
	// buttons rather than the figures that have to clear the Leave button.
	bottom := shopRerollRect(gs, shopPaneRings).Max.Y
	if bottom >= gs.PctY(offerButtonsPct)-offerButtonHeight/2 {
		t.Errorf("a reroll button ends at %d and Leave starts at %d",
			bottom, gs.PctY(offerButtonsPct)-offerButtonHeight/2)
	}
}

// **The band's ring row is the shop's worn row now**, so what has to be checked is that the sell
// figure hung under a ring clears the narration that starts below the band — the collision the old
// two-row layout could not have, and the one the title and hint were silently losing to before
// this screen had a band at all.
func TestTheSellFiguresClearTheNarration(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}

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

// shopState is a run wearing a ring, at the game’s internal resolution.
func shopState(t *testing.T) *state.GlobalState {
	t.Helper()

	gs := &state.GlobalState{
		ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight,
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

// **Every corner control comes off one strip now** *(2026-09-06)*, so what has to hold is that the
// strip and the column above it do not stand on each other and none of it leaves the screen. The
// bug this exists for shipped: the shop's HANDS button was placed by a rule of its own and landed
// under the frame's settings cog, where a click goes to whichever was updated last.
func TestTheCornerControlsDoNotOverlap(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}

	settings := ChromeCornerSlot(gs, ChromeSlotSettings)
	stones := ChromeCornerSlot(gs, ChromeSlotStones)

	if !stones.Intersect(settings).Empty() {
		t.Errorf("the bottom line overlaps: settings %v, stones %v", settings, stones)
	}
	if stones.Min.X < 0 || settings.Max.X > gs.ScreenWidth || settings.Max.Y > gs.ScreenHeight {
		t.Errorf("the bottom line runs off the screen: %v to %v", stones, settings)
	}
	if stones.Min.Y != settings.Min.Y {
		t.Error("the two squares are not on one line")
	}

	// The column stacks upward from the action-point bar and the strip sits under it; the two are
	// the same corner, so they may not meet.
	for i := 0; i < ControlColumnSlots; i++ {
		rung := ControlColumnSlot(gs, i)
		if !rung.Intersect(settings).Empty() {
			t.Errorf("column slot %d at %v stands on the settings button at %v",
				i, rung, settings)
		}
	}
}

// TestTheTabsUnderAnArmedStoneStayInsideThePanel holds the two confirm tabs against the frame they
// hang in. A tab drawn outside the modal panel is a control on a scrim.
func TestTheTabsUnderAnArmedStoneStayInsideThePanel(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}

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

// dealShelfFor deals a shelf for a run the caller has just built, seeding the visit's stream the
// way Init does. **The stream is the scene's now** — a reroll advances its cursor — so a test that
// wants one shelf has to open one, exactly as the screen does.
func dealShelfFor(gs *state.GlobalState) []shelfItem {
	return dealShelf(gs, shopRNG(gs, seeds.ShopStock))
}

// The pile stands to the left of the control column and clear of the shelf above it. **The bug it
// guards against is the one the lettered square had**: a control placed by a rule of its own,
// beside controls placed by another.
func TestTheShopPileStandsClearOfTheColumnAndTheShelf(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}

	pile := shopPileBounds(gs)
	if pile.Max.X >= ControlColumnLeft(gs) {
		t.Errorf("the pile ends at %d and the column starts at %d",
			pile.Max.X, ControlColumnLeft(gs))
	}
	if pile.Min.X < 0 || shopPileCountRect(gs).Max.Y > gs.ScreenHeight {
		t.Errorf("the pile and its count run off the screen: %v", pile)
	}

	for _, p := range shopPaneOrder() {
		back := shopPaneBackRect(gs, p)
		if !back.Intersect(pile).Empty() {
			t.Errorf("the pile at %v stands on pane %d at %v", pile, p, back)
		}
	}
	if !pile.Intersect(shopRerollRect(gs, shopPaneRings)).Empty() {
		t.Error("the pile stands on the rings' reroll button")
	}
}
