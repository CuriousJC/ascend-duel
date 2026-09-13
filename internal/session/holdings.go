package session

// **Holdings: everything a choice made between two fights can change.**
//
// The ledger records rounds, which is what the combat screen hands it — so a run's account went
// silent the moment the duel ended and picked up again at the next one. Everything the player
// actually *decided* happened in that gap: the card they took, the card they cut, the essence they
// spent, the relic they bought, the rung they raised. See ledger.go's After block, which is where
// the words end up.
//
// **A snapshot rather than a notification at each call site** *(owner's call, 2026-09-12)*. The
// screens could announce each choice as they committed it, and that is a list a new mechanic gets
// forgotten from — silently, because a missing announcement and a deliberate silence read the
// same. What actually changed is read back off the run instead, exactly as
// `internal/screens/combat_handmorph.go` reads a parasite's work off the card faces rather than
// off the parasite: "no parasite has a case anywhere in the drawing, which is what stops a new one
// arriving with no picture".
//
// **This package still words nothing.** A snapshot is cards and keys; `internal/screens` turns a
// pair of them into sentences, on the rule that the words are decided in one place.

import "github.com/curiousjc/ascend-duel/internal/combat"

// Holdings is the run's whole answer to "what have I got", taken at one moment.
//
// Every field is a copy. A caller holds one across a choice being committed, which is exactly the
// window in which the run is rewriting its own slices.
type Holdings struct {
	// Cards is the deck, in order. **Told apart by combat.Card.ID**, which is what makes a card
	// that was altered distinguishable from one that was cut and another taken.
	Cards []combat.Card

	// Relics is the worn row, Held the parasites in hand, Pouch the stones not yet spent.
	Relics []string
	Held   []string
	Pouch  []string

	// Stones is how far each rung has been raised, by hand key.
	Stones map[string]int

	Vitae int
}

// Holdings takes the snapshot.
func (s *Session) Holdings() Holdings {
	if s == nil {
		return Holdings{}
	}
	return Holdings{
		Cards:  s.Deck(),
		Relics: s.Worn(),
		Held:   s.Held(),
		Pouch:  s.Carried(),
		Stones: s.StoneCounts(),
		Vitae:  s.Vitae(),
	}
}
