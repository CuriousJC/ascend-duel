package cards

// **A card coming apart, so another one can come through it.**
//
// A mark says something happened *to* a card and leaves it the card it was (see mark.go). A
// dissolve is the other thing that can happen to a face: it stops being this card and starts
// being a different one. A worm eating a card's element, a parasite overwriting one, a ring
// turning every ice card in the hand to fire — all of them are one card replaced by another,
// and what this file holds is the pattern that replacement runs on.
//
// # It is a grid, and the grid is not the picture
//
// The face is cut into small squares, each with its own moment to go. Drawn, that reads as the
// card being eaten away in patches rather than as a grid switching off, because **the delays are
// spatially coherent**: they come from a smooth lattice, so neighbouring squares go at nearly the
// same time and the boundary between what is left and what has gone is a ragged edge that travels.
// Independent per-square delays would read as television static, which is a different — and much
// worse — sentence about what happened to the card.
//
// # One rasteriser today, and the geometry still lives here
//
// Unlike ShatterCracks there is no baked counterpart: a dissolve has no settled state, because the
// settled state is simply the other card. It is here anyway for two reasons. The pattern is derived
// from the card's *name* through the same MarkSeed and the same hasher the break uses, and a second
// copy of that derivation is a second answer to "what does this card look like when it comes
// apart". And it is geometry about a card face, which is this package's subject; internal/screens
// draws it, exactly as it draws the moving half of a break.
//
// **Derived, never rolled**, on the same terms as the crack pattern: it decides nothing, so putting
// it on a stream would be a cosmetic detail advancing a cursor a rule reads. It also means one card
// always comes apart the same way, which is what lets a morph be interrupted, resized or re-entered
// without the pattern shifting under it.

import "image"

// DissolveCellSize is the side of one square of the grid, in pixels.
//
// **Small enough that the edge reads as ragged and large enough that a card is not a thousand draw
// calls.** At 9 pixels a 200x280 card is 22 by 31 squares, which is 682 quads a frame per face —
// nothing to a GPU, and fine enough that the boundary between the old card and the new one looks
// torn rather than pixellated.
const DissolveCellSize = 9

// DissolveCell is one square of a card's face and when it goes.
//
// Delay is 0 for the first square to turn and 1 for the last, so a caller with a progress from 0
// to 1 can ask each square how far through its own change it is. Rect is relative to the card's
// top-left, like a Crack's points.
type DissolveCell struct {
	Rect  image.Rectangle
	Delay float64
}

// dissolveLattice is how many squares there are to a step of the noise grid. **This is the knob
// that decides whether the dissolve reads as patches or as static** — a step of one square is
// static, and a step the width of the card is a straight wipe. Four is a patch about the size of a
// card's cost column, which is the scale the eye reads as "eaten".
const dissolveLattice = 4

// DissolveCells cuts a card of this size into the squares it comes apart in, seeded by its name.
//
// The delays are stretched to fill 0..1 rather than used raw. Interpolated noise clusters toward
// the middle of its range, so unstretched delays would leave a dissolve that had barely started
// when the clock was a third gone and was already finished at two thirds — the transition would
// happen in the middle of its own window and the ends would be dead air.
func DissolveCells(w, h int, seed uint32) []DissolveCell {
	if w <= 0 || h <= 0 {
		return nil
	}
	cols := (w + DissolveCellSize - 1) / DissolveCellSize
	rows := (h + DissolveCellSize - 1) / DissolveCellSize

	// The lattice the delays are sampled off, one value per corner, plus a margin so the far edge
	// of the card is still between two corners rather than off the end.
	lw := cols/dissolveLattice + 2
	lh := rows/dissolveLattice + 2
	r := hasher(seed)
	lat := make([]float64, lw*lh)
	for i := range lat {
		lat[i] = r.unit()
	}

	cells := make([]DissolveCell, 0, cols*rows)
	lo, hi := 1.0, 0.0
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			// **A little per-square jitter on top of the smooth value**, so the travelling edge is
			// frayed rather than a clean contour. Too much and the patches dissolve back into
			// static; a sixth of the range is enough to break the line without breaking the shape.
			d := latticeAt(lat, lw, lh, col, row)*0.84 + r.unit()*0.16
			if d < lo {
				lo = d
			}
			if d > hi {
				hi = d
			}
			x, y := col*DissolveCellSize, row*DissolveCellSize
			cells = append(cells, DissolveCell{
				Rect:  image.Rect(x, y, minInt(x+DissolveCellSize, w), minInt(y+DissolveCellSize, h)),
				Delay: d,
			})
		}
	}

	if span := hi - lo; span > 0 {
		for i := range cells {
			cells[i].Delay = (cells[i].Delay - lo) / span
		}
	}
	return cells
}

// latticeAt is the smooth value under one square: bilinear between the four lattice corners around
// it. Bilinear rather than nearest because nearest would give square patches of exactly the lattice
// size, which reads as a grid of blocks going out — the thing the coherence is here to avoid.
func latticeAt(lat []float64, lw, lh, col, row int) float64 {
	fx := float64(col) / dissolveLattice
	fy := float64(row) / dissolveLattice
	x0, y0 := int(fx), int(fy)
	tx, ty := fx-float64(x0), fy-float64(y0)
	x1, y1 := minInt(x0+1, lw-1), minInt(y0+1, lh-1)

	at := func(x, y int) float64 { return lat[y*lw+x] }
	top := at(x0, y0) + (at(x1, y0)-at(x0, y0))*tx
	bot := at(x0, y1) + (at(x1, y1)-at(x0, y1))*tx
	return top + (bot-top)*ty
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
