package cards

// Marks on a card face: **a picture drawn over a finished card to say something happened to it.**
//
// This is the third way something can be said on a card and it is deliberately unlike the other
// two. The *face* — name, cost ticks, form mark, text — is what the card is. An *upgrade* is what
// the run has permanently made it, and it is art covering the face, laid under everything drawn
// on the card (see upgrade_art.go). A **mark** sits on top of the whole face and is about the
// card's situation rather than its identity: this one is being pointed at.
//
// **The mark is baked into the card image and cached with it**, and Spec.Mark is where a caller
// asks for one, so tools/cardsheet can show a marked card and the game and the sheet cannot
// disagree about what one looks like.
//
// **MarkSeed is the one hash a card's patterns are derived from** — the dissolve's lattice and the
// consumables' motes — so the same card always comes apart the same way. It is a function of the
// card's name, never a draw from a stream: it decides nothing. See CLAUDE.md and the randomness
// skill for the rule this is an explicit, argued exception to.

import (
	"image"
	"image/color"
)

// Mark is what has happened to a card, drawn over its finished face. **A set, not one value** — a
// card can be several things at once, and a bitmask is what says so without the caller having to
// rank them.
//
// **Append-only, and never serialized.** It is the same shape as combat.ConceptID; a file
// writing one down would be a file that means something else after the next bit is claimed. Being
// a mask, an insertion is worse here than elsewhere — every existing value would change meaning
// rather than only the ones after it — so new marks go on the end.
//
// **Order of application is fixed and lives in drawMark**, not in the caller. Two marks on one card
// have to compose the same way every time or the same pair of facts would draw two ways.
type Mark uint16

const (
	// MarkNone is every ordinary card, and it is the zero value so a caller that has never heard
	// of a mark gets an unmarked card.
	MarkNone Mark = 0

	// MarkHighlit is "this card, this one" — the tutorial pointing at what it is talking about.
	//
	// **It replaced a wireframe drawn round the card** *(owner's call, 2026-09-08)*. A red rectangle
	// outside a card is a thing on the screen near the card; a tinted card is the card itself
	// answering. It also composes, which a frame could not: a frame round a set of cards is a box,
	// and a box round non-adjacent cards includes the ones between them — the exact bug the
	// tutorial's matching-cards anchor had.
	MarkHighlit Mark = 1 << iota
)

// Has reports whether a mark set carries one.
func (m Mark) Has(bit Mark) bool { return m&bit != 0 }

// MarkSeed is the number a card's patterns are derived from: its name, hashed. **Exported so every
// caller derives the same pattern from the same card.**
func MarkSeed(name string) uint32 {
	// FNV-1a, written out rather than imported, because hash/fnv returns an interface and this is
	// called per card face.
	var h uint32 = 2166136261
	for i := 0; i < len(name); i++ {
		h ^= uint32(name[i])
		h *= 16777619
	}
	if h == 0 {
		h = 1
	}
	return h
}

// hasher is a tiny deterministic sequence for laying a pattern out. **Not a source of randomness in
// the game's sense** — see the file comment; it decides nothing and advances no stream.
type hasher uint32

func (h *hasher) next() uint32 {
	x := uint32(*h)
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	*h = hasher(x)
	return x
}

func (h *hasher) unit() float64 { return float64(h.next()%10000) / 10000 }

// drawMark paints a card's marks over its finished face, and is the last thing a face has done to
// it.
//
// **They are drawn over everything, the border included**: what a mark says is about the card as a
// whole. A second mark's order against this one is fixed here, so one pair of facts draws one way.
func drawMark(dst *image.RGBA, mark Mark, name string, w, h, radius int) {
	if mark.Has(MarkHighlit) {
		washInside(dst, w, h, radius, HighlightInk, highlightWash)
	}
}

// HighlightInk is the color MarkHighlit washes a card in.
//
// **It is the tutorial's own red and it is exported so there is one of it.** `internal/screens`
// draws the scrim and the bubble's stroke in the same color; a card tinted in a second red would
// read as a different kind of attention. This package still does not know what a tutorial is — it
// is handed a name for a color, exactly as Spec.TextInk is handed one.
var HighlightInk = color.RGBA{R: 232, G: 60, B: 48, A: 255}

// highlightWash is how far a highlit card is pulled toward that red, in percent.
//
// **Enough to be unmistakable across a row of eight and not enough to stop the card being read.**
// The whole reason for tinting the card rather than framing it is that the player is being asked to
// look at *this card*, so the name, the cost and the text have to survive the marking.
const highlightWash = 30

// washInside pulls every pixel of the card toward one color, leaving the transparent corners
// alone. It is dimInside against an arbitrary ink rather than against the card surface.
func washInside(dst *image.RGBA, w, h, radius int, ink color.RGBA, pct int) {
	if pct <= 0 {
		return
	}
	b := dst.Bounds()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !insideRounded(w, h, radius, x, y) {
				continue
			}
			i := dst.PixOffset(b.Min.X+x, b.Min.Y+y)
			if dst.Pix[i+3] == 0 {
				continue
			}
			dst.Pix[i+0] = towardByte(dst.Pix[i+0], ink.R, pct)
			dst.Pix[i+1] = towardByte(dst.Pix[i+1], ink.G, pct)
			dst.Pix[i+2] = towardByte(dst.Pix[i+2], ink.B, pct)
		}
	}
}

func towardByte(from, to uint8, pct int) uint8 {
	return uint8(int(from) + (int(to)-int(from))*pct/100)
}
