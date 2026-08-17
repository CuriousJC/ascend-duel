// Package decks turns the card lists in `data` into cards the rules understand, and holds an
// opponent's three piles while a duel is played.
//
// **It exists so the combat screen and `tools/balance` share one enemy deck** *(2026-08-11)*.
// The screen cannot own this: the balance tool plays whole duels headlessly and
// `internal/screens` links Ebitengine, which on Linux calls `glfw.Init()` from a package
// `init()` and would need a display server to run a table of numbers. Duplicating the deck
// in the tool would have been worse than either — a balance report is only worth reading if
// it is the same opponent the game fights.
//
// **No Ebitengine here, ever**, for that reason. It sits between `data` and
// `internal/combat`, importing both, which neither of them may do.
//
// The player's deck deliberately stays in `internal/screens`. Its cards are drawn on screen and
// move through a hand that can be reordered, neither of which is true of an enemy's, and pulling
// it down here would mean giving this package a screen's vocabulary to reuse a loop.
//
// **This package is where enemy concepts are registered** *(2026-08-16)*. Card definitions became
// data — see `combat.RegisterConcept` — and `data` may not import the rules, so something between
// the two has to hand one to the other. This is the package built for exactly that edge, and it is
// the reason every enemy deck in the game is built here rather than in the screen that draws it.
package decks

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// enemyDecks is every enemy's deck, keyed by record, built once at package init so a bad record
// fails on launch rather than mid-duel.
//
// **One deck per enemy, not one for the roster** *(2026-08-16)*. Every opponent used to draw from
// `enemy_cards.json`, twelve Attacks and twelve Heavies, and its behaviour came from a `PlanStyle`
// string picking one of four planners. Both are gone: an enemy is what it holds.
var enemyDecks = buildEnemyDecks()

// buildEnemyDecks registers every enemy's concepts and expands its deck.
//
// **It walks the roster in sorted order**, per the determinism rules in CLAUDE.md — a map range
// would assign concept IDs in whatever order Go felt like, and while nothing compares IDs across
// processes today, a registry that renumbers itself between runs is a trap laid for the save
// format.
//
// It panics on a bad record for the reason the player's deck builder does: a deck quietly missing
// cards is a balance change nobody made, and a launch failure naming the record is cheaper to fix
// than an enemy that turns out to be harmless three floors in.
func buildEnemyDecks() map[string][]combat.Card {
	records := data.LoadEnemies()
	out := make(map[string][]combat.Card, len(records))

	for _, name := range data.EnemyOrder(records) {
		rec := records[name]
		if len(rec.Cards) == 0 {
			panic(fmt.Sprintf("enemies.json: %s has no cards, so it cannot fight", name))
		}

		var deck []combat.Card
		for _, c := range rec.Cards {
			id, err := combat.RegisterConcept(name, c)
			if err != nil {
				panic("enemies.json: " + err.Error())
			}

			elements := c.Elements
			if len(elements) == 0 {
				// **Empty means basic**, which is what every enemy card is today. An enemy's colour
				// does nothing until an elemental affix attunes it — see MECHANICS.md — so writing
				// one in now would hand it a status it has no source for.
				elements = []string{combat.Basic.String()}
			}
			for _, en := range elements {
				element, ok := combat.ParseElement(en)
				if !ok {
					panic(fmt.Sprintf("enemies.json: %s.%s names unknown element %q", name, c.Label, en))
				}
				for i := 0; i < c.Copies; i++ {
					deck = append(deck, combat.Of(id, element))
				}
			}
		}
		out[name] = deck
	}
	return out
}

// EnemyCards is one opponent's deck, copied, so a caller cannot change what every future duel is
// dealt. An unknown record hands back nothing rather than panicking — the roster is walked from
// the same map, so a miss here means the caller invented a name.
func EnemyCards(record string) []combat.Card {
	return append([]combat.Card(nil), enemyDecks[record]...)
}

// EnemyRecords is every record with a deck, sorted. For a tool walking the roster.
func EnemyRecords() []string {
	out := make([]string, 0, len(enemyDecks))
	for name := range enemyDecks {
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

// EnemySeed is the *pinned* opponent shuffle, so every balance run deals the same cards.
//
// **The game no longer reads it by default.** A fight seeds the opponent's pile from the run
// seed — see `shuffleSeeds` in internal/screens — and falls back to this only while `deckSeed`
// pins the player's hand, because pinning half a duel reproduces nothing. `tools/balance` uses
// it unconditionally: a balance number that moved because the shuffle moved is not a balance
// number.
//
// **A separate stream from the player's deck, and that is the point.** CLAUDE.md's determinism
// rules name "card shuffles" as one stream; sharing one between the two sides would make the
// player's opening hand a function of how many cards the enemy happened to draw, and every
// entry in `internal/screens/seeds.go` would break the first time an enemy deck was retuned. A
// named hand has to stay a fact about the player's deck alone.
//
// It lives here rather than beside deckSeed so the game and tools/balance cannot end up
// fighting differently shuffled opponents.
const EnemySeed int64 = 20260811

// EnemyPile is one opponent's deck through a duel: a draw pile, a hand, and a discard.
//
// **The hand does NOT persist between rounds, unlike the player's — and that is a fix, not
// an oversight.** Persisting it was the first thing tried and it deadlocked: a planner only ever
// takes what it can spend, so every card it could not use stayed in hand. By round three the hand
// was seven dead cards, nothing could be drawn on top of them, and the opponent stood still for
// the rest of the duel. `tools/balance` showed it as a roster nothing could lose to.
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

// NewEnemyPile shuffles one enemy's deck and deals an opening hand.
func NewEnemyPile(record string, seed int64, handSize int) *EnemyPile {
	p := &EnemyPile{
		draw:     EnemyCards(record),
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
