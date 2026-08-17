package combat

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/data"
)

// The combo catalogue is data, and this file is the joint between the file and the rules.
//
// **`data/combos.json` holds the shape; this holds the meaning.** That is the same division
// `CheckCostTiers` already draws for the deck lists — the file can say `"groups": [3,2]`, and only
// this package can say what a turn is and how wide it can be.
//
// **This is the one place `internal/combat` imports `data`, and it is why that edge exists.**
// Everything else in `data/` is read by `screens`, `decks` or `entities` — layers above the
// rules — so the rules never needed it. A combo is different: it is read by the resolver itself.
// `data` imports nothing but the standard library, so the edge costs this package neither its
// testability nor its freedom from Ebitengine.
//
// **A malformed catalogue panics at package init**, exactly as a deck whose declared cost tiers
// disagree with the rules does. A hand silently dropped is a balance change nobody made, and the
// failure has to happen on launch rather than in the one round that would have formed it.

// loadCatalogue reads the file. It panics rather than returning an error: it runs at package init,
// and there is no sensible game to hand back if the rules could not be read.
func loadCatalogue() []Hand {
	handRecs := data.LoadCombos()

	hands := make([]Hand, 0, len(handRecs))
	seenKey := map[string]bool{}
	seenID := map[HandID]string{}

	for _, rec := range handRecs {
		if err := validateHand(rec); err != nil {
			panic(fmt.Sprintf("combat: combos.json hand %q: %v", rec.Key, err))
		}
		if seenKey[rec.Key] {
			panic(fmt.Sprintf("combat: combos.json declares hand key %q twice", rec.Key))
		}
		seenKey[rec.Key] = true

		h := Hand{
			ID:         HandID(rec.ID),
			Key:        rec.Key,
			Name:       rec.Name,
			Groups:     append([]int(nil), rec.Groups...),
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
		panic(fmt.Sprintf("combat: combos.json has no %q hand, so a lone attack could not be named", highCardKey))
	}

	return hands
}

func validateHand(r data.HandData) error {
	switch {
	case r.Key == "":
		return fmt.Errorf("has no key")
	case r.Name == "":
		return fmt.Errorf("has no name")
	case r.ID <= 0:
		return fmt.Errorf("has no ID, or a zero one, which means 'no hand'")
	case len(r.Groups) == 0:
		return fmt.Errorf("names no groups, so it counts nothing")
	case r.Multiplier < 0:
		return fmt.Errorf("has a negative multiplier")
	}

	total := 0
	for _, g := range r.Groups {
		if g < 1 {
			return fmt.Errorf("names a group of %d cards", g)
		}
		total += g
	}
	// **A zero multiplier is legal for the one-card hand and a typo in any other** *(2026-08-15)*.
	// The High Card pays nothing on purpose — what lands is the card's own face damage — and it is
	// the only hand nobody chooses to form. A pair worth no multiplier is a hand nobody would
	// build, so it is refused rather than loaded.
	if total > 1 && r.Multiplier <= 0 {
		return fmt.Errorf("wants %d cards for multiplier %d; a built hand worth nothing is one nobody would form", total, r.Multiplier)
	}
	// **A hand cannot ask for more cards than a turn can hold.** MaxActions is five and frozen,
	// so a six-card hand is one nobody could ever form and is a typo rather than an ambition.
	if total > baseMaxActions {
		return fmt.Errorf("wants %d cards but a turn holds %d", total, baseMaxActions)
	}

	return nil
}
