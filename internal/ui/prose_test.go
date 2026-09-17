package ui

import "testing"

// The wording this package owns, tested where it lives. **It moved out of the ledger's tests when
// the drawing layer split off**: what it checks is how a hand's name is written, which every screen
// and the ledger alike read through, not anything about the panel that happens to print it.

// The hand's name leads with the rung and carries its axis in brackets, because the loudest line
// of the round should not open on the least interesting word in it. A name the catalog does not
// write an axis in front of is left alone.
func TestAHandIsNamedRungFirst(t *testing.T) {
	for _, c := range []struct{ name, want string }{
		{"Form Three of a Kind", "Three of a Kind (Form)"},
		{"Card Pair", "Pair (Card)"},
		{"Elemental Full House", "Full House (Elemental)"},
		{"No Hand", "No Hand"},
	} {
		if got := axisToBack(c.name); got != c.want {
			t.Errorf("%q reads as %q, want %q", c.name, got, c.want)
		}
	}
}
