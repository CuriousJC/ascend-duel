package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The table's geometry, which is arithmetic and needs no window — the same narrow exception
// the other tests in this package take. Nothing here creates an ebiten.Image.
//
// What these guard is the one property the arrangement exists for: **two hands that face each
// other and never touch.** Five cards a side do not fit in a screen at full size, so the rows
// overlap within themselves; the moment they overlap *into* each other the picture stops being
// a confrontation and becomes one row of ten.

// tableSeats is the widest row the rules can produce. Asked of the rules rather than written
// down, because a ring raising the cap is an expected change and must not quietly break the
// layout instead of failing here.
func tableSeats() int { return combat.Duelist{}.MaxActions() }

func TestTheTwoHandsNeverReachEachOther(t *testing.T) {
	gs := testState()

	for n := 1; n <= tableSeats(); n++ {
		player := playedSeatAt(gs, n-1, n)
		enemy := enemySeatAt(gs, 0, n)

		playerRight := player.X + cardWidth
		if playerRight >= enemy.X {
			t.Errorf("%d cards a side: the player's row ends at x=%d and the opponent's starts at x=%d",
				n, playerRight, enemy.X)
		}
	}
}

func TestEachRowIsPinnedToItsOwnEdge(t *testing.T) {
	gs := testState()

	for n := 1; n <= tableSeats(); n++ {
		// The player's grows rightward from the left inset.
		if got := playedSeatAt(gs, 0, n).X; got != tableInset {
			t.Errorf("%d cards: the player's row starts at x=%d, want the %dpx inset", n, got, tableInset)
		}

		// The opponent's is right-aligned, so its *last* card is flush with the right inset
		// whatever the count. That is what makes a hand of two hug its own edge rather than
		// drift toward the middle.
		last := enemySeatAt(gs, n-1, n).X + cardWidth
		if want := gs.ScreenWidth - tableInset; last != want {
			t.Errorf("%d cards: the opponent's row ends at x=%d, want %d", n, last, want)
		}
	}
}

func TestTheTableSitsBetweenTheRingRowAndTheFeed(t *testing.T) {
	gs := testState()
	s := &CombatScene{}

	top := tableRowTop(gs)

	// Below the top row, whose lowest ink is the ring count under the rule.
	ringBottom := s.ringPaneRect(gs).Max.Y + ringRuleWidth + ringCountTopGap + ringCountSize
	if top < ringBottom {
		t.Errorf("the table starts at y=%d, into the ring row's count ending at y=%d", top, ringBottom)
	}

	// And clear of the Resolution feed, which is the one thing a played card may never cover.
	feedTop := gs.PctY(handTopPct) - feedGapAboveCards - feedHeight()
	if bottom := top + cardHeight; bottom > feedTop {
		t.Errorf("the table ends at y=%d, into the Resolution feed at y=%d", bottom, feedTop)
	}
}

func TestTheOpponentsRowIsInResolutionOrder(t *testing.T) {
	// **The row must say what will happen, not what was planned.** ResolutionOrder regroups a
	// turn into prepare, then attacks, then defenses, so a queue planned attack-first comes out
	// of the planner in one order and resolves in another.
	s := &CombatScene{
		enemyActions: combat.PlainCards(combat.Strike, combat.Gather, combat.Brace),
	}

	got := s.enemyQueueOrder()
	want := combat.PlainCards(combat.Gather, combat.Strike, combat.Brace)

	if len(got) != len(want) {
		t.Fatalf("the row holds %d cards, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("seat %d holds %v, want %v", i, got[i], want[i])
		}
	}
}

func TestPlayedSeatsDoNotMoveAsTheRoundPlaysOut(t *testing.T) {
	// The whole row is dealt at round start, so a seat's position is a function of the round's
	// total and never of how many cards have landed. A pitch derived from the latter would
	// shuffle every seated card sideways each time another arrived.
	gs := testState()

	const total = 4
	first := playedSeatAt(gs, 0, total)
	for n := 1; n < total; n++ {
		if got := playedSeatAt(gs, 0, total); got != first {
			t.Fatalf("seat 0 moved to %v while %d cards were down, from %v", got, n, first)
		}
	}
}

func TestSeatingWalksTheSameOrderAsPlayback(t *testing.T) {
	// seatPlayedCards lays the row out and noteResolved lights one of its seats; both count
	// along combat.ResolutionOrder. If they ever took the order from different places the lit
	// card would be the wrong one, which is exactly the bug the old per-event pile could not
	// have — so this is what replaces that safety.
	s := &CombatScene{
		hand: []paletteCard{
			{actionCard: actionCard{Action: combat.Strike, Element: combat.Fire}, selected: true},
			{actionCard: actionCard{Action: combat.Gather, Element: combat.Ice}, selected: true},
			{actionCard: actionCard{Action: combat.Brace, Element: combat.Earth}, selected: true},
		},
		fighterActions: []combat.Card{
			combat.Of(combat.Strike, combat.Fire),
			combat.Of(combat.Gather, combat.Ice),
			combat.Of(combat.Brace, combat.Earth),
		},
	}
	s.seatPlayedCards()

	// Prepare first, then the attack, then the defense — and each seat holds the card the
	// player actually selected for it, not the one in the same position in the hand.
	// The elements come along, so a seat holding the right concept in the wrong colour fails
	// too — which is the whole reason the hand and the queue are one type now.
	want := []combat.Card{
		combat.Of(combat.Gather, combat.Ice),
		combat.Of(combat.Strike, combat.Fire),
		combat.Of(combat.Brace, combat.Earth),
	}
	if len(s.resolved) != len(want) {
		t.Fatalf("%d cards were seated, want %d", len(s.resolved), len(want))
	}
	for i, c := range want {
		if got := s.resolved[i].card; got != c {
			t.Errorf("seat %d holds %v, want %v", i, got, c)
		}
	}

	// And every seat knows which hand slot it came from, which is what the end-of-round throw
	// and the hand row's own hiding both read.
	for i, r := range s.resolved {
		if r.handIndex < 0 || r.handIndex >= len(s.hand) {
			t.Errorf("seat %d came from hand slot %d, which is not in a hand of %d",
				i, r.handIndex, len(s.hand))
		}
	}
}

func TestBothRowsRaiseTheCardThatIsResolving(t *testing.T) {
	// **The gesture has to be the same on both sides.** It arrived on the player's row alone
	// and the opponent's cards sat still through their whole turn, which read as the enemy's
	// hand being scenery rather than the other half of the round.
	gs := testState()

	player := playedSeatAt(gs, 1, 3)
	enemy := enemySeatAt(gs, 1, 3)

	if got := lift(player, true); got.Y != player.Y-tableFireLift {
		t.Errorf("a firing card on the player's row sits at y=%d, want %d", got.Y, player.Y-tableFireLift)
	}
	if got := lift(enemy, true); got.Y != enemy.Y-tableFireLift {
		t.Errorf("a firing card on the opponent's row sits at y=%d, want %d", got.Y, enemy.Y-tableFireLift)
	}

	// And only the x stays put, so a lift can never be mistaken for a card sliding along.
	if got := lift(enemy, true); got.X != enemy.X {
		t.Errorf("lifting moved a card sideways, to x=%d from %d", got.X, enemy.X)
	}
	if got := lift(enemy, false); got != enemy {
		t.Errorf("an idle card was moved to %v from %v", got, enemy)
	}
}

func TestOnlyOneCardOnTheTableIsLitAtATime(t *testing.T) {
	// A turn is contiguous per side, so the lit card walks the left row and then the right
	// one. The event that lights one seat is the event that unlights the other, which is why
	// neither row has to know the other exists.
	s := &CombatScene{
		fighterActions:  combat.PlainCards(combat.Strike),
		enemyActions:    combat.PlainCards(combat.Jab),
		firingSeat:      -1,
		enemyFiringSeat: -1,
		log: []combat.Event{
			{Kind: combat.KindAction, Side: combat.SideA},
			{Kind: combat.KindAction, Side: combat.SideB},
		},
	}

	// The cursor points at the event being applied, not past it — advancePlayback calls
	// applyEvent before it increments. currentSlot counts inclusively for that reason.
	s.cursor = 0
	s.noteResolved(s.log[0])
	if s.firingSeat != 0 || s.enemyFiringSeat != -1 {
		t.Errorf("after the player's card: player seat %d, enemy seat %d — want 0 and -1",
			s.firingSeat, s.enemyFiringSeat)
	}

	s.cursor = 1
	s.noteResolved(s.log[1])
	if s.firingSeat != -1 || s.enemyFiringSeat != 0 {
		t.Errorf("after the opponent's card: player seat %d, enemy seat %d — want -1 and 0",
			s.firingSeat, s.enemyFiringSeat)
	}
}

func TestAPlayedCardFliesFromItsHandSlotToItsSeat(t *testing.T) {
	gs := testState()

	r := resolvedCard{handIndex: 2, handCount: handSize}
	from := slotAt(gs, 2, handSize)
	to := playedSeatAt(gs, 1, 3)

	if got := r.at(gs, 1, 3, false); got != from {
		t.Errorf("a card that has not set off is at %v, want its hand slot %v", got, from)
	}

	r.age = riseTicks
	if got := r.at(gs, 1, 3, false); got != to {
		t.Errorf("a landed card is at %v, want its seat %v", got, to)
	}

	// Lifted while it resolves, and only once it has landed — a card still arriving is already
	// the most moving thing on screen.
	if got := r.at(gs, 1, 3, true); got.Y != to.Y-tableFireLift {
		t.Errorf("a firing card sits at y=%d, want %d", got.Y, to.Y-tableFireLift)
	}
	r.age = 0
	if got := r.at(gs, 1, 3, true); got != from {
		t.Errorf("a card that has not set off was lifted: %v, want %v", got, from)
	}
}
