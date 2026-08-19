package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
)

// These pin the half of the landing figure that has no geometry in it: which events raise one,
// where it says it came from, and the lagging bar. They create no `ebiten.Image` and measure no
// text — the same narrow exception `combat_mathbox_test.go` takes.

// hitScene is a scene with two live combatants and nothing else, which is all the hit logic reads.
func hitScene() *CombatScene {
	return &CombatScene{
		fighter: &entities.Combatant{
			Duelist: combat.Duelist{DMG: 10, Actions: 5, MaxLife: 60, CurrentLife: 60},
		},
		enemy: &entities.Combatant{
			Duelist: combat.Duelist{DMG: 5, Actions: 5, MaxLife: 100, CurrentLife: 100},
		},
	}
}

func TestOnlyDamageRaisesALandingFigure(t *testing.T) {
	// **Every other kind has its own row in the theatre table**, and a burn in particular has a
	// different source: it ticks off the badge standing on its victim rather than out of a blow.
	// Raising a damage figure for it would draw the same gesture for two different causes.
	s := hitScene()
	for _, k := range []combat.EventKind{
		combat.KindAction, combat.KindBurned, combat.KindNegated,
		combat.KindHand, combat.KindStatus, combat.KindMissed,
	} {
		s.noteHit(combat.Event{Kind: k, Amount: 10, Target: combat.SideB, Life: 90}, 100)
	}
	if len(s.hits) != 0 {
		t.Errorf("%d figures were raised by events that are not damage", len(s.hits))
	}

	s.noteHit(combat.Event{Kind: combat.KindDamage, Amount: 10, Target: combat.SideB, Life: 90}, 100)
	if len(s.hits) != 1 {
		t.Fatalf("damage raised %d figures, want 1", len(s.hits))
	}
}

func TestAZeroBlowRaisesNothing(t *testing.T) {
	// A blow of nothing is a figure of nothing flying across the screen. `Amount` can be zero:
	// nothing reduces a blow to zero by the rules, but a shocked turn writes no damage event at
	// all and a future effect might well land a nought.
	s := hitScene()
	s.noteHit(combat.Event{Kind: combat.KindDamage, Amount: 0, Target: combat.SideB, Life: 100}, 100)
	if len(s.hits) != 0 {
		t.Errorf("a blow of nothing raised %d figures", len(s.hits))
	}
}

func TestTheHeldLifeComesFromTheEventNotTheCard(t *testing.T) {
	// **`applyEvent` has already written the new life by the time the figure is raised**, which is
	// the whole division this screen keeps: the model moves first and the drawing lags. So the
	// figure has to reconstruct the pre-blow total from the event — `Life + Amount` — because
	// reading the combatant would give the number the bar is trying not to show yet, and the bar
	// would appear not to lag at all.
	s := hitScene()
	s.enemy.CurrentLife = 70 // as applyEvent would have left it

	s.noteHit(combat.Event{Kind: combat.KindDamage, Amount: 30, Target: combat.SideB, Life: 70}, 100)

	if got := s.shownLife(combat.SideB, s.enemy.CurrentLife); got != 100 {
		t.Errorf("the bar draws %d while the figure is in the air, want the 100 it had before", got)
	}
	if s.enemy.CurrentLife != 70 {
		t.Errorf("the model reads %d - the flight must not touch it", s.enemy.CurrentLife)
	}
}

func TestTheBarCatchesUpWhenTheFigureLands(t *testing.T) {
	s := hitScene()
	s.enemy.CurrentLife = 70
	s.noteHit(combat.Event{Kind: combat.KindDamage, Amount: 30, Target: combat.SideB, Life: 70}, 100)

	for i := 0; i < hitFlyTicks; i++ {
		if got := s.shownLife(combat.SideB, s.enemy.CurrentLife); got != 100 {
			t.Fatalf("tick %d: the bar draws %d before the figure arrived, want 100", i, got)
		}
		s.tickHits()
	}

	if got := s.shownLife(combat.SideB, s.enemy.CurrentLife); got != 70 {
		t.Errorf("the bar draws %d after the figure landed, want the model's 70", got)
	}
	// It is still on screen, being held on the card — the overlap between the figure and the
	// emptier bar is what joins the two.
	if !s.hitsRunning() {
		t.Error("the figure was dropped the instant it arrived, so nothing holds on the card")
	}
}

func TestTheOtherSidesBarIsUnaffected(t *testing.T) {
	// A figure aimed at one card must not hold the other one's bar. Both are drawn from the same
	// function, so this is the mistake a side-blind implementation would make.
	s := hitScene()
	s.enemy.CurrentLife = 70
	s.noteHit(combat.Event{Kind: combat.KindDamage, Amount: 30, Target: combat.SideB, Life: 70}, 100)

	if got := s.shownLife(combat.SideA, s.fighter.CurrentLife); got != 60 {
		t.Errorf("the duelist's bar draws %d while the enemy is being hit, want its own 60", got)
	}
}

func TestTheFigureFinishesAndIsDroppedSoPlaybackCanResume(t *testing.T) {
	// **The cursor waits on `hitsRunning`**, so a figure that never finishes hangs the round on
	// itself. This is the test that says it cannot.
	s := hitScene()
	s.noteHit(combat.Event{Kind: combat.KindDamage, Amount: 30, Target: combat.SideB, Life: 70}, 100)

	for i := 0; i < hitFlyTicks+hitHoldTicks+2; i++ {
		s.tickHits()
	}
	if s.hitsRunning() {
		t.Error("the figure is still running after its whole clock, so playback can never resume")
	}
	if len(s.hits) != 0 {
		t.Errorf("%d finished figures are still on the scene", len(s.hits))
	}
}

func TestAScoredHandsFigureLeavesTheSumAndASoloAttackersLeavesItsCard(t *testing.T) {
	// **This is `anchorBlow`**, and it is the one anchor that is a rule rather than a rectangle.
	// A player's turn is one blow read off a hand, so the total is already on screen in the sum
	// box and the figure travels from there. A solo attacker emits no hand at all — every attack
	// lands its own face damage — so there is no sum, and the figure comes out of the card that
	// swung.
	//
	// **`SoloAttacks` is what decides it, never which side it is.** The engine has no idea which
	// duelist is a person and this screen must not grow a second opinion; the balance tool plays
	// both sides headlessly on the same flag.
	s := hitScene()
	s.enemy.SoloAttacks = true
	s.enemyFiringSeats = []int{2}
	s.firingSeats = []int{1}

	if got := s.blowSeat(combat.Event{Side: combat.SideA}); got != -1 {
		t.Errorf("the player's blow leaves seat %d, want -1 for the sum line", got)
	}
	if got := s.blowSeat(combat.Event{Side: combat.SideB}); got != 2 {
		t.Errorf("the solo attacker's blow leaves seat %d, want the card that is lit, 2", got)
	}

	// And the flag, not the side: a hand-forming opponent's figure comes out of the sum exactly as the
	// player's does.
	s.enemy.SoloAttacks = false
	if got := s.blowSeat(combat.Event{Side: combat.SideB}); got != -1 {
		t.Errorf("a hand-forming opponent's blow leaves seat %d, want -1 for the sum line", got)
	}
}

func TestASoloAttackerWithNothingLitFallsBackToTheSum(t *testing.T) {
	// Nothing should reach this — a solo attacker's damage always follows the action that lit its
	// card — but the fallback has to be a place that exists rather than seat zero, which would
	// point the figure at whichever card happens to sit at the left of the row.
	s := hitScene()
	s.enemy.SoloAttacks = true
	s.enemyFiringSeats = nil

	if got := s.blowSeat(combat.Event{Side: combat.SideB}); got != -1 {
		t.Errorf("a solo attacker with nothing lit leaves seat %d, want the sum line", got)
	}
}

func TestTheLandingFigureIsTheSumsTotalContinuing(t *testing.T) {
	// **Four things have to match for one number to appear to set off rather than two to swap**:
	// the point, the frame, the colour and the size. The point is `handMathRect`'s centre at both
	// ends; the frame is `advancePlayback` clearing the box on the same tick the figure launches;
	// and these are the other two.
	//
	// This is the test that would have caught the version where the figure was its own smaller
	// size in a band the total had left a second and a quarter earlier — which is what it was until
	// the flaw was read back off the code rather than seen on screen.
	if hitFigureSize != mathTotalSize {
		t.Errorf("the landing figure is %v and the sum's total is %v; matching them is what makes "+
			"the flight read as the total travelling", hitFigureSize, mathTotalSize)
	}
	if hitInk() != verbInkFor(combat.CategoryAttack) {
		t.Error("the landing figure is not the colour the sum's total is drawn in")
	}
	if hitFromScale != 1.0 {
		t.Errorf("the figure sets off at %v; it has to start life-size, because its first frame "+
			"replaces the total's last one", hitFromScale)
	}
	if hitToScale >= hitFromScale {
		t.Errorf("the figure goes from %v to %v; it recedes into the card, where a term flying "+
			"into the sum grows toward the reader", hitFromScale, hitToScale)
	}

	// And it is solid from the first frame: a fade-in would blink against the opaque total.
	s := hitScene()
	s.noteHit(combat.Event{Kind: combat.KindDamage, Amount: 30, Target: combat.SideB, Life: 70}, 100)
	if got := hitAlpha(s.hits[0]); got != 1 {
		t.Errorf("the figure sets off at alpha %v, want 1", got)
	}
}

func TestClearingTheSceneDropsFiguresInTheAir(t *testing.T) {
	// **`Init` clears them**, which is the lesson the frozen last round taught: a settled duel does
	// not spend its hand, so anything cleaned up only by the end-of-round spend is still on screen
	// when the next fight starts.
	s := hitScene()
	s.noteHit(combat.Event{Kind: combat.KindDamage, Amount: 30, Target: combat.SideB, Life: 70}, 100)
	s.clearHits()

	if s.hitsRunning() || len(s.hits) != 0 {
		t.Error("a figure survived the scene being cleared")
	}
	if got := s.shownLife(combat.SideB, s.enemy.CurrentLife); got != s.enemy.CurrentLife {
		t.Errorf("the bar still lags at %d after clearing, want the model's %d",
			got, s.enemy.CurrentLife)
	}
}

func TestAKillingBlowHoldsTheLifeThatWasThereNotTheSizeOfTheBlow(t *testing.T) {
	// **The bug this exists for** *(2026-08-19)*: an enemy on 30 of 90 took a pair of Cleaves for
	// 60, and its bar drew 60/90 for the length of the flight before emptying — health visibly
	// going *up* on the one blow that kills.
	//
	// The cause was the held life being worked back from the event: `e.Life + e.Amount`, which is
	// right whenever the blow is smaller than the life it lands on and is the *size of the blow*
	// whenever it is not, `e.Life` being clamped at zero. **Overkill is the only case that shows
	// it**, which is why it was invisible until a fight ended.
	//
	// `applyEvent` reads the life off the combatant before overwriting it now, so this passes the
	// pre-hit life the way the caller does.
	s := hitScene()
	s.enemy.MaxLife, s.enemy.CurrentLife = 90, 30

	before := s.enemy.CurrentLife
	s.enemy.CurrentLife = 0 // what applyEvent writes from e.Life
	s.noteHit(combat.Event{Kind: combat.KindDamage, Amount: 60, Target: combat.SideB, Life: 0}, before)

	if got := s.shownLife(combat.SideB, s.enemy.CurrentLife); got != 30 {
		t.Errorf("the bar draws %d while a 60 lands on 30 of 90, want the 30 that was there", got)
	}

	// And once the figure arrives it is the real life, which is zero — the drop still happens, it
	// just happens on arrival like every other hit.
	for i := 0; i < hitFlyTicks+1; i++ {
		s.tickHits()
	}
	if got := s.shownLife(combat.SideB, s.enemy.CurrentLife); got != 0 {
		t.Errorf("the bar draws %d after the killing figure landed, want 0", got)
	}
}
