package ui

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// Every word on the new-run dialog is set in the figure glyphs, and a line the set does not cover
// quietly falls back to the font — so a new line, or a character dropped from the set, would make
// the dialog two typefaces with nothing failing. This holds each line to the set and to the box.
func TestTheNewRunDialogIsSetInFiguresThatFit(t *testing.T) {
	lines := []struct {
		str    string
		height float64
	}{
		{"NEW RUN", seedTitleHeight},
		{"ACHIEVEMENTS ENABLED", seedCheckHeight},
		{"START ON THIS CODE, OR TURN THE WHEELS TO CHOOSE ONE.", seedLineHeight},
		{"THE JOURNEY IN PROGRESS WILL BE LOST.", seedLineHeight},
	}
	for _, l := range lines {
		if !systems.FigureCovers(l.str) {
			t.Errorf("%q is not covered by the figure set, so it would draw in the font", l.str)
			continue
		}
		if w := systems.MeasureFigure(l.str, l.height); w > seedDialogWidth-2*40 {
			t.Errorf("%q is %.0f pixels wide at %v tall, which does not fit a %d-wide box", l.str, w, l.height, seedDialogWidth)
		}
	}

	// Every character a wheel can land on has a glyph: a full turn of one wheel visits them all.
	code := "000000"
	for i := int64(0); i < seeds.Base; i++ {
		if !systems.FigureCovers(code[:1]) {
			t.Errorf("the wheel character %q has no figure glyph", code[:1])
		}
		code = seeds.Step(code, 0, 1)
	}
}
