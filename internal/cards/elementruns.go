package cards

// The element vocabulary: which words in a piece of prose name something with a colour, and what
// colour that is.
//
// **One table, and it lives here because this is the only windowless package both readers share**
// *(owner's call, 2026-09-08)*. A card face is set by this package and everything else on screen by
// Ebitengine's `text/v2` in internal/screens — two rasterisers with nothing in common — and the
// nine review sheets under `tools/` build their own `Spec` on purpose, so they are a third reader.
// The only way none of the three can disagree about what colour arcane is, is for all of them to
// ask one table; and a table anywhere above this package would be one the sheets cannot reach
// without linking a window.
//
// **It is the one place this package reads `data/`.** The arrow points down like every other and
// `statuses.json` was already carrying presentation the engine ignores — `Badge` is the precedent —
// so a status's colour and the word for it sit beside its picture rather than in a second file.
// What this package still does not learn is anything about how a round resolves: a status is a name
// and a colour here, and nothing asks what it does.

import (
	"image/color"
	"sort"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
)

// elementWords is every word that names an element or one of its statuses, longest first.
//
// **Longest first is load-bearing**, because splitRuns lets the first run to claim a position keep
// it: BURN offered before BURNING would take the front of the word and leave ING in the default
// ink.
//
// **Three words per status and one per element**, all read off `statuses.json` rather than typed
// here: the status's Name (BURNING), its Verb (BURN), and the element it belongs to. A status
// shipping without an Element or a Verb fails a test rather than quietly going uncoloured — see
// TestEveryStatusNamesAnElement.
var elementWords = buildElementWords()

// elementWord is one coloured term, held lower case. Case is not part of the match and a text keeps
// its own spelling, so "Fire" on a relic and "FIRE" on a worm both colour and neither is rewritten.
type elementWord struct {
	word string
	ink  color.RGBA
}

func buildElementWords() []elementWord {
	var out []elementWord
	add := func(word string, e Element) {
		if word == "" {
			return
		}
		out = append(out, elementWord{word: strings.ToLower(word), ink: BorderOf(e)})
	}

	// **The five, not Elements().** Basic is the absence of an element and its grey is what an
	// uncoloured word already looks like, so a vocabulary entry for it would spend a highlight seat
	// to change nothing. Relic is not an element at all.
	for _, e := range []Element{Fire, Ice, Lightning, Earth, Arcane} {
		add(e.String(), e)
	}

	for _, s := range data.LoadStatuses() {
		e, ok := ParseElement(s.Element)
		if !ok {
			continue
		}
		add(s.Name, e)
		add(s.Verb, e)
	}

	sort.SliceStable(out, func(i, j int) bool {
		return len(out[i].word) > len(out[j].word)
	})
	return out
}

// ElementRuns is the runs of this text that name something with a colour, longest first — ready to
// hand to a Spec.
//
// **The match is ContainsRun, which is the same rule that paints them.** A run harvested by one
// rule and declined by another would be a colour that silently does nothing, which is the hardest
// kind of missing to notice.
func ElementRuns(text string) []TextRun {
	var out []TextRun
	for _, w := range elementWords {
		if ContainsRun(text, w.word) {
			out = append(out, TextRun{Run: w.word, Ink: w.ink})
		}
	}
	return out
}

// ElementHighlights is ElementRuns packed into the fixed array a Spec carries, and is what every
// caller building a card should use.
//
// **A text naming more terms than the array holds loses the last of them**, and
// TestEveryTextFitsItsHighlights is what catches that before a player sees a half-lit sentence.
// Truncating rather than growing is the posture the card's text band already takes: the strings are
// authored in this repo, so an overrun is an authoring mistake to fix and not a case to handle.
func ElementHighlights(text string) [MaxTextHighlights]TextRun {
	var out [MaxTextHighlights]TextRun
	copy(out[:], ElementRuns(text))
	return out
}

// ParseElement resolves the names written in `data/`, reporting failure rather than falling back to
// Basic — the same contract combat.ParseElement has, and for the same reason: a word quietly read as
// the wrong element is a colour nobody chose.
//
// **Relic is not parseable.** It is in elementNames because a relic card is drawn through this type,
// but no data file names it and a text saying "relic" means the jewellery.
func ParseElement(name string) (Element, bool) {
	for i, n := range elementNames {
		if n == name && Element(i) != Relic {
			return Element(i), true
		}
	}
	return Basic, false
}
