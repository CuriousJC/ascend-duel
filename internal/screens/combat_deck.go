package screens

// The cards and the deck they come out of: what a card is made of, what the starting deck
// holds, the four piles and the movement between them.
//
// Split out of combat.go on 2026-08-07. The deck lives on the scene rather than in
// internal/combat on purpose — see CLAUDE.md on determinism. That is what keeps the rules
// package pure, testable and free of a shuffle, and it is why this file exists at all.
//
// **The overlay that shows the deck left on 2026-08-22**, for deckpanel.go, because three screens
// want it. What stayed is the piles — a fact about a fight — and `fightContents`, which is the
// one place that says how a fight's three piles map onto the two the panel draws.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/scenario"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/trace"
)

// actionCard is one instance in the piles: a concept and the element it is made of.
//
// **It is an alias for `combat.Card` as of 2026-08-12, not a struct of its own.** The screen
// used to own this *and* an unexported `element` type, on the honest grounds that neither meant
// anything to the rules — a card's colour painted a border and `ResolveRound` never saw it.
// Elements are mechanical now, so the rules own the type and the piles hold exactly what the
// engine resolves.
//
// That is what an alias is for here: the hand, the queue and the round are one type, so a card
// cannot be converted wrongly on the way between them because it is never converted at all. The
// name stays because the screen is full of it, and `actionCard` still says what a pile holds
// better than `Card` does next to `paletteCard`, `pileEntry` and `cardFlight`.
//
// **What went with the old type: `elementColors` and `element.color()`.** They were the surface
// colours from when a card *was* a coloured rectangle, and nothing had called either since the
// border took over the element on 2026-08-09 and `internal/cards` took over the drawing. The
// live colour table is `cards.BorderOf`.
type actionCard = combat.Card

// The hand drawn from the deck each round.
//
// Eight from five on 2026-08-04. Eight cards do not fit the screen side by side, which is
// what the overlap in handPitch is for — the hand is expected to go past eight sometimes.
//
// **Eight was sized against a 30-card deck and the deck is now 48.** That is 17% of the deck
// in hand where it used to be 27%, so consistency fell without this number moving.
//
// **It is the base rather than the size as of 2026-08-15**, because a Plan card widens one hand
// by two. See handTarget, which is what the refill actually reads; this is what it widens from.
// Eight is left alone deliberately: `discardsPerRound` and now Plan are the levers meant to
// answer draw variance, and moving all three at once would leave no way to tell which did the
// work. A brand growing hand size is the recorded permanent version.
const handSize = 8

// deckSize is how many cards the player owns right now, counting all three piles.
//
// **It counts rather than reading a constant** *(2026-08-17)*. It used to total the authored
// starting list, which was the same number every time and stopped being true the moment a worm
// could remove a card. The piles are conserved — nothing is created or destroyed mid-fight — so
// their sum is the run's deck size for as long as the fight lasts.
func (s *CombatScene) deckSize() int {
	return len(s.deck) + len(s.discard) + len(s.hand)
}

// deckSeedName pins every launch to one catalogued opening hand. **Empty means unpinned**,
// which is the default: the shuffles are rolled from the run seed instead and each fight deals
// a fresh deck. Set it to a name while working on something that needs a particular hand — the
// names and what each deals are in seeds.go, and `go run ./tools/seeds` prints them.
//
// Naming it rather than writing a bare number is the whole point: `deckSeed = 15` says nothing
// about why 15, and the next person to change it has no way to know they have just stopped
// dealing the hand a demo depended on.
const deckSeedName = ""

// deckSeed is the pinned shuffle, or **zero for "roll one from the run seed"** — the same
// toggle shape as `fixedRunSeed` in main.go, and the counterpart of it: that one pins which
// enemies a run meets, this one pins which cards it draws. Pinning makes a layout problem
// reproducible; unpinned is what an actual run looks like.
//
// When it is non-zero the *opponent's* shuffle is pinned too, to `seeds.EnemyDeckPin` — see
// shuffleSeeds. Half a pinned duel is not reproducible, and the scripted demo sets only this
// one.
//
// A var rather than a const so a build-tagged file can point it at a catalogue entry in
// `init` — which is how the scripted demo picks its hand without the game growing a flag it
// would have to keep.
var deckSeed = pinnedDeckSeed()

// pinnedDeckSeed reads deckSeedName, treating the empty name as no pin. seedFor panics on a
// name it does not know, which is right for a typo and wrong for "no name at all", so the
// empty case is answered here rather than by adding a not-found path to the catalogue.
func pinnedDeckSeed() int64 {
	if deckSeedName == "" {
		return 0
	}
	return seedFor(deckSeedName)
}

// discardSelected throws the selected cards away: they leave the hand for the discard
// pile, and the hand is dealt back up to size from the draw pile.
//
// Selection does double duty here — it is both "queue this for the round" and "this is
// the one I mean" — so throwing a card out costs the action points it was holding until
// the moment it leaves. That is a consequence of overloading selection rather than a
// designed cost; see TODO.md.
func (s *CombatScene) discardSelected() {
	if !s.planning() || s.discardsLeft <= 0 {
		return
	}
	s.discardsLeft--
	s.spendSelected()

	trace.Logf("input", "discard pressed, %d left this round, hand now %s",
		s.discardsLeft, handLabel(s.hand))
}

// spendSelected moves every selected card to the discard pile, deals the hand back up to
// size, and rebuilds the queue from what is left.
//
// **Selected is the only thing that leaves a hand.** Both ways a card can go — thrown away
// by Discard, or played by DUEL! — are the same movement over the same predicate, so they
// are one function rather than two that have to be kept in agreement.
// **The cards move now and the animation catches up.** Every flight this raises is a ghost
// of a card that has already gone — the hand, the piles and the queue are correct the moment
// this returns, so planning(), the action-point budget and the row's own layout never have
// to know about a card that is neither in the hand nor out of it. Holding a card in place
// until its animation finished would have put that question into all three.
func (s *CombatScene) spendSelected() {
	// The row the discarded cards are leaving. Captured before the filter, because a flight
	// starts from the slot a card had in the hand that existed when it was thrown — and that
	// hand stops existing on the next line. See slotAt.
	leaving := len(s.hand)

	// Filtering in place over the hand's own array. Safe because kept never runs ahead of
	// the read cursor, and it keeps the surviving cards in the order the player left them.
	//
	// keptFrom records where each survivor was standing, because a card that stays in the hand
	// still *moves*: the cards around it have gone and the row closes up under it. That is a
	// slide, and it needs the slot the card is leaving as much as a discard does.
	kept := s.hand[:0]
	var keptFrom []int
	for i, c := range s.hand {
		if c.selected {
			s.discard = append(s.discard, c.actionCard)

			// A card that was played is sitting in its seat on the table, not in its old hand
			// slot, so that is where it has to set off from. Sending it out of a slot it
			// visibly left ten seconds ago would make it jump back across the screen to be
			// thrown. The count goes with it: a seat's x is a function of how many cards the
			// row holds, and the row is cleared three lines below this.
			flight := cardFlight{
				travel:   newTravel(0, flightTicks),
				card:     c.actionCard,
				outbound: true,
				index:    i, count: leaving,
			}
			if p, ok := s.playedSeatOf(i); ok {
				flight.index, flight.count, flight.fromTable = p, len(s.theatre.resolved), true
				flight.split = s.playedSplit()
			}
			s.addFlight(flight)
			continue
		}
		kept = append(kept, c)
		keptFrom = append(keptFrom, i)
	}
	s.hand = kept

	// The round's history goes with the cards it was made of. Cleared here rather than at the
	// start of the next round because this is the moment those cards actually leave, and a
	// pile outliving them would be a picture of a round that is over.
	s.theatre.resolved = nil

	// Everything appended past this point was dealt, which is what makes the drawn cards
	// identifiable without drawHand having to report them.
	dealt := len(s.hand)
	s.drawHand()

	// **The sort runs before anything is animated, not after.** A dealt card lands in the slot
	// the sort gives it rather than on the right-hand end, so it flies to where it will
	// actually be sitting — and `inboundTo`, which blanks a slot that is still filling, is
	// looking at that same index.
	//
	// The permutation is what survives the rearrangement. `order[to]` is where the card now at
	// `to` came from: past `dealt` it came out of the draw pile and flies in, below it the card
	// was already in the hand and slides across the row from wherever it was standing.
	order := s.sortHand()

	staggered := 0
	for to, from := range order {
		if from >= dealt {
			s.addFlight(cardFlight{
				travel: newTravel(staggered*flightStaggerPer, flightTicks),
				card:   s.hand[to].actionCard,
				index:  to, count: len(s.hand),
			})
			staggered++
			continue
		}

		// A survivor. It has moved if its slot changed or if the row it is standing in did —
		// eight cards centred is not the same place as six centred, so a card that kept its
		// index still has ground to cover.
		was := keptFrom[from]
		if was == to && leaving == len(s.hand) {
			continue
		}
		s.addSlide(handSlide{
			travel:    newTravel(0, slideTicks),
			card:      s.hand[to].actionCard,
			selected:  s.hand[to].selected,
			fromIndex: was, fromCount: leaving,
			toIndex: to, toCount: len(s.hand),
		})
	}

	s.syncQueue()
}

// toggleDeck shows or hides the deck overlay.
func (s *CombatScene) toggleDeck() {
	s.showDeck = !s.showDeck
	trace.Logf("input", "deck overlay %v (draw %d, discard %d)",
		s.showDeck, len(s.deck), len(s.discard))
}

// resetDeck fills the draw pile from the run, shuffles it, empties the discard and deals an
// opening hand.
//
// **The deck comes from the run now, not from the authored list** *(2026-08-17)*. That is the
// whole of what makes a worm stick: the piles are rebuilt on every `Init`, and `Init` is how the
// next fight starts, so a deck edit held anywhere on this scene would be thrown away between
// rooms.
//
// **A nil run means the starting deck**, which is not a fallback for the game — `main` always
// builds one — but for the callers that deal a hand without a run around it: `OpeningHand` in
// seeds.go, `tools/seeds`, and the flight tests. A named seed is a fact about the *starting*
// deck, so those must not read a run even when one exists.
func (s *CombatScene) resetDeck(run *session.Session) {
	s.deck = s.deck[:0]
	if run != nil {
		// **FightDeck rather than Deck**: this is the `deck-built` moment, so a flip ring recolours
		// what is dealt without touching what the run owns. See session.FightDeck.
		s.deck = append(s.deck, run.FightDeck()...)
	} else {
		s.deck = append(s.deck, session.StartingDeck()...)
	}

	s.discard = s.discard[:0]
	s.hand = s.hand[:0]

	s.shuffleDeck()
	s.drawHand()

	// **A scenario's hand is dealt over the shuffle, not through it** — the draw pile is left
	// exactly as it was, so the second hand of the fight is a normal one and the fixture is only
	// the opening. Compiled out of every normal build; see internal/scenario.
	if scenario.Active() {
		s.plugHand(scenario.Hand())
	}

	// An opening hand arrives sorted, exactly as a refilled one does. The default is cost, so
	// this is true of a scene that has never had a sort button pressed on it.
	s.sortHand()
}

// plugHand replaces the hand with an authored one. **A debug seat and nothing else** — it is
// called from one place, behind `scenario.Active()`, and it is a no-op in every build that has not
// asked for the tag.
//
// **The replaced cards go nowhere.** They are not discarded and not put back: the draw pile is
// untouched, so the round after this one refills from a deck that never knew. A fixture is meant
// to be one hand, not a rewritten deck.
func (s *CombatScene) plugHand(cards []combat.Card) {
	if len(cards) == 0 {
		return
	}

	s.hand = s.hand[:0]
	for _, c := range cards {
		s.hand = append(s.hand, paletteCard{actionCard: c})
	}
}

// shuffleDeck shuffles the draw pile using the scene's own source. Never rand.Shuffle,
// which draws from the global source and would make the deal unreproducible.
func (s *CombatScene) shuffleDeck() {
	s.rng.Shuffle(len(s.deck), func(i, j int) {
		s.deck[i], s.deck[j] = s.deck[j], s.deck[i]
	})
}

// handTarget is how many cards this round's refill draws to: the usual size, plus whatever a
// Plan banked in the round before.
//
// **This is where a Plan actually draws** *(2026-08-15)*. `internal/combat` has no deck, so a
// Plan records `BonusDraw` on the duelist and the size of the next hand is the whole of what it
// bought. It has to be the refill target rather than an immediate two cards: the hand refills to
// a fixed size at the round boundary anyway, so cards handed over mid-round would be two fewer
// drawn at the boundary and the card would do nothing at all.
//
// The engine assigns `BonusDraw` rather than adding to it, so it lasts exactly one round — a
// hand of ten, and then a hand of eight again.
//
// **A scene with no fighter draws the usual eight.** `OpeningHand` and the flight tests build a
// bare CombatScene to deal a hand without a duel around it, which is exactly the case a Plan
// cannot have applied in.
func (s *CombatScene) handTarget() int {
	if s.fighter == nil {
		return handSize
	}
	return handSize + s.fighter.BonusDraw
}

// drawHand fills the hand up to handTarget, reshuffling the discard back into the draw pile
// when it runs dry. A hand can come up short only if every card the player owns is already
// in it, which cannot happen with a deck larger than the hand.
func (s *CombatScene) drawHand() {
	for len(s.hand) < s.handTarget() {
		if len(s.deck) == 0 {
			if len(s.discard) == 0 {
				return
			}
			s.deck = append(s.deck, s.discard...)
			s.discard = s.discard[:0]
			s.shuffleDeck()
		}

		last := len(s.deck) - 1
		s.hand = append(s.hand, paletteCard{actionCard: s.deck[last]})
		s.deck = s.deck[:last]
	}
}

// endRoundHand spends what was played and refills. **Only the cards that were actually
// played leave** — anything still sitting unselected in the hand stays exactly where it is,
// and the draw tops the hand back up to size.
//
// **Do not go back to discarding the whole hand each round.** The argument for it is that a
// hand kept back would let a plan be prepared once and repeated — but the *queue* already
// empties every round, so no plan repeats by default, and clearing the hand as well produces
// a hand you cannot build on: cards you deliberately held are taken away for having been
// held, so the only way to keep anything is to play it. Refilling only what was used is what
// makes a hand something you shape across rounds rather than a fresh deal you react to.
//
// It also gives Discard a real job. A card you never want now sits in your hand until you
// throw it out, so the discard button is how you clear it rather than a shortcut for
// something the round boundary was going to do anyway.
func (s *CombatScene) endRoundHand() {
	s.spendSelected()
	s.discardsLeft = discardsPerRound
}

// fightContents is the deck as a fight sees it: what is left to draw, what is spoken for, and the
// duelist whose rings price the faces.
//
// **The hand and the discard are one list here**, which is the panel's own split — see
// deckpanel.go. Both are cards you cannot draw, and merging them is what lets a card stay where it
// is and simply dim when it is played.
func (s *CombatScene) fightContents() deckContents {
	d := deckContents{
		draw:   s.deck,
		spent:  make([]combat.Card, 0, len(s.discard)+len(s.hand)),
		holder: s.fighter.Duelist,
		counts: fmt.Sprintf("draw %d  ·  discard %d  ·  %d in hand  ·  %d owned",
			len(s.deck), len(s.discard), len(s.hand), s.deckSize()),
		hint: "Deck again to close",
	}
	d.spent = append(d.spent, s.discard...)
	for _, c := range s.hand {
		d.spent = append(d.spent, c.actionCard)
	}
	return d
}
