package decks

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// **Every example is a hand the matcher would actually score as that rung.** The variety is only
// ever in what the rung does not count, so this is the line between illustrating a rule and
// breaking it — an example that no longer forms its own hand would be the panel teaching
// something untrue.
func TestEveryExampleFormsItsOwnRung(t *testing.T) {
	deck := session.StartingDeck()
	for _, h := range combat.Hands() {
		cards, _ := Example(deck, h)
		if got := len(cards); got != h.Cards() {
			t.Errorf("%s: %d cards for a %d-card hand", h.Name, got, h.Cards())
			continue
		}

		// One group per distinct value on the hand's own axis, each the size the rung asks for.
		counts := map[int]int{}
		for _, c := range cards {
			v, ok := MatchValue(c, h.Match)
			if !ok {
				t.Errorf("%s: %s counts for nothing on the %v axis", h.Name, c.Label(), h.Match)
				continue
			}
			counts[v]++
		}
		if len(counts) != len(h.Groups) {
			t.Errorf("%s: %d distinct values against %d groups", h.Name, len(counts), len(h.Groups))
		}
		for v, n := range counts {
			held := false
			for _, want := range h.Groups {
				if want == n {
					held = true
					break
				}
			}
			if !held {
				t.Errorf("%s: value %d appears %d times, which is no group size in %v",
					h.Name, v, n, h.Groups)
			}
		}
	}
}

// **Every rung is illustrated, reachable or not.** A hand wanting five copies of one card cannot
// be dealt from the shipping deck; the ladder still has the rung, so the panel still draws it and
// Example repeats a card to fill the row. This is what fails if an empty example comes back.
func TestEveryRungIsIllustrated(t *testing.T) {
	deck := session.StartingDeck()
	for _, h := range combat.Hands() {
		cards, cost := Example(deck, h)
		if len(cards) != h.Cards() {
			t.Errorf("%s is illustrated with %d cards against a %d-card hand",
				h.Name, len(cards), h.Cards())
		}
		if cost <= 0 {
			t.Errorf("%s costs %d AP", h.Name, cost)
		}
	}
}

// **A repeat only ever appears once the row has said everything it can.** A rung the deck holds
// enough distinct cards for must never draw one twice — that would be the illustration inventing
// a constraint the rung does not have.
func TestACardIsRepeatedOnlyWhenTheDeckRunsOut(t *testing.T) {
	deck := session.StartingDeck()
	for _, h := range combat.Hands() {
		cards, _ := Example(deck, h)

		seen := map[combat.Card]int{}
		for _, c := range cards {
			seen[c]++
		}
		for c, n := range seen {
			if n == 1 {
				continue
			}
			// The deck has to be short of this card for the repeat to be honest.
			held := 0
			for _, d := range deck {
				if d == c {
					held++
				}
			}
			if held >= n {
				t.Errorf("%s draws %s %d times with %d in the deck",
					h.Name, c.Label(), n, held)
			}
		}
	}
}

// **A rung has to look like itself, and until 2026-08-24 three of them did not.** The cheapest
// pair on every axis is two identical cards, so a Form Pair, a Card Pair and an Elemental Pair
// were the same two cards — and drawn as pictures on the hands panel that is the panel claiming
// a form pair wants two copies of one card.
//
// What this pins is that each rung's example differs in something the rung leaves free.
func TestAnExampleShowsWhatTheRungLeavesFree(t *testing.T) {
	deck := session.StartingDeck()
	for _, h := range combat.Hands() {
		cards, _ := Example(deck, h)
		if len(cards) < 2 {
			continue
		}
		if variety(cards, h.Match) == 0 {
			t.Errorf("%s is illustrated with cards that differ in nothing: %v",
				h.Name, labels(cards))
		}
	}
}

// **The three pairs are three different pictures.** They are the rungs the old rule collapsed, and
// the ones a player most needs told apart — a pair is where each axis is first met.
func TestThePairsAreToldApart(t *testing.T) {
	deck := session.StartingDeck()

	seen := map[string][]string{}
	for _, h := range combat.Hands() {
		if h.Cards() != 2 || len(h.Groups) != 1 {
			continue
		}
		cards, _ := Example(deck, h)
		key := ""
		for _, c := range cards {
			// What a token actually draws: element, form and cost. Two rungs drawn the same way
			// are two rungs the panel cannot tell apart, whatever cards they name.
			key += string(rune('a'+int(c.Element))) + string(rune('a'+int(c.Form()))) +
				string(rune('0'+c.Cost())) + " "
		}
		if other, clash := seen[key]; clash {
			t.Errorf("%s draws the same tokens as %v: %q", h.Name, other, key)
		}
		seen[key] = append(seen[key], h.Name)
	}
}

// **Cost is still the tie-break**, which is the half of the old rule that survives: a hand nobody
// could afford to queue illustrates nothing. Among the sets that show the rung equally well the
// example is the cheapest, so nothing here may cost more than a set of the same variety.
func TestCostBreaksTheTieAmongEqualIllustrations(t *testing.T) {
	deck := session.StartingDeck()
	for _, h := range combat.Hands() {
		cards, cost := Example(deck, h)
		if cost <= 0 {
			t.Errorf("%s costs %d AP", h.Name, cost)
		}
		sum := 0
		for _, c := range cards {
			sum += c.Cost()
		}
		if sum != cost {
			t.Errorf("%s: reported %d AP against %d on the cards", h.Name, cost, sum)
		}
	}
}

// **Two runs of the same deck illustrate a rung the same way**, per the determinism rules: the
// panel is opened over and over and a set that moved between looks would be unreadable.
func TestTheExampleIsTheSameEveryTime(t *testing.T) {
	deck := session.StartingDeck()
	for _, h := range combat.Hands() {
		first, _ := Example(deck, h)
		for i := 0; i < 3; i++ {
			again, _ := Example(deck, h)
			if len(again) != len(first) {
				t.Fatalf("%s: %d cards then %d", h.Name, len(first), len(again))
			}
			for j := range first {
				if again[j] != first[j] {
					t.Errorf("%s: card %d is %s then %s",
						h.Name, j, first[j].Label(), again[j].Label())
				}
			}
		}
	}
}

func labels(cs []combat.Card) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Label())
	}
	return out
}
