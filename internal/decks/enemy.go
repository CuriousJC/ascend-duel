package decks

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// enemyConcepts is every record's cards as registered concepts, keyed by record, built once at
// package init so a bad record fails on launch rather than mid-duel.
//
// **A concept, not a card.** A creature's colour is its floor's rather than its own, so the deck
// it actually fights with is not known until a fight is built — see EnemyCards, which is the one
// place a concept and an element become cards.
var enemyConcepts = buildEnemyConcepts()

// conceptCopies is one registered concept and how many of it a deck holds.
type conceptCopies struct {
	id     combat.ConceptID
	copies int
}

// buildEnemyConcepts registers every motif record's cards.
//
// **It walks the roster in sorted order**, per the determinism rules in CLAUDE.md — a map range
// would assign concept IDs in whatever order Go felt like, and while nothing compares IDs across
// processes today, a registry that renumbers itself between runs is a trap laid for the save
// format.
//
// It panics on a bad record for the reason the player's deck builder does: a deck quietly missing
// cards is a balance change nobody made, and a launch failure naming the record is cheaper to fix
// than a creature that turns out to be harmless three floors in.
func buildEnemyConcepts() map[string][]conceptCopies {
	motifs := data.LoadMotifs()

	out := map[string][]conceptCopies{}
	for _, motif := range data.MotifOrder(motifs) {
		for _, rec := range motifs[motif].Records {
			if len(rec.Cards) == 0 {
				panic(fmt.Sprintf("motifs: %s has no cards, so it cannot fight", rec.Record))
			}
			var list []conceptCopies
			for _, c := range rec.Cards {
				id, err := combat.RegisterConcept(rec.Record, c)
				if err != nil {
					panic("motifs: " + err.Error())
				}
				list = append(list, conceptCopies{id: id, copies: c.Copies})
			}
			out[rec.Record] = list
		}
	}
	return out
}

// EnemyCards is one opponent's deck as it is dealt on a given floor: its own concepts, every one
// of them in the floor's element.
//
// **The element is the floor's, not the card's.** A creature is instantiated as one element and
// its whole deck takes it, the same way a duelist's Jab is a concept that ships in five colours —
// so there is no element anywhere in data/motifs, and a card cannot carry one of its own.
//
// An unnamed element deals a basic deck, which is what a fixture with no floor behind it gets.
// An unknown record hands back nothing rather than panicking — the roster is walked from the same
// map, so a miss here means the caller invented a name.
func EnemyCards(record, element string) []combat.Card {
	el := combat.Basic
	if element != "" {
		parsed, ok := combat.ParseElement(element)
		if !ok {
			panic("motifs: " + record + " cannot be dealt as " + element)
		}
		el = parsed
	}

	var out []combat.Card
	for _, c := range enemyConcepts[record] {
		for i := 0; i < c.copies; i++ {
			out = append(out, combat.Of(c.id, el))
		}
	}
	return out
}

// EnemyRecords is every record with a deck, sorted. For a tool walking the roster.
func EnemyRecords() []string {
	out := make([]string, 0, len(enemyConcepts))
	for name := range enemyConcepts {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// EnemyHandSize is how many cards an opponent is dealt to plan from.
//
// **Bigger than MaxActions on purpose**, which is 5 across the roster today. A hand exactly
// the size of the action cap would make the deck a formality — every card drawn would be
// played — and the point of dealing a hand is that a planner has to choose from it, and
// sometimes cannot find what it wants.
const EnemyHandSize = 7

// **This package declares no seed.** `EnemySeed` lived here until 2026-08-17, only because a
// headless caller could not import `internal/screens` and the pinned opponent shuffle had to sit
// somewhere both could reach. That pressure is what produced `internal/seeds`, so it is
// `seeds.EnemyDeckPin` now — and a package whose job is turning card data into rules types is
// better for owning no randomness at all. `NewEnemyPile` still takes its seed as a parameter,
// which is what kept the move to one line.

// EnemyPile is one opponent's deck through a duel: a draw pile, a hand, and a discard.
//
// **The hand does NOT persist between rounds, unlike the player's — and that is a fix, not
// an oversight.** Persisting it was the first thing tried and it deadlocked: a planner only ever
// takes what it can spend, so every card it could not use stayed in hand. By round three the hand
// was seven dead cards, nothing could be drawn on top of them, and the opponent stood still for
// the rest of the duel — a roster nothing could lose to.
//
// The player's hand may persist because Discard exists — since 2026-08-06 it is the *only*
// way an unwanted card leaves a hand, which is exactly the lever an enemy does not have.
// Persistence without it is not a harder deck, it is a lock. If enemies ever get a discard
// of their own, this is the decision to revisit.
type EnemyPile struct {
	draw    []combat.Card
	hand    []combat.Card
	discard []combat.Card

	handSize int

	// rng is explicit and per-pile rather than the math/rand package-level functions, which
	// draw from a global shared with every other caller. See the determinism rules in
	// CLAUDE.md.
	rng *rand.Rand
}

// NewEnemyPile shuffles one enemy's deck, dealt in the floor's element, and deals an opening hand.
func NewEnemyPile(record, element string, seed int64, handSize int) *EnemyPile {
	p := &EnemyPile{
		draw:     EnemyCards(record, element),
		handSize: handSize,
		rng:      rand.New(rand.NewSource(seed)),
	}
	p.shuffle()
	p.fill()
	return p
}

// Plan deals the opponent back up to a full hand, asks the planner what to do with it, and
// moves what it chose to the discard.
//
// **The cards leave the hand here, before the round resolves.** That mirrors the player's
// queue: what is committed is spent, whether or not a stagger later deletes it from the
// round. A card the engine refuses to play is still a card that was thrown.
func (p *EnemyPile) Plan(d combat.Duelist) []combat.Card {
	p.fill()

	plan := combat.PlanFor(d, p.hand)
	for _, c := range plan {
		p.spend(c)
	}

	// What was not played goes back too. See the type's comment: without a discard of its
	// own, an opponent that kept its leftovers would fill its hand with cards no plan reads
	// and stop acting entirely.
	p.discard = append(p.discard, p.hand...)
	p.hand = nil

	return plan
}

// Counts reports the three piles, for a trace dump or a debug line.
func (p *EnemyPile) Counts() (draw, hand, discard int) {
	return len(p.draw), len(p.hand), len(p.discard)
}

// spend moves one played card out of the hand and into the discard. It matches the whole card,
// element included, so a plan that took a fire Jab does not discard the basic one beside it.
func (p *EnemyPile) spend(card combat.Card) {
	for i, c := range p.hand {
		if c == card {
			p.hand = append(p.hand[:i], p.hand[i+1:]...)
			p.discard = append(p.discard, card)
			return
		}
	}
}

// fill tops the hand back up, folding the discard back in when the draw pile runs out —
// the same rule the player's deck follows.
func (p *EnemyPile) fill() {
	for len(p.hand) < p.handSize {
		if len(p.draw) == 0 {
			if len(p.discard) == 0 {
				return // an empty deck list; nothing to deal, and not worth crashing over
			}
			p.draw, p.discard = p.discard, nil
			p.shuffle()
		}
		p.hand = append(p.hand, p.draw[0])
		p.draw = p.draw[1:]
	}
}

func (p *EnemyPile) shuffle() {
	p.rng.Shuffle(len(p.draw), func(i, j int) {
		p.draw[i], p.draw[j] = p.draw[j], p.draw[i]
	})
}
