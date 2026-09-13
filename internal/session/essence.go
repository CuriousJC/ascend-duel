package session

// Essences: the alterations a run can make to its own deck.
//
// **An essence targets one aspect of a card and gives it a new value**, which is the card language's
// shape pointed at a card that already exists rather than at a card being defined. The catalog
// is `data/essences.json`; this file is where a record becomes something applicable, and where a bad
// record is refused.
//
// **It lives here rather than in `internal/combat` because an essence acts on the *run's* deck.** The
// rules resolve rounds and have no deck; the run has one. That is the same who-consumes-it test
// every file in `data/` answers.

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// EssenceTarget is which aspect of a card an essence changes.
//
// **A closed vocabulary, and closing it is the point** — the same posture `combat.Verb` takes.
// The set is short for a structural reason rather than a lack of imagination: `combat.Card` is a
// concept plus an element, and **the element is the only per-instance field**. Cost, damage,
// form and label all live on the shared concept, so an essence targeting one of those would change
// every copy of that card in the deck rather than the one the player picked.
//
// **Cost and amount became per-card on 2026-08-17** — `Card.CostDelta` and `Card.AmountPct` — and
// it was cheaper than it looked: `Cost()` and `Damage()` were already methods on the card, so the
// override went in one place each, and `Amount()` was added beside them for the three sites that
// read `Spec().Amount` directly. **Form and label are still concept-wide**, and an essence reaching
// for one of those is the moment to make the same argument again from scratch.
type EssenceTarget int

const (
	// TargetElement recolors a card. The concept is untouched, so what changes is which color
	// it counts as in a mix and which status it can apply.
	TargetElement EssenceTarget = iota

	// TargetRemove takes a card out of the run for good.
	TargetRemove

	// TargetDuplicate puts a second copy of a card into the run. **Copies are the sharpest dial
	// in the game** — four of one concept in a turn is a Barrage — so this is the essence most
	// likely to need a cost.
	TargetDuplicate

	// TargetCost changes what a card costs, by a signed delta. **Floored at zero, not at one**
	// (owner's call, 2026-08-17): a free card is still bounded, by the count cap rather than by
	// the budget, and that shift was taken with its eyes open.
	TargetCost

	// TargetAmount scales a card's figure, as a percentage — 150 is half again. What the figure
	// *is* depends on the verb, which is what makes one essence reach every card in the deck: a
	// defense percentage, a shield count, or a damage multiplier. A defense is clamped
	// under 100 by `Card.Amount`, because nothing stops a blow outright.
	TargetAmount

	// TargetPromote moves a card one rung up its form's ladder — Jab to Bash to Smash. It
	// costs more and hits harder, and the ladder is a consequence of what `duelist_cards.json`
	// declares rather than a table beside it.
	TargetPromote

	// TargetDemote moves a card one rung down: cheaper and weaker. **Not a downgrade so much as a
	// consistency play** — a hand of cheap cards plays more of itself, and the hand ladder counts
	// copies rather than damage.
	TargetDemote

	// TargetForm changes what one card counts as on the form axis — the element essences' trick on
	// the other matching axis.
	//
	// **It writes `Card.FormOverride`, which a rune already owns**, so this is the vocabulary
	// arriving at a mechanic rather than a new one: form is concept-wide and an essence swapping the
	// concept's form would change every copy in the deck, which is the argument the target list
	// was closed against. The override is per-card and only `Card.Form` reads it.
	//
	// **A Brace told to be a crush still defends**, and now matches on an attack axis — the legal
	// weird thing the owner ruled in when runes got the field.
	TargetForm
)

// EssenceTargets is every target in a fixed order, for anything that walks them.
func EssenceTargets() []EssenceTarget {
	return []EssenceTarget{TargetElement, TargetRemove, TargetDuplicate,
		TargetCost, TargetAmount, TargetPromote, TargetDemote, TargetForm}
}

func (t EssenceTarget) String() string {
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
	case TargetForm:
		return "form"
	default:
		return "element"
	}
}

// ParseEssenceTarget resolves a target from its name. It reports failure rather than falling back: a
// essence quietly registered as a recolor because its target was misspelled is a mechanic nobody
// designed.
func ParseEssenceTarget(name string) (EssenceTarget, bool) {
	for _, t := range EssenceTargets() {
		if t.String() == name {
			return t, true
		}
	}
	return TargetElement, false
}

// Essence is one alteration, resolved against the rules.
//
// Comparable, so a screen can hold one by value and compare two without reaching for the key.
type Essence struct {
	Record string
	Name   string
	Text   string
	Target EssenceTarget

	// Art is the assets key of the picture this essence draws, already resolved through
	// data.EssenceData.ArtKey — so it is never empty, and an essence nobody has drawn carries the
	// placeholder rather than a hole. Carried here so the reward screen and tools/essencesheet read
	// one answer.
	Art string

	// Family and Draw are authored, ignored by everything that plays the game, and read only by
	// tools/essencesheet — the motif the record was written under, and the art brief for its picture.
	// They ride along here so the sheet does not have to open data/essences.json a second time.
	Family string
	Draw   string

	// Element is the new color, and is only meaningful for TargetElement.
	Element combat.Element

	// Form is the axis a card is told to count on, and is only meaningful for TargetForm.
	Form combat.Form

	// Number is the value read against a numeric target: a signed delta for `cost`, a percentage
	// for `amount`. Meaningless for the rest, which are refused if they carry one.
	Number int
}

// essences is the validated catalog, built once at package init.
//
// **A bad record panics at init**, so it fails on launch rather than the first time a player wins
// a fight — the same severity a bad card record takes, and for the same reason: an essence that does
// nothing is a reward that silently is not one.
var essences, essenceOrder = loadEssences()

// Essences is every essence in the catalog, in a fixed sorted order.
func Essences() []Essence {
	out := make([]Essence, 0, len(essenceOrder))
	for _, key := range essenceOrder {
		out = append(out, essences[key])
	}
	return out
}

// EssenceByKey finds one by its record key.
func EssenceByKey(key string) (Essence, bool) {
	w, ok := essences[key]
	return w, ok
}

func loadEssences() (map[string]Essence, []string) {
	recs := data.LoadEssences()

	out := make(map[string]Essence, len(recs))
	for _, key := range data.EssenceOrder(recs) {
		w, err := resolveEssence(recs[key])
		if err != nil {
			panic("essences.json: " + err.Error())
		}
		out[key] = w
	}

	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) < 2 {
		// The offer is two essences, so a catalog of one cannot fill it. Caught here rather than
		// producing a screen with a gap in it.
		panic(fmt.Sprintf("essences.json: %d essences, and an offer needs two", len(keys)))
	}
	return out, keys
}

// resolveEssence turns a record into an essence, or says why it cannot.
//
// **It refuses a value on a target that takes none**, rather than ignoring it. A `remove` essence
// carrying `"Value": "fire"` is somebody expecting something the mechanic does not do, and
// accepting it silently is how a catalog comes to disagree with the game.
func resolveEssence(r data.EssenceData) (Essence, error) {
	if r.EssenceRecord == "" {
		return Essence{}, fmt.Errorf("an essence has no record key")
	}
	if r.Name == "" {
		return Essence{}, fmt.Errorf("%s has no name", r.EssenceRecord)
	}
	if r.Text == "" {
		// The card is a name and a line of text and nothing else — there is no glyph and no
		// figure — so an essence with no text is a card that does not say what it does.
		return Essence{}, fmt.Errorf("%s has no text, so its card says nothing", r.EssenceRecord)
	}

	target, ok := ParseEssenceTarget(r.Target)
	if !ok {
		return Essence{}, fmt.Errorf("%s names target %q, which is not one of %s",
			r.EssenceRecord, r.Target, targetList())
	}

	w := Essence{Record: r.EssenceRecord, Name: r.Name, Text: r.Text, Target: target,
		Art: r.ArtKey(), Family: r.Family, Draw: r.Draw}

	switch target {
	case TargetCost:
		n, err := strconv.Atoi(r.Value)
		if err != nil {
			return Essence{}, fmt.Errorf("%s targets cost and its value %q is not a number",
				r.EssenceRecord, r.Value)
		}
		if n == 0 {
			return Essence{}, fmt.Errorf("%s changes a cost by nothing", r.EssenceRecord)
		}
		w.Number = n
		return w, nil

	case TargetAmount:
		n, err := strconv.Atoi(r.Value)
		if err != nil {
			return Essence{}, fmt.Errorf("%s targets amount and its value %q is not a percentage",
				r.EssenceRecord, r.Value)
		}
		if n <= 0 {
			return Essence{}, fmt.Errorf("%s scales an amount to %d%%, which is nothing at all",
				r.EssenceRecord, n)
		}
		if n == 100 {
			return Essence{}, fmt.Errorf("%s scales an amount to 100%%, which changes nothing",
				r.EssenceRecord)
		}
		w.Number = n
		return w, nil
	}

	if target == TargetForm {
		f, ok := combat.ParseForm(r.Value)
		if !ok {
			return Essence{}, fmt.Errorf("%s names form %q, which the rules do not have",
				r.EssenceRecord, r.Value)
		}
		if f == combat.FormNone {
			// A card counting on no axis is a card that can never join a form hand, which is a
			// essence that takes something away rather than giving it.
			return Essence{}, fmt.Errorf("%s takes a card's form away", r.EssenceRecord)
		}
		w.Form = f
		return w, nil
	}

	if target == TargetElement {
		e, ok := combat.ParseElement(r.Value)
		if !ok {
			return Essence{}, fmt.Errorf("%s names element %q, which the rules do not have",
				r.EssenceRecord, r.Value)
		}
		if e == combat.Basic {
			// An essence that grayed a card out would be a way to *lose* a color rather than choose
			// one, and no card in the player's deck is drab — the defenses stopped being the
			// exception on 2026-08-23.
			return Essence{}, fmt.Errorf("%s turns a card basic, which takes a color away", r.EssenceRecord)
		}
		w.Element = e
		return w, nil
	}

	if r.Value != "" {
		return Essence{}, fmt.Errorf("%s targets %s and carries the value %q, which nothing reads",
			r.EssenceRecord, target, r.Value)
	}
	return w, nil
}

func targetList() string {
	out := ""
	for i, t := range EssenceTargets() {
		if i > 0 {
			out += ", "
		}
		out += t.String()
	}
	return out
}

// Apply performs an essence on one card of the run, by its position in the deck.
//
// **The one place the deck is altered by an essence**, so there is one place that can get it wrong.
// It reports whether anything happened: an index the deck does not hold is refused rather than
// silently landing on a neighbor, which matters because the offer hands out positions and the
// deck thins under them.
func (s *Session) Apply(w Essence, i int) bool {
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
		// back at 150, so a second essence on the same card is worth something — and `Card.Amount`
		// does the clamping, so a Defend walked up repeatedly stops at its ceiling instead of
		// being refused.
		if card.AmountPct == 0 {
			card.AmountPct = 100
		}
		card.AmountPct = card.AmountPct * w.Number / 100
		s.deck[i] = card
		return true

	case TargetForm:
		card.FormOverride = w.Form
		s.deck[i] = card
		return true

	case TargetPromote, TargetDemote:
		step := 1
		if w.Target == TargetDemote {
			step = -1
		}
		next, ok := combat.Neighbor(card.Concept, step)
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

// CanApply reports whether this essence would do anything to this card. **The screen asks before it
// offers**, because an essence that lands and changes nothing is a reward taken away: a Pulverize cannot
// be promoted, and neither can a Guard — the defenses are a ladder of their own since 2026-09-06,
// so the ends stop the same way rather than the whole verb being refused.
func (s *Session) CanApply(w Essence, i int) bool {
	card, ok := s.Card(i)
	if !ok {
		return false
	}

	switch w.Target {
	case TargetPromote:
		_, ok := combat.Neighbor(card.Concept, 1)
		return ok
	case TargetDemote:
		_, ok := combat.Neighbor(card.Concept, -1)
		return ok
	case TargetElement:
		return card.Element != w.Element
	case TargetForm:
		return card.Form() != w.Form
	default:
		return true
	}
}
