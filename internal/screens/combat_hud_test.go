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

	// And the row is wide enough to be a row. Five ring cards at full size is what the band
	// was widened for, and a pane narrower than that would start overlapping them — which
	// ringSlotPitch does deliberately, so nothing else would report it.
	if want := maxRings * cards.RingStyle.Width; rings.Dx() < want {
		t.Errorf("the ring row is %dpx wide, less than the %d cards it holds at %dpx each",
			rings.Dx(), maxRings, cards.RingStyle.Width)
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

	// The rule under the rings hangs below the cards themselves, so the row's rectangle is
	// taller than a ring card by exactly that gap.
	if want := cards.RingStyle.Height + ringRuleGap; rings.Dy() != want {
		t.Errorf("the ring row is %dpx tall, want %d — a %dpx card plus the %dpx rule gap",
			rings.Dy(), want, cards.RingStyle.Height, ringRuleGap)
	}
}

// The two tower lines sit in the band between the duelist card and the table's left-hand row,
// and that band is only about seventy pixels deep. Both of its edges move on their own — the
// card's off topRowTopPct, the table's off handTopPct and the feed's height — so the fit is
// exactly the kind of thing that goes stale silently.
func TestTheTowerLinesFitBetweenTheCardAndTheTable(t *testing.T) {
	gs := testState()
	s := &CombatScene{}

	card, place := s.duelistCardRect(gs), s.towerPlaceRect(gs)

	if place.Min.Y <= card.Max.Y {
		t.Errorf("the tower lines start at y=%d, inside the duelist card ending at y=%d",
			place.Min.Y, card.Max.Y)
	}
	if place.Min.X != card.Min.X {
		t.Errorf("the tower lines start at x=%d and the card at x=%d — they share a left edge",
			place.Min.X, card.Min.X)
	}

	// The player's row of played cards starts at tableInset, well left of the card's right
	// edge, so the lines have to finish above it.
	if top := tableRowTop(gs); place.Max.Y > top {
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

	// The `3/5` fraction belongs to the row it counts, so the surface has to be deep enough to
	// hold it — a number hanging off the bottom edge would read as loose underneath the panel.
	countBottom := rings.Max.Y + ringRuleWidth + ringCountTopGap + ringCountSize
	if back.Max.Y < countBottom {
		t.Errorf("the backing ends at y=%d, above the cap fraction ending at y=%d",
			back.Max.Y, countBottom)
	}
}
