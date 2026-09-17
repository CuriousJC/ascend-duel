package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// morphScene is a combat screen holding a hand dealt straight off a run, with nothing else going on.
func morphScene(t *testing.T, deck []combat.Card) (*state.GlobalState, *CombatScene) {
	t.Helper()
	run := session.New(deck)
	gs := &state.GlobalState{
		ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight, Run: run,
	}
	s := &CombatScene{run: run, fighter: &entities.Combatant{}}
	for _, c := range run.Deck() {
		s.hand = append(s.hand, paletteCard{Card: c, selected: true})
	}
	return gs, s
}

// spend applies a rune by key the way the screen does, and hands back the faces from before it
// landed so a test can raise the morphs off them.
func spend(t *testing.T, gs *state.GlobalState, s *CombatScene, key string) {
	t.Helper()
	p, ok := session.RuneByKey(key)
	if !ok {
		t.Skipf("no %s in the catalog", key)
	}
	was, seats := s.handFaces(gs)
	if !gs.Run.ApplyRuneRolling(p, s.selectedCardIDs(), nil) {
		t.Fatalf("%s refused the selection", key)
	}
	s.resyncHandFromRun(gs)
	for _, copied := range gs.Run.Duplicated() {
		s.hand = append(s.hand, paletteCard{Card: copied})
	}
	s.raiseHandMorphs(gs, was, seats)
}

// TestARecoloringRuneMorphsEveryCardItTook. **One beat for all of them** — the morphs are
// raised together, so a rune that named two cards puts two on stage at once rather than one
// after the other.
func TestARecoloringRuneMorphsEveryCardItTook(t *testing.T) {
	gs, s := morphScene(t, combat.PlainCards(combat.Bash, combat.Bash))
	spend(t, gs, s, "hexmark")

	if got := len(s.Theater.morphs); got != 2 {
		t.Fatalf("a two-card borer raised %d morphs, want 2", got)
	}
	for _, h := range s.Theater.morphs {
		if !h.m.Replaces() {
			t.Errorf("card %d: a recolored card should be one face turning into another", h.id)
		}
		if _, still := s.handMorphFor(h.id); !still {
			t.Errorf("card %d: the morph cannot be found by the id the row will look it up with", h.id)
		}
	}
}

// TestTheMorphsRunTogether. The clocks are what make it one beat rather than a queue of them, so
// they have to be started on the same frame and be the same length.
func TestTheMorphsRunTogether(t *testing.T) {
	gs, s := morphScene(t, combat.PlainCards(combat.Bash, combat.Bash))
	spend(t, gs, s, "hexmark")

	first := s.Theater.morphs[0].m.Clock()
	for _, h := range s.Theater.morphs[1:] {
		if h.m.Clock() != first {
			t.Errorf("one morph is on %+v and another on %+v; they should share a beat", first, h.m.Clock())
		}
	}
}

// TestAnEatenCardMorphsAwayAtTheSeatItHad. The row closes over it immediately, so there is no seat
// to look up — this is the one case the captured rectangle exists for, and a zero rectangle would
// draw the card in the top-left corner of the screen.
func TestAnEatenCardMorphsAwayAtTheSeatItHad(t *testing.T) {
	gs, s := morphScene(t, combat.PlainCards(combat.Bash, combat.Bash))
	spend(t, gs, s, "unmake")

	if len(s.hand) != 0 {
		t.Fatalf("unmake left %d cards in the hand, want none", len(s.hand))
	}
	if got := len(s.Theater.morphs); got != 2 {
		t.Fatalf("unmake raised %d morphs, want 2", got)
	}
	for _, h := range s.Theater.morphs {
		if !h.m.Eats() {
			t.Errorf("card %d: an eaten card has a face to lose and none to gain", h.id)
		}
		if h.at.Empty() {
			t.Errorf("card %d: no seat was captured, so the ghost would be drawn at the origin", h.id)
		}
	}
}

// TestACopyMorphsInOutOfNothing. It has no earlier face, which is exactly what tells it apart from
// a card that was changed — see raiseHandMorphs, which reads the difference rather than the
// rune.
func TestACopyMorphsInOutOfNothing(t *testing.T) {
	gs, s := morphScene(t, combat.PlainCards(combat.Bash))
	spend(t, gs, s, "mimic")

	if got := len(s.Theater.morphs); got != 1 {
		t.Fatalf("a copy raised %d morphs, want 1", got)
	}
	m := s.Theater.morphs[0].m
	if !m.Arrives() {
		t.Error("a copy arrives out of nothing; it has no face to lose")
	}
}

// TestARuneThatChangedNothingRaisesNothing. The morphs come off a comparison, so a card whose
// face is untouched must not flash — which is what would happen if the raise walked the rune's
// targets rather than the difference.
func TestARuneThatChangedNothingRaisesNothing(t *testing.T) {
	gs, s := morphScene(t, combat.PlainCards(combat.Bash, combat.Bash))
	was, seats := s.handFaces(gs)
	s.raiseHandMorphs(gs, was, seats)

	if got := len(s.Theater.morphs); got != 0 {
		t.Errorf("an unchanged hand raised %d morphs, want none", got)
	}
}
