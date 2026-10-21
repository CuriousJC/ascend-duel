package achieve

// **What has been earned: the three questions, and the tallies a turn adds to.**
//
// Every function here is pure. It is handed what happened and hands back keys; nothing is awarded,
// nothing is written, and nothing is filtered against what the player already has — see
// `internal/screens/achieve.go`, which is where a key meets a profile. Keeping the filter out of
// here is what makes the whole catalog walkable in a test with no profile in sight.

import (
	"strconv"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// Moment is a named thing having happened, with whatever it carries.
type Moment struct {
	Name string

	// Value is a direction on MomentLadderWrapped, and a rung's hands.json key on MomentHandFormed.
	Value string

	// N is the figure the moment carries: the realm on MomentRealmReached, the shield count on
	// MomentShieldsRaised.
	N int
}

// The raisers, so a call site spells a moment once and cannot misspell it.
func DuelWon() Moment               { return Moment{Name: MomentDuelWon} }
func DuelWonFirstRound() Moment     { return Moment{Name: MomentDuelWonFirstRound} }
func TutorialFinished() Moment      { return Moment{Name: MomentTutorialFinished} }
func RealmReached(realm int) Moment { return Moment{Name: MomentRealmReached, N: realm} }

// The two directions a ladder-wrapped moment carries.
const (
	WrapDown = "down"
	WrapUp   = "up"
)

// LadderWrapped is the moment a card's change raises when it went from one end of its own ladder
// to the other — the bottom rung to the top is `down`, since only a demote arrives there, and the top
// to the bottom is `up`. Anything else reports false.
//
// **It reads the two cards and not the essence**, because the cards are what the player sees and
// they say the direction on their own: nothing but a demote takes a 1 AP Jab to a 3 AP Skewer.
// **A ladder of two never wraps**, since its ends are also neighbors — Brace demoted to Block is
// indistinguishable from Brace promoted — so the moment is about the three-rung attack ladders.
func LadderWrapped(before, after combat.Card) (Moment, bool) {
	rungs := combat.Ladder(before.Concept)
	if len(rungs) < 3 {
		return Moment{}, false
	}
	bottom, top := rungs[0], rungs[len(rungs)-1]
	switch {
	case before.Concept == bottom && after.Concept == top:
		return Moment{Name: MomentLadderWrapped, Value: WrapDown}, true
	case before.Concept == top && after.Concept == bottom:
		return Moment{Name: MomentLadderWrapped, Value: WrapUp}, true
	}
	return Moment{}, false
}

// HandFormed is the rung the player's blow was scored on, by its hands.json key.
func HandFormed(key string) Moment { return Moment{Name: MomentHandFormed, Value: key} }

// ShieldsRaised is the count a duelist ended up standing behind.
func ShieldsRaised(n int) Moment { return Moment{Name: MomentShieldsRaised, N: n} }

// HitLanded is the hardest single hit the player landed in a round.
func HitLanded(n int) Moment { return Moment{Name: MomentHitLanded, N: n} }

// JourneyWon is the last realm's boss falling.
func JourneyWon() Moment { return Moment{Name: MomentJourneyWon} }

// HardestHit is the largest single hit the player landed on the creature in a resolved round, or
// zero. **Off the log**, so a hit a shield ate — which raises no KindDamage — is not counted.
func HardestHit(log []combat.Event) int {
	best := 0
	for _, e := range log {
		if e.Kind == combat.KindDamage && e.Target != combat.SideA && e.Amount > best {
			best = e.Amount
		}
	}
	return best
}

// DeckShape is the moments a deck's make-up satisfies: one element across every card, one form
// across every attack.
//
// **A wildcard fits whatever the rest agree on**, on its own axis: a deck of fire cards and one
// wild-element card is a fire deck, and a deck whose attacks are stabs and a wild-form card is a
// stab deck. A deck with nothing to agree — no card for the element, no attack for the form — is
// not one shape, since there is nothing in it to be one of.
func DeckShape(deck []combat.Card) []Moment {
	var out []Moment
	if oneElement(deck) {
		out = append(out, Moment{Name: MomentDeckOneElement})
	}
	if oneForm(deck) {
		out = append(out, Moment{Name: MomentDeckOneForm})
	}
	return out
}

func oneElement(deck []combat.Card) bool {
	seen := combat.Basic
	found := false
	for _, c := range deck {
		found = true
		if c.Wild(combat.AxisElement) {
			continue
		}
		if c.Element == combat.Basic || (seen != combat.Basic && c.Element != seen) {
			return false
		}
		seen = c.Element
	}
	return found
}

func oneForm(deck []combat.Card) bool {
	var seen combat.Form
	attacks := 0
	for _, c := range deck {
		if c.Spec().Verb != combat.VerbAttack {
			continue
		}
		attacks++
		if c.Wild(combat.AxisForm) {
			continue
		}
		if seen != combat.FormNone && c.Form() != seen {
			return false
		}
		seen = c.Form()
	}
	return attacks > 0
}

// Collection is the moments the roster's whole record of beaten fights satisfies. `beaten` says
// whether a record has ever been beaten dealt as an element, by the profile's own reckoning.
//
// **A realm is a motif in an element**, one per pair any of the motif's records can be dealt as —
// the realm book's list — **and it is beaten when one of its bosses falls in that element**. Five
// elements with a boss down is a boss of each; every pair a record can be dealt as, down, is the
// bestiary. **The target is the roster as it now stands**, so a motif authored later raises it.
func Collection(motifs map[string]data.MotifData, beaten func(record, element string) bool) []Moment {
	bestiary, realms := true, true
	bossIn := map[string]bool{}
	found := false
	for _, m := range motifs {
		for _, element := range data.AffinityElements {
			deals, realm := false, false
			for _, r := range m.Records {
				if !r.HasAffinity(element) {
					continue
				}
				deals, found = true, true
				won := beaten(r.Record, element)
				if !won {
					bestiary = false
				}
				if won && r.Tier == data.TierBoss {
					realm = true
					bossIn[element] = true
				}
			}
			if deals && !realm {
				realms = false
			}
		}
	}
	var out []Moment
	if found && bestiary {
		out = append(out, Moment{Name: MomentBestiaryComplete})
	}
	if found && realms {
		out = append(out, Moment{Name: MomentRealmsComplete})
	}
	if len(bossIn) == len(data.AffinityElements) {
		out = append(out, Moment{Name: MomentBossEveryElement})
	}
	return out
}

// ByMoment is every achievement this moment satisfies.
func (c *Catalog) ByMoment(m Moment) []string {
	var out []string
	for _, a := range c.list {
		t := a.trigger
		if t.kind != data.TriggerMoment || t.moment != m.Name {
			continue
		}
		switch m.Name {
		case MomentRealmReached, MomentShieldsRaised, MomentHitLanded:
			// A threshold, not an equality. See the constants.
			if m.N < t.n {
				continue
			}
		case MomentLadderWrapped, MomentHandFormed:
			if m.Value != t.value {
				continue
			}
		}
		out = append(out, a.APIName)
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
func (c *Catalog) ByCounts(counts map[string]int) []string {
	var out []string
	for _, a := range c.list {
		if a.trigger.kind != data.TriggerCount {
			continue
		}
		if counts[a.trigger.counter] >= a.trigger.n {
			out = append(out, a.APIName)
		}
	}
	return out
}

// ByTurn is every achievement the cards played in one turn satisfy.
//
// **The turn, not the hand.** A hand counts the cards that scored it and leaves the rest out; these
// achievements are about what the player put on the table together — "four elements at once" — so
// Kitchen Sink can ask for a defense beside three attack forms, which no hand on the ladder can say.
func (c *Catalog) ByTurn(turn []combat.Card) []string {
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
				out = append(out, a.APIName)
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
		// **Every card must carry a value on the axis**, so a basic card cannot slip through an
		// element `same` clause by having nothing to disagree with — the same reading matchValue
		// takes in the hand matcher, where Basic is an absence rather than a value.
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
// different: "how many slashing cards have I played" is the form, and "how many Bashes" is the
// concept. Counting both costs one map entry each and is what lets an achievement be authored on
// either without a Go change.
func CountersFor(turn []combat.Card) map[string]int {
	out := map[string]int{}
	for _, c := range turn {
		out[counterConcept+c.Spec().Key]++
		out[counterForm+c.Form().String()]++
		switch {
		case c.Rider().Kind == combat.RiderWildElement:
			for _, e := range combat.AllElements {
				if e != combat.Basic {
					out[counterElement+e.String()]++
				}
			}
		case c.Element != combat.Basic:
			out[counterElement+c.Element.String()]++
		}
	}
	return out
}

// CounterLabel is a tally's name written for a person: `form:slash` as "slash cards", and
// `concept:Bash` as "Bash". It exists for a progress line the achievements page may yet want,
// and for an error message that has to name one.
func CounterLabel(name string) string {
	switch {
	case len(name) > len(counterForm) && name[:len(counterForm)] == counterForm:
		return name[len(counterForm):] + " cards"
	case len(name) > len(counterConcept) && name[:len(counterConcept)] == counterConcept:
		return name[len(counterConcept):]
	case len(name) > len(counterElement) && name[:len(counterElement)] == counterElement:
		return name[len(counterElement):] + " cards"
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
