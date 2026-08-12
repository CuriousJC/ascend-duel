package screens

import (
	"math/rand"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// **Named deck seeds, so a specific opening hand can be asked for by name.**
//
// The shuffle is deterministic from `deckSeed`, which means a seed *is* an opening hand.
// Before this, checking anything that needed a particular hand — a combo you can actually
// click, a round with all three card categories in it — meant relaunching and hoping, or
// reaching past the input layer to force the queue. A named seed replaces both with a number.
//
// **These are found, not chosen.** `go run ./tools/seeds` searches for hands matching each
// description and prints what it finds, including the hand each catalogued seed currently
// deals. **Re-run it after touching `startingDeck` or `handSize`** — a seed is a fact about a
// particular deck, and changing the deck silently invalidates every number below. The tool
// reports a catalogue entry whose hand no longer matches its description rather than leaving
// it to be discovered by a demo that quietly stopped testing what it claimed to.
//
// A slice rather than a map on purpose: `tools/seeds` prints these in order, and Go randomises
// map iteration. See the determinism rules in CLAUDE.md.
type namedSeed struct {
	name string
	seed int64
	want string // what the tool searched for, and what the entry is re-checked against
}

// Every number here came out of `tools/seeds`, and two of them replaced guesses the re-check
// rejected on its first run — which is the tool earning its place immediately.
//
// **Every entry was replaced on 2026-08-08**, when the deck went from 30 cards to 60. All five
// re-checks failed at once, which is exactly what the re-check exists for: a seed is a fact
// about one particular deck, and doubling the deck silently re-deals every catalogued hand. The
// old numbers are left in the comments below as a record of what they used to deal.
var seedCatalog = []namedSeed{
	// 1xGather 1xGuard 2xRitual 3xStrike 1xDodge. Three Strikes is exactly a 6 AP budget, so
	// the Flurry is clickable on round one with nothing set up first. *(was seed 15)*
	{"strike-flurry", 38, "three or more Strikes: a Strike Flurry that can be clicked"},

	// 1xGuard 2xRitual 5xStrike. The *cards* for an Onslaught; five Strikes is 10 AP, so it
	// still needs a Gather round to afford — and this hand holds two Rituals instead, which is
	// +10 AP next round and the fastest route to it the deck can deal.
	//
	// **This took 600,000 seeds to find, against 20,000 before.** Five Strikes in a hand of
	// eight is 5 of 5 copies in a 60-card deck: about 1 hand in 98,000, where the 30-card deck
	// made it 1 in 3,000. The Onslaught family is now effectively undrawable without
	// deckbuilding, which is arguably correct for the game's most absurd combo — but it is a
	// consequence of the deck doubling that nobody chose, and it is worth knowing before
	// tuning the combo itself. *(was seed 1569)*
	{"strike-onslaught", 54198, "five Strikes: a Strike Onslaught, which takes a whole turn"},

	// 1xGather 1xSift 2xJab 3xHeavy 1xMirror. *(was seed 5)*
	{"heavy-flurry", 1, "three or more Heavys: 12 AP, unaffordable, but the cards are there"},

	// 1xGather 1xRitual 1xJab 1xHeavy 1xBrace 1xDodge 1xRiposte 1xMirror — eight distinct
	// concepts, which is the widest hand in the catalogue and the best look at the new cards.
	// Gather + Jab + Dodge is 4 AP, so a white, a red and a blue verb chip fit in one round with
	// room to spare. *(was seed 1)*
	{"all-categories", 6, "a Gather, an attack and a defend: every verb chip in one round"},

	// 1xGather 1xRitual 1xFeint 1xHeavy 2xDodge 2xRiposte. *(was seed 3)*
	{"defensive", 44, "two Dodges and a Riposte: negation against a swarm"},
}

// seedFor looks a catalogued seed up by name. **It panics on an unknown name**, which is the
// right failure for a table compiled into the binary: every caller is a constant in this
// repository, so a miss is a typo that should never reach a running game rather than a
// condition to handle.
func seedFor(name string) int64 {
	for _, s := range seedCatalog {
		if s.name == name {
			return s.seed
		}
	}
	panic("screens: no seed named " + name)
}

// Seeds is the catalogue as (name, seed, description) triples, for tools/seeds to print and
// re-check. Exported because that tool lives outside this package; nothing in the game reads
// it.
func Seeds() (names []string, values []int64, wants []string) {
	for _, s := range seedCatalog {
		names = append(names, s.name)
		values = append(values, s.seed)
		wants = append(wants, s.want)
	}
	return names, values, wants
}

// OpeningHand deals the hand a seed produces, using **the real deck and the real shuffle** —
// it builds a bare CombatScene and calls the same resetDeck the game does, so the tool cannot
// drift from what a launch actually deals. Nothing here touches Ebitengine, which is what
// lets a windowless tool call it.
func OpeningHand(seed int64) []combat.ActionKind {
	var s CombatScene
	s.rng = rand.New(rand.NewSource(seed))
	s.resetDeck()

	out := make([]combat.ActionKind, 0, len(s.hand))
	for _, c := range s.hand {
		out = append(out, c.actionCard.Action)
	}
	return out
}
