package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

// **The check CLAUDE.md said did not exist, and the bug it would have caught.**
//
// The lesson tells the player four of their cards burn the same colour, to take all four, and that
// the blow ends the fight. Every one of those is a promise about one particular deal off the
// shipping deck against one particular creature — and for a while nothing held them: the script was
// pinned to a seed by a scenario, and when the profile became the real trigger the lesson ran on
// whatever the clock had rolled, describing a set the hand did not contain.
//
// **Pinning the seed in `data/tutorial.json` is only half the fix.** The other half is this. The
// seed is a fact about `duelist_cards.json`, `startingDeck` and `handSize`; the kill is a fact about
// those *plus* the hand ladder, the duelist's DMG and the creature's HP. Six files can each move for
// their own good reasons and quietly turn Bob into a liar, and nothing else fails when they do.
//
// **If this goes red, the fix is a new seed, not a weaker check.**
func TestTheTutorialsSeedDealsTheHandTheLessonDescribes(t *testing.T) {
	const wantSet = 4

	script := tutorial.Load()
	if script.Seed == "" {
		t.Fatal("the tutorial script must pin a seed, or the lesson describes a hand it did not deal")
	}
	if script.Match != tutorial.MatchElement {
		t.Fatalf("this test is written against the elemental lesson; the script matches on %v", script.Match)
	}
	runSeed, err := seeds.Parse(script.Seed)
	if err != nil {
		t.Fatalf("tutorial.json seed %q: %v", script.Seed, err)
	}

	// The same derivation a launch uses: the player's deck stream, at fight zero. See
	// CombatScene.shuffleSeeds, which is the policy this has to agree with.
	hand := OpeningCards(seeds.ForFight(runSeed, seeds.PlayerDeck, 0))

	counts := map[combat.Element]int{}
	for _, c := range hand {
		counts[c.Element]++
	}
	best, bestN := combat.Basic, 0
	for _, c := range hand {
		if n := counts[c.Element]; n > bestN {
			best, bestN = c.Element, n
		}
	}

	if bestN != wantSet {
		t.Fatalf("run %s deals a largest elemental set of %d (%v), and the lesson promises %d\n%s",
			script.Seed, bestN, best, wantSet, describe(hand))
	}

	// **The set has to be the only one of its size.** The step points at "the matching cards" and
	// locks everything else, so a second set just as big would make the square and the sentence
	// disagree about which cards are meant.
	for e, n := range counts {
		if e != best && n >= bestN {
			t.Errorf("run %s deals two sets of %d — %v and %v — so the lit square is ambiguous",
				script.Seed, n, best, e)
		}
	}

	// **The first card the lesson asks for has to be one of the set.** Step `the-hand` queues
	// `first-card`, and if that card is not in the set the player is holding a stray when
	// `take-them-all` arrives — which breaks the budget and the hand at once.
	if hand[0].Element != best {
		t.Errorf("run %s deals %v first and the taught set is %v; the opening click queues a stray",
			script.Seed, hand[0].Element, best)
	}

	// **And the whole set has to be affordable and lethal**, which is the promise the lesson ends
	// on. The arithmetic is combat's: the cards' own damage, multiplied by the hand.
	me, ok := data.LoadDuelists()["Fighter1"]
	if !ok {
		t.Fatal("no duelist record Fighter1")
	}
	foe, ok := data.LoadEnemies()[script.Enemy]
	if !ok {
		t.Fatalf("tutorial.json names enemy %q, which is in no roster", script.Enemy)
	}

	cost, sum := 0, 0
	for _, c := range hand {
		if c.Element != best {
			continue
		}
		concept := combat.ConceptOf(c.Concept)
		cost += c.Cost()
		if concept.Verb == combat.VerbAttack {
			sum += concept.Amount
		} else {
			// A plan in the taught set is not fatal, but it is worth saying out loud: it joins the
			// hand and brings no damage, so the lesson would be asking the player to play a card
			// that does nothing while telling them it multiplies the blow.
			t.Errorf("the taught set holds %s, which is a %v and deals nothing",
				concept.Label, concept.Verb)
		}
	}

	// **The four cards are named, so this test and combat's cannot drift.**
	// `TestTheTutorialsBlowKillsTheTutorialsEnemy` writes the taught turn out by hand — it has to,
	// since the shuffle needs a scene and that package must stay window-free — and this is what
	// holds its literal against what the seed actually deals.
	want := map[string]bool{"Cut": true, "Bash": true, "Thrust": true, "Strike": true}
	for _, c := range hand {
		if c.Element != best {
			continue
		}
		label := combat.ConceptOf(c.Concept).Label
		if !want[label] {
			t.Errorf("the taught set holds %s, which internal/combat's copy of this turn does not; "+
				"update TestTheTutorialsBlowKillsTheTutorialsEnemy to match", label)
		}
		delete(want, label)
	}
	for label := range want {
		t.Errorf("internal/combat's taught turn expects %s and the seed does not deal it", label)
	}

	if cost > me.Actions {
		t.Errorf("the taught set costs %d AP and the duelist has %d, so it cannot be played at all",
			cost, me.Actions)
	}
	if blow := sum * me.DMG / 100 * multiplierFor(t, bestN) / 100; blow < foe.HP {
		t.Errorf("the taught blow is %d against %s's %d HP, and the lesson promises a kill in one",
			blow, script.Enemy, foe.HP)
	}
}

// multiplierFor is what the ladder pays an elemental set of n, read off the catalogue rather than
// written down here — the whole point being that a rebalanced ladder shows up as this test failing.
func multiplierFor(t *testing.T, n int) int {
	t.Helper()
	for _, h := range combat.Hands() {
		if h.Match == combat.AxisElement && len(h.Groups) == 1 && h.Groups[0] == n {
			return h.Multiplier
		}
	}
	t.Fatalf("the ladder has no elemental set of %d", n)
	return 0
}

func describe(hand []combat.Card) string {
	out := "hand:"
	for _, c := range hand {
		out += " " + combat.ConceptOf(c.Concept).Label + "/" + c.Element.String()
	}
	return out
}
