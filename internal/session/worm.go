package session

// Worms: the alterations a run can make to its own deck.
//
// **A worm targets one aspect of a card and gives it a new value**, which is the card language's
// shape pointed at a card that already exists rather than at a card being defined. The catalogue
// is `data/worms.json`; this file is where a record becomes something applicable, and where a bad
// record is refused.
//
// **It lives here rather than in `internal/combat` because a worm acts on the *run's* deck.** The
// rules resolve rounds and have no deck; the run has one. That is the same who-consumes-it test
// every file in `data/` answers.

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// WormTarget is which aspect of a card a worm changes.
//
// **A closed vocabulary, and closing it is the point** — the same posture `combat.Verb` takes.
// The set is short for a structural reason rather than a lack of imagination: `combat.Card` is a
// concept plus an element, and **the element is the only per-instance field**. Cost, damage,
// form and label all live on the shared concept, so a worm targeting one of those would change
// every copy of that card in the deck rather than the one the player picked.
//
// **Cost and amount became per-card on 2026-08-17** — `Card.CostDelta` and `Card.AmountPct` — and
// it was cheaper than it looked: `Cost()` and `Damage()` were already methods on the card, so the
// override went in one place each, and `Amount()` was added beside them for the three sites that
// read `Spec().Amount` directly. **Form and label are still concept-wide**, and a worm reaching
// for one of those is the moment to make the same argument again from scratch.
type WormTarget int

const (
	// TargetElement recolours a card. The concept is untouched, so what changes is which colour
	// it counts as in a mix and which status it can apply.
	TargetElement WormTarget = iota

	// TargetRemove takes a card out of the run for good.
	TargetRemove

	// TargetDuplicate puts a second copy of a card into the run. **Copies are the sharpest dial
	// in the game** — four of one concept in a turn is a Barrage — so this is the worm most
	// likely to need a cost.
	TargetDuplicate

	// TargetCost changes what a card costs, by a signed delta. **Floored at zero, not at one**
	// (owner's call, 2026-08-17): a free card is still bounded, by the count cap rather than by
	// the budget, and that shift was taken with its eyes open.
	TargetCost

	// TargetAmount scales a card's figure, as a percentage — 150 is half again. What the figure
	// *is* depends on the verb, which is what makes one worm reach every card in the deck: a
	// defence percentage, points banked, cards drawn, or a damage multiplier. A defence is clamped
	// under 100 by `Card.Amount`, because nothing stops a blow outright.
	TargetAmount

	// TargetPromote moves a card one rung up its form's ladder — Jab to Strike to Smash. It
	// costs more and hits harder, and the ladder is a consequence of what `duelist_cards.json`
	// declares rather than a table beside it.
	TargetPromote

	// TargetDemote moves a card one rung down: cheaper and weaker. **Not a downgrade so much as a
	// consistency play** — a hand of cheap cards plays more of itself, and the hand ladder counts
	// copies rather than damage.
	TargetDemote
)

// WormTargets is every target in a fixed order, for anything that walks them.
func WormTargets() []WormTarget {
	return []WormTarget{TargetElement, TargetRemove, TargetDuplicate,
		TargetCost, TargetAmount, TargetPromote, TargetDemote}
}

func (t WormTarget) String() string {
	switch t {
	case TargetRemove:
		return "remove"
	case TargetDuplicate:
		return "duplicate"
	case TargetCost:
		return "cost"
	case TargetAmount:
		return "amount"
	case TargetPromote:
		return "promote"
	case TargetDemote:
		return "demote"
	default:
		return "element"
	}
}

// ParseWormTarget resolves a target from its name. It reports failure rather than falling back: a
// worm quietly registered as a recolour because its target was misspelled is a mechanic nobody
// designed.
func ParseWormTarget(name string) (WormTarget, bool) {
	for _, t := range WormTargets() {
		if t.String() == name {
			return t, true
		}
	}
	return TargetElement, false
}

// Worm is one alteration, resolved against the rules.
//
// Comparable, so a screen can hold one by value and compare two without reaching for the key.
type Worm struct {
	Record string
	Name   string
	Text   string
	Target WormTarget

	// Element is the new colour, and is only meaningful for TargetElement.
	Element combat.Element

	// Number is the value read against a numeric target: a signed delta for `cost`, a percentage
	// for `amount`. Meaningless for the rest, which are refused if they carry one.
	Number int
}

// worms is the validated catalogue, built once at package init.
//
// **A bad record panics at init**, so it fails on launch rather than the first time a player wins
// a fight — the same severity a bad card record takes, and for the same reason: a worm that does
// nothing is a reward that silently is not one.
var worms, wormOrder = loadWorms()

// Worms is every worm in the catalogue, in a fixed sorted order.
func Worms() []Worm {
	out := make([]Worm, 0, len(wormOrder))
	for _, key := range wormOrder {
		out = append(out, worms[key])
	}
	return out
}

// WormByKey finds one by its record key.
func WormByKey(key string) (Worm, bool) {
	w, ok := worms[key]
	return w, ok
}

func loadWorms() (map[string]Worm, []string) {
	recs := data.LoadWorms()

	out := make(map[string]Worm, len(recs))
	for _, key := range data.WormOrder(recs) {
		w, err := resolveWorm(recs[key])
		if err != nil {
			panic("worms.json: " + err.Error())
		}
		out[key] = w
	}

	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) < 2 {
		// The offer is two worms, so a catalogue of one cannot fill it. Caught here rather than
		// producing a screen with a gap in it.
		panic(fmt.Sprintf("worms.json: %d worms, and an offer needs two", len(keys)))
	}
	return out, keys
}

// resolveWorm turns a record into a worm, or says why it cannot.
//
// **It refuses a value on a target that takes none**, rather than ignoring it. A `remove` worm
// carrying `"Value": "fire"` is somebody expecting something the mechanic does not do, and
// accepting it silently is how a catalogue comes to disagree with the game.
func resolveWorm(r data.WormData) (Worm, error) {
	if r.WormRecord == "" {
		return Worm{}, fmt.Errorf("a worm has no record key")
	}
	if r.Name == "" {
		return Worm{}, fmt.Errorf("%s has no name", r.WormRecord)
	}
	if r.Text == "" {
		// The card is a name and a line of text and nothing else — there is no glyph and no
		// figure — so a worm with no text is a card that does not say what it does.
		return Worm{}, fmt.Errorf("%s has no text, so its card says nothing", r.WormRecord)
	}

	target, ok := ParseWormTarget(r.Target)
	if !ok {
		return Worm{}, fmt.Errorf("%s names target %q, which is not one of %s",
			r.WormRecord, r.Target, targetList())
	}

	w := Worm{Record: r.WormRecord, Name: r.Name, Text: r.Text, Target: target}

	switch target {
	case TargetCost:
		n, err := strconv.Atoi(r.Value)
		if err != nil {
			return Worm{}, fmt.Errorf("%s targets cost and its value %q is not a number",
				r.WormRecord, r.Value)
		}
		if n == 0 {
			return Worm{}, fmt.Errorf("%s changes a cost by nothing", r.WormRecord)
		}
		w.Number = n
		return w, nil

	case TargetAmount:
		n, err := strconv.Atoi(r.Value)
		if err != nil {
			return Worm{}, fmt.Errorf("%s targets amount and its value %q is not a percentage",
				r.WormRecord, r.Value)
		}
		if n <= 0 {
			return Worm{}, fmt.Errorf("%s scales an amount to %d%%, which is nothing at all",
				r.WormRecord, n)
		}
		if n == 100 {
			return Worm{}, fmt.Errorf("%s scales an amount to 100%%, which changes nothing",
				r.WormRecord)
		}
		w.Number = n
		return w, nil
	}

	if target == TargetElement {
		e, ok := combat.ParseElement(r.Value)
		if !ok {
			return Worm{}, fmt.Errorf("%s names element %q, which the rules do not have",
				r.WormRecord, r.Value)
		}
		if e == combat.Basic {
			// A worm that greyed a card out would be a way to *lose* a colour rather than choose
			// one, and no card in the player's deck is drab — the plans stopped being the
			// exception on 2026-08-23.
			return Worm{}, fmt.Errorf("%s turns a card basic, which takes a colour away", r.WormRecord)
		}
		w.Element = e
		return w, nil
	}

	if r.Value != "" {
		return Worm{}, fmt.Errorf("%s targets %s and carries the value %q, which nothing reads",
			r.WormRecord, target, r.Value)
	}
	return w, nil
}

func targetList() string {
	out := ""
	for i, t := range WormTargets() {
		if i > 0 {
			out += ", "
		}
		out += t.String()
	}
	return out
}

// Apply performs a worm on one card of the run, by its position in the deck.
//
// **The one place the deck is altered by a worm**, so there is one place that can get it wrong.
// It reports whether anything happened: an index the deck does not hold is refused rather than
// silently landing on a neighbour, which matters because the offer hands out positions and the
// deck thins under them.
func (s *Session) Apply(w Worm, i int) bool {
	card, ok := s.Card(i)
	if !ok {
		return false
	}

	switch w.Target {
	case TargetRemove:
		return s.Remove(i)

	case TargetDuplicate:
		s.Add(card)
		return true

	case TargetCost:
		card.CostDelta += w.Number
		s.deck[i] = card
		return true

	case TargetAmount:
		// **Percentages compound rather than replace.** A card scaled twice by 150 is at 225, not
		// back at 150, so a second worm on the same card is worth something — and `Card.Amount`
		// does the clamping, so a Defend walked up repeatedly stops at its ceiling instead of
		// being refused.
		if card.AmountPct == 0 {
			card.AmountPct = 100
		}
		card.AmountPct = card.AmountPct * w.Number / 100
		s.deck[i] = card
		return true

	case TargetPromote, TargetDemote:
		step := 1
		if w.Target == TargetDemote {
			step = -1
		}
		next, ok := combat.Neighbour(card.Concept, step)
		if !ok {
			return false
		}
		card.Concept = next
		s.deck[i] = card
		return true

	default:
		return s.SetElement(i, w.Element)
	}
}

// CanApply reports whether this worm would do anything to this card. **The screen asks before it
// offers**, because a worm that lands and changes nothing is a reward taken away: a Smash cannot
// be promoted and a defend card has no ladder at all.
func (s *Session) CanApply(w Worm, i int) bool {
	card, ok := s.Card(i)
	if !ok {
		return false
	}

	switch w.Target {
	case TargetPromote:
		_, ok := combat.Neighbour(card.Concept, 1)
		return ok
	case TargetDemote:
		_, ok := combat.Neighbour(card.Concept, -1)
		return ok
	case TargetElement:
		return card.Element != w.Element
	default:
		return true
	}
}
