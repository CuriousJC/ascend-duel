package cards

// **Writing a word in an upgrade's own wash**, which is the one place in the game a single word is
// set in more than one colour.
//
// It is here rather than in `internal/screens` for `ElementRuns`' reason and `internal/carddesc`'s:
// the review sheets print the same titles the game does, and a tool cannot import a package that
// links Ebitengine. Two rasterisers draw this game's words and they share no code — so what is
// shared is the vocabulary, and this is one more entry in it.

import (
	"image"
	"image/color"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/systems"
)

// SplitWash cuts one segment around a word, replacing that word with one segment per letter, each in
// the colour the upgrade's wash is at that point across a card.
//
// **CHROMATIC is what wanted it** *(owner's call, 2026-09-09)*. `carddesc.Chromatic` exists because
// the wheel has no hue left for "all of them", and it took the panel's plain ink on the argument
// that the *word* was the signal — which is true of the word and untrue of the panel around it: a
// card washed in five colours, explained by a title in grey, is the one tooltip that does not look
// like the thing under the cursor. Five colours across nine letters is not a sixth hue being
// claimed. It is the five.
//
// **The colours are sampled out of the wash rather than listed anywhere.** `systems.UpgradeInk` is
// what paints the card, so a repaint of the ink moves the word with it; writing the five element
// inks out instead would be a second palette that agrees with the card only until one is retuned.
//
// **Only an uncoloured segment is looked at.** A segment that already carries an ink was coloured by
// the element vocabulary, and a word cannot be two things — which is what makes the order of the two
// passes at the call site a rule rather than a preference.
//
// **Whole words only**, like ContainsRun, so CHROMATICS is not lit.
func SplitWash(seg Segment, word string, u systems.Upgrade) []Segment {
	if seg.Ink.A != 0 || word == "" {
		return []Segment{seg}
	}

	i := indexWholeWord(seg.Text, word)
	if i < 0 {
		return []Segment{seg}
	}

	var out []Segment
	if before := seg.Text[:i]; before != "" {
		out = append(out, Segment{Text: before})
	}
	out = append(out, WashSegments(word, u)...)
	if after := seg.Text[i+len(word):]; after != "" {
		out = append(out, Segment{Text: after})
	}
	return out
}

// WashSegments is one word as one segment per letter, across the upgrade's ink.
//
// **Per letter, not per band.** A segment boundary can only fall between letters, so a word shorter
// than the bands still reads as a spectrum rather than as two letters in two colours. A word with
// nothing to sample comes back uncoloured rather than black.
func WashSegments(word string, u systems.Upgrade) []Segment {
	ink := systems.UpgradeInk(u)
	letters := []rune(word)
	if ink == nil || len(letters) == 0 {
		return []Segment{{Text: word}}
	}

	out := make([]Segment, 0, len(letters))
	for i, letter := range letters {
		at := (float64(i) + 0.5) / float64(len(letters))
		out = append(out, Segment{Text: string(letter), Ink: washInk(ink, at)})
	}
	return out
}

// WashLift is how far a sampled colour is moved toward the light, and toward what.
//
// **The wash was mixed for an off-white card and a tooltip panel is nearly black.** The bands are
// chosen to read as colour *over paper*; the darkest of them sits at about the panel's own surface in
// weight, so a title set in it reads as dim rather than as coloured. This is the light-ground rule
// the other way round — on a dark panel the move is toward white, the way ColorToward moves
// everything on the table toward the table.
//
// **A third of the way**, which is enough to bring the darkest band up to the weight the rest of a
// title is set at and little enough that every band is still recognisably the colour on the card.
var (
	WashLight   = color.RGBA{R: 245, G: 242, B: 236, A: 255}
	WashLiftPct = 33
)

// washInk is the ink's colour a fraction of the way across it, lifted for a dark panel.
//
// **Sampled down the middle**, because the wildcard's ink is bands running top to bottom and the
// row is therefore the one axis carrying no information. systems.UpgradeInk hands the square out
// whole precisely so a caller can sample it against its own rectangle.
func washInk(ink *image.RGBA, at float64) color.RGBA {
	b := ink.Bounds()
	x := b.Min.X + int(at*float64(b.Dx()))
	if x >= b.Max.X {
		x = b.Max.X - 1
	}
	return systems.ColorToward(ink.RGBAAt(x, b.Min.Y+b.Dy()/2), WashLight, WashLiftPct)
}

// indexWholeWord is where word starts in text as a whole word, or -1. **Case-folded, like
// ContainsRun**, so a caller writing Chromatic and a card writing CHROMATIC share one entry.
func indexWholeWord(text, word string) int {
	folded, want := strings.ToLower(text), strings.ToLower(word)
	for at := 0; at+len(want) <= len(folded); at++ {
		if folded[at:at+len(want)] == want && wholeWord(folded, at, len(want)) {
			return at
		}
	}
	return -1
}

// WashWord is a word in a tooltip that names an upgrade, and the upgrade it names.
type WashWord struct {
	Word    string
	Upgrade systems.Upgrade
}

// MetalWords is the two words a tooltip writes to name a card's metal, and is this package's copy
// of `carddesc.Gold` and `carddesc.Silver`.
//
// **A knowingly accepted duplicate, held by a test.** `internal/carddesc` writes the words and this
// package colours them, and the arrow between the two goes neither way: carddesc must not import a
// package that reaches Ebitengine, and this one must not learn what a `combat.Card` is.
// TestTheMetalWordsAgree in `internal/screens` — the one package that imports both — is what stops
// carddesc writing GOLD while this looks for GOLDEN, which would be a word never lit and nothing
// failing.
var MetalWords = []WashWord{
	{Word: "GOLD", Upgrade: systems.UpgradeGolden},
	{Word: "SILVER", Upgrade: systems.UpgradeSilver},
}

// SplitName cuts one segment around a word, replacing that word with a single segment in the one
// colour the upgrade's ink names it — GOLD in gold, on a dark panel.
//
// **One colour rather than SplitWash's spectrum, because the metals are one colour.** The wildcard's
// ink is five bands and reads as a spectrum only if the letters are cut apart; a sheen is one hue
// with a light running over it, and a letter-by-letter gold would be a gradient nobody could see
// spent on nine runs. Which of the two a word takes is a fact about its ink, so the two cuts are
// two functions rather than one with a flag.
//
// **Lifted like SplitWash**, because a tooltip panel is nearly black and these inks were mixed to
// wash an off-white card. See WashLight.
func SplitName(seg Segment, word string, u systems.Upgrade) []Segment {
	if seg.Ink.A != 0 || word == "" {
		return []Segment{seg}
	}

	i := indexWholeWord(seg.Text, word)
	if i < 0 {
		return []Segment{seg}
	}

	ink := systems.UpgradeInk(u)
	if ink == nil {
		return []Segment{seg}
	}

	var out []Segment
	if before := seg.Text[:i]; before != "" {
		out = append(out, Segment{Text: before})
	}
	out = append(out, Segment{Text: seg.Text[i : i+len(word)], Ink: washInk(ink, 0.5)})
	if after := seg.Text[i+len(word):]; after != "" {
		out = append(out, Segment{Text: after})
	}
	return out
}

// SplitMetals is SplitName over every word in MetalWords, which is what a caller colouring a line of
// panel text wants — one call rather than a loop each site has to get the order of.
func SplitMetals(segs []Segment) []Segment {
	for _, w := range MetalWords {
		var next []Segment
		for _, seg := range segs {
			next = append(next, SplitName(seg, w.Word, w.Upgrade)...)
		}
		segs = next
	}
	return segs
}
