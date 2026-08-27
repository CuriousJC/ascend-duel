package session

// **A run, written down and read back.**
//
// `Session` says in its own doc comment that it is deliberately not persisted, and the reason given
// there — a deck edit is a choice rather than something the seed derives, so replay needs a seed
// plus a choice log — is still true. It is an argument about *replay*. **Resuming is a different
// question**: it needs the state the player is in, not the path they took to it, and a snapshot is
// exactly that. The reversal is recorded in MECHANICS.md rather than left as a contradiction
// between a comment and the code beneath it.
//
// **This package knows what a run is; `internal/profile` knows how to put one on disk.** So the
// conversion lives here and the file handling lives there, and profile goes on importing nothing of
// ours. The same division `decks` draws between a JSON card list and a rules type.
//
// **What is *not* here is the climb.** It is rebuilt from the run code — see Resume, and the note
// in profile/run.go about the day that stops being true.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/seeds"
)

// Snapshot is this run, frozen, ready to be saved.
//
// **The run seed is passed in rather than held.** A session has never known its own seed — the
// climb was built from a source handed to `Start` and nothing since has needed the number — and
// adding a field for the sake of the save would put a second copy of it beside `GlobalState.RunSeed`
// for the two to drift apart.
func (s *Session) Snapshot(runSeed int64) *profile.RunSnapshot {
	out := &profile.RunSnapshot{
		Seed:       seeds.Code(runSeed),
		Fight:      s.fight,
		Phase:      s.phase.String(),
		Vitae:      s.vitae,
		LifeLeft:   s.lifeLeft,
		Worn:       s.Worn(),
		Grown:      map[string]int{},
		Stones:     s.StoneCounts(),
		NextCardID: s.nextCardID,
		Spoils: profile.SpoilsSnapshot{
			Propagated: s.spoils.Propagated,
			FromLife:   s.spoils.FromLife,
			FromRoom:   s.spoils.FromRoom,
		},
		Deck: make([]profile.CardSnapshot, 0, len(s.deck)),
	}

	// **Only rings actually worn.** `grown` can hold an accumulator for a ring since sold, and
	// writing one down would be recording state for something the run no longer has.
	for _, key := range s.worn {
		if n := s.grown[key]; n != 0 {
			out.Grown[key] = n
		}
	}

	for _, c := range s.deck {
		out.Deck = append(out.Deck, profile.CardSnapshot{
			ID:        c.ID,
			Concept:   combat.ConceptOf(c.Concept).Key,
			Element:   c.Element.String(),
			CostDelta: c.CostDelta,
			AmountPct: c.AmountPct,
		})
	}
	return out
}

// Resume rebuilds a run from a snapshot, and reports the run seed it was saved under.
//
// **The climb is rebuilt from the seed rather than restored**, which is what keeps the file small
// and keeps one answer to who stands in which room — see profile/run.go.
//
// **Every name is resolved rather than trusted**, exactly as a ring record is: a concept key, an
// element, a phase and a ring record are four vocabularies, and a snapshot naming something this
// build has not got is a resumed run that is quietly wrong. It is refused instead, which costs the
// player one run and is reported to the caller as a fresh start.
//
// **A ring the catalogue no longer holds is refused rather than dropped**, on the same grounds: a
// run silently resuming without the ring it was wearing is a run the player would have to work out
// had changed.
func Resume(enemies map[string]data.EnemyData, bosses map[string]data.BossData, snap *profile.RunSnapshot) (*Session, int64, error) {
	if snap == nil {
		return nil, 0, fmt.Errorf("no run to resume")
	}

	runSeed, err := seeds.Parse(snap.Seed)
	if err != nil {
		return nil, 0, fmt.Errorf("run seed %q: %w", snap.Seed, err)
	}

	phase, ok := ParsePhase(snap.Phase)
	if !ok {
		return nil, 0, fmt.Errorf("phase %q is not a station of the loop", snap.Phase)
	}

	deck := make([]combat.Card, 0, len(snap.Deck))
	for _, c := range snap.Deck {
		concept, ok := combat.ConceptByKey(c.Concept)
		if !ok {
			return nil, 0, fmt.Errorf("card %q is not in the deck this build has", c.Concept)
		}
		element, ok := combat.ParseElement(c.Element)
		if !ok {
			return nil, 0, fmt.Errorf("card %q is %q, which is not an element", c.Concept, c.Element)
		}
		deck = append(deck, combat.Card{
			ID:        c.ID,
			Concept:   concept,
			Element:   element,
			CostDelta: c.CostDelta,
			AmountPct: c.AmountPct,
		})
	}

	// **Built bare rather than through New or Start.** Both of those mint fresh card identities and
	// put `StartingRings` on, which is right for a run beginning and wrong for one being put back
	// exactly as it was.
	s := &Session{
		deck:       deck,
		nextCardID: snap.NextCardID,
		fight:      snap.Fight,
		vitae:      snap.Vitae,
		lifeLeft:   snap.LifeLeft,
		phase:      phase,
		grown:      map[string]int{},
		stones:     map[string]int{},
		spoils: Spoils{
			Propagated: snap.Spoils.Propagated,
			FromLife:   snap.Spoils.FromLife,
			FromRoom:   snap.Spoils.FromRoom,
		},
	}
	s.climb = newClimb(enemies, bosses, runSeed)

	for _, key := range snap.Worn {
		if !s.Wear(key) {
			return nil, 0, fmt.Errorf("ring %q is not one this build can wear", key)
		}
	}
	// **A stone naming a rung this build has not got is refused rather than dropped**, exactly as a
	// ring the catalogue no longer holds is: a run resumed quietly paying less for its Card Pairs is
	// a run the player would have to work out had changed.
	for hand, n := range snap.Stones {
		if _, ok := combat.HandSlot(hand); !ok {
			return nil, 0, fmt.Errorf("hand %q has stones on it, and is not a rung this build has", hand)
		}
		if n > 0 {
			s.stones[hand] = n
		}
	}

	for key, n := range snap.Grown {
		if _, ok := registeredRings[key]; !ok {
			return nil, 0, fmt.Errorf("ring %q has grown, and is not in the catalogue", key)
		}
		s.grown[key] = n
	}

	// **A counter below the deck's highest id is refused.** It would hand a number out twice, and
	// everything that looks a card up by id would find whichever came first — a corruption that
	// only shows up later, as the wrong card being drawn in a panel.
	for _, c := range s.deck {
		if c.ID > s.nextCardID {
			return nil, 0, fmt.Errorf("card id %d is above the run's counter %d", c.ID, s.nextCardID)
		}
	}

	return s, runSeed, nil
}
