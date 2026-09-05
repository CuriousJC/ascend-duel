package combat

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/data"
)

// The hand catalogue is data, and this file is the joint between the file and the rules.
//
// **`data/hands.json` holds the shape; this holds the meaning.** That is the same division
// `CheckCostTiers` already draws for the deck lists — the file can say `"groups": [3,2]`, and only
// this package can say what a turn is and how wide it can be.
//
// **This is the one place `internal/combat` imports `data`, and it is why that edge exists.**
// Everything else in `data/` is read by `screens`, `decks` or `entities` — layers above the
// rules — so the rules never needed it. A hand is different: it is read by the resolver itself.
// `data` imports nothing but the standard library, so the edge costs this package neither its
// testability nor its freedom from Ebitengine.
//
// **A malformed catalogue panics at package init**, exactly as a deck whose declared cost tiers
// disagree with the rules does. A hand silently dropped is a balance change nobody made, and the
// failure has to happen on launch rather than in the one round that would have formed it.

// loadCatalogue reads the file. It panics rather than returning an error: it runs at package init,
// and there is no sensible game to hand back if the rules could not be read.
func loadCatalogue() []Hand {
	handRecs := data.LoadHands()

	hands := make([]Hand, 0, len(handRecs))
	seenKey := map[string]bool{}
	seenID := map[HandID]string{}

	for _, rec := range handRecs {
		axis, err := validateHand(rec)
		if err != nil {
			panic(fmt.Sprintf("combat: hands.json hand %q: %v", rec.Key, err))
		}
		vary, varies, err := validateVary(rec, axis)
		if err != nil {
			panic(fmt.Sprintf("combat: hands.json hand %q: %v", rec.Key, err))
		}
		if seenKey[rec.Key] {
			panic(fmt.Sprintf("combat: hands.json declares hand key %q twice", rec.Key))
		}
		seenKey[rec.Key] = true

		h := Hand{
			ID:         HandID(rec.ID),
			Key:        rec.Key,
			Name:       rec.Name,
			Match:      axis,
			Groups:     append([]int(nil), rec.Groups...),
			Vary:       vary,
			Varies:     varies,
			Multiplier: rec.Multiplier,
		}
		if prev, dup := seenID[h.ID]; dup {
			panic(fmt.Sprintf("combat: hand ID %d is used by both %q and %q", h.ID, prev, h.Name))
		}
		seenID[h.ID] = h.Name
		hands = append(hands, h)
	}

	// **The fallback has to be in the file.** Any turn with an attack in it produces a hand, and
	// the one it produces when nothing was built is the High Card — so a catalogue without it is a
	// catalogue that cannot name the commonest result in the game, which is the one failure this
	// model can have.
	if !seenKey[highCardKey] {
		panic(fmt.Sprintf("combat: hands.json has no %q hand, so a lone attack could not be named", highCardKey))
	}

	return hands
}

func validateHand(r data.HandData) (Axis, error) {
	switch {
	case r.Key == "":
		return AxisConcept, fmt.Errorf("has no key")
	case r.Name == "":
		return AxisConcept, fmt.Errorf("has no name")
	case r.ID <= 0:
		return AxisConcept, fmt.Errorf("has no ID, or a zero one, which means 'no hand'")
	case len(r.Groups) == 0:
		return AxisConcept, fmt.Errorf("names no groups, so it counts nothing")
	// **100 is the identity and 0 deletes the blow** *(2026-08-18)*. The multiplier multiplies the
	// hand's own cards now rather than a separate swing added on top of them, so a hand at 0 is not
	// "pays no bonus" — it is an attack phase that deals nothing. Every hand needs a real number,
	// the High Card included, and its 100 is what makes a lone attack land its face damage.
	//
	// Anything below 100 is a *penalty* and is deliberately still legal: refusing it would take a
	// tuning lever away from the file, which is the one place the ladder is meant to be tuned.
	case r.Multiplier <= 0:
		return AxisConcept, fmt.Errorf("has a multiplier of %d; the multiplier scales the hand's own cards, so that is an attack phase dealing nothing", r.Multiplier)
	}

	total := 0
	for _, g := range r.Groups {
		if g < 1 {
			return AxisConcept, fmt.Errorf("names a group of %d cards", g)
		}
		total += g
	}
	// **A built hand has to beat the cards that formed it.** The High Card sits at the identity, so
	// a multi-card hand at or below 100 is one a player would be punished for making — a typo
	// rather than an ambition, and refused rather than loaded.
	if total > 1 && r.Multiplier <= multiplierScale {
		return AxisConcept, fmt.Errorf("wants %d cards for multiplier %d, which is no better than playing them as a High Card", total, r.Multiplier)
	}
	// **A hand cannot ask for more cards than a turn can hold.** MaxActions is five and frozen,
	// so a six-card hand is one nobody could ever form and is a typo rather than an ambition.
	if total > baseMaxActions {
		return AxisConcept, fmt.Errorf("wants %d cards but a turn holds %d", total, baseMaxActions)
	}

	// **Nor for more distinct values than its axis has** *(2026-08-19)*. Three forms reach a blow
	// and five elements do, so a hand wanting four groups on the form axis is unclimbable — the
	// same class of typo as one wanting six cards, and invisible without the check because the
	// matcher would simply never find it.
	axis, ok := ParseAxis(r.Match)
	if !ok {
		return AxisConcept, fmt.Errorf("matches on %q, which is not one of %s", r.Match, axisList())
	}
	if spread := axis.spread(); spread > 0 && len(r.Groups) > spread {
		return AxisConcept, fmt.Errorf("wants %d groups but only %d %ss can reach a blow", len(r.Groups), spread, axis)
	}

	return axis, nil
}

// validateVary resolves the optional `vary` clause and refuses the two ways it can be nonsense.
//
// **Varying on the axis the hand already counts is a contradiction**, not a curiosity: every card
// in a group carries the same value there by construction, so such a hand could never form and
// would be invisible rather than refused. **And a group cannot ask for more distinct values than
// the axis has** — three forms varying across four cards is the same unclimbable typo the groups
// check above catches, one axis over.
func validateVary(r data.HandData, match Axis) (Axis, bool, error) {
	if r.Vary == "" {
		return AxisConcept, false, nil
	}
	vary, ok := ParseAxis(r.Vary)
	if !ok {
		return AxisConcept, false, fmt.Errorf("varies on %q, which is not one of %s", r.Vary, axisList())
	}
	if vary == match {
		return AxisConcept, false, fmt.Errorf("varies on %s, which is the axis it already counts on, so no hand could ever form it", vary)
	}
	if spread := vary.spread(); spread > 0 {
		for _, g := range r.Groups {
			if g > spread {
				return AxisConcept, false, fmt.Errorf("wants a group of %d cards varying on %s, but only %d %ss can reach a blow", g, vary, spread, vary)
			}
		}
	}
	return vary, true, nil
}

// axisList is the axes written out for an error message.
func axisList() string {
	out := ""
	for i, a := range AllAxes {
		if i > 0 {
			out += ", "
		}
		out += a.String()
	}
	return out
}
