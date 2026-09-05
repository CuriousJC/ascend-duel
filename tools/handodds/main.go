// handodds says how often each rung of the hand ladder can actually be built, which is what the
// multipliers in data/hands.json are priced against.
//
//	go run ./tools/handodds
//	go run ./tools/handodds -n 200000        # a quicker, noisier sample
//	go run ./tools/handodds -ap 8            # a turn holding cost discounts, priced as a bigger budget
//
// **It exists for the reason tools/seeds does: the numbers are facts about one particular deck.**
// A hand's rarity is a function of how many cards share a concept, a form and an element, so
// changing `data/duelist_cards.json`, the hand size or the action budget silently invalidates every
// figure the ladder was tuned against. Re-run it after touching any of those, exactly as the seed
// catalogue is re-checked.
//
// **The arithmetic is tools/hands, shared with tools/handsheet** *(2026-09-05)*, which prints the
// same figures beside the cards each rung is made of. This command is the tuning view: every rung
// in one table, with the axes kept apart, plus the two flags. See that package for what
// "reachable" means and what the model leaves out.
//
// It is not a test. There is nothing here to assert — the output is a table to tune against, and
// the tuning is a judgement call about how much a rarer hand should pay.
package main

import (
	"flag"
	"fmt"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/tools/hands"
)

func main() {
	trials := flag.Int("n", hands.Trials, "how many hands to deal")
	// **-ap models a turn holding discounts, and it is a proxy rather than a stat.** Nothing in
	// the game raises the budget; what a run buys is cheaper *cards* — a `card-cost` ring at -1,
	// a worm's `CostDelta`, Atrophy demoting a card a rung — and a turn playing five cards each a
	// point cheaper spends what a five-point-larger budget would. It is exact for a discount every
	// card in the set qualifies for and generous for one that only some do, so a number taken from
	// a run of it is a number about a *kitted* turn and has to be labelled as one.
	//
	// **It is not needed to reach any rung of the ladder**, which was the reason it was added and
	// stopped being true. Every rung has a build a 6 AP round can pay for — an elemental five of a
	// kind is Jab, Cut, Bash, Ward and Thrust in one colour, 6 AP exactly, since the defences carry
	// a colour and join hands. What `-ap` answers now is how much a discount widens the *set* of
	// hands that reach a rung, which is a different and still useful question.
	budget := flag.Int("ap", hands.Budget(), "action points a turn may spend")
	flag.Parse()

	deck := hands.StartingDeck()
	table := hands.Measure(deck, *budget, hands.HandSize, *trials)

	fmt.Printf("%d attack cards of %d, hand of %d, %d AP, %d cards to a turn, %d hands dealt\n\n",
		hands.Attacks(deck), len(deck), table.HandSize, table.Budget, hands.MaxCards, table.Trials)
	fmt.Printf("cards sharing a value: %d per concept, %d per form, %d per element\n\n",
		hands.PerValue(deck, combat.AxisConcept),
		hands.PerValue(deck, combat.AxisForm),
		hands.PerValue(deck, combat.AxisElement))

	fmt.Printf("%-28s %6s %10s %10s %12s\n", "hand", "pays", "reachable", "1 in", "best in hand")
	lastAxis := combat.Axis(-1)
	for _, o := range table.Rungs {
		if o.Hand.Match != lastAxis {
			fmt.Println()
			lastAxis = o.Hand.Match
		}
		fmt.Printf("%-28s %6d %8.3f%% %10.0f %11.2f%%\n",
			o.Hand.Name, o.Hand.Multiplier, o.Reachable, o.OneIn(), o.Best)
	}
	fmt.Printf("\nno built hand of any kind: %.3f%% - the turns the High Card names\n", table.Nothing)
}
