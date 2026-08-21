package combat

// The opponent's planner: what an enemy does with the hand it was dealt.
//
// **One planner, and the deck is what makes an enemy itself** *(2026-08-16)*. PlanFor scores
// every affordable combination of the hand's attacks through the same blowFor the resolver uses
// — so it finds that three cheap cards forming a Three of a Kind beat one expensive card — and
// then spends whatever budget is left on defences and banks, which is what keeps a non-attack
// card in an enemy deck from being dead content. It replaced four named behaviours chosen by a
// string on the enemy record, three of which were unreachable.
//
// **A planner may only play what it was dealt.** The shuffle that produced the hand is outside
// this package, in internal/decks, which is what keeps the rules free of randomness and of a
// clock — this file is pure arithmetic over a hand it is handed.
//
// **It is the enemy's, not the player's.** Nothing here is consulted for a person's turn; the
// player's plan is whatever they dragged into the action box.
//
// Split out of combat.go on 2026-08-21.

// PlanFor builds one round's plan out of the hand an opponent was dealt.
//
// **One planner as of 2026-08-16, and the deck is what makes an enemy itself.** There were four
// styles — brute, swarm, warden, tactician — chosen by a string on the enemy record, and they went
// with the shared enemy deck. An opponent that holds six cheap copies of one card *is* a swarm; one
// holding three expensive ones *is* a brute. The personality moved into the thing the player can
// actually read, which is what the cards do rather than which branch a switch took.
//
// Two things went wrong with styles that this cannot repeat. Three of the four were unreachable —
// the warden asked for a Defend by name and the shared list held none, so every enemy in the game
// fought as a brute. And none of them was rewritten when a turn stopped resolving several blows, so
// the shape the roster treated as weakest (a swarm's four cheap attacks, now a Barrage at 5x) had
// quietly become the strongest.
//
// **The rule is: hit as hard as this hand can, then spend what is left over.** The attack half is
// exact rather than greedy — it scores every affordable combination through `blowFor`, the same
// function that resolves the round, so the plan the opponent plays is the plan the engine will
// score. A greedy "take the dearest" pass cannot see that three Ooze beat one Dissolve.
//
// **Bounded by the budget and by MaxActions**, both, and it may never return a card it was not
// dealt — TestThePlannerObeysBothBounds holds it.
//
// **The shuffle stays outside this package.** What arrives is a hand, already dealt, exactly as the
// player's hand reaches the screen: `internal/combat` keeps no randomness and no clock, which is
// what TestRoundIsDeterministic pins and what lets the balance tool run whole duels headlessly.
//
// **The planner reasons about concepts and carries elements along.** Every choice is made on cost
// and damage; the element rides on the card it was dealt on and reaches the round untouched. An
// enemy's colours do nothing until an affix attunes them, so preferring one would be preferring a
// border.
func PlanFor(d Duelist, hand []Card) []Card {
	return planFor(d, hand, handTable)
}

// planFor is PlanFor with the catalogue injected, so a test can drive a planner against a
// synthetic ladder.
func planFor(d Duelist, hand []Card, hands []Hand) []Card {
	budget, slots := d.ActionPoints(), d.MaxActions()

	chosen, spent := bestAttacks(d, hand, budget, slots, hands)
	return append(chosen, spareCards(hand, chosen, budget-spent, slots-len(chosen))...)
}

// bestAttacks is the hardest-hitting affordable combination of attacks in the hand, and what it
// cost. It returns the cards in the order they were dealt.
//
// **Exhaustive over subsets, which is affordable because a hand is small.** EnemyHandSize is 7, so
// this is at most 128 candidates, each scored by the real `blowFor`. Above `maxSearchableAttacks`
// it falls back to taking the biggest cards that fit — a balance sim deliberately handing an
// opponent twenty attacks should get a plan rather than a hung process.
func bestAttacks(d Duelist, hand []Card, budget, slots int, hands []Hand) ([]Card, int) {
	var offence []int
	for i, c := range hand {
		s := c.Spec()
		if s.Verb == VerbAttack && s.Target == TargetOpponent {
			offence = append(offence, i)
		}
	}
	if len(offence) == 0 {
		return nil, 0
	}
	if len(offence) > maxSearchableAttacks {
		return greedyAttacks(d, hand, offence, budget, slots)
	}

	bestScore, bestCost := -1, 0
	var best []int

	// Ascending masks, and a strictly-better test, so a tie goes to the combination whose cards
	// were dealt earliest. That is deterministic without inventing a rule — the same tie-break
	// `matchCountOf` and `biggestAttack` take.
	for mask := 1; mask < 1<<len(offence); mask++ {
		var pick []int
		cost := 0
		for bit, idx := range offence {
			if mask&(1<<bit) == 0 {
				continue
			}
			pick = append(pick, idx)
			cost += d.CardCost(hand[idx])
		}
		if len(pick) > slots || cost > budget {
			continue
		}
		if score := blowScore(d, hand, pick, hands); score > bestScore {
			bestScore, bestCost, best = score, cost, pick
		}
	}

	out := make([]Card, 0, len(best))
	for _, i := range best {
		out = append(out, hand[i])
	}
	return out, bestCost
}

// maxSearchableAttacks bounds the exhaustive search. Seven is a full enemy hand; this is well
// above it and exists only so a sim probing outside the rules degrades rather than hangs.
const maxSearchableAttacks = 14

// blowScore is what one candidate turn would actually land, run through the same matcher the
// resolver uses. It is the blow before any defence, which is all a planner can know — it cannot
// see what the other side has raised.
func blowScore(d Duelist, hand []Card, pick []int, hands []Hand) int {
	// **A solo attacker's turn is worth the sum of its cards and nothing else.** No hand is read,
	// so there is no multiplier to chase and no card that earns nothing — which also means the
	// search is now looking for the most damage the budget buys rather than the best combination.
	if d.SoloAttacks {
		total := 0
		for _, idx := range pick {
			total += d.CardDamage(hand[idx])
		}
		return total
	}

	turn := make([]Slot, 0, len(pick))
	for i, idx := range pick {
		turn = append(turn, Slot{Index: i, Card: hand[idx]})
	}

	blow := blowFor(turn, hands)
	if len(blow.Cards) == 0 {
		return 0
	}

	base := 0
	for _, i := range blow.Cards {
		base += d.CardDamage(turn[i].Card)
	}
	return scaleDamage(base, blow.Multiplier)
}

// greedyAttacks is the fallback for a hand too big to search: the dearest cards that fit, which is
// what the old brute did.
func greedyAttacks(d Duelist, hand []Card, offence []int, budget, slots int) ([]Card, int) {
	used := make([]bool, len(hand))
	var out []Card
	spent := 0

	for len(out) < slots {
		best, bestCost := -1, 0
		for _, i := range offence {
			if used[i] {
				continue
			}
			if cost := d.CardCost(hand[i]); cost <= budget-spent && cost > bestCost {
				best, bestCost = i, cost
			}
		}
		if best < 0 {
			break
		}
		used[best] = true
		out = append(out, hand[best])
		spent += bestCost
	}
	return out, spent
}

// spareCards fills the slots and points the attacks did not want with whatever else the hand holds.
//
// **This is what keeps a non-attack card in an enemy deck from being dead content.** A planner that
// only maximised damage would never raise a shield or bank a point, so every `Congeal` authored
// into the roster would sit in a discard pile forever. Attacking is still the whole of the plan;
// this is the change left over.
//
// **Defences go up first, then the hand's own order.** A defence is the one leftover that changes
// whether the enemy is alive to use the next one, so it earns the tie-break; past that the deck
// author decides by what they put in, and the draw decides which of it turned up.
func spareCards(hand []Card, chosen []Card, budget, slots int) []Card {
	if slots <= 0 || budget <= 0 {
		return nil
	}

	used := make([]bool, len(hand))
	for _, c := range chosen {
		for i, h := range hand {
			if !used[i] && h == c {
				used[i] = true
				break
			}
		}
	}

	var out []Card
	for _, wantDefend := range []bool{true, false} {
		for i, c := range hand {
			if used[i] || len(out) >= slots {
				continue
			}
			s := c.Spec()
			if s.Verb == VerbAttack {
				continue // an attack the search already declined is an attack that did not help
			}
			if (s.Verb == VerbDefend) != wantDefend {
				continue
			}
			if s.Cost > budget {
				continue
			}
			used[i] = true
			out = append(out, c)
			budget -= s.Cost
		}
	}
	return out
}
