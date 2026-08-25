package screens

// **The combat screen's half of the tutorial**: what it can say is true, and where it draws the
// things Bob points at.
//
// It is a file of its own rather than three methods scattered through combat.go, because the
// whole of the feature's footprint on this screen is here — the overlay's own drawing is in
// tutorial.go and the state machine is in `internal/tutorial`. A tutorial that grew tendrils
// through the screen would be one nobody could take back out.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

// tutorialFacts is what this screen can honestly report about the run this frame.
//
// **Everything here is read, never derived for the tutorial's benefit.** `Resolving` is the
// screen's own playback cursor and `RoundsPlayed` is the counter it already keeps, so a condition
// cannot end up watching a number that exists only because a lesson wanted one — which would be
// a fact about the tutorial rather than about the game.
func (s *CombatScene) tutorialFacts(gs *state.GlobalState) tutorial.Facts {
	f := tutorial.Facts{
		Queued: len(s.fighterActions),

		// **Unqueued, not the hand's length.** A queued card stays in the row with `selected` set —
		// see `toggle` and `syncQueue` — so `len(s.hand)` is the same number before and after the
		// player has picked everything, and a step waiting on that waited forever.
		Unqueued:     len(s.hand) - s.selectedCount(),
		Resolving:    !s.planning() && !s.duelSettled(),
		RoundsPlayed: s.round,
	}
	match := s.matchingCards()
	f.Matching = len(match)
	for _, i := range match {
		if s.hand[i].selected {
			f.MatchingQueued++
		}
	}
	if gs.Run != nil {
		f.Phase = gs.Run.Phase().String()
	}
	return f
}

// tutorialRect is where each of this screen's anchors is drawn.
//
// **The rectangles are asked of the same functions that draw the things**, never re-derived from
// the constants they were laid out with. A spotlight around where a control used to be is the one
// failure this whole feature cannot survive, and it is exactly what a second copy of a layout
// produces the first time the first copy moves.
func (s *CombatScene) tutorialRect(gs *state.GlobalState, a tutorial.Anchor) (image.Rectangle, bool) {
	switch a {
	case tutorial.AnchorEnemyCard:
		return s.enemyCardRect(gs), true
	case tutorial.AnchorDuelistCard:
		return s.duelistCardRect(gs), true
	case tutorial.AnchorTowerPlace:
		return s.towerPlaceRect(gs), true
	case tutorial.AnchorHand:
		return handZone(gs), true
	case tutorial.AnchorFirstCard:
		// **The card, not the band.** `cardSlot` is the same rectangle the click is hit-tested
		// against, so the lit square and the one legal click cannot describe different pixels.
		// An empty hand has no first card and reports false, which drops the gate rather than
		// shielding the screen around a seat with nothing in it.
		if len(s.hand) == 0 {
			return image.Rectangle{}, false
		}
		return s.cardSlot(gs, 0), true
	case tutorial.AnchorMatchingCards:
		// **The seats the matching cards occupy, as one rectangle.** The lit square and the one
		// legal click are the same rectangle here as everywhere else, which is what makes the
		// lesson true by construction: the only cards the player can reach are the ones Bob just
		// said match.
		//
		// **They are contiguous because the hand is sorted**, and every sort mode ends in the
		// same key chain — cards sharing a concept differ only by element, so they land side by
		// side whichever key leads. The union is taken over the seats rather than assumed to be a
		// span, so a sort that ever broke them apart would light a wider square rather than the
		// wrong one.
		match := s.matchingCards()
		if len(match) == 0 {
			return image.Rectangle{}, false
		}
		r := s.cardSlot(gs, match[0])
		for _, i := range match[1:] {
			r = r.Union(s.cardSlot(gs, i))
		}
		return r, true

	case tutorial.AnchorDeckStack:
		return deckStackBounds(gs), true
	case tutorial.AnchorMathBand:
		// The band the blow is added up in. **The whole band rather than the figures in it**: the
		// sum is laid out centred and its width is a function of how many terms the round produced,
		// so a rectangle round the figures would be a different size every round and the square
		// would appear to twitch.
		return s.handMathRect(gs), true

	case tutorial.AnchorAPBar:
		// The bar is drawn from the hand band's bottom edge — see drawAPBar's caller — and it is
		// eight pixels tall, which is too thin to spotlight on its own. The rectangle returned is
		// the bar plus the figure written under it, since "3/6 AP" is the half of the pair a
		// player can actually read.
		band := handZone(gs)
		return image.Rect(band.Min.X, band.Max.Y+apBarBelow-4,
			band.Max.X, band.Max.Y+apBarBelow+apBarHeight+apFigureBelowBar+20), true

	case tutorial.AnchorDuelButton:
		return buttonRect(s.duelButton), true
	case tutorial.AnchorHandsButton:
		return buttonRect(s.hands.button), true
	}
	return image.Rectangle{}, false
}

// tutorialCovered is whether one of this screen's three dialogs is up: the deck overlay, the fight
// log or the hands ladder. **It is `modalUp` and nothing else**, rather than a second list of the
// same three — a spotlight that kept pointing after a fourth dialog was added would be exactly the
// bug this method exists to fix.
func (s *CombatScene) tutorialCovered(*state.GlobalState) bool { return s.modalUp() }

// matchingCards is the largest set of cards in the hand sharing a concept, as hand indices in
// order. It is what [tutorial.AnchorMatchingCards] points at and what [tutorial.CondMatchQueued]
// counts, and both read this one function so the square and the condition cannot describe
// different cards.
//
// **A tie goes to whichever concept appears first in the hand**, which is a rule rather than a
// preference: ranging the tally would be map order, and the tutorial would point at a different
// pair of cards on different launches of the same seed. See the determinism rules in CLAUDE.md.
//
// **Fewer than two is no set at all.** One card matches nothing, and a step asking the player to
// find a hand in a hand that has not got one would gate the screen down to a single card and then
// wait for a condition that is already satisfied.
func (s *CombatScene) matchingCards() []int {
	counts := make(map[combat.ConceptID]int, len(s.hand))
	for _, c := range s.hand {
		counts[c.actionCard.Concept]++
	}

	best, bestN := combat.ConceptID(0), 0
	for _, c := range s.hand {
		if n := counts[c.actionCard.Concept]; n > bestN {
			best, bestN = c.actionCard.Concept, n
		}
	}
	if bestN < 2 {
		return nil
	}

	var out []int
	for i, c := range s.hand {
		if c.actionCard.Concept == best {
			out = append(out, i)
		}
	}
	return out
}
