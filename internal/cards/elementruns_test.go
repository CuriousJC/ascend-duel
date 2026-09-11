package cards

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// The element vocabulary: that every status can be coloured, that no authored text names more
// coloured terms than a card can carry, and that the words are offered longest first.

func TestEveryStatusNamesAnElement(t *testing.T) {
	// A status shipping without an Element or a Verb goes uncoloured while every other one is lit,
	// which reads as a rendering fault rather than as a missing field. This is the counterpart of
	// TestEveryStatusElementHasABadge in internal/screens, on the other presentation axis.
	for _, s := range data.LoadStatuses() {
		if _, ok := ParseElement(s.Element); !ok {
			t.Errorf("status %q has Element %q, which is not an element", s.StatusRecord, s.Element)
		}
		if s.Verb == "" {
			t.Errorf("status %q has no Verb, so prose saying it happens rather than stands "+
				"goes uncoloured", s.StatusRecord)
		}
		if s.Name == "" {
			t.Errorf("status %q has no Name", s.StatusRecord)
		}
	}
}

func TestEveryStatusWordIsColoured(t *testing.T) {
	// The words themselves, through the real matcher: a status's Name and its Verb both have to
	// come back as runs, or half the relic catalogue's sentences colour and half do not.
	for _, s := range data.LoadStatuses() {
		for _, word := range []string{s.Name, s.Verb} {
			found := false
			for _, r := range ElementRuns("attacks " + word + " the target.") {
				if strings.EqualFold(r.Run, word) {
					found = true
				}
			}
			if !found {
				t.Errorf("%q, from status %q, is not in the coloured vocabulary", word, s.StatusRecord)
			}
		}
	}
}

func TestTheFiveElementsAreColouredInBothCases(t *testing.T) {
	// Relics write "Fire" and worms write "FIRE". Both colour, through one vocabulary entry, because
	// the match ignores case on both sides.
	for _, e := range []Element{Fire, Ice, Lightning, Earth, Arcane} {
		for _, word := range []string{strings.ToUpper(e.String()), e.String()} {
			runs := ElementRuns("CARD BECOMES " + word)
			if len(runs) != 1 {
				t.Errorf("%q produced %v, want one run", word, runs)
				continue
			}
			if runs[0].Ink != BorderOf(e) {
				t.Errorf("%q is not drawn in %v's colour", word, e)
			}
		}
	}
}

func TestBasicAndRelicAreNotColouredWords(t *testing.T) {
	// Basic is the absence of an element and its grey is what an uncoloured word already looks
	// like. Relic is not an element at all, and a text saying "relic" means the jewellery.
	for _, word := range []string{"BASIC", "RELIC"} {
		if runs := ElementRuns("CARD BECOMES " + word); len(runs) != 0 {
			t.Errorf("%q was coloured: %v", word, runs)
		}
	}
}

func TestAnElementInsideALongerWordIsNotColoured(t *testing.T) {
	// ICE is inside SLICE, and every parasite that turns a card into a Slice says so. A substring
	// match would light three letters of a card's name in the ice blue.
	if runs := ElementRuns("CARD BECOMES SLICE"); len(runs) != 0 {
		t.Errorf("a word inside SLICE was coloured: %v", runs)
	}
}

func TestTheVocabularyIsLongestFirst(t *testing.T) {
	// splitRuns lets the first run to claim a position keep it, so BURNING has to be offered before
	// BURN or the longer word draws as a coloured BURN and a default-ink ING.
	for i := 1; i < len(elementWords); i++ {
		if len(elementWords[i-1].word) < len(elementWords[i].word) {
			t.Fatalf("the vocabulary is not longest first: %q before %q",
				elementWords[i-1].word, elementWords[i].word)
		}
	}
}

func TestEveryTextFitsItsHighlights(t *testing.T) {
	// Spec carries a fixed array, so a text naming more coloured terms than it holds loses the last
	// of them silently. The strings are authored in this repo, so this is the place that says an
	// author has run out of room rather than a player finding a half-lit sentence.
	check := func(what, text string) {
		t.Helper()
		if n := len(ElementRuns(text)); n > MaxTextHighlights {
			t.Errorf("%s names %d coloured terms and a card holds %d: %q",
				what, n, MaxTextHighlights, text)
		}
	}

	for key, w := range data.LoadWorms() {
		check("worm "+key, w.Text)
	}
	for key, p := range data.LoadParasites() {
		check("parasite "+key, p.Text)
	}
	for key, st := range data.LoadStones() {
		check("stone "+key, st.Text)
	}
	for _, p := range data.LoadPotions() {
		check("potion "+p.Name, p.Text)
	}
	for key, r := range data.LoadRelics() {
		check("relic "+key, r.Text)
	}
	for _, s := range data.LoadStatuses() {
		check("status "+s.StatusRecord, s.Text)
	}
}
