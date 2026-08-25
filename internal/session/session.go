package session

import (
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/pyramid"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

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

	// nextCardID is the counter behind every card's identity. **It only ever goes up**, so a
	// number is never handed out twice inside one run and an id belonging to a removed card is
	// never quietly reused by a card added later.
	//
	// Identity is the run's to give: the rules have no idea a run exists, and every card outside
	// one — an enemy's deck, a test literal — keeps the zero id and is none the worse for it. See
	// combat.Card.ID for what the number buys.
	nextCardID int

	// fight is how many rooms in the run has got: zero on the first fight, incremented on a win.
	//
	// **It lives here rather than on the combat screen because two scenes read it** — the screen
	// picks the opponent and scales it, and the post-battle screen seeds its offer from it. It
	// used to be `CombatScene.fightIndex`, which survived `Init` and was invisible to everything
	// else. The screen keeps a copy per visit for its draw paths; this is the authority.
	fight int

	// vitae is the purse. **Run-level**: awarded by the post-battle screen and by propagation, and
	// spent in the shop. It stopped being a constant on the combat screen on 2026-08-17 and gained
	// something to be spent on four days later.
	vitae int

	// worn is what the player is wearing, by record key, **in worn order** — which is a rule and not
	// a presentation detail: rings fire left to right and compound, so the order has to be one the
	// player can see. See ring.go.
	worn []string

	// climb is the run's fight order — who stands in each room, in the order they will be met.
	// Nil on a run built by New, which is a test's run; a real one comes from Start. See climb.go.
	climb *pyramid.Pyramid

	// phase is where in the loop the run is: the fight, the reward, the shop, the room choice.
	// See flow.go, which is the one place that moves it.
	phase Phase

	// lifeLeft is the life the fighter finished the last fight on. **Run-level because a screen
	// after the fight has to draw the duelist as they came out of it** — the reward screen puts the
	// player's card up beside their rings, and there is no combatant left to ask by then.
	lifeLeft int

	// spoils is what the last win still owes the player, decided by WonFight and paid out by the
	// post-battle screen as it narrates each part. See spoils.go.
	spoils Spoils

	// grown is each growing ring's accumulator, keyed by record. **Keyed by record rather than by
	// position**, because it is the first ring state that will have to be serialized and a position
	// would mean nothing in a save file.
	grown map[string]int

	// tutorial is the teaching run, or nil for a run nobody is being taught. See tutorial.go for
	// why a step cursor belongs to the run rather than to the screen that happens to be up.
	tutorial *tutorial.Run
}

// New starts a run from a deck list — `startingDeck`, in practice, expanded to one entry per
// card. The slice is copied, so the caller's starting list cannot be edited by a worm.
//
// **It opens wearing StartingRings**, which is empty as shipped — see ring.go, where the list and
// the reason live. A run buys its rings.
func New(deck []combat.Card) *Session {
	s := &Session{deck: make([]combat.Card, len(deck)), vitae: startingVitae, grown: map[string]int{}}
	copy(s.deck, deck)

	// **Identity is stamped here and nowhere else on the way in.** `StartingDeck()` hands over a
	// list of descriptions — four copies of a fire Strike are four equal values — and a run is
	// where they stop being interchangeable.
	for i := range s.deck {
		s.deck[i].ID = s.mintCardID()
	}

	for _, key := range StartingRings {
		s.Wear(key)
	}
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

// SpendVitae takes from the purse, and **reports whether it could**. A run cannot go into debt:
// a caller that does not check the result has bought something for free.
//
// **It is the one place a purse goes down**, which is why AddVitae refuses a negative rather than
// being the same method twice. `Buy` is its only caller.
func (s *Session) SpendVitae(n int) bool {
	if n <= 0 || n > s.vitae {
		return false
	}
	s.vitae -= n
	return true
}

// LifeLeft is the life the fighter walked out of the last fight with.
func (s *Session) LifeLeft() int { return s.lifeLeft }

// Fight is how far up the tower the run has got, zero-based.
func (s *Session) Fight() int { return s.fight }

// WonFight advances to the next room. **Losing does not call this**, which is what makes a defeat
// put the same opponent back up rather than skipping past it.
//
// **It is the `fight-won` moment**, so it is also where the win's payout is decided and where every
// growing ring takes its step. Both happen before the room counter moves, which is the order
// MECHANICS.md states: interest is on what the run walked out of the fight holding, not on what the
// win is about to pay it.
//
// **It decides the payout and pays none of it** *(2026-08-22)*. `lifeLeft` is what the fighter
// finished on — a tenth of it is part of the prize — and the three figures are frozen here and
// handed over by the post-battle screen a sentence at a time. See spoils.go.
func (s *Session) WonFight(lifeLeft int) {
	s.lifeLeft = lifeLeft
	s.spoils = s.spoilsFor(lifeLeft)
	s.growRings()
	s.fight++
}

// Add puts a card into the run. Nothing offers this yet — REMOVE and MODIFY are the two worms
// that exist — but the third one named in the design is "add", and it is one line.
//
// **The card is given a fresh identity, whatever it arrived carrying.** A caller handing over a
// copy of a card the run already owns would otherwise put two cards with one id into the deck, and
// everything that looks a card up by id would find whichever came first.
func (s *Session) Add(c combat.Card) {
	c.ID = s.mintCardID()
	s.deck = append(s.deck, c)
}

// mintCardID hands out the next identity.
func (s *Session) mintCardID() int {
	s.nextCardID++
	return s.nextCardID
}

// CardByID is the card the run owns under an identity — **what a card looked like before any ring
// touched it**, which is the question a drawn card cannot answer for itself.
//
// It reports false for an id the run has not got, which covers the two honest cases: a card with
// no identity at all (an enemy's, a test's) and a card whose original has since been eaten by a
// worm. A caller that cannot find the original should draw the card it actually has.
//
// **A linear walk, deliberately.** The deck is fifty-odd cards and this is asked while a panel is
// open, so an index would be a second structure to keep in step with `Remove` for no measurable
// gain.
func (s *Session) CardByID(id int) (combat.Card, bool) {
	if id == 0 {
		return combat.Card{}, false
	}
	for _, c := range s.deck {
		if c.ID == id {
			return c, true
		}
	}
	return combat.Card{}, false
}
