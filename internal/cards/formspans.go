package cards

// **The form vocabulary: stab, slash and crush, each in the material the palette gives it.**
//
// It is a sibling of the element vocabulary in elementspans.go and of the metals in wash.go, and it
// is here for the reason both of those are: the review sheets and the two rasterizers have to agree
// about what color a word is, and this is the only windowless package all of them can reach.
//
// **The three materials are the last three ramps of `docs/art/palette.json`** — ivory for stab,
// steel for slash, granite for crush — and the ink below is each one's `core`. The same three
// materials are what `assets/texture/` is a picture of, so a word's color and a word's grain are
// one decision read two ways.
//
// **A word carries a Texture as well as an Ink, and only a panel can use it.** This package draws a
// card face into a plain Go image with no graphics context, so it names the texture and paints the
// flat ink; `internal/systems` is where a name becomes a tile behind the glyphs. That split is the
// one `systems.ArtMark` and `ArtMarkImage` already draw.
//
// **These are mixed for a dark panel and nothing else may use them.** Every other color in this
// package is chosen against the off-white card surface. Granite is the one to look at first: it is
// the darkest of the three and a panel is nearly black, so it is the word most likely to sink into
// its own ground.
//
// **Defend is deliberately absent.** The three forms above are an axis a hand can be counted on and
// the palette gives each a material; defend is a verb, blue already belongs to it, and there is no
// fourth ramp. A word here would be a color with nothing behind it.

import "image/color"

// The three materials, at their palette core.
var (
	// Ivory: a warm off-white, the palest of the three and stab's.
	InkIvory = color.RGBA{R: 220, G: 207, B: 174, A: 255}

	// Steel: the cool gray of a brushed blade, slash's.
	InkSteel = color.RGBA{R: 157, G: 168, B: 176, A: 255}

	// Granite: the dark speckled stone a maul is, crush's.
	InkGranite = color.RGBA{R: 107, G: 109, B: 112, A: 255}
)

// The asset key each material's tile is filed under. **Named rather than derived from the word**,
// so a material can be re-cut or renamed without every form word having to agree about the
// filename it produces.
const (
	TextureIvory   = "texture-ivory"
	TextureSteel   = "texture-steel"
	TextureGranite = "texture-granite"
)

// FormWord is a word naming a form, and the stone the catalogs draw that form in.
type FormWord struct {
	Word string
	Ink  color.RGBA

	// Texture is the asset key of the material this word is set in, for a drawing that can composite
	// one. **Empty means there is none and the Ink is the whole answer**, which is what every rarity
	// word is and what a card face sees whatever the word.
	Texture string
}

// FormWords is the whole vocabulary, and the order it is offered in.
//
// **No two of them are a prefix of another**, so unlike the element words this list needs no
// longest-first sort — and a fourth entry that did would have to earn one.
var FormWords = []FormWord{
	{Word: "STAB", Ink: InkIvory, Texture: TextureIvory},
	{Word: "SLASH", Ink: InkSteel, Texture: TextureSteel},
	{Word: "CRUSH", Ink: InkGranite, Texture: TextureGranite},
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
		out = append(out, Segment{Text: rest[i : i+len(f.Word)], Ink: f.Ink, Texture: f.Texture})
		rest = rest[i+len(f.Word):]
	}
	if rest != "" || len(out) == 0 {
		out = append(out, Segment{Text: rest})
	}
	return out
}
