package screens

import (
	"encoding/json"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/ui"
)

// The combat screen is the one worth describing, and this is the compile-time half of that.
var _ ui.Reporter = (*CombatScene)(nil)

// TestTheCombatScreenDescribesAHalfBuiltScene is the case the whole interface exists for. A crash
// report is taken from a screen that has just panicked, which may be a screen whose Init never
// finished — so the account has to hold up with no opponent, no piles and no run behind it.
//
// **A zero scene is the worst one available in a test**, and it catches exactly the mistake
// ui.Reporter's doc comment forbids: a field read cannot fail on one and a derivation can.
func TestTheCombatScreenDescribesAHalfBuiltScene(t *testing.T) {
	var s CombatScene

	got := s.Report()
	if got == nil {
		t.Fatal("a scene that implements Reporter said nothing at all")
	}
	if _, ok := got["enemy"]; ok {
		t.Fatal("a scene with no opponent named one")
	}
	for _, key := range []string{"round", "cursor", "log", "hand", "deck", "discard", "selected", "fight"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("the account leaves out %q", key)
		}
	}
}

// TestTheAccountIsFlatAndWritable holds the envelope rule: a report is one JSON document, so a
// screen that handed back something json cannot write would take the tier down with it — and,
// because a nested value is how a whole hand gets in here by accident, nothing in it may be a map
// or a slice.
func TestTheAccountIsFlatAndWritable(t *testing.T) {
	var s CombatScene
	s.enemyElement = "fire"

	raw, err := json.Marshal(s.Report())
	if err != nil {
		t.Fatalf("the account will not encode: %v", err)
	}

	var back map[string]any
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("the account does not read back: %v", err)
	}
	for k, v := range back {
		switch v.(type) {
		case float64, string, bool:
		default:
			t.Fatalf("%q is %T; a crash report holds named values, never a structure", k, v)
		}
	}
	if back["enemyElement"] != "fire" {
		t.Fatalf("enemyElement = %v, want what the scene was holding", back["enemyElement"])
	}
}
