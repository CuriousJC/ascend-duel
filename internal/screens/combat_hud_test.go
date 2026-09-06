package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
)

// The top row is three things sharing one line: the duelist card, the ring row, the enemy
// card. It is arithmetic and therefore checkable without a window — the same narrow
// exception the other tests in this package take. Nothing here creates an ebiten.Image.
//
// It exists because the ring row's right edge was a hardcoded 79% chosen to clear an enemy
// card centred at 88%, and that card moved to the corner the next day. A percentage standing
// in for something else's position goes stale silently: the failure is a ring drawn
// underneath a card, which looks like a drawing bug rather than a stale constant.

// A zero CombatScene is enough for all three: none of the placements reads the fighter, the
// deck or the round.

func TestTheTopRowIsThreeThingsThatDoNotOverlap(t *testing.T) {
	gs := testState()
	s := &CombatScene{}

	duelist, rings, enemy := s.duelistCardRect(gs), s.ringPaneRect(gs), s.enemyCardRect(gs)

	if rings.Min.X <= duelist.Max.X {
		t.Errorf("the ring row starts at x=%d, inside the duelist card ending at x=%d",
			rings.Min.X, duelist.Max.X)
	}
	if rings.Max.X >= enemy.Min.X {
		t.Errorf("the ring row ends at x=%d, inside the enemy card starting at x=%d",
			rings.Max.X, enemy.Min.X)
	}

	// **What is deliberately not checked is that the cards do not overlap** *(owner's call,
	// 2026-09-06)*. They do, and that is the design: the top row packs five ring seats and two
	// consumable seats at one shared pitch, and between the two fighter cards that pitch is two
	// pixels tighter than a card. ringSlotPitch closes a row up rather than shrinking a card,
	// because a smaller ring is a different drawing — so an assertion that five cards fit at full
	// width was a fact about a row with nothing beside it, and it went stale the moment the row was
	// divided.
	//
	// The floor kept here is a sanity bound and not a layout rule: a pitch that has collapsed to
	// nothing means the span arithmetic is wrong rather than that the row is snug.
	if pitch := ringSlotPitch(rings, maxRings); pitch <= 0 {
		t.Errorf("%d rings sit at a pitch of %dpx in a %dpx row", maxRings, pitch, rings.Dx())
	}

	// The consumables pane is the other half of the row, and it must clear the enemy card and the
	// rings on either side of it. It is fixed width, so this is what catches the row being narrowed
	// under it rather than the pane being resized.
	if cons := s.consumablePaneRect(gs); cons.Min.X <= rings.Max.X || cons.Max.X > enemy.Min.X {
		t.Errorf("the consumables pane runs %d..%d, against a ring row ending at %d and an enemy card at %d",
			cons.Min.X, cons.Max.X, rings.Max.X, enemy.Min.X)
	}
}

func TestBothCornerCardsAreOnScreenWithEqualMargins(t *testing.T) {
	gs := testState()
	s := &CombatScene{}

	duelist, enemy := s.duelistCardRect(gs), s.enemyCardRect(gs)

	if duelist.Min.X < 0 || duelist.Min.Y < 0 {
		t.Errorf("the duelist card starts at (%d,%d), off the top-left of the screen",
			duelist.Min.X, duelist.Min.Y)
	}
	if enemy.Max.X > gs.ScreenWidth {
		t.Errorf("the enemy card reaches x=%d, past the right of the %d-pixel screen",
			enemy.Max.X, gs.ScreenWidth)
	}

	// **Equal margins are the whole of what "in the corners" means.** Two cards in opposite
	// corners at different insets read as one placed and one drifted.
	left := duelist.Min.X
	right := gs.ScreenWidth - enemy.Max.X
	if d := left - right; d > 1 || d < -1 {
		t.Errorf("the duelist card is %dpx from the left edge and the enemy card is %dpx from the right",
			left, right)
	}

	// They share a top, which is what makes the row a row rather than three things near the
	// top of the screen.
	if duelist.Min.Y != enemy.Min.Y {
		t.Errorf("the duelist card starts at y=%d and the enemy card at y=%d",
			duelist.Min.Y, enemy.Min.Y)
	}
}

func TestTheRingRowSitsBelowTheCardsBesideIt(t *testing.T) {
	gs := testState()
	s := &CombatScene{}

	duelist, rings := s.duelistCardRect(gs), s.ringPaneRect(gs)

	// **The drop is deliberate and the amount is the point.** The row was flush with the two
	// cards' tops until the backing panel arrived: three things on one line, with a surface
	// behind the middle one, read as a single wide object with two cards embedded in it. The
	// offset breaks that line.
	if got := rings.Min.Y - duelist.Min.Y; got != ringPaneTopDrop {
		t.Errorf("the ring row sits %dpx below the duelist card, want %d", got, ringPaneTopDrop)
	}

	// **The row is exactly a card deep** *(2026-09-04)*. It used to be a card plus ringRuleGap,
	// reserving room under itself for the rule and the worn count; both moved into the caption
	// column beside the duelist card, which is the height that let the card grow to five quarters.
	if want := cards.RingStyle.Height; rings.Dy() != want {
		t.Errorf("the ring row is %dpx tall, want %d — exactly a ring card", rings.Dy(), want)
	}
}

// The two tower lines sit under the duelist card, in its column, and the band they have to fit in
// is bounded by the played row below them. Both edges move on their own — the card's off
// topRowTopPct, the table's off handTop — so the fit is exactly the kind of thing that goes stale
// silently.
func TestTheTowerLinesFitBetweenTheCardAndTheTable(t *testing.T) {
	gs := testState()
	s := &CombatScene{}

	card, place := s.duelistCardRect(gs), s.towerPlaceRect(gs)

	// **The caption is back under the card** *(2026-09-04, owner's call)*. It stood in a column
	// beside it for a day, which bought the top band height and cost the ring row — and therefore
	// the hand, which is laid out to it — 166 pixels of width. See towerPlaceRect.
	if place.Min.X != card.Min.X || place.Max.X != card.Max.X {
		t.Errorf("the tower lines run x=%d..%d, want the duelist card's column %d..%d",
			place.Min.X, place.Max.X, card.Min.X, card.Max.X)
	}
	if place.Min.Y != card.Max.Y+towerLineGap {
		t.Errorf("the tower lines start at y=%d, want %dpx under the card at y=%d",
			place.Min.Y, towerLineGap, card.Max.Y)
	}

	// And clear of the ring row, which now starts at the card's own right edge.
	if pane := s.ringPaneRect(gs); place.Max.X > pane.Min.X {
		t.Errorf("the tower lines reach x=%d, into the ring row at x=%d", place.Max.X, pane.Min.X)
	}

	// The whole top band has to finish above the table row — the caption included, since it is
	// under the card now rather than beside it.
	top := tableRowTop(gs)
	if s.ringCountRect(gs).Max.Y > top {
		t.Errorf("the ring count reaches y=%d, into the table row at y=%d",
			s.ringCountRect(gs).Max.Y, top)
	}
	if place.Max.Y > top {
		t.Errorf("the tower lines reach y=%d, into the table row at y=%d", place.Max.Y, top)
	}
}

func TestEveryRoomOnAFloorIsNamed(t *testing.T) {
	// Three fights to a floor and three names for them: towerRoom indexes one array by the
	// other's modulus, so a fourth fight per floor would panic rather than draw a blank line.
	if len(towerRoomNames) != fightsPerFloor {
		t.Fatalf("%d room names for %d fights a floor", len(towerRoomNames), fightsPerFloor)
	}

	// The floor turns over on the fight after the last room, and the first fight is the first
	// room of floor one — an off-by-one here would name the boss room "Outer".
	for fight, want := range map[int]string{
		0: "Outer Room", 1: "Inner Room", 2: "Stairway",
		3: "Outer Room", 5: "Stairway", 6: "Outer Room",
	} {
		if got := towerRoom(fight); got != want {
			t.Errorf("fight %d is the %s, want the %s", fight, got, want)
		}
	}
	for fight, want := range map[int]int{0: 1, 2: 1, 3: 2, 5: 2, 6: 3, 23: 8} {
		if got := towerFloor(fight); got != want {
			t.Errorf("fight %d is on floor %d, want %d", fight, got, want)
		}
	}
}

func TestTheRingBackingHoldsTheWholeRowWithoutTouchingTheCards(t *testing.T) {
	gs := testState()
	s := &CombatScene{}

	duelist, enemy := s.duelistCardRect(gs), s.enemyCardRect(gs)
	rings, back := s.ringPaneRect(gs), s.ringPaneBackRect(gs)

	// It has to reach past the row on every side, or the two end cards sit on its edge and it
	// reads as a border drawn around them.
	if !rings.In(back) {
		t.Errorf("the backing %v does not cover the row %v", back, rings)
	}

	// And it must not reach the fighter cards, or the three-part row becomes one object again.
	if back.Min.X <= duelist.Max.X {
		t.Errorf("the backing starts at x=%d, touching the duelist card ending at x=%d",
			back.Min.X, duelist.Max.X)
	}
	if back.Max.X >= enemy.Min.X {
		t.Errorf("the backing ends at x=%d, touching the enemy card starting at x=%d",
			back.Max.X, enemy.Min.X)
	}

	// **The fraction is no longer under the row** *(2026-09-04)*, so the backing is simply the row
	// padded rather than extended to cover it. What still has to be true is that it holds the cards
	// on every side; see ringCountRect for where the count went and why.
	if back.Max.Y < rings.Max.Y {
		t.Errorf("the backing ends at y=%d, above the row it stands behind at y=%d",
			back.Max.Y, rings.Max.Y)
	}
}
