package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/carddesc"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// **CHROMATIC is written in the card's own colours**, which is the whole point of the word — a card
// washed in five colours explained by a grey title is the one tooltip that does not look like the
// thing under the cursor.
func TestTheChromaticTitleIsWrittenInTheWash(t *testing.T) {
	wild := combat.Plain(combat.Skewer).SetRider(combat.Rider{Kind: combat.RiderWildElement})
	runs := tipLine(carddesc.Title(wild))

	if got := runs.Text(); got != carddesc.Chromatic+" SKEWER" {
		t.Fatalf("the title reads %q, want %q", got, carddesc.Chromatic+" SKEWER")
	}

	inks := map[string]bool{}
	for _, run := range runs {
		if run.Ink.A != 0 {
			inks[string([]byte{run.Ink.R, run.Ink.G, run.Ink.B})] = true
		}
	}
	// Five bands, and a run boundary can only fall between letters — so a nine-letter word cannot
	// land on fewer than a few of them however the bands are drawn.
	if len(inks) < 3 {
		t.Errorf("CHROMATIC is written in %d colours, which does not read as a spectrum", len(inks))
	}
}

// **The rest of the line is left alone.** The word is lit, not the sentence around it — the same
// rule every element word in a tooltip is under.
func TestOnlyTheWordChromaticIsLit(t *testing.T) {
	runs := tipLine(carddesc.Chromatic + " SKEWER")

	lit := ""
	plain := ""
	for _, run := range runs {
		if run.Ink.A != 0 {
			lit += run.Text
			continue
		}
		plain += run.Text
	}
	if lit != carddesc.Chromatic {
		t.Errorf("the lit run is %q, want %q", lit, carddesc.Chromatic)
	}
	if plain != " SKEWER" {
		t.Errorf("the plain run is %q, want %q", plain, " SKEWER")
	}
}

// **A word that merely contains the letters is not lit**, which is what whole-word matching buys —
// the same posture cards.ElementRuns takes.
func TestAWordContainingChromaticIsNotLit(t *testing.T) {
	for _, line := range []string{"CHROMATICS", "ACHROMATIC"} {
		for _, run := range tipLine(line) {
			if run.Ink.A != 0 {
				t.Errorf("%q was lit at %q", line, run.Text)
			}
		}
	}
}

// **An ordinary line is still one run.** Most tooltip lines have nothing to colour and must be
// drawn exactly as they were before any of this existed.
func TestAPlainLineIsStillOneRun(t *testing.T) {
	if runs := tipLine("3 AP"); len(runs) != 1 || runs[0].Ink.A != 0 {
		t.Errorf("a plain line came back as %v", runs)
	}
}
