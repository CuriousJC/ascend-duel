package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
)

// TestEverySquareIsWholeAtBothEndsOfAMorph. The window is what softens the travelling edge, and it
// is also what would leave the first squares gone before the clock started and the last ones still
// going after it stopped — cellProgress stretches the clock by one window for exactly that reason.
// This is the tripwire on the stretch, because on screen the failure is a card that pops on the
// frame the morph hands over.
func TestEverySquareIsWholeAtBothEndsOfAMorph(t *testing.T) {
	for _, delay := range []float64{0, 0.13, 0.5, 0.87, 1} {
		if p := cellProgress(delay, 0); p != 0 {
			t.Errorf("delay %.2f is %.3f through at the start of the morph, want 0", delay, p)
		}
		if p := cellProgress(delay, 1); p != 1 {
			t.Errorf("delay %.2f is %.3f through at the end of the morph, want 1", delay, p)
		}
	}
}

// TestASquareOnlyEverGoesForwards. Every square is a cross-fade between two faces, so a progress
// that went back would be a card coming back out of the one that replaced it.
func TestASquareOnlyEverGoesForwards(t *testing.T) {
	for _, delay := range []float64{0, 0.31, 0.74, 1} {
		last := 0.0
		for step := 0; step <= 100; step++ {
			p := cellProgress(delay, float64(step)/100)
			if p < last {
				t.Fatalf("delay %.2f went from %.3f back to %.3f", delay, last, p)
			}
			last = p
		}
	}
}

// TestAnEarlySquareLeadsALateOne. The delays are the whole pattern; a cellProgress that ignored
// them would compile, pass the two tests above and draw a plain cross-fade.
func TestAnEarlySquareLeadsALateOne(t *testing.T) {
	const mid = 0.5
	early, late := cellProgress(0.1, mid), cellProgress(0.9, mid)
	if early <= late {
		t.Errorf("at half way an early square is %.3f through and a late one %.3f; the early one "+
			"should be ahead", early, late)
	}
}

// TestWhichFacesAMorphHasIsWhatItDoes. There is no style enum on purpose — see cardmorph.go — so
// the three constructors have to disagree about their two faces or all three are the same morph.
func TestWhichFacesAMorphHasIsWhatItDoes(t *testing.T) {
	a := cards.Spec{Name: "Jab"}
	b := cards.Spec{Name: "Thrust"}

	if m := morphInto(a, b, cards.Hand); !m.hasBefore || !m.hasAfter {
		t.Error("a card replaced by another needs both faces")
	}
	if m := morphAway(a, cards.Hand); !m.hasBefore || m.hasAfter {
		t.Error("an eaten card has a face to lose and none to gain")
	}
	if m := morphIn(b, cards.Hand); m.hasBefore || !m.hasAfter {
		t.Error("a card arriving out of nothing has no face to lose")
	}
}

// TestAMorphWaitsBeforeItChanges, so the card that landed is still for a beat and can be read.
// **The pause is the travel's delay**, which is what keeps one clock for the whole transition; a
// morph reporting itself finished before it had waited would be one that never ran at all.
func TestAMorphWaitsBeforeItChanges(t *testing.T) {
	m := morphInto(cards.Spec{Name: "Jab"}, cards.Spec{Name: "Thrust"}, cards.Hand)
	if !m.waiting() {
		t.Fatal("a fresh morph should be holding on the old card")
	}
	for i := 0; i < morphWaitTicks; i++ {
		m.tick()
	}
	if m.waiting() {
		t.Error("still waiting after the whole pause")
	}
	if m.done() {
		t.Fatal("finished before the dissolve had run")
	}
	for i := 0; i < morphTicks; i++ {
		m.tick()
	}
	if !m.done() {
		t.Error("not finished after the pause and the dissolve together")
	}
}
