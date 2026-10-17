package session

// The one rune that does not act on a card: the echo.
//
// It is here rather than in rune.go because it is not an alteration to the deck, which is what
// that file is about — a chimera moves nothing at all and becomes another rune when it is used.
//
// **The gamble used to live here too and moved into `internal/combat` on 2026-09-09** *(owner's
// call)*. It stopped being a consumable that rolled once between turns and became an upgrade a card
// permanently carries, so the roll now happens where the card is played. What is left on this side
// is Grant, which is how the rules hand a permanent bonus back to the run that owns it.

// Echoes is the rune a chimera would become. For everything else it is the identity.
//
// It reports false when a chimera has nothing to copy: a run where no rune has ever been spent.
// That is a refusal rather than a fallback, because a chimera that became nothing is a consumable
// the player paid for and did not get.
func (s *Session) Echoes(p Rune) (Rune, bool) {
	if p.Target != RuneChimera {
		return p, true
	}
	if s.lastRune == "" {
		return Rune{}, false
	}
	echoed, ok := RuneByKey(s.lastRune)
	if !ok || echoed.Target == RuneChimera {
		// A key the catalog no longer holds, or a chimera that somehow recorded itself. Neither
		// can happen today — `rememberRune` writes a resolved record and `Resume` refuses an
		// unknown one — and both would be a chimera firing nothing if they did.
		return Rune{}, false
	}
	return echoed, true
}

// EchoedName is what a chimera would become, for a tooltip to say. Empty when there is
// nothing to copy.
func (s *Session) EchoedName(p Rune) string {
	echoed, ok := s.Echoes(p)
	if !ok || echoed.Record == p.Record {
		return ""
	}
	return echoed.Name
}

// rememberRune records what was just spent, so a chimera has something to copy.
//
// **Never the chimera**, which is never spent — see BecomeLastRune. The guard stays so a chimera
// reaching here by some other door cannot leave the next one copying it into nothing.
func (s *Session) rememberRune(resolved Rune) {
	if resolved.Target == RuneChimera {
		return
	}
	s.lastRune = resolved.Record
}

// LastRune is the record a chimera would copy, by key. Empty on a run that has spent none.
func (s *Session) LastRune() string { return s.lastRune }
