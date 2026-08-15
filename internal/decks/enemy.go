// Package decks turns a card list in `data` into cards the rules understand, and holds an
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
// **The element used to be the other half of that argument and is not any more** *(2026-08-12)*.
// It was a screen concept; it is a rule now, `combat.Element`, so both decks deal
// `combat.Card` and this package reads the colour out of its JSON like any other field.
package decks

import (
	"math/rand"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// enemyCards is what every opponent draws from, built once at package init so a bad record
// fails on launch rather than mid-duel.
//
// **One list for every enemy**, not one per record. Which of them a Warden actually plays is
// decided by its style, which is the smallest thing that makes an enemy deck real; per-enemy
// lists want a field on data.EnemyData and are the obvious next step.
var enemyCards = buildEnemyCards()

// EnemyCards is the list, copied, for anything that wants to look at it without being able
// to change what every future duel is dealt.
func EnemyCards() []combat.Card {
	return append([]combat.Card(nil), enemyCards...)
}

// buildEnemyCards turns the data records into a flat list, in file order.
//
// It panics on a bad record for exactly the reasons the player's deck builder does: an
// unknown concept, or a cost tier that disagrees with the rules, would otherwise produce a
// deck quietly missing cards — and an enemy silently short its Heavies is a balance change
// nobody made.
//
// **Elements are carried now, and today they are all basic** *(2026-08-12)*. The list used to
// multiply the copies and throw the colour away, on the grounds that an enemy card was never
// drawn on screen. It is drawn now — the table lays both queues out — and the element is a rule
// rather than a border, so the file is read as written. Every entry says `basic`, which is
// MECHANICS.md's plan working rather than a gap: an affix *transforms* an enemy's basic deck
// into an element, so a colour typed into this file would pre-empt a mechanic that does not
// exist. Nothing stops one being added the day affixes land.
func buildEnemyCards() []combat.Card {
	records := data.LoadEnemyCards()

	problems := data.CheckCostTiers("enemy_cards.json", records,
		func(concept string) (int, bool) {
			a, ok := combat.ParseAction(concept)
			if !ok {
				return 0, false
			}
			return a.Cost(), true
		},
		func(concept string) (string, bool) {
			a, ok := combat.ParseAction(concept)
			if !ok {
				return "", false
			}
			return a.Category().String(), true
		},
		func(concept string) (string, bool) {
			a, ok := combat.ParseAction(concept)
			if !ok {
				return "", false
			}
			return a.Family().String(), true
		},
	)
	if len(problems) > 0 {
		msg := "enemy_cards.json disagrees with the rules:"
		for _, p := range problems {
			msg += "\n  " + p.Error()
		}
		panic(msg)
	}

	var out []combat.Card
	for _, c := range records {
		action, ok := combat.ParseAction(c.Concept)
		if !ok {
			panic("enemy_cards.json: unknown concept " + c.Concept)
		}
		for _, name := range c.Elements {
			element, ok := combat.ParseElement(name)
			if !ok {
				panic("enemy_cards.json: " + c.Concept + " names unknown element " + name)
			}
			for i := 0; i < c.Copies; i++ {
				out = append(out, combat.Of(action, element))
			}
		}
	}
	return out
}

// EnemyHandSize is how many cards an opponent is dealt to plan from.
//
// **Bigger than MaxActions on purpose**, which is 5 across the roster today. A hand exactly
// the size of the action cap would make the deck a formality — every card drawn would be
// played — and the point of dealing a hand is that a style has to choose from it, and
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
// an oversight.** Persisting it was the first thing tried and it deadlocked: a style only
// ever takes attacks, plus a Defend or a Prepare, so every card it could not use stayed in
// hand. By round three the hand was seven dead cards, nothing could be drawn on top of them,
// and the opponent stood still for the rest of the duel. `tools/balance` showed it as a
// roster nothing could lose to.
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

// NewEnemyPile shuffles a fresh deck and deals an opening hand.
func NewEnemyPile(seed int64, handSize int) *EnemyPile {
	p := &EnemyPile{
		draw:     EnemyCards(),
		handSize: handSize,
		rng:      rand.New(rand.NewSource(seed)),
	}
	p.shuffle()
	p.fill()
	return p
}

// Plan deals the opponent back up to a full hand, asks its style what to do with it, and
// moves what it chose to the discard.
//
// **The cards leave the hand here, before the round resolves.** That mirrors the player's
// queue: what is committed is spent, whether or not a stagger later deletes it from the
// round. A card the engine refuses to play is still a card that was thrown.
func (p *EnemyPile) Plan(style combat.PlanStyle, d combat.Duelist) []combat.Card {
	p.fill()

	plan := combat.PlanFor(style, d, p.hand)
	for _, c := range plan {
		p.spend(c)
	}

	// What was not played goes back too. See the type's comment: without a discard of its
	// own, an opponent that kept its leftovers would fill its hand with cards no style reads
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
