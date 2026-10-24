package screens

import "testing"

// **A stone's tooltip writes a multiplier as the player reads it**, to the tenth.
func TestAStoneWritesItsMultiplierToTheTenth(t *testing.T) {
	for pct, want := range map[int]string{100: "1.0", 110: "1.1", 130: "1.3", 480: "4.8", 1000: "10.0"} {
		if got := stoneMultiplier(pct); got != want {
			t.Errorf("%d reads %q, want %q", pct, got, want)
		}
	}
}
