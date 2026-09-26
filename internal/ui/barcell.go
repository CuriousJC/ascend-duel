package ui

// barcell.go: one cell of a resource bar, drawn from authored art and stretched to fit.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// BarCell is which of the three authored cells to draw.
type BarCell int

const (
	BarCellEmpty BarCell = iota // a point still to be spent: a sunken socket
	BarCellSpent                // a point spent within what there was
	BarCellOver                 // a point spent past it, in the game's one red
)

// barCellKeys is where each cell's picture is in the asset map.
var barCellKeys = [...]string{
	BarCellEmpty: "barCellEmpty_png",
	BarCellSpent: "barCellSpent_png",
	BarCellOver:  "barCellOver_png",
}

// barCellCapPct is how much of a cell's source width each end cap takes, in percent. The art is
// authored at 256 wide with its caps in the outer 32 pixels at each side, and every column between
// them identical — which is what lets the middle stretch without changing.
const barCellCapPct = 12.5

// DrawBarCell draws one cell into r: the two caps at the height's own scale, and the middle
// stretched across whatever is left.
//
// **The caps scale with the height and never with the width**, so a cell is the same rounded
// shape at any width. A cell narrower than its two caps squeezes the caps rather than overlapping
// them — at that width it is a dot, and a dot is still countable.
//
// A missing picture draws nothing, which is the honest failure: a blank bar means a file that did
// not load rather than a count of zero.
func DrawBarCell(gs *state.GlobalState, dst *ebiten.Image, kind BarCell, r image.Rectangle) {
	src := gs.Assets[barCellKeys[kind]]
	if src == nil || r.Dx() <= 0 || r.Dy() <= 0 {
		return
	}
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	srcCap := int(float64(sw) * barCellCapPct / 100)

	capW := srcCap * r.Dy() / sh
	if 2*capW > r.Dx() {
		capW = r.Dx() / 2
	}
	mid := r.Dx() - 2*capW

	piece := func(sx0, sx1, dx, dw int) {
		if dw <= 0 || sx1 <= sx0 {
			return
		}
		sub := src.SubImage(image.Rect(sx0, 0, sx1, sh)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(dw)/float64(sx1-sx0), float64(r.Dy())/float64(sh))
		op.GeoM.Translate(float64(dx), float64(r.Min.Y))
		op.Filter = ebiten.FilterLinear
		dst.DrawImage(sub, op)
	}
	piece(0, srcCap, r.Min.X, capW)
	piece(srcCap, sw-srcCap, r.Min.X+capW, mid)
	piece(sw-srcCap, sw, r.Min.X+capW+mid, capW)
}
