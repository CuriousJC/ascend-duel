package cards

import "testing"

// **The three form words come out in their three stones**, which is the whole of what this
// vocabulary promises. Written against the exported inks rather than against literal colors, so
// retuning marble is one edit rather than two.
func TestEachFormWordTakesItsOwnStone(t *testing.T) {
	for _, tc := range []struct {
		line string
		want [3]int // spans, index of the colored one, and nothing else
	}{
		{"STAB, 1 AP", [3]int{2, 0, 0}},
		{"Every slash card lands twice", [3]int{3, 1, 0}},
	} {
		got := SplitForms([]Segment{{Text: tc.line}})
		if len(got) != tc.want[0] {
			t.Errorf("%q cut into %d segments, want %d: %v", tc.line, len(got), tc.want[0], got)
			continue
		}
		if got[tc.want[1]].Ink.A == 0 {
			t.Errorf("%q left its form word uncolored: %v", tc.line, got)
		}
	}
}

// **A word a form word merely starts is not lit.** SLASHED is not a slash, and a vocabulary that
// matched inside words would color half of it and teach the player the color means nothing.
func TestAFormWordInsideAnotherWordIsNotLit(t *testing.T) {
	for _, line := range []string{"SLASHED", "crushing", "stabbing"} {
		got := SplitForms([]Segment{{Text: line}})
		for _, seg := range got {
			if seg.Ink.A != 0 {
				t.Errorf("%q lit a run it should not have: %v", line, got)
			}
		}
	}
}

// **A segment the element vocabulary already claimed is left alone**, which is the rule that makes
// the order of the passes a fact rather than a preference. A word cannot be two things.
func TestAColoredSegmentIsNotRecolored(t *testing.T) {
	claimed := Segment{Text: "SLASH", Ink: BorderOf(Fire)}
	got := SplitForms([]Segment{claimed})
	if len(got) != 1 || got[0].Ink != claimed.Ink {
		t.Errorf("a claimed segment was recut: %v", got)
	}
}

// **Every form the catalogs draw in a stone has an entry, and nothing else does.** Defend is
// deliberately absent — see the file header — so this fails on a fourth entry arriving without
// the argument for it.
func TestTheFormVocabularyIsTheThreeAttackForms(t *testing.T) {
	want := map[string]bool{"STAB": true, "SLASH": true, "CRUSH": true}
	if len(FormWords) != len(want) {
		t.Fatalf("the vocabulary is %d words, want %d: %v", len(FormWords), len(want), FormWords)
	}
	for _, f := range FormWords {
		if !want[f.Word] {
			t.Errorf("%q is in the form vocabulary and should not be", f.Word)
		}
		if f.Ink.A == 0 {
			t.Errorf("%q has no ink", f.Word)
		}
	}
}
