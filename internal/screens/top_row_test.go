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
