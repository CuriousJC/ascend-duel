package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
)

// shieldScene is a screen with defend cards on the player's side of the table, one per element
// given, and two attacks on the opponent's.
func shieldScene(els ...combat.Element) *CombatScene {
	s := &CombatScene{}
	for _, el := range els {
		s.Theater.resolved = append(s.Theater.resolved, resolvedCard{card: combat.Card{Concept: combat.Block, Element: el}})
	}
	s.Theater.enemyDealt = make([]dealtCard, 2)
	s.fighter = &entities.Combatant{}
	s.enemy = &entities.Combatant{}
	// **Both standing**: a fight with a fighter down forms no shields at all.
	s.fighter.CurrentLife, s.enemy.CurrentLife = 100, 100
	return s
}

// **Once a fighter is down the defend phase is over**: the turn's shields, a helm's included, are
// dropped rather than formed over a duel that has ended.
func TestAKillingBlowFormsNoShields(t *testing.T) {
	s := shieldScene(combat.Fire)
	s.noteShieldRaise(raisedAt(0, 2, combat.Fire))
	s.noteWard(combat.Event{Kind: combat.KindWarded, Side: combat.SideA, Amount: 1, Element: combat.Earth})
	s.enemy.CurrentLife = 0

	if s.formPendingShields() {
		t.Error("shields formed after the killing blow")
	}
	if heaped(s, combat.SideA) != 0 || len(s.Theater.pendingRaises) != 0 {
		t.Errorf("%d shields standing and %d still pending after the duel ended",
			heaped(s, combat.SideA), len(s.Theater.pendingRaises))
	}
}

// raisedAt is the event the engine emits for a card at a seat putting shields up.
func raisedAt(seat, count int, el combat.Element) combat.Event {
	return combat.Event{Kind: combat.KindRaised, Side: combat.SideA, Slot: seat, Amount: count, Element: el}
}

// heaped is how many shields a side has standing over table seats, rings and the duelist card.
func heaped(s *CombatScene, side combat.Side) int {
	n := 0
	for _, p := range s.Theater.piles {
		if p.side == side {
			n += len(p.shields)
		}
	}
	return n
}

// raise forms the shields of one raise at once, as the end of a turn would.
func raise(s *CombatScene, e combat.Event) {
	s.noteShieldRaise(e)
	s.formPendingShields()
}

// **A turn's shields form once its arithmetic is done.** A raise reached forms nothing; the turn
// over, every raise in it forms together, each heap over its own card.
func TestTheTurnsShieldsFormWhenTheTurnIsOver(t *testing.T) {
	s := shieldScene(combat.Fire, combat.Ice)
	s.log = []combat.Event{
		raisedAt(0, 2, combat.Fire),
		raisedAt(1, 1, combat.Ice),
		{Kind: combat.KindHand, Side: combat.SideA},
		{Kind: combat.KindAction, Side: combat.SideB},
	}

	s.noteShieldRaise(s.log[0])
	s.noteShieldRaise(s.log[1])
	if heaped(s, combat.SideA) != 0 {
		t.Fatalf("%d shields formed before the turn's arithmetic", heaped(s, combat.SideA))
	}

	s.cursor = 2
	if s.turnOver() {
		t.Fatal("the turn is over while its hand is still to be reached")
	}
	s.cursor = 3
	if !s.turnOver() || !s.formPendingShields() {
		t.Fatal("the turn ended and its shields did not form")
	}
	if len(s.Theater.piles) != 2 || heaped(s, combat.SideA) != 3 {
		t.Errorf("formed %d heaps of %d shields, want 2 heaps of 3", len(s.Theater.piles), heaped(s, combat.SideA))
	}
}

// **A block throws a shield of the element the engine spent** to meet the attack it ate, and the
// heap it came off is one shorter.
func TestABlockThrowsTheShieldItSpent(t *testing.T) {
	gs := testState()
	s := shieldScene(combat.Fire, combat.Ice)
	raise(s, raisedAt(0, 1, combat.Fire))
	raise(s, raisedAt(1, 1, combat.Ice))

	s.noteBlock(gs, combat.Event{Kind: combat.KindBlocked, Side: combat.SideA, Target: combat.SideA,
		Element: combat.Ice, Slot: 1, Surged: true})

	if len(s.Theater.intercepts) != 1 {
		t.Fatalf("%d shields thrown, want 1", len(s.Theater.intercepts))
	}
	m := s.Theater.intercepts[0]
	if m.shield != cards.Ice || m.stopped != 1 || m.label != "SURGE" {
		t.Errorf("threw a %s shield at seat %d (label %q), want the ice one at seat 1, SURGE", m.shield, m.stopped, m.label)
	}
	if p := s.pileAtSeat(combat.SideA, 1); p != nil {
		t.Errorf("the ice heap still holds %d shields", len(p.shields))
	}
	if p := s.pileAtSeat(combat.SideA, 0); p == nil || len(p.shields) != 1 {
		t.Error("the fire heap lost a shield it did not spend")
	}
}

// **The cursor waits for a thrown shield to meet its hit and break**, then lets go, and the attack it
// stopped is counted for the tutorial.
func TestAThrownShieldHoldsTheCursorUntilItLands(t *testing.T) {
	gs := testState()
	s := shieldScene(combat.Fire)
	raise(s, raisedAt(0, 1, combat.Fire))
	for s.Theater.shieldsMoving() {
		s.Theater.Tick()
	}

	s.noteBlock(gs, combat.Event{Kind: combat.KindBlocked, Target: combat.SideA, Element: combat.Fire, Slot: 0})
	if !s.Theater.Running() {
		t.Fatal("a shield in the air does not hold playback")
	}
	for i := 0; i < s.Theater.intercepts[0].t.Ticks; i++ {
		s.Theater.Tick()
	}
	if s.Theater.Running() {
		t.Error("a finished intercept still holds playback")
	}
	if !s.stoppedSeats()[0] {
		t.Error("the stopped attack is not counted")
	}
}

// **At the end of the round the heaps are squared with the model.** What lapsed goes, and what the
// model still holds outlasts the card that raised it.
func TestTheRoundEndKeepsWhatTheModelHolds(t *testing.T) {
	gs := testState()
	s := shieldScene(combat.Fire)
	raise(s, raisedAt(0, 2, combat.Fire))

	var standing combat.ShieldStack
	standing[combat.Fire] = 1
	s.settleShields(gs, combat.SideA, standing)

	if got := heaped(s, combat.SideA); got != 1 {
		t.Fatalf("%d shields still standing, want the 1 the model holds", got)
	}
	if s.pileAtSeat(combat.SideA, 0) != nil {
		t.Error("a shield is still standing over a card that has left the table")
	}
	if len(s.Theater.fades) == 0 {
		t.Error("the lapsed shield vanished rather than fading")
	}

	s.settleShields(gs, combat.SideA, combat.ShieldStack{})
	if got := heaped(s, combat.SideA); got != 0 {
		t.Errorf("%d shields standing after everything lapsed", got)
	}
}

// **An expiry leaves what a worn relic kept**, which is the event's `Life`.
func TestAnExpiryLeavesWhatWasKept(t *testing.T) {
	gs := testState()
	s := shieldScene(combat.Fire)
	raise(s, raisedAt(0, 3, combat.Fire))

	s.noteExpired(gs, combat.Event{Kind: combat.KindExpired, Target: combat.SideA, Amount: 2, Life: 1})
	if got := heaped(s, combat.SideA); got != 1 {
		t.Errorf("%d shields standing after an expiry that kept 1", got)
	}
}

// **The shield count an achievement is asked about comes off the resolved log, not the playback**
// *(2026-09-09)*. Same rule recordHandsPlayed and payHeldVitae are written under: the round is
// decided before a frame of it is drawn, so a record earned as the animation reached a pip would be
// one the player could miss by walking away from the screen.
//
// **It is the peak rather than the count at the end.** KindRaised carries what is standing after
// its own card, so two Guards announce 3 and then 6 — and a shield eaten later in the round was
// still raised.
func TestTheShieldPeakIsReadOffTheLog(t *testing.T) {
	s := &CombatScene{}

	s.noteShieldsRaised([]combat.Event{
		{Kind: combat.KindRaised, Side: combat.SideA, Life: 3},
		{Kind: combat.KindRaised, Side: combat.SideA, Life: 6},
		// The opponent's row is not the player's achievement.
		{Kind: combat.KindRaised, Side: combat.SideB, Life: 11},
		// A blow eats one, and the peak does not follow it down.
		{Kind: combat.KindBlocked, Side: combat.SideB, Life: 5},
	})
	if s.shieldPeak != 6 {
		t.Errorf("the round peaked at %d shields, want the 6 the second raise announced", s.shieldPeak)
	}

	// A round that raised nothing leaves nothing to award.
	s.noteShieldsRaised([]combat.Event{{Kind: combat.KindHand, Side: combat.SideA}})
	if s.shieldPeak != 0 {
		t.Errorf("a round with no raise left a peak of %d, want none", s.shieldPeak)
	}
}
