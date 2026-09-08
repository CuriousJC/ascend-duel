package cards

import (
	"image"
	"math"
	"testing"
)

// TestTheGridCoversTheWholeCardExactlyOnce. A square missing is a hole that neither face is drawn
// in; a square counted twice is a patch drawn at double alpha over whichever face is under it.
// Both are invisible in a still and obvious in motion, which is exactly the kind of thing a test
// should be holding.
func TestTheGridCoversTheWholeCardExactlyOnce(t *testing.T) {
	const w, h = 162, 224
	seen := make([]int, w*h)
	for _, c := range DissolveCells(w, h, MarkSeed("Jab")) {
		for y := c.Rect.Min.Y; y < c.Rect.Max.Y; y++ {
			for x := c.Rect.Min.X; x < c.Rect.Max.X; x++ {
				seen[y*w+x]++
			}
		}
	}
	for i, n := range seen {
		if n != 1 {
			t.Fatalf("pixel (%d,%d) covered %d times, want exactly 1", i%w, i/w, n)
		}
	}
}

// TestTheDissolveStartsAtZeroAndFinishesAtOne. The delays are stretched to fill the range on
// purpose — see DissolveCells. Without it a morph would be over before its own clock was, and the
// ends of the window would be dead air on screen.
func TestTheDissolveStartsAtZeroAndFinishesAtOne(t *testing.T) {
	for _, name := range []string{"Jab", "Ward", "Thrust", "Smash", "Guard"} {
		cells := DissolveCells(162, 224, MarkSeed(name))
		if len(cells) == 0 {
			t.Fatalf("%s: no cells", name)
		}
		lo, hi := 1.0, 0.0
		for _, c := range cells {
			lo, hi = math.Min(lo, c.Delay), math.Max(hi, c.Delay)
		}
		if lo > 1e-9 || hi < 1-1e-9 {
			t.Errorf("%s: delays span %.4f..%.4f, want the full 0..1", name, lo, hi)
		}
	}
}

// TestNeighbouringSquaresGoTogether is the property that makes this a dissolve rather than static.
//
// **It is the whole reason the delays come off a lattice** — independent per-square delays would
// pass every other test in this file and read as television noise. Adjacent squares should be far
// closer in time than two squares picked at random.
func TestNeighbouringSquaresGoTogether(t *testing.T) {
	const w, h = 162, 224
	cells := DissolveCells(w, h, MarkSeed("Jab"))
	cols := (w + DissolveCellSize - 1) / DissolveCellSize
	rows := (h + DissolveCellSize - 1) / DissolveCellSize

	var near, nearN float64
	for row := 0; row < rows; row++ {
		for col := 0; col+1 < cols; col++ {
			a := cells[row*cols+col].Delay
			b := cells[row*cols+col+1].Delay
			near += math.Abs(a - b)
			nearN++
		}
	}
	near /= nearN

	// Two squares with no relation average a third of the range apart, which is what a uniform
	// spread gives. Neighbours have to be well inside that.
	const unrelated = 1.0 / 3.0
	if near > unrelated/2 {
		t.Errorf("neighbouring squares differ by %.3f on average; unrelated ones differ by about "+
			"%.3f, so the pattern is closer to noise than to patches", near, unrelated)
	}
}

// TestTheSameCardComesApartTheSameWay. A pattern that shifted between two draws would move on the
// frame a morph was interrupted, re-entered or resized — the same reason a break's cracks are
// derived from the card's name rather than rolled.
func TestTheSameCardComesApartTheSameWay(t *testing.T) {
	first := DissolveCells(162, 224, MarkSeed("Jab"))
	again := DissolveCells(162, 224, MarkSeed("Jab"))
	if len(first) != len(again) {
		t.Fatalf("two calls gave %d and %d cells", len(first), len(again))
	}
	for i := range first {
		if first[i] != again[i] {
			t.Fatalf("cell %d differs: %+v then %+v", i, first[i], again[i])
		}
	}

	other := DissolveCells(162, 224, MarkSeed("Ward"))
	same := 0
	for i := range first {
		if first[i].Delay == other[i].Delay {
			same++
		}
	}
	if same == len(first) {
		t.Error("two different cards come apart identically; the pattern is not reading the seed")
	}
}

// TestADegenerateCardHasNoGrid, rather than panicking or dividing by zero. Nothing asks for one
// today; a card size arriving as zero is the kind of thing a layout change produces.
func TestADegenerateCardHasNoGrid(t *testing.T) {
	for _, size := range []image.Point{{X: 0, Y: 10}, {X: 10, Y: 0}, {X: -3, Y: -3}} {
		if cells := DissolveCells(size.X, size.Y, 7); cells != nil {
			t.Errorf("%v gave %d cells, want none", size, len(cells))
		}
	}
}
