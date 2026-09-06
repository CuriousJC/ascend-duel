package achieve

// **What has been earned: the three questions, and the tallies a turn adds to.**
//
// Every function here is pure. It is handed what happened and hands back keys; nothing is awarded,
// nothing is written, and nothing is filtered against what the player already has — see
// `internal/screens/achieve.go`, which is where a key meets a profile. Keeping the filter out of
// here is what makes the whole catalogue walkable in a test with no profile in sight.

import (
	"strconv"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// Moment is a named thing having happened, with whatever it carries.
type Moment struct {
	Name string

	// Value is a concept's label, on MomentCardAltered.
	Value string

	// N is the floor, on MomentFloorReached.
	N int
}

// DuelWon, TutorialFinished, FloorReached and CardAltered are the four raisers, so a call site
// spells a moment once and cannot misspell it.
func DuelWon() Moment               { return Moment{Name: MomentDuelWon} }
func TutorialFinished() Moment      { return Moment{Name: MomentTutorialFinished} }
func FloorReached(floor int) Moment { return Moment{Name: MomentFloorReached, N: floor} }
func CardAltered(label string) Moment {
	return Moment{Name: MomentCardAltered, Value: label}
}

// ByMoment is every achievement this moment satisfies.
func (c *Catalogue) ByMoment(m Moment) []string {
	var out []string
	for _, a := range c.list {
		t := a.trigger
		if t.kind != data.TriggerMoment || t.moment != m.Name {
			continue
		}
		switch m.Name {
		case MomentFloorReached:
			// A threshold, not an equality. See the constant.
			if m.N < t.n {
				continue
			}
		case MomentCardAltered:
			if m.Value != t.value {
				continue
			}
		}
		out = append(out, a.Key)
	}
	return out
}

// ByCounts is every count achievement the tallies have reached.
//
// **It is asked at the end of a duel rather than as a card is played** *(owner's call,
// 2026-09-06)*. Counters move in memory all through a fight and are settled when it ends, so this
// is asked once against the settled figures — which is also why it reports everything that is *at
// or over* its threshold rather than everything that crossed it on the last card. Whether a key is
// new is the profile's question, not this one's.
func (c *Catalogue) ByCounts(counts map[string]int) []string {
	var out []string
	for _, a := range c.list {
		if a.trigger.kind != data.TriggerCount {
			continue
		}
		if counts[a.trigger.counter] >= a.trigger.n {
			out = append(out, a.Key)
		}
	}
	return out
}

// ByTurn is every achievement the cards played in one turn satisfy.
//
// **The turn, not the hand.** A hand counts the cards that scored it and leaves the rest out; these
// achievements are about what the player put on the table together — "four elements at once" — so
// Arsenal can ask for a defence beside three attack forms, which no hand on the ladder can say.
func (c *Catalogue) ByTurn(turn []combat.Card) []string {
	if len(turn) == 0 {
		return nil
	}
	var out []string
	for _, a := range c.list {
		if a.trigger.kind != data.TriggerTurn {
			continue
		}
		for _, p := range a.trigger.patterns {
			if p.holds(turn) {
				out = append(out, a.Key)
				break
			}
		}
	}
	return out
}

// holds reports whether every clause of a pattern is satisfied by the turn.
func (p pattern) holds(turn []combat.Card) bool {
	for _, c := range p.clauses {
		if !c.holds(turn) {
			return false
		}
	}
	return true
}

// holds reports whether one clause is satisfied.
func (c clause) holds(turn []combat.Card) bool {
	cards := c.of.filter(turn)

	switch c.mode {
	case data.ModeCount:
		return len(cards) >= c.n

	case data.ModeDistinct:
		return len(distinctOn(cards, c.axis)) >= c.n

	case data.ModeSame:
		// **An empty selection satisfies nothing.** "All of them agree" is vacuously true of no
		// cards, and an achievement earned by playing nothing is not one.
		if len(cards) == 0 {
			return false
		}
		// **Every card must carry a value on the axis**, so a formless card cannot slip through a
		// `same` clause by having nothing to disagree with — the same reading matchValue takes in
		// the hand matcher, where FormNone and Basic are absences rather than values.
		for _, card := range cards {
			if _, ok := axisValue(card, c.axis); !ok {
				return false
			}
		}
		return len(distinctOn(cards, c.axis)) == 1

	default:
		return false
	}
}

// filter picks the cards a clause looks at.
func (c category) filter(turn []combat.Card) []combat.Card {
	if c == anyCategory {
		return turn
	}
	want := combat.CategoryAttack
	if c == defendOnly {
		want = combat.CategoryDefend
	}
	out := make([]combat.Card, 0, len(turn))
	for _, card := range turn {
		if card.Category() == want {
			out = append(out, card)
		}
	}
	return out
}

// distinctOn is how many different values the cards carry on one axis.
//
// **A card with no value on the axis is not a value.** An enemy card has no form and a drab card no
// element; neither may count toward "four different elements", exactly as neither counts toward a
// hand.
func distinctOn(cards []combat.Card, axis combat.Axis) map[int]bool {
	seen := map[int]bool{}
	for _, c := range cards {
		if v, ok := axisValue(c, axis); ok {
			seen[v] = true
		}
	}
	return seen
}

// axisValue is what one card counts as on an axis, and whether it counts at all.
//
// **It asks the matcher rather than remembering what the matcher does.** `combat.MatchValue` was
// exported for this on 2026-09-06: `internal/decks` had already mirrored the rule in three lines,
// and a second mirror here would have made three readings of one rule that must never disagree — an
// achievement counting a hand the ladder does not is a silent wrong answer, not a crash.
func axisValue(c combat.Card, axis combat.Axis) (int, bool) { return combat.MatchValue(c, axis) }

// CountersFor is what one played turn adds to the lifetime tallies.
//
// **Two names per card, on two axes**, because the two questions a player asks are genuinely
// different: "how many slashing cards have I played" is the form, and "how many Strikes" is the
// concept. Counting both costs one map entry each and is what lets an achievement be authored on
// either without a Go change.
//
// **A card with no form contributes no form tally**, and never a `form:none` — a counter naming an
// absence is a counter nothing should ever ask about.
func CountersFor(turn []combat.Card) map[string]int {
	out := map[string]int{}
	for _, c := range turn {
		out[counterConcept+c.Spec().Key]++
		if f := c.Form(); f != combat.FormNone {
			out[counterForm+f.String()]++
		}
	}
	return out
}

// CounterLabel is a tally's name written for a person: `form:slash` as "slash cards", and
// `concept:Strike` as "Strike". It exists for a progress line the achievements page may yet want,
// and for an error message that has to name one.
func CounterLabel(name string) string {
	switch {
	case len(name) > len(counterForm) && name[:len(counterForm)] == counterForm:
		return name[len(counterForm):] + " cards"
	case len(name) > len(counterConcept) && name[:len(counterConcept)] == counterConcept:
		return name[len(counterConcept):]
	default:
		return name
	}
}

// Progress is how far along a count achievement is, as "141 / 300", or empty for a trigger that is
// not a tally. **Only a count can show progress** — a turn either happened or did not, and a moment
// is not a fraction of anything.
func (a Achievement) Progress(counts map[string]int) string {
	if a.trigger.kind != data.TriggerCount {
		return ""
	}
	got := counts[a.trigger.counter]
	if got > a.trigger.n {
		got = a.trigger.n
	}
	return strconv.Itoa(got) + " / " + strconv.Itoa(a.trigger.n)
}
