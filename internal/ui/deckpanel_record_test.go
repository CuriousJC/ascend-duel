package ui

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// **The run's record fits under the grid**: it starts below the last row, the tonic cards stand
// inside it, and the figures' last column stops short of the band's right edge.
func TestTheRunRecordFitsUnderTheGrid(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: 1920, ScreenHeight: 1080}
	band := deckRecordRect(gs)

	_, _, top := deckGridRegion(gs)
	if gridBottom := int(top) + deckRowCount*(cards.Mini.Height+deckRowGap); band.Min.Y <= gridBottom {
		t.Errorf("the band starts at %d, inside the grid", band.Min.Y)
	}
	if seat := tonicSeats(band, 1)[0]; seat.Max.Y > band.Max.Y {
		t.Errorf("a tonic card ends at %d, below the band's floor at %d", seat.Max.Y, band.Max.Y)
	}
	rows := deckColumnHeadDrop + deckRecordStatRows*deckRecordLinePit
	if band.Min.Y+rows > band.Max.Y {
		t.Errorf("the figures need %d of the band's %d", rows, band.Dy())
	}
	cols := (len(deckRecordStats(nil)) + deckRecordStatRows - 1) / deckRecordStatRows
	right := band.Min.X + deckRecordTonicsW + deckRecordStatGap + cols*deckRecordStatColW + (cols-1)*deckRecordStatGap
	if right > band.Max.X {
		t.Errorf("the figures run to %d, past the band's edge at %d", right, band.Max.X)
	}
}

// **However many tonics a run drinks, they stay inside the tonics' share** — they overlap instead.
func TestManyTonicsOverlapRatherThanRunOn(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: 1920, ScreenHeight: 1080}
	band := deckRecordRect(gs)
	for _, n := range []int{1, 4, 12} {
		seats := tonicSeats(band, n)
		if last := seats[n-1]; last.Max.X > band.Min.X+deckRecordTonicsW {
			t.Errorf("%d tonics end at %d, past their share at %d", n, last.Max.X, band.Min.X+deckRecordTonicsW)
		}
	}
}
