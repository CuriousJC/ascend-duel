package screens

// **What the combat screen says about itself in a crash report.**
//
// See ui.Reporter for the contract and internal/crashlog for where this lands. The rule the whole
// file is written under: **every value below is a plain field already on the struct, or a length,
// or a comparison of two.** Nothing here calls a method of this screen's, because the screen being
// asked has just panicked and a derivation is a second crash inside the first.

// Report is this screen's account of itself.
//
// **The question it exists to answer is which of the screen's state machines disagreed with
// another.** A duel is four of them running at once, and nearly every fault this screen can raise
// is one of them holding an index into a thing another one has since changed:
//
//   - playback against the log it walks — a cursor past the end, or a round counter that does not
//     match the events being replayed;
//   - the three piles against the hand — a seat index into a row that has been re-sorted, or a card
//     that left the hand while something still pointed at it;
//   - the queue against the budget, which is where a turn resolved with cards it could not pay for
//     would show;
//   - the exits, which are what says whether the duel was still being played at all.
//
// **The cards themselves are deliberately not here.** The journal already holds every selection by
// identity and the ledger holds what the engine made of them, so a list of labels in this map would
// be a third and worse copy of both — and a report that carried the whole hand would be
// combat.Event's sparse arrays arriving by another door. What is wanted is the shape of the state,
// not its contents.
func (s *CombatScene) Report() map[string]any {
	m := map[string]any{
		// Playback. `log` is the length rather than the events: what a reader needs is whether the
		// cursor is inside it.
		"round":         s.round,
		"cursor":        s.cursor,
		"log":           len(s.log),
		"ticks":         s.ticks,
		"ledgerWritten": s.ledgerWritten,
		"ledgerClosed":  s.ledgerClosed,

		// The piles, and how much of the hand is spoken for.
		"deck":     len(s.deck),
		"hand":     len(s.hand),
		"discard":  len(s.discard),
		"selected": s.selectedCount(),

		// The round being planned, and what both sides are standing behind. The shields come off
		// the adopted end-of-round state, which is the only place they are held outside the rules.
		"queued":       len(s.fighterActions),
		"enemyQueued":  len(s.enemyActions),
		"discardsLeft": s.discardsLeft,
		"shields":      s.fighterAfter.Shields.Count(),
		"enemyShields": s.enemyAfter.Shields.Count(),

		// Which fight this is, and the way out of it. **The opponent's record key is what makes a
		// report reproducible**: it is the name a scenario fixture would write down.
		"fight":        s.fightIndex,
		"enemyElement": s.enemyElement,
		"won":          s.won,
		"died":         s.died,
		"showDeck":     s.showDeck,

		// Whether Bob is on the screen. A lesson mid-step gates input and holds the round where it
		// is, so a crash during one is a different screen from a crash during a duel — and the
		// panel's rectangle is the one plain field that says so.
		"taught": !s.tut.panel.Empty(),
	}

	// **Guarded rather than read straight**, because the opponent is a pointer and a crash early in
	// Init is a crash before there is one. It is the only value here that can be absent.
	if s.enemy != nil {
		m["enemy"] = s.enemy.Record
	}
	return m
}
