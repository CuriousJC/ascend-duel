package cards

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// The element vocabulary: that every element is colored, that no authored text names more colored
// terms than a card can carry, and that the words are offered longest first.

func TestTheFiveElementsAreColoredInBothCases(t *testing.T) {
	// Relics write "Fire" and essences write "FIRE". Both color, through one vocabulary entry, because
	// the match ignores case on both sides.
	for _, e := range []Element{Fire, Ice, Lightning, Earth, Arcane} {
		for _, word := range []string{strings.ToUpper(e.String()), e.String()} {
			runs := ElementSpans("CARD BECOMES " + word)
			if len(runs) != 1 {
				t.Errorf("%q produced %v, want one run", word, runs)
				continue
			}
			if runs[0].Ink != BorderOf(e) {
				t.Errorf("%q is not drawn in %v's color", word, e)
			}
		}
	}
}

func TestBasicAndRelicAreNotColoredWords(t *testing.T) {
	// Basic is the absence of an element and its gray is what an uncolored word already looks
	// like. Relic is not an element at all, and a text saying "relic" means the jewelry.
	for _, word := range []string{"BASIC", "RELIC"} {
		if runs := ElementSpans("CARD BECOMES " + word); len(runs) != 0 {
			t.Errorf("%q was colored: %v", word, runs)
		}
	}
}

func TestAnElementInsideALongerWordIsNotColored(t *testing.T) {
	// ICE is inside SLICE, and every rune that turns a card into a Slice says so. A substring
	// match would light three letters of a card's name in the ice blue.
	if runs := ElementSpans("CARD BECOMES SLICE"); len(runs) != 0 {
		t.Errorf("a word inside SLICE was colored: %v", runs)
	}
}

func TestTheVocabularyIsLongestFirst(t *testing.T) {
	// SplitSpans lets the first run to claim a position keep it, so a longer word has to be offered
	// before a shorter one inside it, or the longer word draws half colored.
	for i := 1; i < len(elementWords); i++ {
		if len(elementWords[i-1].word) < len(elementWords[i].word) {
			t.Fatalf("the vocabulary is not longest first: %q before %q",
				elementWords[i-1].word, elementWords[i].word)
		}
	}
}

func TestEveryTextFitsItsHighlights(t *testing.T) {
	// Spec carries a fixed array, so a text naming more colored terms than it holds loses the last
	// of them silently. The strings are authored in this repo, so this is the place that says an
	// author has run out of room rather than a player finding a half-lit sentence.
	check := func(what, text string) {
		t.Helper()
		if n := len(ElementSpans(text)); n > MaxTextHighlights {
			t.Errorf("%s names %d colored terms and a card holds %d: %q",
				what, n, MaxTextHighlights, text)
		}
	}

	for key, w := range data.LoadEssences() {
		check("essence "+key, w.Text)
	}
	for key, p := range data.LoadRunes() {
		check("rune "+key, p.Text)
	}
	for key, st := range data.LoadStones() {
		check("stone "+key, st.Text)
	}
	for key, r := range data.LoadRelics() {
		check("relic "+key, r.Text)
	}
}
