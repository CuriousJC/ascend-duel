package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// A chosen seed is a journey that could have been looked up in advance, so whatever the game
// withholds from one has to still know after the run is closed and resumed.
func TestAChosenSeedSurvivesAResume(t *testing.T) {
	for _, chosen := range []bool{false, true} {
		s := New(nil)
		if chosen {
			s.ChooseSeed()
		}
		back, _, err := Resume(nil, data.JourneyData{}, s.Snapshot(0))
		if err != nil {
			t.Fatalf("resume: %v", err)
		}
		if got := back.SeedChosen(); got != chosen {
			t.Errorf("a run saved with a chosen seed of %v resumed as %v", chosen, got)
		}
	}
}
