package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

// **The lit area is the clickable area, so a lit card the lesson did not name is a card the player
// may play.** That is not a cosmetic slip: the taught hand costs the whole action budget, so one
// stray queued card makes one taught card unpayable and the round commits a hand the step has just
// promised would be something else.
//
// It went wrong exactly once, and this is the shape of it *(2026-09-08)*. `matching-cards` handed
// back the bounding box over the matching seats, on a note arguing that the sort keeps them
// contiguous — which is true of cards sharing a *concept* and not of cards sharing an *element*,
// which is what the lesson matches on. Under the default cost sort the taught four land at seats
// 0, 1, 2 and 4 with an arcane card at seat 3, inside the square.
func TestTheMatchingCardsGateLightsOnlyTheTaughtCards(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}

	s := newTaughtScene(t, gs)
	match := s.matchingCards(gs)
	if len(match) < 2 {
		t.Fatalf("the taught hand holds a matching set of %d; the lesson needs one to point at",
			len(match))
	}

	taught := map[int]bool{}
	for _, i := range match {
		taught[i] = true
	}

	rects, ok := s.tutorialRects(gs, tutorial.AnchorMatchingCards)
	if !ok {
		t.Fatal("the matching-cards anchor reports no rectangle for a hand that has a set in it")
	}

	// Every seat the hand holds is asked whether the gate would let it be clicked, using the same
	// predicate every control on this screen asks.
	gs.InputGated, gs.InputFocus = true, rects
	for seat := range s.hand {
		at := center(s.cardSlot(gs, seat))
		allowed := gs.InputAllowed(at)
		switch {
		case taught[seat] && !allowed:
			t.Errorf("seat %d is one of the taught cards (%s/%s) and the gate refuses it",
				seat, combat.ConceptOf(s.hand[seat].Concept).Label, s.hand[seat].Element)
		case !taught[seat] && allowed:
			t.Errorf("seat %d is a %s/%s, which the lesson did not name, and the gate allows it — "+
				"queueing it spends budget the taught set needs",
				seat, combat.ConceptOf(s.hand[seat].Concept).Label, s.hand[seat].Element)
		}
	}
}

// **And the taught cards really are scattered**, which is what makes the test above worth having
// rather than a tautology. If a future sort ever made them contiguous this would go red, and the
// answer is to delete it rather than to weaken the one above: the gate has to hold either way.
func TestTheTaughtCardsAreNotContiguous(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}

	s := newTaughtScene(t, gs)
	match := s.matchingCards(gs)
	if len(match) < 2 {
		t.Fatal("no matching set to reason about")
	}
	if match[len(match)-1]-match[0] == len(match)-1 {
		t.Skip("the taught set is contiguous in this hand, so the bounding box would have been " +
			"harmless here; the per-card gate is still the rule")
	}
	t.Logf("the taught set sits at seats %v — the box round it also covers %d other card(s)",
		match, match[len(match)-1]-match[0]+1-len(match))
}

// newTaughtScene is a combat scene holding the hand the tutorial's own seed deals, sorted the way
// the game sorts it. **The real hand and the real sort**, because the bug was a fact about the two
// together and a fixture would have hidden it.
func newTaughtScene(t *testing.T, gs *state.GlobalState) *CombatScene {
	t.Helper()

	script := tutorial.Load()
	runSeed, err := seeds.Parse(script.Seed)
	if err != nil {
		t.Fatalf("tutorial.json seed %q: %v", script.Seed, err)
	}

	gs.Run = session.New(nil)

	s := stubCombat()
	s.hand = nil
	for _, c := range OpeningCards(seeds.ForFight(runSeed, seeds.PlayerDeck, 0)) {
		s.hand = append(s.hand, paletteCard{actionCard: c})
	}
	s.sortHand()

	teachRun(t, gs, script)
	return s
}

// teachRun puts the shipped script on the run, so matchingCards has an axis to match on.
func teachRun(t *testing.T, gs *state.GlobalState, script tutorial.Script) {
	t.Helper()
	gs.Run.Teach(script)
}

// **"Take the other three" must not point at the one already taken.** The anchor is the click gate
// as well as the mark, so a queued card left in the set is a queued card the step invites you to
// click — and clicking it undoes the step before this one. It read as four red cards under a
// sentence about three.
func TestTheStepAsksOnlyForTheCardsStillToTake(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}

	s := newTaughtScene(t, gs)
	match := s.matchingCards(gs)
	if len(match) < 2 {
		t.Fatal("no matching set to reason about")
	}

	// The lesson's own opening move: the first card of the set is queued.
	s.hand[match[0]].selected = true

	left, ok := s.tutorialRects(gs, tutorial.AnchorMatchingCardsLeft)
	if !ok {
		t.Fatal("the anchor reports nothing while three cards are still to take")
	}
	if len(left) != len(match)-1 {
		t.Errorf("%d of the %d taught cards are still to take and the step lights %d",
			len(match)-1, len(match), len(left))
	}

	gs.InputGated, gs.InputFocus = true, left
	if gs.InputAllowed(center(s.cardSlot(gs, match[0]))) {
		t.Error("the card the player already queued is still clickable, so the one thing this " +
			"step invites is undoing the last one")
	}
	for _, i := range match[1:] {
		if !gs.InputAllowed(center(s.cardSlot(gs, i))) {
			t.Errorf("seat %d is still to be taken and the step refuses it", i)
		}
	}

	// **And the whole set still has an anchor**, because the step after this one describes all four
	// — including the Brace, which is the card that step is about.
	whole, ok := s.tutorialRects(gs, tutorial.AnchorMatchingCards)
	if !ok || len(whole) != len(match) {
		t.Errorf("the describing step lights %d of the %d taught cards", len(whole), len(match))
	}
}

// Once every taught card is queued the step has nothing left to ask for, and says so rather than
// lighting the row it has finished with.
func TestTheStepEmptiesAsTheCardsAreTaken(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}

	s := newTaughtScene(t, gs)
	for _, i := range s.matchingCards(gs) {
		s.hand[i].selected = true
	}
	if _, ok := s.tutorialRects(gs, tutorial.AnchorMatchingCardsLeft); ok {
		t.Error("every taught card is queued and the step is still asking for one")
	}
}
