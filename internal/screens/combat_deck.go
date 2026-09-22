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
	"log"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/journal"
	"github.com/curiousjc/ascend-duel/internal/scenario"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// **The piles hold `combat.Card` itself, and every screen writes that name.**
//
// The screen used to own an `combat.Card` type *and* an unexported `element`, on the honest grounds
// that neither meant anything to the rules — a card's color painted a border and `ResolveRound`
// never saw it. Elements are mechanical now, so the rules own the type and the piles hold exactly
// what the engine resolves. It was then an *alias* for a year, which made the two names one type
// and made the rule invisible at the same time.
//
// The rule is what matters: the hand, the queue and the round are one type, so a card cannot be
// converted wrongly on the way between them because it is never converted at all. Written out at
// every site, `combat.Card` says that; a local synonym said the opposite at a glance.

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
// starting list, which was the same number every time and stopped being true the moment an essence
// could remove a card. The piles are conserved — nothing is created or destroyed mid-fight — so
// their sum is the run's deck size for as long as the fight lasts.
func (s *CombatScene) deckSize() int {
	return len(s.deck) + len(s.discard) + len(s.hand)
}

// deckSeedName pins every launch to one cataloged opening hand. **Empty means unpinned**,
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
// A var rather than a const so a build-tagged file can point it at a catalog entry in
// `init` — which is how the scripted demo picks its hand without the game growing a flag it
// would have to keep.
var deckSeed = pinnedDeckSeed()

// pinnedDeckSeed reads deckSeedName, treating the empty name as no pin. seedFor panics on a
// name it does not know, which is right for a typo and wrong for "no name at all", so the
// empty case is answered here rather than by adding a not-found path to the catalog.
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

	// **A discard is a choice and the plan's list did not have a word for it** *(2026-09-22)*.
	// Every kind in internal/journal was derived from a line the ledger already words, and a
	// discard writes no ledger line at all — but it takes cards out of a hand and deals others in,
	// so a retrace without it diverges on the very next turn.
	s.choices.Write(journal.Record{
		Kind:    journal.KindDiscard,
		Targets: s.selectedCardIDs(),
		Amount:  s.discardsLeft,
	})

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
			s.discard = append(s.discard, c.Card)

			// A card that was played is sitting in its seat on the table, not in its old hand
			// slot, so that is where it has to set off from. Sending it out of a slot it
			// visibly left ten seconds ago would make it jump back across the screen to be
			// thrown. The count goes with it: a seat's x is a function of how many cards the
			// row holds, and the row is cleared three lines below this.
			flight := cardFlight{
				Travel:   ui.NewTravel(0, flightTicks()),
				card:     c.Card,
				outbound: true,
				index:    i, count: leaving,
			}
			if p, ok := s.playedSeatOf(i); ok {
				flight.index, flight.count, flight.fromTable = p, len(s.Theater.resolved), true
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
	s.Theater.resolved = nil

	// Everything appended past this point was dealt, which is what makes the drawn cards
	// identifiable without drawHand having to report them.
	dealt := len(s.hand)
	pile := s.drawHand()

	// **The survivors close the row up first, and the sort comes after the deal**
	// *(owner's call, 2026-09-15)*. This used to sort before anything was animated so a dealt card
	// flew straight to the slot it would end up in — one journey per card, and a hand that never
	// showed the player what it was dealt. The row now shuts up to the left, the new cards arrive
	// on the right in pile order, the flip cascade plays over them, and the whole row sorts last.
	// See combat_deal.go.
	for to := 0; to < dealt; to++ {
		was := keptFrom[to]
		if was == to && leaving == len(s.hand) {
			continue
		}
		s.addSlide(ui.CardSlide{
			Travel:    ui.NewTravel(0, ui.SlideTicks()),
			Card:      s.hand[to].Card,
			FromLift:  selectedLift(s.hand[to].selected),
			ToLift:    selectedLift(s.hand[to].selected),
			FromIndex: was, FromCount: leaving,
			ToIndex: to, ToCount: len(s.hand),
		})
	}

	s.startDeal(dealt, pile)

	// **The queue goes with the cards it named.** This is the moment the played or discarded cards
	// leave the hand, and the queue is a list of exactly those cards — so one that outlived them
	// was a round still being described by a hand that no longer exists.
	//
	// **It was `finishDeal`'s until now, and that was a whole deal too late** *(2026-09-18)*. The
	// sort moved to the end of the deal on 2026-09-15 and took the queue's rebuild with it, so for
	// the length of every deal `previewBlow` re-derived the hand that had just been played and
	// `drawPlannedHand` painted it back into its planning seat — the name reappearing on the table
	// after the creature had already answered it. The doc above has always said this function
	// leaves the queue correct; this is what makes that true again.
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
// whole of what makes an essence stick: the piles are rebuilt on every `Init`, and `Init` is how the
// next fight starts, so a deck edit held anywhere on this scene would be thrown away between
// rooms.
//
// **A nil run means the starting deck**, which is not a fallback for the game — `main` always
// builds one — but for the callers that deal a hand without a run around it: `OpeningHand` in
// seeds.go, `tools/seeds`, and the flight tests. A named seed is a fact about the *starting*
// deck, so those must not read a run even when one exists.
func (s *CombatScene) resetDeck(run *session.Session) {
	s.run = run
	s.deck = s.deck[:0]
	if run != nil {
		// **FightDeck rather than Deck**: this is the `deck-built` moment, so a demoting relic steps
		// what is dealt without touching what the run owns. The element flips are *not* here — they
		// fire per card in drawHand — so this pile holds cards in the colors the run owns. See
		// session.FightDeck and combat.MomentCardDrawn.
		s.deck = append(s.deck, run.FightDeck()...)
	} else {
		s.deck = append(s.deck, session.StartingDeck()...)
	}

	s.discard = s.discard[:0]
	s.hand = s.hand[:0]

	s.shuffleDeck()
	pile := s.drawHand()

	// **A scenario's hand is dealt over the shuffle, not through it** — the draw pile is left
	// exactly as it was, so the second hand of the fight is a normal one and the fixture is only
	// the opening. Compiled out of every normal build; see internal/scenario.
	//
	// **Only a fixture that actually plugs a hand loses the deal** *(2026-09-15)*. This dropped the
	// pile for every scenario launch, `plugHand` being a no-op on an empty list — so a fixture with
	// a `Deck` and no `Hand` opened with eight cards already standing there and the deal never ran
	// at all. Which is most of them, and it is the one path a fixture is *for*.
	if scenario.Active() && s.plugHand(run, scenario.Hand()) {
		// The cards the fixture put in the hand are not the cards that came off the pile, so the
		// deal has nothing to walk. It sorts them and stands down.
		pile = nil
	}

	// **An opening hand is dealt exactly as a refilled one is** *(owner's call, 2026-09-15)* — out
	// of the pile, through the flip cascade, and sorted last. It used to be filled and sorted here
	// with nothing on screen, which made the first hand of a run the one hand in the game that
	// simply appeared. `startDeal` sorts at the end of the sequence, so this no longer sorts.
	s.startDeal(0, pile)
}

// plugHand replaces the hand with an authored one. **A debug seat and nothing else** — it is
// called from one place, behind `scenario.Active()`, and it is a no-op in every build that has not
// asked for the tag.
//
// **The replaced cards go nowhere.** They are not discarded and not put back: the draw pile is
// untouched, so the round after this one refills from a deck that never knew. A fixture is meant
// to be one hand, not a rewritten deck.
// **It reports whether it replaced anything**, which the caller needs: a hand the fixture wrote did
// not come off the draw pile, so there is no deal to play over it — and a fixture that plugged
// *nothing* must still get the ordinary one.
//
// **Every card it seats is one the run owns** *(owner's call, 2026-09-17)*. It seated the values
// `internal/scenario` built with `combat.Of`, which leaves `Card.ID` at zero — so a plugged hand
// was a row of cards the run had never heard of, and `Session.CardByID` could not find any of them.
// Everything aimed by identity therefore refused it: **every rune in the sack drew dim**, and two
// plugged cards both carrying the zero identity also tripped `CanApplyRune`'s named-twice guard.
// Silently, because a rune that cannot be spent looks exactly like a rune whose selection is wrong.
//
// **Claimed from the deck first, minted only if the deck has not got it.** A fixture almost always
// names cards its own `Deck` (or the authored one) already holds, and claiming keeps the run the
// size its `Deck` line says — which is what the deck panel's total is read against when a copying
// rune is the thing being looked at. A card the run does not hold is added to it, because the
// alternative is seating an identity-less card again and the fixture asked for that card.
//
// **Matched on everything but the identity**, so a fixture naming a card with riders claims a run
// card carrying the same riders rather than a bare one wearing the same name.
func (s *CombatScene) plugHand(run *session.Session, cards []combat.Card) bool {
	if len(cards) == 0 {
		return false
	}

	s.hand = s.hand[:0]
	if run == nil {
		// No run to own them — the windowless callers, which never plug a hand. Seated as they
		// arrive rather than dropped, so this stays a hand rather than an empty row.
		for _, c := range cards {
			s.hand = append(s.hand, paletteCard{Card: c})
		}
		return true
	}

	owned := run.Deck()
	claimed := make(map[int]bool, len(cards))
	for _, want := range cards {
		seat := -1
		for i, have := range owned {
			if claimed[have.ID] {
				continue
			}
			probe := have
			probe.ID = want.ID
			if probe == want {
				seat = i
				break
			}
		}
		if seat < 0 {
			// **Minted rather than refused**, and said out loud: a fixture naming a card outside
			// its own deck has grown the run by one, which is worth knowing when the deck panel's
			// total is the thing being read.
			run.Add(want)
			owned = run.Deck()
			seat = len(owned) - 1
			log.Printf("scenario hand: the run did not hold %s, so it does now — the deck is %d",
				owned[seat], run.Size())
		}
		claimed[owned[seat].ID] = true
		s.hand = append(s.hand, paletteCard{Card: owned[seat]})
	}
	return true
}

// shuffleDeck shuffles the draw pile using the scene's own source. Never rand.Shuffle,
// which draws from the global source and would make the deal unreproducible.
func (s *CombatScene) shuffleDeck() {
	s.rng.Shuffle(len(s.deck), func(i, j int) {
		s.deck[i], s.deck[j] = s.deck[j], s.deck[i]
	})
}

// handTarget is how many cards this round's refill draws to.
//
// **It is a constant, and the function survives the thing that made it one** *(2026-08-31)*. A
// Plan card used to bank a wider hand for the round after, so this read `handSize + BonusDraw`;
// nothing widens a hand any more. It stays a function rather than becoming `handSize` at every
// call site because "how many cards does a refill draw to" is a question with one answer and one
// place to change it, and the relic grammar has a seat for a card-drawn moment already.
func (s *CombatScene) handTarget() int { return handSize }

// drawHand fills the hand up to handTarget, reshuffling the discard back into the draw pile
// when it runs dry. A hand can come up short only if every card the player owns is already
// in it, which cannot happen with a deck larger than the hand.
//
// **This is the `card-drawn` moment** *(2026-08-24)*. A flip relic recolors a card here, one card
// at a time on its way out of the pile, which is what its text has always said — "every earth card
// is dealt as a fire card". It used to recolor the whole fight deck in one pass at `deck-built`;
// the cards dealt are the same either way, since a flip is unconditional over an element.
//
// **The invariant that makes it safe: the draw pile holds cards as the run owns them.** A flip
// reads a card's original color, so a discarded ice-that-was-lightning card folded back into the
// pile and drawn again would be read as ice — and a second flip keyed on ice would fire, chaining
// two relics into a deck of one color, which is exactly what firing at `deck-built` prevented for
// free. `restoreToDeck` is what pays for it now.
// **It reports the cards as the pile held them**, parallel to the cards it appended. The hand gets
// the finished card, which is the one every rule reads; the deal gets the face the cascade starts
// from, so a flip can be watched happening rather than having already happened. See
// combat_deal.go.
func (s *CombatScene) drawHand() []combat.Card {
	var pile []combat.Card

	for len(s.hand) < s.handTarget() {
		if len(s.deck) == 0 {
			if len(s.discard) == 0 {
				return pile
			}
			// **Put back the way they were found.** See restoreToDeck.
			for _, c := range s.discard {
				s.deck = append(s.deck, s.restoreToDeck(c))
			}
			s.discard = s.discard[:0]
			s.shuffleDeck()
		}

		last := len(s.deck) - 1
		raw := s.deck[last]
		s.hand = append(s.hand, paletteCard{Card: s.drawnAs(raw)})
		s.deck = s.deck[:last]
		pile = append(pile, raw)
	}
	return pile
}

// drawnAs is the card as it is dealt into the hand: the worn flips applied, or the card untouched
// when the scene has no run behind it.
//
// **The card it is handed is a draw-pile card**, which the invariant above says is a card in the
// color the run owns — so the flip reads the original, as combat.FlipElement requires.
func (s *CombatScene) drawnAs(c combat.Card) combat.Card {
	if s.run == nil {
		return c
	}
	return s.run.DrawnAs(c)
}

// restoreToDeck undoes a draw: a card going back into the draw pile is put back in the color the
// run owns it in, so the next flip that reads it reads the original rather than the last flip's
// answer.
//
// **It restores the color and nothing else.** The concept is deliberately left as it is: a
// demotion is a `deck-built` rule, applied once as this pile was built, and a card that came out of
// this pile as a 2 AP Thrust is a 2 AP Thrust for the whole fight. Only the flip fires per draw, so
// only the flip has anything to undo.
//
// **A card the run has never heard of is left alone.** That covers a scene dealt with no run and a
// card whose original an essence has since eaten; drawing what is actually in hand is the honest
// answer to both.
func (s *CombatScene) restoreToDeck(c combat.Card) combat.Card {
	if s.run == nil {
		return c
	}
	owned, ok := s.run.CardByID(c.ID)
	if !ok {
		return c
	}
	c.Element = owned.Element
	return c
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
// duelist whose relics price the faces.
//
// **The hand and the discard are one list here**, which is the panel's own split — see
// deckpanel.go. Both are cards you cannot draw, and merging them is what lets a card stay where it
// is and simply dim when it is played.
func (s *CombatScene) fightContents() ui.DeckContents {
	d := ui.DeckContents{
		Draw:  s.deck,
		Spent: make([]combat.Card, 0, len(s.discard)+len(s.hand)),

		// **The run, so the panel can find a card's original.** A card in the hand or the discard
		// has been through a draw and holds only the color a flip relic made it; the ID is the way
		// back to what the run owns. See deckContents.run.
		Run:     s.run,
		InFight: true,

		Holder: s.fighter.Duelist,
	}
	d.Spent = append(d.Spent, s.discard...)
	for _, c := range s.hand {
		d.Spent = append(d.Spent, c.Card)
	}
	return d
}

// fightHands is the ladder as a fight sees it: **every card you own, whichever pile it is in**,
// and the duelist holding them.
//
// **The piles are merged rather than reported**, which is the opposite of fightContents and is
// deliberate. The deck panel is about where your cards are right now; the hands panel is about
// what your cards can build, and a rung that has gone out of reach because three of its cards are
// in the discard this round is not a fact about your deck. The piles are conserved, so their sum
// is the run's deck for as long as the fight lasts.
func (s *CombatScene) fightHands() ui.HandsContents {
	deck := make([]combat.Card, 0, s.deckSize())
	deck = append(deck, s.deck...)
	deck = append(deck, s.discard...)
	for _, c := range s.hand {
		deck = append(deck, c.Card)
	}
	return ui.HandsContents{Deck: deck, Holder: s.fighter.Duelist, Plays: ui.RunPlays(s.run)}
}
