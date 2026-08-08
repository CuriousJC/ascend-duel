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
var seedCatalog = []namedSeed{
	// 2xGather 1xGuard 3xStrike 1xDodge 1xRiposte. Three Strikes is exactly a 6 AP budget, so
	// the Flurry is clickable on round one with nothing set up first.
	{"strike-flurry", 15, "three or more Strikes: a Strike Flurry that can be clicked"},

	// 1xGather 5xStrike 2xRiposte. The *cards* for an Onslaught; five Strikes is 10 AP, so it
	// still takes two Gather rounds to afford, and the hand holds only one Gather. Playing it
	// means gathering, discarding into more Gathers, and holding the Strikes — which is the
	// engine-building the combo is meant to be gated behind, so it is the right shape of hard.
	{"strike-onslaught", 1569, "five Strikes: a Strike Onslaught, which takes a whole turn"},

	// 2xStrike 4xHeavy 1xDodge 1xRiposte.
	{"heavy-flurry", 5, "three or more Heavys: 12 AP, unaffordable, but the cards are there"},

	// 2xGather 1xStrike 2xHeavy 3xRiposte. Gather + Riposte + Strike is 6 AP exactly, which
	// puts a white, a blue and a red verb chip in one round.
	{"all-categories", 1, "a Gather, an attack and a defend: every verb chip in one round"},

	// 1xGuard 2xStrike 1xHeavy 3xDodge 1xRiposte.
	{"defensive", 3, "two Dodges and a Riposte: negation against a swarm"},
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
		out = append(out, c.actionCard.action)
	}
	return out
}
