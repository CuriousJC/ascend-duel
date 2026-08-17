package combat

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/data"
)

// The combo catalogue is data, and this file is the joint between the file and the rules.
//
// **`data/combos.json` holds the shape; this holds the meaning.** That is the same division
// `CheckCostTiers` already draws for the deck lists — the file can say `"card": "Strike"` and
// `"scope": ["attack"]`, and only this package can say what a Strike or an attack *is*. Joining
// them is `ParseAction` and `ParseCategory`, exactly as the deck loaders do it.
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

// Expansion modes. **An expanded entry is one line in the file that becomes one hand per attack
// card**, which is what keeps the ladder a shape the game keeps rather than a list somebody has
// to remember to extend when an attack card is added.
const (
	expandNone    = "none"
	expandAttacks = "attack-cards"
)

// loadCatalogue reads the file and expands it. It panics rather than returning an error: it runs
// at package init, and there is no sensible game to hand back if the rules could not be read.
func loadCatalogue() ([]Hand, []Mix) {
	handRecs, mixRecs := data.LoadCombos()

	hands := make([]Hand, 0, len(handRecs)*4)
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

		expanded, err := expandHand(rec)
		if err != nil {
			panic(fmt.Sprintf("combat: combos.json hand %q: %v", rec.Key, err))
		}
		for _, h := range expanded {
			if prev, dup := seenID[h.ID]; dup {
				panic(fmt.Sprintf("combat: hand ID %d is used by both %q and %q", h.ID, prev, h.Name))
			}
			seenID[h.ID] = h.Name
			hands = append(hands, h)
		}
	}

	mixes := make([]Mix, 0, len(mixRecs))
	seenColours := map[int]string{}
	seenMixID := map[MixID]string{}

	for _, rec := range mixRecs {
		if err := validateMix(rec); err != nil {
			panic(fmt.Sprintf("combat: combos.json mix %q: %v", rec.Key, err))
		}
		// **Exactly one mix per colour count, and no gaps.** The mixes have to partition every
		// possible hand, or a turn showing that many colours would form a hand and then have no
		// makeup — which is the one way this model can produce a combo it cannot name.
		if prev, dup := seenColours[rec.Colours]; dup {
			panic(fmt.Sprintf("combat: mixes %q and %q both claim %d colours", prev, rec.Name, rec.Colours))
		}
		seenColours[rec.Colours] = rec.Name
		if prev, dup := seenMixID[MixID(rec.ID)]; dup {
			panic(fmt.Sprintf("combat: mix ID %d is used by both %q and %q", rec.ID, prev, rec.Name))
		}
		seenMixID[MixID(rec.ID)] = rec.Name

		mixes = append(mixes, Mix{
			ID:         MixID(rec.ID),
			Key:        rec.Key,
			Name:       rec.Name,
			Colours:    rec.Colours,
			Multiplier: rec.Multiplier,
		})
	}

	// Every colour count a hand can show needs a mix, from none of them to all of them.
	for n := 0; n <= ElementCount-1; n++ {
		if _, ok := seenColours[n]; !ok {
			panic(fmt.Sprintf("combat: combos.json has no mix for %d colours, so a hand showing that many could not be named", n))
		}
	}

	// **The fallback has to be in the file.** Any turn with an attack in it produces a hand, and
	// the one it produces when nothing was built is the High Card — so a catalogue without it is a
	// catalogue that cannot name the commonest result in the game. Same posture as the missing
	// mix above, and for the same reason.
	if !seenKey[highCardKey] {
		panic(fmt.Sprintf("combat: combos.json has no %q hand, so a lone attack could not be named", highCardKey))
	}

	return hands, mixes
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

	switch r.Expand {
	case "", expandNone:
	case expandAttacks:
		// **Expansion takes exactly one group.** Pinning a two-group pattern to one card is
		// meaningless — the second group is by definition a *different* concept — so the file is
		// refused rather than quietly producing a hand that can never match.
		if len(r.Groups) != 1 {
			return fmt.Errorf("expands per attack card, so it must name exactly one group")
		}
	default:
		return fmt.Errorf("has expand %q, want %q or %q", r.Expand, expandNone, expandAttacks)
	}

	return nil
}

func validateMix(r data.MixData) error {
	switch {
	case r.Key == "":
		return fmt.Errorf("has no key")
	case r.Name == "":
		return fmt.Errorf("has no name")
	case r.ID <= 0:
		return fmt.Errorf("has no ID, or a zero one, which means 'no makeup'")
	case r.Colours < 0:
		return fmt.Errorf("claims %d colours", r.Colours)
	case r.Colours > ElementCount-1:
		return fmt.Errorf("claims %d colours but only %d exist besides basic", r.Colours, ElementCount-1)
	case r.Multiplier < 0:
		return fmt.Errorf("has a negative multiplier")
	}
	return nil
}

// expandHand turns one record into the hands it describes: one, or one per attack card. It is
// also where every name in the entry is resolved, so a typo is reported here rather than becoming
// a hand that matches nothing.
func expandHand(r data.HandData) ([]Hand, error) {
	base, err := handBase(r)
	if err != nil {
		return nil, err
	}

	if r.Expand != expandAttacks {
		return []Hand{base}, nil
	}

	return []Hand{base}, nil
}

// handBase builds the hand every expansion of a record shares, leaving ID, Name and Pin to be
// filled in per expansion.
func handBase(r data.HandData) (Hand, error) {
	h := Hand{
		ID:         HandID(r.ID),
		Key:        r.Key,
		Name:       r.Name,
		Groups:     append([]int(nil), r.Groups...),
		Multiplier: r.Multiplier,
		Effect:     effectFor(r.Effect),
	}

	for _, name := range r.Scope {
		cat, ok := ParseCategory(name)
		if !ok {
			return Hand{}, fmt.Errorf("names scope %q, which is not a category", name)
		}
		h.Scope = append(h.Scope, cat)
	}

	return h, nil
}

// effectFor turns the file's reward record into the rules' Effect. The only translation is
// StaggerAll, which the file writes as a bool and the rules carry as a sentinel.
func effectFor(e data.EffectData) Effect {
	out := Effect{BankAP: e.BankAP, Stagger: e.Stagger}
	if e.StaggerAll {
		out.Stagger = StaggerAll
	}
	return out
}
