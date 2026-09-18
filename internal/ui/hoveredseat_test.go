package ui

import (
	"image"
	"testing"
)

// **The card on top is the last one drawn, and that is what a point lands on.**
//
// This is the bug the function exists for, reported 2026-09-18: with a row packed tight enough to
// overlap, a forward walk stops at the first seat covering the cursor — which is the card *behind*
// the one the player can see. The worn relics, the sack and the build band each had their own
// forward loop, so a tooltip named one card while the row raised another.
func TestAPointLandsOnTheLastSeatDrawnOverIt(t *testing.T) {
	// Five cards 200 wide at a pitch of 50: every seat but the last is four fifths covered.
	seat := func(i int) image.Rectangle { return image.Rect(i*50, 0, i*50+200, 280) }

	for _, c := range []struct {
		x, want int
		what    string
	}{
		{x: 10, want: 0, what: "the sliver only the first card reaches"},
		{x: 60, want: 1, what: "where the first two overlap"},
		{x: 210, want: 4, what: "under every card at once"},
		{x: 350, want: 4, what: "the last card's own tail"},
		{x: -1, want: -1, what: "left of the row"},
		{x: 500, want: -1, what: "right of the row"},
	} {
		if got := HoveredSeat(image.Pt(c.x, 100), 5, seat); got != c.want {
			t.Errorf("%s (x=%d): seat %d, want %d", c.what, c.x, got, c.want)
		}
	}
}

// **A seat that draws nothing answers an empty rectangle and is walked past**, which is how the
// shop's sold relics and drunk potions stay out of the way. A seat that blocked a tooltip while
// showing nothing would be a worse version of the bug this function fixes.
func TestAnEmptySeatDoesNotSwallowTheCardBehindIt(t *testing.T) {
	seat := func(i int) image.Rectangle {
		if i == 2 {
			return image.Rectangle{}
		}
		return image.Rect(i*50, 0, i*50+200, 280)
	}

	// x=210 is under seats 1..4; seat 4 answers, and with 3 and 4 gone it falls to 1 rather than 2.
	if got := HoveredSeat(image.Pt(210, 100), 5, seat); got != 4 {
		t.Errorf("the top seat answered %d, want 4", got)
	}
	if got := HoveredSeat(image.Pt(210, 100), 3, seat); got != 1 {
		t.Errorf("with the top seat empty the row answered %d, want 1", got)
	}
}

// A row with nothing in it has no seat to land on, rather than panicking on an index.
func TestAnEmptyRowLandsOnNothing(t *testing.T) {
	if got := HoveredSeat(image.Pt(0, 0), 0, func(int) image.Rectangle {
		t.Fatal("an empty row asked for a seat")
		return image.Rectangle{}
	}); got != -1 {
		t.Errorf("an empty row answered %d, want -1", got)
	}
}
