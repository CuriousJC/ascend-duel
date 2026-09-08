package cards

// Marks on a card face: **a picture drawn over a finished card to say something happened to it.**
//
// This is the third way something can be said on a card and it is deliberately unlike the other
// two. The *face* — name, cost ticks, form mark, text — is what the card is. An *upgrade* is what
// the run has permanently made it, and it works by taking the left column over rather than by
// covering anything up (see upgrade.go). A **mark** sits on top of the whole face and is about the
// card's situation rather than its identity: this one was stopped, this one was eaten, this one is
// being altered.
//
// # Two flavours, and they are not the same drawing twice
//
// **The final state** is what this file rasterises: the mark baked into the card image, cached with
// it, and true for as long as the card is drawn. Spec.Mark is where a caller asks for one, so
// tools/cardsheet can show a marked card and the game and the sheet cannot disagree about what one
// looks like.
//
// **The transition** is the mark arriving — the crack spreading, the card being struck. That cannot
// live here: it changes every frame, and baking a new card image per frame would blow the face
// cache that internal/screens keys on the whole Spec. So this package exports the mark's *geometry*
// (ShatterCracks) and the screen draws the same lines on the GPU while they are moving, handing
// over to the baked version once it settles. **One geometry, two rasterisers**, so the animation
// cannot end on a picture different from the one it hands to.
//
// # The pattern is derived, never rolled
//
// A crack pattern is a function of the card's name and its size, through a small hash — not a draw
// from any stream. That is on purpose and is not a determinism rule being skipped: this decides
// nothing, and putting it on a stream would mean a cosmetic detail advancing a cursor that a rule
// reads. It also means the same card always shatters the same way, so a mark is stable across a
// redraw, a resize and a sheet regeneration. See CLAUDE.md and the randomness skill for the rule
// this is an explicit, argued exception to.

import (
	"image"
	"image/color"
	"math"
)

// Mark is what has happened to a card, drawn over its finished face. **A set, not one value** — a
// card can be several things at once, and a bitmask is what says so without the caller having to
// rank them.
//
// **Append-only, and never serialized.** It is the same shape as GlyphKind and ConceptID; a file
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

	// MarkShattered is an attack that will not fire — a shield ate it. **A broken window rather
	// than a cross or a grey-out**: the card is still there and still readable, which is the
	// point, because the player needs to see *what* was stopped as well as that something was.
	MarkShattered Mark = 1 << iota

	// MarkHighlit is "this card, this one" — the tutorial pointing at what it is talking about.
	//
	// **It replaced a wireframe drawn round the card** *(owner's call, 2026-09-08)*. A red rectangle
	// outside a card is a thing on the screen near the card; a tinted card is the card itself
	// answering. It also composes, which a frame could not: a frame round a set of cards is a box,
	// and a box round non-adjacent cards includes the ones between them — the exact bug the
	// tutorial's matching-cards anchor had.
	MarkHighlit
)

// Has reports whether a mark set carries one.
func (m Mark) Has(bit Mark) bool { return m&bit != 0 }

// shatterDim is how far the card under a shatter is pulled toward the surface, in percent of the
// way there. **The card has to stay readable** — the whole reason the mark is a crack and not a
// blackout — so this is a knock rather than a fade: enough that a shattered card reads as spent
// beside a live one, not enough to make the reader lean in.
const shatterDim = 18

// crackInk is what a crack is drawn in: a near-black at partial alpha, so it darkens whatever it
// crosses rather than replacing it. **Not an element colour and not the ring pink** — a break is
// not a fifth thing wanting a hue, and the wheel is full (see CLAUDE.md). It reads as absence of
// card rather than as a mark someone put there.
var crackInk = color.RGBA{R: 30, G: 27, B: 34, A: 190}

// crackGlint is the light side of a crack, one pixel off it, which is what makes the break read as
// glass rather than as ink. It is the same trick BevelEdges plays on every other surface in the
// game: a broken edge catches light on one side and shadows on the other.
var crackGlint = color.RGBA{R: 255, G: 255, B: 255, A: 120}

// CrackInk is what a break is drawn in, for the caller that strokes the same geometry on the GPU
// while it is still moving. **Exported so there is one answer** — a transition in a different black
// from the mark it hands to would step on the frame the two swap.
func CrackInk() color.RGBA { return crackInk }

// Crack is one straight run of a break, in pixels relative to the card's top-left.
//
// **Delay is when it appears during a transition**, 0 at the moment of impact and 1 at the end of
// the spread — so the screen can propagate the break outward from where it was struck instead of
// stamping the whole pattern at once. The baked final state ignores it and draws them all.
type Crack struct {
	From, To image.Point
	Width    int
	Delay    float64
}

// ShatterCracks is the geometry of a broken window over a card of this size, seeded by its name.
//
// **Radials plus chords, and the chords are what make it a window.** Lines running from an impact
// point to the edges alone read as a starburst — a thing that happened *at* a point. Joining
// adjacent radials with short chords at two radii turns the same lines into panes of glass, which
// is the shape a reader recognises without being told.
//
// The impact point is offset from centre, because a break centred on a card reads as a decoration
// laid on it rather than as something that struck it.
func ShatterCracks(w, h int, seed uint32) []Crack {
	if w <= 0 || h <= 0 {
		return nil
	}
	r := hasher(seed)

	// The impact, in the middle third of the card so no radial is too short to read.
	cx := w/3 + int(r.next()%uint32(maxInt(1, w/3)))
	cy := h/3 + int(r.next()%uint32(maxInt(1, h/3)))

	// Enough radials to close a few panes and few enough that each is legible at card size.
	const radials = 7
	angles := make([]float64, radials)
	step := 2 * math.Pi / float64(radials)
	for i := range angles {
		// Evenly spread, then jittered by up to a third of the gap either way. Even spacing alone
		// reads as a manufactured star; unbounded jitter closes two radials into one thick line.
		angles[i] = float64(i)*step + (r.unit()-0.5)*step*0.66
	}

	span := float64(w+h) / 2
	out := make([]Crack, 0, radials*3)
	ends := make([]image.Point, radials)

	for i, a := range angles {
		// Long enough to leave the card on every heading, so a radial always reaches an edge and
		// is clipped there rather than stopping in open space, which would read as a scratch.
		end := image.Pt(cx+int(math.Cos(a)*span), cy+int(math.Sin(a)*span))
		ends[i] = end
		out = append(out, Crack{From: image.Pt(cx, cy), To: end, Width: 2, Delay: 0})
	}

	// The chords, at three radii, joining each radial to the next one round. They arrive after the
	// radials during a transition — the break spreads out from the impact, then the panes close.
	//
	// **Each vertex sits at a jittered radius rather than on a true circle.** Perfect rings read as
	// a spider web, which is a thing that was woven; glass breaks into panes of unequal size, and
	// the unevenness is most of what tells the two apart at card size.
	for _, ring := range []struct {
		at    float64
		delay float64
	}{{0.30, 0.40}, {0.56, 0.62}, {0.84, 0.82}} {
		at := make([]image.Point, len(angles))
		for i := range angles {
			at[i] = along(cx, cy, ends[i], ring.at*(0.78+r.unit()*0.44))
		}
		for i := range at {
			j := (i + 1) % len(at)
			out = append(out, Crack{From: at[i], To: at[j], Width: 2, Delay: ring.delay})
		}
	}
	return out
}

// along is the point a fraction of the way from an impact to a radial's end.
func along(cx, cy int, end image.Point, f float64) image.Point {
	return image.Pt(
		cx+int(float64(end.X-cx)*f),
		cy+int(float64(end.Y-cy)*f),
	)
}

// MarkSeed is the number a card's mark geometry is derived from: its name, hashed.
//
// **It is exported because both rasterisers need the same one.** The screen draws the moving
// version and this package bakes the settled one, and a break that rearranged itself on the frame
// the animation handed over would be the one failure this whole split exists to prevent.
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// drawMark paints a card's marks over its finished face, and is the last thing a face has done to
// it.
//
// **They are drawn over everything, the border included.** A break that stopped at the frame would
// read as being inside the card, and what a mark says is about the card as a whole.
//
// **The order is fixed here so two marks compose one way.** The break goes down first and the
// highlight washes over it: a shattered card the tutorial is pointing at should read as pointed-at
// *and* broken, and a break drawn over the wash would be the louder of the two when the sentence
// being said is about the pointing.
func drawMark(dst *image.RGBA, mark Mark, name string, w, h, radius int) {
	if mark.Has(MarkShattered) {
		dimInside(dst, w, h, radius, shatterDim)
		for _, c := range ShatterCracks(w, h, MarkSeed(name)) {
			DrawCrack(dst, c, w, h, radius)
		}
	}
	if mark.Has(MarkHighlit) {
		washInside(dst, w, h, radius, HighlightInk, highlightWash)
	}
}

// HighlightInk is the colour MarkHighlit washes a card in.
//
// **It is the tutorial's own red and it is exported so there is one of it.** `internal/screens`
// draws the scrim and the bubble's stroke in the same colour; a card tinted in a second red would
// read as a different kind of attention. This package still does not know what a tutorial is — it
// is handed a name for a colour, exactly as Spec.TextInk is handed one.
var HighlightInk = color.RGBA{R: 232, G: 60, B: 48, A: 255}

// highlightWash is how far a highlit card is pulled toward that red, in percent.
//
// **Enough to be unmistakable across a row of eight and not enough to stop the card being read.**
// The whole reason for tinting the card rather than framing it is that the player is being asked to
// look at *this card*, so the name, the cost and the text have to survive the marking.
const highlightWash = 30

// washInside pulls every pixel of the card toward a colour, leaving the transparent corners alone.
// It is dimInside against an arbitrary ink rather than against the card surface.
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

// dimInside pulls every pixel of the card toward the card surface. **Toward the surface rather than
// toward black**, which is ColorToward's rule: this is a light card, and scaling it down would make
// it louder than the live card beside it.
func dimInside(dst *image.RGBA, w, h, radius, pct int) {
	washInside(dst, w, h, radius, Surface, pct)
}

func towardByte(from, to uint8, pct int) uint8 {
	return uint8(int(from) + (int(to)-int(from))*pct/100)
}

// DrawCrack rasterises one crack into a card image, clipped to the card's rounded silhouette.
//
// **Exported so the geometry above has exactly one plain-Go rasteriser** rather than one per
// caller — anything wanting a still frame of a break draws it through here.
func DrawCrack(dst *image.RGBA, c Crack, w, h, radius int) {
	dx, dy := c.To.X-c.From.X, c.To.Y-c.From.Y
	steps := abs(dx)
	if abs(dy) > steps {
		steps = abs(dy)
	}
	if steps == 0 {
		return
	}
	for s := 0; s <= steps; s++ {
		x := c.From.X + dx*s/steps
		y := c.From.Y + dy*s/steps
		stamp(dst, x, y, c.Width, w, h, radius, crackInk)
		// The glint sits one pixel off the break on the side the rest of the game is lit from.
		stamp(dst, x+1, y-1, 1, w, h, radius, crackGlint)
	}
}

// stamp puts a square of ink down, clipped to the card.
func stamp(dst *image.RGBA, x, y, size, w, h, radius int, c color.RGBA) {
	if size < 1 {
		size = 1
	}
	b := dst.Bounds()
	for oy := 0; oy < size; oy++ {
		for ox := 0; ox < size; ox++ {
			px, py := x+ox, y+oy
			if px < 0 || py < 0 || px >= w || py >= h || !insideRounded(w, h, radius, px, py) {
				continue
			}
			i := dst.PixOffset(b.Min.X+px, b.Min.Y+py)
			if dst.Pix[i+3] == 0 {
				continue
			}
			a := int(c.A)
			dst.Pix[i+0] = uint8((int(c.R)*a + int(dst.Pix[i+0])*(255-a)) / 255)
			dst.Pix[i+1] = uint8((int(c.G)*a + int(dst.Pix[i+1])*(255-a)) / 255)
			dst.Pix[i+2] = uint8((int(c.B)*a + int(dst.Pix[i+2])*(255-a)) / 255)
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
