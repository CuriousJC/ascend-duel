package cards

// **The form vocabulary: stab, slash and crush, each in the stone the art already gives it.**
//
// It is a sibling of the element vocabulary in elementspans.go and of the metals in wash.go, and it
// is here for the reason both of those are: the review sheets and the two rasterizers have to agree
// about what color a word is, and this is the only windowless package all of them can reach.
//
// **The materials are not invented here — they are read off the catalogs.** `data/relics.json` and
// `data/runes.json` both brief their three-way form families in the same three stones: the Twisted
// rings are marble, malachite and moonstone for stab, slash and crush, and the Thornstave,
// Edgestave and Maulstave repeat it. The art has been saying which stone belongs to which form for
// as long as there has been art; this is the same answer in type.
//
// **These inks are mixed for the tooltip panel and nothing else may use them.** Every other color
// in this package is chosen against the off-white card surface; a panel is nearly black, which is
// why the metals are lifted toward WashLight before they are drawn there. These are mixed at the
// panel's weight to begin with, because a stone has no card-side color to lift from — there is no
// marble anything on the table.
//
// **Marble is the weak one and it is worth knowing why before retuning it.** Stab's stone is a warm
// off-white, and off-white is what an uncolored word on this panel already is: the cream below is
// as far from `tipInk` as it can go and still be marble. If it reads as plain, the answer is not a
// louder cream — it is the neutral form mark in `assets/form/` set inline before the word, which is
// the swatch the hue rule asks for and a picture rather than a sixth claim on the wheel.
//
// **Defend is deliberately absent.** The three forms above are an axis a hand can be counted on and
// the art gives each a stone; defend is a verb, blue already belongs to it, and there is no fourth
// stone in either catalog. A word here would be a color with nothing behind it.

import "image/color"

// The three stones, at the weight a nearly black panel needs.
var (
	// Marble: a warm off-white, pushed as far from the panel's plain ink as the stone allows.
	InkMarble = color.RGBA{R: 240, G: 222, B: 186, A: 255}

	// Malachite: the banded green, bright enough to hold at body size.
	InkMalachite = color.RGBA{R: 86, G: 200, B: 142, A: 255}

	// Moonstone: the milky blue sheen that floats under the cabochon.
	InkMoonstone = color.RGBA{R: 138, G: 186, B: 236, A: 255}
)

// FormWord is a word naming a form, and the stone the catalogs draw that form in.
type FormWord struct {
	Word string
	Ink  color.RGBA
}

// FormWords is the whole vocabulary, and the order it is offered in.
//
// **No two of them are a prefix of another**, so unlike the element words this list needs no
// longest-first sort — and a fourth entry that did would have to earn one.
var FormWords = []FormWord{
	{Word: "STAB", Ink: InkMarble},
	{Word: "SLASH", Ink: InkMalachite},
	{Word: "CRUSH", Ink: InkMoonstone},
}

// SplitForms colors every form word in a line that nothing has colored already.
//
// **Only an uncolored segment is looked at**, exactly as SplitWash and SplitName are: a word cannot
// be two things, and that rule is what makes the order of the passes at the call site a fact rather
// than a preference.
//
// **Whole words only and case-folded**, so a relic writing "Every slash card" and a tooltip writing
// "SLASH" share one entry and SLASHED is not lit.
func SplitForms(segs []Segment) []Segment {
	for _, f := range FormWords {
		var next []Segment
		for _, seg := range segs {
			next = append(next, splitForm(seg, f)...)
		}
		segs = next
	}
	return segs
}

// splitForm cuts one segment around one form word. It finds every occurrence rather than the first,
// because a relic's sentence can name its form twice and a half-colored line is worse than a plain
// one — the reader learns the color means nothing.
func splitForm(seg Segment, f FormWord) []Segment {
	if seg.Ink.A != 0 || f.Word == "" {
		return []Segment{seg}
	}

	var out []Segment
	rest := seg.Text
	for {
		i := indexWholeWord(rest, f.Word)
		if i < 0 {
			break
		}
		if before := rest[:i]; before != "" {
			out = append(out, Segment{Text: before})
		}
		out = append(out, Segment{Text: rest[i : i+len(f.Word)], Ink: f.Ink})
		rest = rest[i+len(f.Word):]
	}
	if rest != "" || len(out) == 0 {
		out = append(out, Segment{Text: rest})
	}
	return out
}
