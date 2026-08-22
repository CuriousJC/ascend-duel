package session

// **Putting a run somewhere it did not play its way to.**
//
// This exists for one caller: the `scenario` fixture, which opens the game on a chosen screen so a
// between-fights scene can be looked at without playing three duels to reach it. See
// internal/scenario, and CLAUDE.md on why that package is behind a build tag.
//
// **It is deliberately not a general setter.** Everything else that moves a run is a *rule* —
// WonFight, Advance, Buy — and a screen that could put the run wherever it liked is the
// hardcoded-successor problem with an extra step. One method, documented as a fixture, is the
// smallest shape that does the job and the easiest to delete.

// JumpTo drops the run into a room with a given purse and a given amount of life left, and decides
// the spoils that room's win would have paid.
//
// **It goes through the same `spoilsFor` the real win does**, rather than making up a payout. A
// fixture that pays differently from the game is a fixture that shows the wrong screen.
func (s *Session) JumpTo(fight, vitae, lifeLeft int) {
	if fight < 0 {
		fight = 0
	}
	if vitae > 0 {
		s.vitae = vitae
	}
	s.fight = fight
	s.lifeLeft = lifeLeft

	// The spoils are the ones the room *below* this one paid, because the run is standing after a
	// win — the same off-by-one WonFight has, where the counter moves after the payout is decided.
	behind := fight - 1
	if behind < 0 {
		behind = 0
	}
	was := s.fight
	s.fight = behind
	s.spoils = s.spoilsFor(lifeLeft)
	s.fight = was
}
