package screens

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

// **The search the other four tutorial tests send you to.**
//
// Every one of them ends "if this goes red, the fix is a new seed, not a weaker check" — and none
// of them said how to find one. That gap cost an afternoon the first time a seed had to be
// replaced, because the constraints live in four files and a candidate has to satisfy all of them
// at once: the hand the lesson describes, the blow that wounds without killing, the creature's
// heaviest blow being the one the shield eats, and a shop the taught purse can afford.
//
// It is a test rather than a tool for one reason: `dealShelf`, `shopRNG` and `shopPotions` are
// unexported, and exporting three shop internals so a tool could shop is a worse trade than a test
// that skips. `tools/seeds` is the counterpart for an ordinary deck seed and cannot answer this
// one — it tallies concepts and the tutorial matches on element.
//
//	SEEDSEARCH=1 go test ./internal/screens -run TestFindATutorialSeed -v -timeout 900s
//	SEEDSEARCH=exact ...   # only candidates keeping the cards the steps name
//
// **It prints candidates and asserts nothing.** Take one, put it in `data/tutorial.json`, and run
// the suite — the four real tests are what confirm it, and two of the constraints (the wound, the
// blocked blow) are deliberately not duplicated here so this cannot drift from them.
//
// **Candidates keeping the taught cards are marked, and are the ones to want.** A seed dealing a
// different four is legal and means rewriting the steps that name a card — `the-ap-bar` says Jab,
// `the-shield-card` says Brace, `the-broken-card` says Drain — so a matching set is the difference
// between changing one string and re-authoring the lesson.
func TestFindATutorialSeed(t *testing.T) {
	if os.Getenv("SEEDSEARCH") == "" {
		t.Skip("a search, not a check: set SEEDSEARCH=1 to run it")
	}

	// What the taught run walks into the shop holding, and what it is told to buy. Kept in step
	// with TestTheTutorialsShopCanAffordWhatTheLessonDemands by hand — this one only proposes.
	const purse, relicsToBuy, wantSet = 13, 2, 4

	script := tutorial.Load()
	taught := map[string]bool{"Jab": true, "Brace": true, "Thrust": true, "Bash": true}

	me, ok := data.LoadDuelists()["Fighter1"]
	if !ok {
		t.Fatal("no duelist record Fighter1")
	}

	draught := 0
	for _, p := range shopPotions() {
		if p.Effect == session.PotionDMG {
			draught = p.Price
		}
	}
	if draught == 0 {
		t.Fatal("no potion in the catalogue adds DMG, and the lesson tells the player to drink one")
	}

	// **`exact` is the mode to reach for, and the plain search will not find one for you.**
	// A candidate keeping the taught four is roughly one in a few thousand seeds where any
	// candidate is one in fifty thousand, so a run capped at the first twenty hits reports twenty
	// seeds that all mean re-authoring the lesson. Ask for the ones you can actually use.
	exact := os.Getenv("SEEDSEARCH") == "exact"

	found := 0
	for v := int64(0); v < seeds.Space && found < 20; v++ {
		hand := OpeningCards(seeds.ForFight(v, seeds.PlayerDeck, 0))

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
		// The set is the taught size, the opening click lands inside it, and no second set of the
		// same size makes the lit square ambiguous.
		if bestN != wantSet || hand[0].Element != best {
			continue
		}
		ambiguous := false
		for e, n := range counts {
			if e != best && n >= bestN {
				ambiguous = true
			}
		}
		if ambiguous {
			continue
		}

		// Affordable inside one turn, and holding exactly the one shield the lesson explains.
		cost, shields := 0, 0
		names := map[string]bool{}
		for _, c := range hand {
			if c.Element != best {
				continue
			}
			concept := combat.ConceptOf(c.Concept)
			cost += c.Cost()
			names[concept.Label] = true
			if concept.Verb != combat.VerbAttack {
				shields++
			}
		}
		if shields != 1 || cost > me.Actions {
			continue
		}

		// **The shop the lesson actually reaches**, which is the one after the taught fight — the
		// shelf is drawn per fight, so reading it at fight zero is reading a different shop.
		gs := &state.GlobalState{
			ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight, RunSeed: v,
		}
		gs.Run = session.New(nil)
		gs.Run.Teach(script)
		gs.Run.WonFight(48, 60)

		shelf := dealShelf(gs, shopRNG(gs, seeds.ShopStock))
		if len(shelf) < relicsToBuy {
			continue
		}
		prices := make([]int, 0, len(shelf))
		for _, item := range shelf {
			p, _ := session.RelicPrice(item.key)
			prices = append(prices, p)
		}
		sort.Ints(prices)
		spend := draught
		for _, p := range prices[:relicsToBuy] {
			spend += p
		}
		if spend > purse {
			continue
		}

		mark := " "
		if len(names) == len(taught) {
			mark = "*"
			for n := range names {
				if !taught[n] {
					mark = " "
				}
			}
		}
		if exact && mark != "*" {
			continue
		}
		found++
		t.Logf("%s %s  %-9v %d AP  shelf %v, spends %d of %d  %v",
			mark, seeds.Code(v), best, cost, prices, spend, purse, sorted(names))
	}

	if found == 0 {
		t.Fatalf("no seed below %d satisfies the hand and the shop at once", seeds.Space)
	}
	fmt.Printf("\n%d candidate seed(s); * keeps the cards the steps name\n", found)
}

func sorted(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
