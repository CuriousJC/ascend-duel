package screens

import (
	"math/rand"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// **Named deck seeds, so a specific opening hand can be asked for by name.**
//
// The shuffle is deterministic from `deckSeed`, which means a seed *is* an opening hand.
// Before this, checking anything that needed a particular hand — a hand you can actually
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

// Every number here came out of `tools/seeds`.
//
// **Every entry has been replaced three times**: on 2026-08-08 when the deck went from 30 cards to
// 60, on 2026-08-15 when it was rebuilt as three attack forms plus the defences, and again the same
// day when the drab attacks were cut and the deck fell to 48. Every re-check failed at once each
// time, which is exactly what the re-check exists for — a seed is a fact about one particular
// deck, and changing the deck silently re-deals every catalogued hand.
//
// **Arcane took the deck back to 60 on 2026-08-25 and only two entries fell over**, which is worth
// knowing rather than surprising: a fifth colour adds a card to every concept without changing what
// a concept *is*, so the hands described in terms of concepts — three Strikes, four Strikes — were
// re-dealt to the same shape by a different shuffle. What broke was the two that wanted a specific
// combination of concepts. **A Card Five of a Kind is dealable for the first time**, at one hand in
// about 22,000, because five copies of a concept now exist; there is no catalogued seed for it and
// the default 20,000-seed search is too short to expect one.

var seedCatalog = []namedSeed{
	// 1xThrust 1xLunge 1xCut 1xCleave 1xBash 3xStrike. Three Strikes is 6 AP, exactly the opening
	// budget, so the Three of a Kind is clickable on round one with nothing set up first — and being
	// three colours it lands three statuses with it, which is what the hand and its colours look
	// like when they arrive together.
	{"three-strikes", 56, "three or more Strikes: a Three of a Kind that can be clicked"},

	// 1xThrust 1xCut 1xSlash 1xBash 4xStrike. **The biggest hand the deck can deal**: four copies is
	// every Strike there is, so this is the top of the ladder. Eight AP, so it needs a banked round,
	// and it draws four of the five colours — four copies of a concept are four different elements.
	{"four-strikes", 904, "four Strikes: a Four of a Kind, the top of the ladder"},

	// 1xThrust 1xSlash 2xCleave 3xSmash 1xWard. The same shape one form over and at the top of
	// the ladder: three Smashes is 9 AP and cannot be paid for out of an opening budget, so this is
	// the hand that shows a hand you can see and cannot afford.
	{"three-smashes", 152, "three or more Smashes: 9 AP, unaffordable, but the cards are there"},

	// 1xJab 1xThrust 2xSlash 3xStrike 1xGuard. Both phases in one round, which is what the two
	// categories have to be told apart by: a defend card and an attack, so a blue "defends" and a
	// red "attacks" appear in the same feed.
	{"both-verbs", 1, "a defend card and an attack: both phases in one round"},

	// 1xJab 1xCut 1xSlash 1xStrike 1xSmash 1xWard 1xBrace 1xGuard — eight distinct
	// concepts, which is the widest hand in the catalogue and the best look at the deck. It holds
	// all three defences, which is what the demo wants: six shields in one turn, and the pip row on
	// the duelist card full. Ward and Brace and Guard together are 6 AP, so the whole defend
	// vocabulary is one round.
	{"all-shields", 6, "a Ward, a Brace and a Guard: the whole defend vocabulary in hand"},
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
func OpeningHand(seed int64) []combat.ConceptID {
	out := make([]combat.ConceptID, 0, handSize)
	for _, c := range OpeningCards(seed) {
		out = append(out, c.Concept)
	}
	return out
}

// OpeningCards is the same deal as whole cards, so a caller can see the colours as well as the
// concepts.
//
// **The tutorial's seed is chosen on the element axis**, which the concept list above cannot
// answer — see TestTheTutorialsSeedDealsTheHandTheLessonDescribes. Both go through one `resetDeck`
// so a tool and a launch cannot deal differently.
func OpeningCards(seed int64) []combat.Card {
	var s CombatScene
	s.rng = rand.New(rand.NewSource(seed))
	// A nil run: a named seed is a fact about the *starting* deck, so the catalogue must not
	// read whatever a run has been altered into.
	s.resetDeck(nil)

	out := make([]combat.Card, 0, len(s.hand))
	for _, c := range s.hand {
		out = append(out, c.actionCard)
	}
	return out
}
