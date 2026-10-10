package ui

import (
	"reflect"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/session"
)

func TestSayFillsHolesWithTheirOwnInks(t *testing.T) {
	got := Say("You have {total}.", Slots{"total": Inked("12 vitae", session.InkVitae)})
	want := []session.LedgerSpan{
		{Text: "You have "},
		{Text: "12 vitae", Ink: session.InkVitae},
		{Text: "."},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestSayWritesInkedAndMarkedLiterals(t *testing.T) {
	got := Say("{mark:took} {vitae:Vitae}", nil)
	want := []session.LedgerSpan{
		{Text: "took", Mark: true},
		{Text: " "},
		{Text: "Vitae", Ink: session.InkVitae},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// A hole nobody filled is drawn as its own name, so the mistake is on screen rather than a gap.
func TestAnUnfilledHoleShowsItsName(t *testing.T) {
	if got := SayText("{who} falls", nil); got != "{who} falls" {
		t.Errorf("got %q", got)
	}
}

// Plain text either side of a plain slot is one run, so a sentence is not drawn in pieces.
func TestPlainRunsJoin(t *testing.T) {
	got := Say("{who} falls", Slots{"who": Plain("Duelist")})
	if len(got) != 1 || got[0].Text != "Duelist falls" {
		t.Errorf("got %+v", got)
	}
}

// A sentence in data/wording.json that nothing asks for is one the author is editing to no effect.
func TestEveryWordingKeyIsUsed(t *testing.T) {
	for section, keys := range wording {
		for name := range keys {
			if key := section + "." + name; !wordingAsked[key] {
				t.Errorf("wording.json: %q is in the file and nothing says it", key)
			}
		}
	}
}

func TestTemplateHolesSkipsInkedWords(t *testing.T) {
	got := templateHoles("{vitae:Vitae} for {per}, {gain} and {per}")
	if !reflect.DeepEqual(got, []string{"gain", "per"}) {
		t.Errorf("got %v", got)
	}
}

func TestAnUnclosedBraceIsText(t *testing.T) {
	if got := SayText("odd {brace", nil); got != "odd {brace" {
		t.Errorf("got %q", got)
	}
}
