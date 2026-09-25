package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// These pin the half of the landing figure that has no geometry in it: which events raise one,
// where it says it came from, and the lagging bar. They create no `ebiten.Image` and measure no
// text — the same narrow exception `combat_mathbox_test.go` takes.

// hitScene is a scene with two live combatants and nothing else, which is all the hit logic reads.
//
// **Both are solo attackers**, because a solo attacker's damage event is the one `noteHit` flies;
// a hand-forming side's hits are thrown by the hand dialog's lines — see the throwColumn tests.
func hitScene() *CombatScene {
	return &CombatScene{
		fighter: &entities.Combatant{
			Duelist: combat.Duelist{DMG: 10, Actions: 5, MaxLife: 60, CurrentLife: 60, SoloAttacks: true},
		},
		enemy: &entities.Combatant{
			Duelist: combat.Duelist{DMG: 5, Actions: 5, MaxLife: 100, CurrentLife: 100, SoloAttacks: true},
		},
	}
}

func TestOnlyDamageRaisesALandingFigure(t *testing.T) {
	// **Every other kind has its own row in the theater table**, and a burn in particular has a
	// different source: it ticks off the badge standing on its victim rather than out of a blow.
	// Raising a damage figure for it would draw the same gesture for two different causes.
	s := hitScene()
	for _, k := range []combat.EventKind{
		combat.KindAction, combat.KindBurned, combat.KindBlocked,
		combat.KindHand, combat.KindStatus, combat.KindMissed,
	} {
		s.noteHit(combat.Event{Kind: k, Amount: 10, Target: combat.SideB, Life: 90}, 100)
	}
	if len(s.Theater.hits) != 0 {
		t.Errorf("%d figures were raised by events that are not damage", len(s.Theater.hits))
	}

	s.noteHit(combat.Event{Kind: combat.KindDamage, Amount: 10, Target: combat.SideB, Life: 90}, 100)
	if len(s.Theater.hits) != 1 {
		t.Fatalf("damage raised %d figures, want 1", len(s.Theater.hits))
	}
}

func TestAZeroBlowRaisesNothing(t *testing.T) {
	// A blow of nothing is a figure of nothing flying across the screen. `Amount` can be zero:
	// nothing reduces a blow to zero by the rules, but a shocked turn writes no damage event at
	// all and a future effect might well land a nought.
	s := hitScene()
	s.noteHit(combat.Event{Kind: combat.KindDamage, Amount: 0, Target: combat.SideB, Life: 100}, 100)
	if len(s.Theater.hits) != 0 {
		t.Errorf("a blow of nothing raised %d figures", len(s.Theater.hits))
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

	for i := 0; i < hitFlyTicks(); i++ {
		if got := s.shownLife(combat.SideB, s.enemy.CurrentLife); got != 100 {
			t.Fatalf("tick %d: the bar draws %d before the figure arrived, want 100", i, got)
		}
		s.Theater.Tick()
	}

	if got := s.shownLife(combat.SideB, s.enemy.CurrentLife); got != 70 {
		t.Errorf("the bar draws %d after the figure landed, want the model's 70", got)
	}
	// It is still on screen, being held on the card — the overlap between the figure and the
	// emptier bar is what joins the two.
	if !s.Theater.Running() {
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
	// **The cursor waits on the theater still running**, so a figure that never finishes hangs the round on
	// itself. This is the test that says it cannot.
	s := hitScene()
	s.noteHit(combat.Event{Kind: combat.KindDamage, Amount: 30, Target: combat.SideB, Life: 70}, 100)

	for i := 0; i < hitFlyTicks()+hitHoldTicks()+2; i++ {
		s.Theater.Tick()
	}
	if s.Theater.Running() {
		t.Error("the figure is still running after its whole clock, so playback can never resume")
	}
	if len(s.Theater.hits) != 0 {
		t.Errorf("%d finished figures are still on the scene", len(s.Theater.hits))
	}
}

func TestASoloAttackersFigureLeavesItsLitCard(t *testing.T) {
	// **This is `anchorBlow` for a solo attacker**: every attack lands its own face damage, so the
	// figure comes out of the card that swung — the one that is lit.
	s := hitScene()
	s.Theater.enemyFiringSeats = []int{2}

	if got := s.blowSeat(combat.Event{Side: combat.SideB}); got != 2 {
		t.Errorf("the solo attacker's hit leaves seat %d, want the card that is lit, 2", got)
	}
}

func TestAHandFormingSidesDamageIsNotFlownByPlayback(t *testing.T) {
	// **`SoloAttacks` decides it, never which side it is.** A hand-forming side's hits are thrown by
	// the hand dialog's lines as they finish, so the damage event reached later in playback flies
	// nothing — a second figure would be the same hit landing twice.
	s := hitScene()
	s.fighter.SoloAttacks = false
	s.noteHit(combat.Event{Kind: combat.KindDamage, Side: combat.SideA, Amount: 10, Target: combat.SideB, Life: 90}, 100)
	if len(s.Theater.hits) != 0 {
		t.Errorf("a hand-forming side's damage raised %d figures in playback", len(s.Theater.hits))
	}
}

func TestASoloAttackerWithNothingLitFallsBackToNoSeat(t *testing.T) {
	// Nothing should reach this — a solo attacker's damage always follows the action that lit its
	// card — but the fallback has to be a place that exists rather than seat zero, which would
	// point the figure at whichever card happens to sit at the left of the row.
	s := hitScene()
	s.Theater.enemyFiringSeats = nil

	if got := s.blowSeat(combat.Event{Side: combat.SideB}); got != -1 {
		t.Errorf("a solo attacker with nothing lit leaves seat %d, want -1", got)
	}
}

// --- a hand's hits, thrown by their lines ----------------------------------------------------

// throwScene is a hand-forming player mid-dialog: two lines, and a log whose hits are a landing and
// a miss.
func throwScene() *CombatScene {
	s := hitScene()
	s.fighter.SoloAttacks = false
	s.log = []combat.Event{
		{Kind: combat.KindHand, Side: combat.SideA},
		{Kind: combat.KindDamage, Side: combat.SideA, Target: combat.SideB, Hit: 0, Amount: 30, Life: 70},
		{Kind: combat.KindMissed, Side: combat.SideA, Hit: 1},
		{Kind: combat.KindRoundEnd},
	}
	total := []mathItem{{text: "30"}}
	s.Theater.mathBox = handMathBox{active: true, side: combat.SideA, columns: []mathColumn{
		{hit: 0, items: total, at: 1, logAt: 1},
		{hit: 1, items: total, at: 1, logAt: 2},
		{hit: 2, items: total, at: 1, logAt: -1},
	}}
	return s
}

func TestALandedLineFliesAndTheBarWaitsForIt(t *testing.T) {
	s := throwScene()
	s.throwColumn(0)

	if len(s.Theater.hits) != 1 || s.Theater.hits[0].amount != 30 {
		t.Fatalf("the landed line raised %v, want one figure of 30", s.Theater.hits)
	}
	if s.enemy.CurrentLife != 70 {
		t.Errorf("the model reads %d, want the 70 the hit left", s.enemy.CurrentLife)
	}
	if got := s.shownLife(combat.SideB, s.enemy.CurrentLife); got != 100 {
		t.Errorf("the bar draws %d while the figure is in the air, want 100", got)
	}
	if !s.Theater.walked[1] {
		t.Error("the damage event is not marked as shown, so playback would fly it a second time")
	}
	if !s.Theater.mathBox.columns[0].spent {
		t.Error("the line still draws its total while the figure is in the air")
	}
}

func TestAMissedLineSaysSoAndFliesNothing(t *testing.T) {
	s := throwScene()
	s.throwColumn(1)

	if col := s.Theater.mathBox.columns[1]; col.verdict != "MISS" {
		t.Errorf("the missed line says %q, want MISS", col.verdict)
	}
	if len(s.Theater.hits) != 0 {
		t.Error("a missed hit raised a figure")
	}
	if !s.Theater.walked[2] {
		t.Error("the miss is not marked as shown")
	}
}

func TestALineWhoseHitWasNeverThrownFades(t *testing.T) {
	s := throwScene()
	s.throwColumn(2)

	if !s.Theater.mathBox.columns[2].unthrown {
		t.Error("a hit that never came is drawn as though it did")
	}
}

func TestParallelFiguresEmptyTheBarAsEachArrives(t *testing.T) {
	// **Figures in parallel land in any order**, so the bar is the model plus whatever is still in
	// the air: it drops by each figure as that figure arrives.
	s := hitScene()
	s.enemy.CurrentLife = 50 // two hits of 30 and 20 already written to the model
	s.Theater.hits = []hitFlight{
		{amount: 30, target: combat.SideB, held: 100, t: ui.NewTravel(0, hitFlyTicks()+hitHoldTicks())},
		{amount: 20, target: combat.SideB, held: 70, t: ui.NewTravel(0, hitFlyTicks()+hitHoldTicks())},
	}
	if got := s.shownLife(combat.SideB, s.enemy.CurrentLife); got != 100 {
		t.Errorf("the bar draws %d with both figures in the air, want 100", got)
	}

	// The later figure lands first.
	s.Theater.hits[1].t.Age = hitFlyTicks()
	if got := s.shownLife(combat.SideB, s.enemy.CurrentLife); got != 80 {
		t.Errorf("the bar draws %d once the 20 has landed, want 80", got)
	}
}

func TestTheLandingFigureIsTheSumsTotalContinuing(t *testing.T) {
	// **Four things have to match for one number to appear to set off rather than two to swap**:
	// the point, the frame, the color and the size. The point is `handMathRect`'s center at both
	// ends; the frame is `advancePlayback` clearing the box on the same tick the figure launches;
	// and these are the other two.
	//
	// This is the test that would have caught the version where the figure was its own smaller
	// size in a band the total had left a second and a quarter earlier — which is what it was until
	// the flaw was read back off the code rather than seen on screen.
	if hitFigureSize != mathTotalSize {
		t.Errorf("the landing figure is %v and the sum's total is %v; matching them is what makes "+
			"the flight read as the total traveling", hitFigureSize, mathTotalSize)
	}
	if hitInk() != ui.VerbInkFor(combat.CategoryAttack) {
		t.Error("the landing figure is not the color the sum's total is drawn in")
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
	if len(s.Theater.hits) != 1 {
		t.Fatalf("a solo attacker's damage raised %d figures, want 1", len(s.Theater.hits))
	}
	if got := hitAlpha(s.Theater.hits[0]); got != 1 {
		t.Errorf("the figure sets off at alpha %v, want 1", got)
	}
}

func TestClearingTheSceneDropsFiguresInTheAir(t *testing.T) {
	// **`Init` clears them**, which is the lesson the frozen last round taught: a settled duel does
	// not spend its hand, so anything cleaned up only by the end-of-round spend is still on screen
	// when the next fight starts.
	s := hitScene()
	s.noteHit(combat.Event{Kind: combat.KindDamage, Amount: 30, Target: combat.SideB, Life: 70}, 100)
	s.Theater.Clear()

	if s.Theater.Running() || len(s.Theater.hits) != 0 {
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
	for i := 0; i < hitFlyTicks()+1; i++ {
		s.Theater.Tick()
	}
	if got := s.shownLife(combat.SideB, s.enemy.CurrentLife); got != 0 {
		t.Errorf("the bar draws %d after the killing figure landed, want 0", got)
	}
}
