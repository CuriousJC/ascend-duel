package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// These tests walk a table and compare constants. They create no `ebiten.Image` and need no
// window, which is the narrow exception `internal/screens` tests are allowed under — see
// CLAUDE.md. They exist because the invariant is a cross-package one the compiler cannot see: an
// event kind added in `internal/combat` has to be answered on this screen, and nothing about the
// two packages compiling says it was.

// TestEveryEventKindIsChoreographed is the point of the theatre table.
//
// **An event nobody draws is an event the player is never told about**, once the Resolution feed
// is behind a button. Today a new kind would still narrate itself in the feed and the omission
// would be invisible; after the button it is silence. This fails at the moment the kind is added
// rather than at the moment somebody notices the screen went quiet.
func TestEveryEventKindIsChoreographed(t *testing.T) {
	for k := combat.EventKind(0); k <= combat.KindRoundEnd; k++ {
		spec, ok := theatre[k]
		if !ok {
			t.Errorf("event kind %d has no entry in the theatre table - give it one, "+
				"with anchorNone and a reason if it genuinely draws nothing", k)
			continue
		}
		if spec.why == "" {
			t.Errorf("event kind %d is choreographed with no reason written down", k)
		}
	}

	// The other direction: an entry for a kind that no longer exists is a drawing nothing can
	// trigger, and it would sit here looking like coverage.
	for k := range theatre {
		if k < 0 || k > combat.KindRoundEnd {
			t.Errorf("the theatre table describes event kind %d, which the engine does not emit", k)
		}
	}
}

// TestEveryAnchorAndGestureIsNamed keeps the table's vocabulary closed. A value outside the enum
// is a place or a movement nothing knows how to resolve, and the String methods are what a failing
// test elsewhere prints — an unnamed one reports "?" and says nothing.
func TestEveryAnchorAndGestureIsNamed(t *testing.T) {
	for a := anchorNone; a < anchorCount; a++ {
		if a.String() == "?" {
			t.Errorf("anchor %d has no name", a)
		}
	}
	for g := gestureNone; g < gestureCount; g++ {
		if g.String() == "?" {
			t.Errorf("gesture %d has no name", g)
		}
	}
}

// TestTheChoreographyIsInternallyConsistent checks each entry against the grammar rather than
// against a list of expected values — a test naming the anchors back would only restate the table.
//
// The three rules, and each is a mistake that would draw something incoherent:
//
//   - something that flies needs both ends. A flight from nowhere is the appearing-in-the-middle
//     failure this whole file exists to prevent.
//   - something that pops or is struck out needs a target and nothing else. It did not come from
//     anywhere; that is what separates it from a flight.
//   - something that draws nothing needs neither. An anchor on a gestureNone entry is a place
//     nobody goes, left behind by a drawing that was reconsidered.
func TestTheChoreographyIsInternallyConsistent(t *testing.T) {
	for k := combat.EventKind(0); k <= combat.KindRoundEnd; k++ {
		spec, ok := theatre[k]
		if !ok {
			continue // TestEveryEventKindIsChoreographed reports this
		}

		switch spec.how {
		case gestureFly:
			if spec.from == anchorNone || spec.to == anchorNone {
				t.Errorf("event kind %d flies from %v to %v - a flight needs both ends",
					k, spec.from, spec.to)
			}
		case gesturePop, gestureStrike, gestureTally:
			if spec.to == anchorNone {
				t.Errorf("event kind %d %v with no target", k, spec.how)
			}
			if spec.from != anchorNone {
				t.Errorf("event kind %d %v but sets a source (%v) - only a flight has one",
					k, spec.how, spec.from)
			}
		case gestureNone:
			if spec.from != anchorNone || spec.to != anchorNone {
				t.Errorf("event kind %d draws nothing but names %v and %v",
					k, spec.from, spec.to)
			}
		default:
			t.Errorf("event kind %d has gesture %d, which is outside the enum", k, spec.how)
		}

		if spec.from >= anchorCount || spec.to >= anchorCount || spec.from < 0 || spec.to < 0 {
			t.Errorf("event kind %d names an anchor outside the enum: %d -> %d",
				k, spec.from, spec.to)
		}
	}
}

// TestTheEngineIsTheAuthorityOnDamagesSource pins the one anchor that is a rule rather than a
// rectangle. `anchorBlow` exists because `KindDamage` means two pictures, and which one is decided
// by whether the acting side scores hands — never by which side it is.
//
// If this ever needs changing, the thing to check first is that `SoloAttacks` is still a flag on
// the duelist. The day the screen starts asking "is this SideB" instead, the balance tool — which
// plays both sides headlessly — has been quietly left behind.
func TestTheEngineIsTheAuthorityOnDamagesSource(t *testing.T) {
	spec := theatre[combat.KindDamage]
	if spec.from != anchorBlow {
		t.Errorf("damage flies from %v; it has to come from wherever the turn's figure "+
			"already is, which differs between a scored hand and a solo attacker", spec.from)
	}
	if spec.to != anchorTargetCard {
		t.Errorf("damage lands on %v, not on the card whose bar it empties", spec.to)
	}
}
