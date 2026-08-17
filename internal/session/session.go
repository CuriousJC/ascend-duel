// Package session holds what belongs to a *run* rather than to a fight or a screen.
//
// **It is the hole every run-level feature has been blocked on.** Rings cannot be bought, the
// deck cannot be altered and vitae cannot be spent for one reason: nothing survives a fight.
// `CombatScene` is rebuilt on every entry — `Init` is how the next fight starts — so anything
// kept there is thrown away between rooms. This is where a run keeps its things.
//
// **No Ebitengine, ever**, and no screen state. What lands here is what more than one scene
// needs and what has to outlive a fight: the deck today, the rings and the purse next. If a
// field is only read by one screen, it belongs on that screen.
//
// The package sits below `screens` and above `combat`, and `state.GlobalState` carries a
// pointer to it — which is why `state` transitively imports `combat` as of 2026-08-17. That
// reverses a line in CLAUDE.md written to stop *screen* state leaking into global state; a run
// is not screen state, and it is exactly what `ActiveScreen` and `NewScreen` already sit beside.
package session

import "github.com/curiousjc/ascend-duel/internal/combat"

// startingVitae is what a run opens with. It was a constant on the combat screen, reset on every
// visit, which is exactly what "run-level state living on a scene" looks like.
const startingVitae = 5

// Session is one run: everything the player is carrying up the tower.
//
// **It is deliberately not persisted.** Two runs from the same seed may end up holding
// different decks, because a deck edit is a *choice* rather than something derived from the
// seed — the replay story is a seed plus a choice log. See the `randomness` skill.
type Session struct {
	// deck is every card the player owns, in no particular order, including the ones currently
	// sitting in a pile on the combat screen. **The piles are copies of this**, dealt fresh at
	// the start of each fight; this is the list they are dealt from.
	//
	// Unexported so it cannot be appended to from a screen. Everything that changes it goes
	// through a method, which is what keeps an index handed out by `Deck()` meaningful for as
	// long as a caller holds it.
	deck []combat.Card

	// fight is how many rooms in the run has got: zero on the first fight, incremented on a win.
	//
	// **It lives here rather than on the combat screen because two scenes read it** — the screen
	// picks the opponent and scales it, and the post-battle screen seeds its offer from it. It
	// used to be `CombatScene.fightIndex`, which survived `Init` and was invisible to everything
	// else. The screen keeps a copy per visit for its draw paths; this is the authority.
	fight int

	// vitae is the purse. **Run-level and nothing spends it yet** — the shop is the scene that
	// will — but it is awarded now, by the post-battle screen, which is what made it stop being a
	// constant on the combat screen.
	vitae int
}

// New starts a run from a deck list — `startingDeck`, in practice, expanded to one entry per
// card. The slice is copied, so the caller's starting list cannot be edited by a worm.
func New(deck []combat.Card) *Session {
	s := &Session{deck: make([]combat.Card, len(deck)), vitae: startingVitae}
	copy(s.deck, deck)
	return s
}

// Deck is every card the player owns, as a copy.
//
// **A copy, for the reason `decks.EnemyCards` hands one back**: anything that sorted or
// shuffled the result would otherwise be reordering what every future fight is dealt, and the
// damage would outlive whatever did it.
func (s *Session) Deck() []combat.Card {
	out := make([]combat.Card, len(s.deck))
	copy(out, s.deck)
	return out
}

// Size is how many cards the run holds. The deck thins as worms remove cards, so this is not a
// constant and nothing should treat 48 as one.
func (s *Session) Size() int { return len(s.deck) }

// Card is one entry by index, and reports whether the index exists.
func (s *Session) Card(i int) (combat.Card, bool) {
	if i < 0 || i >= len(s.deck) {
		return combat.Card{}, false
	}
	return s.deck[i], true
}

// Remove takes a card out of the run for good.
//
// **Indices shift, and that is why a caller may not hold two of them across a call.** The
// alteration screen offers a hand, the player picks one card, and the offer is done — one
// action against one index. If that ever becomes several actions against one offer, the offer
// has to be re-resolved after each, or carry something stabler than a position.
func (s *Session) Remove(i int) bool {
	if i < 0 || i >= len(s.deck) {
		return false
	}
	s.deck = append(s.deck[:i], s.deck[i+1:]...)
	return true
}

// SetElement recolours a card. The concept is untouched: a worm varies a card the game already
// defines rather than inventing one, so what changes is which colour it counts as in a mix and
// which status it can apply.
func (s *Session) SetElement(i int, e combat.Element) bool {
	if i < 0 || i >= len(s.deck) {
		return false
	}
	s.deck[i].Element = e
	return true
}

// Vitae is what the run is carrying.
func (s *Session) Vitae() int { return s.vitae }

// AddVitae puts some in the purse. **Never negative** — spending is the shop's business and it
// will want its own method, so that the one place a purse can go down is the one place that has to
// check it can.
func (s *Session) AddVitae(n int) {
	if n <= 0 {
		return
	}
	s.vitae += n
}

// Fight is how far up the tower the run has got, zero-based.
func (s *Session) Fight() int { return s.fight }

// WonFight advances to the next room. **Losing does not call this**, which is what makes a defeat
// put the same opponent back up rather than skipping past it.
func (s *Session) WonFight() { s.fight++ }

// Add puts a card into the run. Nothing offers this yet — REMOVE and MODIFY are the two worms
// that exist — but the third one named in the design is "add", and it is one line.
func (s *Session) Add(c combat.Card) {
	s.deck = append(s.deck, c)
}
