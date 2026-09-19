package cards

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"golang.org/x/image/font"
)

// These tests assert *pixels*, not that the code ran.
//
// A contact sheet is the usual way to check drawing here, and it is the right tool for
// "does this look good". It is the wrong tool for "is the border six pixels thick" and
// "did the disabled card come out louder than the live one", because those are true or
// false and nobody should have to squint at a PNG to find out. The corners are
// deliberately hard-edged, so every pixel is exactly one color and can be compared
// without tolerances.

func faces(t *testing.T) *Faces {
	t.Helper()
	f, err := NewFaces(assets.LoadFontData()["kubasta"])
	if err != nil {
		t.Fatalf("loading the card font: %v", err)
	}
	return f
}

// deckVisibleWidth is how much of a mini card the deck overlay actually shows, given the
// pitch it lays rows out at. Duplicated from internal/screens rather than imported —
// screens imports cards, never the reverse — and TestDeckPitchMatchesTheCard in that
// package fails if the two drift.
const deckVisibleWidth = 75

// deckNames is every concept in the deck, for the layout tests that have to check the
// worst case rather than a representative one.
var deckNames = []string{
	"Jab", "Thrust", "Skewer",
	"Cut", "Slice", "Cleave",
	"Thump", "Bash", "Smash",
	"Brace", "Block", "Guard",
}

func strike(e Element) Spec {
	return Spec{Name: "Bash", Form: FormCrush, Cost: 2, Element: e, Enabled: true}
}

func render(t *testing.T, s Spec, st Style) *image.RGBA {
	t.Helper()
	img, err := Render(s, st, faces(t))
	if err != nil {
		t.Fatalf("rendering %s: %v", s.Name, err)
	}
	return img
}

func TestCardIsExactlyTheStyleSize(t *testing.T) {
	for name, st := range map[string]Style{"hand": Hand, "mini": Mini} {
		img := render(t, strike(Fire), st)
		if got := img.Bounds(); got.Dx() != st.Width || got.Dy() != st.Height {
			t.Errorf("%s card is %dx%d, want %dx%d", name, got.Dx(), got.Dy(), st.Width, st.Height)
		}
	}
}

func TestCornersAreTransparentAndEdgesAreNot(t *testing.T) {
	st := Hand
	img := render(t, strike(Fire), st)

	// The very corner pixel is outside the radius, so it has to be clear or the card is
	// a rectangle with a decorative curve painted on it.
	corners := map[string]image.Point{
		"top-left":     {X: 0, Y: 0},
		"top-right":    {X: st.Width - 1, Y: 0},
		"bottom-left":  {X: 0, Y: st.Height - 1},
		"bottom-right": {X: st.Width - 1, Y: st.Height - 1},
	}
	for name, p := range corners {
		if a := img.RGBAAt(p.X, p.Y).A; a != 0 {
			t.Errorf("%s corner has alpha %d, want 0 — the corner is not rounded", name, a)
		}
	}

	// The middle of each edge must be solid, or the shape has been clipped away.
	edges := map[string]image.Point{
		"top":    {X: st.Width / 2, Y: 0},
		"bottom": {X: st.Width / 2, Y: st.Height - 1},
		"left":   {X: 0, Y: st.Height / 2},
		"right":  {X: st.Width - 1, Y: st.Height / 2},
	}
	for name, p := range edges {
		if a := img.RGBAAt(p.X, p.Y).A; a != 255 {
			t.Errorf("%s edge midpoint has alpha %d, want 255", name, a)
		}
	}
}

// The border stopped naming the element on 2026-08-23 — the form mark does that now, and
// borderBase says why. What this still pins is the width and the resting state, which is what
// the border is for.
//
// **Its outer BorderBevel pixels are the card's light** *(2026-08-24)*, so the walk starts inside
// them; the bevel has its own test below.
func TestBorderIsTheNeutralColorAtItsDeclaredWidth(t *testing.T) {
	st := Hand
	mid := st.Height / 2

	for _, e := range Elements() {
		img := render(t, strike(e), st)
		want := systems.ColorToward(borderBase(e, ""), Surface, borderRestToward)

		// Walk in from the left edge at the card's waist, where there is no curvature.
		for x := BorderBevel; x < st.BorderWidth; x++ {
			if got := img.RGBAAt(x, mid); got != want {
				t.Errorf("%s border at x=%d is %v, want %v", e, x, got, want)
				break
			}
		}
		// One pixel past the border is the surface. This is the assertion that actually
		// pins the width — without it a border of any thickness would pass above.
		if got := img.RGBAAt(st.BorderWidth, mid); got != Surface {
			t.Errorf("%s: pixel just inside the %dpx border is %v, want the surface %v",
				e, st.BorderWidth, got, Surface)
		}
	}
}

// **The card is lit from the top left, like every bevelled thing on screen.** The left edge at the
// waist is lit and the right edge is shadowed, and neither is the border's own color — a bevel
// that had quietly become two copies of the fill would draw identically to no bevel at all.
func TestTheCardBorderIsLitOnTheTopLeftAndShadowedOnTheBottomRight(t *testing.T) {
	st := Hand
	img := render(t, strike(Fire), st)
	mid := st.Height / 2

	face := systems.ColorToward(borderBase(Fire, ""), Surface, borderRestToward)
	light, shade := systems.BevelEdges(face)

	if got := img.RGBAAt(0, mid); got != light {
		t.Errorf("the left edge is %v, want the lit %v", got, light)
	}
	if got := img.RGBAAt(st.Width-1, mid); got != shade {
		t.Errorf("the right edge is %v, want the shadowed %v", got, shade)
	}
	if got := img.RGBAAt(st.Width/2, 0); got != light {
		t.Errorf("the top edge is %v, want the lit %v", got, light)
	}
	if got := img.RGBAAt(st.Width/2, st.Height-1); got != shade {
		t.Errorf("the bottom edge is %v, want the shadowed %v", got, shade)
	}

	// **The bevel is the outside of the border, not the whole of it.** Four of the six pixels are
	// still the state color, which is the signal the border exists to carry.
	if got := img.RGBAAt(BorderBevel, mid); got != face {
		t.Errorf("the pixel just inside the bevel is %v, want the border's own %v", got, face)
	}
}

func TestEveryElementBorderIsDistinct(t *testing.T) {
	// Basic is a mid gray rather than the near-white the screen uses as a surface,
	// precisely so it is not the same color as the card it sits on. If someone
	// "restores" it to {235,235,235} this fails.
	seen := map[color.RGBA]Element{}
	for _, e := range Elements() {
		c := BorderOf(e)
		if prev, dup := seen[c]; dup {
			t.Errorf("%s and %s share the border color %v", prev, e, c)
		}
		seen[c] = e

		if c == Surface {
			t.Errorf("%s border is the same color as the card surface — it would be invisible", e)
		}
	}
}

func TestSurfaceIsConstantAcrossElements(t *testing.T) {
	// The whole point of the redesign: the face no longer says which element it is.
	st := Hand
	probe := image.Point{X: st.Width - st.BorderWidth - 4, Y: st.Height - st.BorderWidth - 4}

	for _, e := range Elements() {
		img := render(t, strike(e), st)
		if got := img.RGBAAt(probe.X, probe.Y); got != Surface {
			t.Errorf("%s card surface is %v, want the constant %v", e, got, Surface)
		}
	}
}

// tickInk is the average color of one cost tick, over the whole rectangle the style gives it.
//
// **A tick is a drawing rather than a filled rectangle as of 2026-09-16**, so no single pixel is
// the element's color any more: there is a lit top edge, a body, a dark under edge and a
// near-black contour. What survived the change is what the left column is *for* — the average
// still carries the element's hue, and every state still moves it the same distance the border
// moves. The four tests below therefore measure the average and the hue rather than one pixel.
// See drawDashes.
// tickCore is the pixel at the middle of one cost tick: opaque, well inside the drawing, and
// away from every anti-aliased edge. It is what the state test measures, because a fade is
// applied per pixel and only a pixel the surface has no share in can be checked exactly.
func tickCore(img *image.RGBA, st Style, i int) color.RGBA {
	y := st.DashTop + i*(st.DashHeight+st.DashGap) + st.DashHeight/2
	return img.RGBAAt(st.DashLeft+st.DashWidth/2, y)
}

// tickDrawn says whether a tick was drawn at all: its rectangle holds something other than the
// card's own surface. It is the count test's question, which used to be "is this pixel exactly
// the element color" and cannot be any more.
func tickDrawn(img *image.RGBA, st Style, i int) bool {
	y0 := st.DashTop + i*(st.DashHeight+st.DashGap)
	for y := y0; y < y0+st.DashHeight; y++ {
		for x := st.DashLeft; x < st.DashLeft+st.DashWidth; x++ {
			if c := img.RGBAAt(x, y); c != Surface && c != SurfaceDisabled {
				return true
			}
		}
	}
	return false
}

// nearColor allows a channel or two of rounding, which averaging a drawing and then fading it
// produces and which fading it and then averaging it does not. The states are tens of levels
// apart, so this cannot hide one landing on another's figure.
func nearColor(a, b color.RGBA, tol int) bool {
	d := func(x, y uint8) int {
		if x > y {
			return int(x) - int(y)
		}
		return int(y) - int(x)
	}
	return d(a.R, b.R) <= tol && d(a.G, b.G) <= tol && d(a.B, b.B) <= tol
}

func TestCostDrawsOneDashPerPoint(t *testing.T) {
	st := Hand

	for cost := 0; cost <= 4; cost++ {
		s := strike(Fire)
		s.Cost = cost
		img := render(t, s, st)

		count := 0
		for i := 0; i < 6; i++ {
			if tickDrawn(img, st, i) {
				count++
			}
		}
		if count != cost {
			t.Errorf("cost %d drew %d dashes", cost, count)
		}
	}
}

func TestLeftColumnDoesNotCollide(t *testing.T) {
	// **The left column is a glyph over a stack of dashes**, and it used to have a damage badge
	// under that. Adding the category glyph is what made this worth a test — before it there
	// was one badge and a corner, and nothing could overlap anything.
	//
	// Four dashes is the most the rules can produce today. A fifth cost tier grows the stack
	// past the bottom of the card, which is a layout change and not just a bigger number. This
	// fails when that happens rather than rendering it.
	const maxCost = 4

	st := Hand

	// The mark's box is what the layout names, so it is what has to fit. The mark is centered in
	// it and clipped to it, so nothing can be drawn outside it.
	formBottom := st.FormTop + st.FormSize
	dashBottom := st.DashTop + (maxCost-1)*(st.DashHeight+st.DashGap) + st.DashHeight

	if formBottom > st.DashTop {
		t.Errorf("form mark ends at y=%d, %dpx into the dash stack at y=%d",
			formBottom, formBottom-st.DashTop, st.DashTop)
	}
	if inside := st.Height - st.BorderWidth; dashBottom > inside {
		t.Errorf("%d dashes end at y=%d, past the inside of the bottom border at y=%d",
			maxCost, dashBottom, inside)
	}
}

func TestTheCostColumnStaysOutOfTheTextColumn(t *testing.T) {
	// **The text is centered in what the cost column leaves**, so the column's *width* is now
	// load-bearing in a way it never was while the text ran the full width of the card. One
	// thing shares a horizontal with the text and therefore sets that width: the dash marks.
	// That is what a cost column ought to mean, and it is why the column is as narrow as it is.
	st := Hand

	if right := st.DashLeft + st.DashWidth; right > st.TextColumnLeft {
		t.Errorf("the cost dashes reach x=%d, into the text column at x=%d",
			right, st.TextColumnLeft)
	}
}

func TestTheFormMarkClearsTheTextBand(t *testing.T) {
	// **The mark is deliberately wider than the column it sits above**, so what keeps it off
	// the text is height rather than width: it sits in the top-left corner and the text band
	// starts below it. If either moves far enough to overlap, the mark is drawn first and the
	// first line of text lands on top of it.
	st := Hand

	if bottom := st.FormTop + st.FormSize; bottom > st.TextBandTop {
		t.Errorf("the form mark ends at y=%d, %dpx into the text band at y=%d",
			bottom, bottom-st.TextBandTop, st.TextBandTop)
	}
}

func TestAGlyphOffTheCornerDoesNotSquareTheCorner(t *testing.T) {
	// The glyph is placed at a negative offset so it runs off the top-left edge. Cropping it at
	// the image's bounding box rather than at the card's own curve would fill the transparent
	// rounded corner with glyph pixels and the card would read as having one square corner.
	st := Hand
	if st.GlyphInset >= 0 && st.FormTop >= 0 {
		t.Skip("the hand's form mark no longer hangs off the corner")
	}

	// A defend card, because the kite shield is the widest glyph and reaches furthest into the
	// corner. Disabled as well as enabled: the fade pass walks the same rectangle.
	for _, enabled := range []bool{true, false} {
		s := Spec{Name: "Defend", Form: FormDefend, Cost: 3, Element: Ice, Enabled: enabled}
		img := render(t, s, st)

		// The outermost corner pixel of the bounding box is outside a 12px radius and must stay
		// transparent whatever is drawn over it.
		if got := img.RGBAAt(0, 0); got.A != 0 {
			t.Errorf("enabled=%v: the top-left corner pixel is %v, want transparent", enabled, got)
		}
	}
}

func TestMiniIsHandAtHalfSize(t *testing.T) {
	// **The whole of Mini's definition, pinned as one equality** *(2026-08-23)*. Mini used to be
	// authored beside Hand and had drifted field by field — a full-size form box on a half-size
	// card being the one that showed — so this replaces a list of individual assertions with the
	// rule the owner actually asked for: the deck panel draws the hand's card, only smaller.
	//
	// A field added to Hand and forgotten here is now impossible rather than merely unlikely.
	if got, want := Mini, Hand.Scaled(1, 2); got != want {
		t.Errorf("Mini is not Hand at half size:\n got %+v\nwant %+v", got, want)
	}
}

func TestMiniSaysWhatItNeedsToInsideTheVisibleStrip(t *testing.T) {
	// **What the overlay can say is set by what fits in the visible strip**, because the rows
	// overlap and only the left of each card shows. The name and the form mark are what identify
	// a card here, so both have to land inside it; the tempting regression is to re-tighten the
	// pitch for a longer row and silently lose one.
	visible := deckVisibleWidth

	if !Mini.ShowName {
		t.Error("Mini does not show the name, which is the only thing identifying a concept")
	}
	if !Mini.ShowForm {
		t.Error("Mini does not show the form mark")
	}
	if markRight := Mini.GlyphInset + Mini.FormSize; markRight > visible {
		t.Errorf("the form mark ends at x=%d but only %d pixels show through the overlap",
			markRight, visible)
	}
	if dashRight := Mini.DashLeft + Mini.DashWidth; dashRight > visible {
		t.Errorf("the dashes end at x=%d but only %d pixels show", dashRight, visible)
	}

	// Every name in the deck has to fit the card at the size Mini asks for.
	f := faces(t)
	face, err := f.at(Mini.NameSize)
	if err != nil {
		t.Fatal(err)
	}
	usable := Mini.Width - 2*Mini.BorderWidth - 4
	for _, n := range deckNames {
		if w := font.MeasureString(face, n).Ceil(); w > usable {
			t.Errorf("%q is %dpx at %gpt, wider than the %dpx a mini card has",
				n, w, Mini.NameSize, usable)
		}
	}
}

func TestMiniRendersEverythingInsideTheVisibleStrip(t *testing.T) {
	// A geometry test proves the numbers; this proves the pixels. Anything drawn past the
	// visible strip is hidden by the next card in the row, so a layout that drifted right
	// would silently stop showing what it claims to show.
	st := Mini
	s := Spec{Name: "Prepare", Form: FormDefend, Cost: 1, Element: Lightning, Enabled: true}
	img := render(t, s, st)

	// **Inside the rounded surface, not inside its bounding box.** A rectangle inset by the
	// border width still clips the four curves, so the border's own pixels near a corner would
	// be counted as content — which is what a thinner border made visible.
	iw, ih := st.Width-2*st.BorderWidth, st.Height-2*st.BorderWidth
	ink := 0
	for y := st.BorderWidth; y < st.Height-st.BorderWidth; y++ {
		for x := deckVisibleWidth; x < st.Width-st.BorderWidth; x++ {
			if !insideRounded(iw, ih, st.CornerRadius-st.BorderWidth,
				x-st.BorderWidth, y-st.BorderWidth) {
				continue
			}
			if c := img.RGBAAt(x, y); c != Surface {
				ink++
			}
		}
	}
	if ink > 0 {
		t.Errorf("%d pixels of content sit past x=%d, where the next card covers them",
			ink, deckVisibleWidth)
	}
}

func TestNameClearsTheFormMark(t *testing.T) {
	// **The name is centered on the card, not on the space left beside the mark.** So a
	// long enough name reaches back into the corner the mark now occupies. Every concept
	// in the deck is checked, because the failure is invisible on "Jab" and obvious on the
	// longest one.
	st := Hand
	f := faces(t)
	face, err := f.at(st.NameSize)
	if err != nil {
		t.Fatal(err)
	}
	markRight := st.GlyphInset + st.FormSize

	for _, n := range deckNames {
		w := font.MeasureString(face, n).Ceil()
		left := (st.Width - w) / 2
		if left <= markRight {
			t.Errorf("%q is %dpx wide, so centered it starts at x=%d and runs into the mark ending at x=%d",
				n, w, left, markRight)
		}
	}
}

func TestFormMarkIsDrawnAndDiffers(t *testing.T) {
	// Four forms, four marks. If two ever render identically the corner is saying nothing, which
	// is worse than leaving it empty — so this holds the four drawings apart from each other.
	st := Hand
	seen := map[string]Form{}

	for _, fam := range Forms() {
		spec := strike(Fire)
		spec.Form = fam
		img := render(t, spec, st)

		// Hash the mark's own box rather than the whole card, so the name and dashes
		// cannot mask a mark that failed to draw.
		var sum uint64
		for y := st.FormTop; y < st.FormTop+st.FormSize; y++ {
			for x := st.GlyphInset; x < st.GlyphInset+st.FormSize; x++ {
				p := img.RGBAAt(x, y)
				sum = sum*31 + uint64(p.R)<<16 + uint64(p.G)<<8 + uint64(p.B)
			}
		}
		key := fmt.Sprintf("%x", sum)
		if prev, dup := seen[key]; dup {
			t.Errorf("%s and %s draw the same form mark", prev, fam)
		}
		seen[key] = fam
	}

	// And FormNone leaves the slot alone, which is what a relic and the opponent's cards need.
	spec := strike(Fire)
	spec.Form = FormNone
	img := render(t, spec, st)
	x := st.GlyphInset + st.FormSize/2
	y := st.FormTop + st.FormSize/2
	if got := img.RGBAAt(x, y); got != Surface {
		t.Errorf("FormNone drew something at the mark slot: %v, want the surface %v", got, Surface)
	}
}

func TestEveryFormHasItsOwnMark(t *testing.T) {
	// The pixel test above would also catch this, slowly and by a hash. This says the actual rule:
	// every form has a mark in every element, no two forms share one, and FormNone has none at all
	// — a relic and both fighter cards belong to no form and the slot must stay empty for them.
	//
	// **It asks about asset keys now** *(2026-09-16)*. A form used to name a generated glyph and the
	// drawing was tinted; there is one authored drawing per form per element, so what a form has is
	// a key per element plus a neutral one, and MarkArtKey is the thing that builds them.
	for _, e := range append(Elements(), Basic) {
		seen := map[string]Form{}
		for _, fam := range Forms() {
			k := MarkArtKey(fam, e)
			if k == "" {
				t.Errorf("%s in %s has no mark, so its corner would be blank", fam, e)
				continue
			}
			if systems.ArtMark(k, Hand.FormSize, Hand.FormSize) == nil {
				t.Errorf("%s in %s names %q, which is not in the assets", fam, e, k)
			}
			if prev, dup := seen[k]; dup {
				t.Errorf("%s and %s in %s both mark themselves with %q", prev, fam, e, k)
			}
			seen[k] = fam
		}
	}
	for _, e := range append(Elements(), Basic) {
		if k := MarkArtKey(FormNone, e); k != "" {
			t.Errorf("FormNone in %s names %q; it must draw nothing", e, k)
		}
	}
}

// TestEveryElementHasItsOwnTick is the tick half of the rule above: every element resolves to a
// drawing, no two elements share one, and there is no card anywhere that falls through to nothing.
func TestEveryElementHasItsOwnTick(t *testing.T) {
	seen := map[string]Element{}
	for _, e := range Elements() {
		k := TickArtKey(e)
		if k == "" {
			t.Fatalf("%s has no tick key at all", e)
		}
		if systems.ArtMark(k, Hand.DashWidth, Hand.DashHeight) == nil {
			t.Errorf("%s names tick %q, which is not in the assets", e, k)
		}
		// Basic and Relic deliberately share the neutral bar: neither is an element a hand is
		// counted on, and neither draws a cost the player is reading for color.
		if prev, dup := seen[k]; dup && hasArt(e) && hasArt(prev) {
			t.Errorf("%s and %s both draw tick %q", prev, e, k)
		}
		seen[k] = e
	}
}

func TestRelicBorderIsUnmistakable(t *testing.T) {
	// The one thing that must never happen is reaching for a relic thinking it is a card
	// you can play, so the pink has to be a long way from every element border.
	const minDistance = 120

	pink := BorderOf(Relic)
	for _, e := range Elements() {
		if d := distance(pink, BorderOf(e)); d < minDistance {
			t.Errorf("the relic border is only %d from %s's — too close to tell apart at a glance", d, e)
		}
	}
	// And it is not in Elements(), because anything iterating elements means cards.
	for _, e := range Elements() {
		if e == Relic {
			t.Error("Relic is in Elements(); a relic is not an element a card can have")
		}
	}
}

func TestRelicDrawsArtAndNoCardFurniture(t *testing.T) {
	st := RelicStyle
	art := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for i := range art.Pix {
		art.Pix[i] = 255 // opaque white block, easy to find
	}

	s := Spec{Name: "Fire", Element: Relic, Art: art, Enabled: true}
	img := render(t, s, st)

	// The art lands in the middle of its box.
	cx := st.Width / 2
	cy := st.ArtTop + st.ArtMaxH/2
	if got := img.RGBAAt(cx, cy); got.A == 0 || got == Surface {
		t.Errorf("nothing drawn at the center of the art box (%d,%d): %v", cx, cy, got)
	}

	// And none of the card's own furniture is on it: a relic has no cost and no phase, so
	// a stray dash or glyph would be claiming something untrue about it.
	if st.ShowForm || st.TextLineHeight > 0 {
		t.Error("the relic style claims to draw a form mark or effect text")
	}
	noCost := Spec{Name: "Fire", Element: Relic, Art: art, Enabled: true, Cost: 0}
	plain := render(t, noCost, st)
	tick := systems.ColorToward(BorderOf(Relic), Surface, borderRestToward)
	for y := 0; y < 40; y++ {
		if plain.RGBAAt(st.DashLeft, y) == tick && st.DashWidth > 0 {
			t.Errorf("a cost dash was drawn on a relic at y=%d", y)
		}
	}
}

func TestDashesDoNotOverprintTheName(t *testing.T) {
	// The geometry test above proves the columns do not overlap. This proves it in
	// pixels, on the longest name in the deck at the widest cost — the case where a
	// mistake would actually show.
	st := Hand
	s := Spec{Name: "Prepare", Form: FormDefend, Cost: 4, Element: Lightning, Enabled: true}
	img := render(t, s, st)

	// **Compared against the same card with no name at all**, rather than against a color. A tick
	// is a drawing now, so "every pixel is the element" is no longer the question — the question
	// is whether setting the name changed any pixel inside the cost column, and a nameless twin
	// answers it without the test having to know what a tick looks like.
	bare := s
	bare.Name = ""
	clean := render(t, bare, st)

	for i := 0; i < s.Cost; i++ {
		y0 := st.DashTop + i*(st.DashHeight+st.DashGap)
		for y := y0; y < y0+st.DashHeight; y++ {
			for x := st.DashLeft; x < st.DashLeft+st.DashWidth; x++ {
				if got, want := img.RGBAAt(x, y), clean.RGBAAt(x, y); got != want {
					t.Fatalf("tick %d is broken at (%d,%d): %v, want %v — the name is printing over it",
						i, x, y, got, want)
				}
			}
		}
	}
}

func TestTheCostColumnHoldsNothingButDashes(t *testing.T) {
	// **The card carries no damage figure** — what it deals is in the effect text — so below the
	// last dash the column is bare surface. This is the pixel half of the geometry test above:
	// it catches anything drawn into the column that the layout constants do not describe.
	st := Hand
	s := Spec{Name: "Cleave", Form: FormSlash, Cost: 3, Element: Ice, Enabled: true}
	img := render(t, s, st)

	firstFree := st.DashTop + s.Cost*(st.DashHeight+st.DashGap)
	for y := firstFree; y < st.Height-st.BorderWidth; y++ {
		for x := st.DashLeft; x < st.TextColumnLeft; x++ {
			if got := img.RGBAAt(x, y); got != Surface {
				t.Fatalf("(%d,%d) is %v, want bare surface %v below the cost dashes", x, y, got, Surface)
			}
		}
	}
}

func TestDisabledIsQuieterThanRestingOnALightCard(t *testing.T) {
	// **The regression this whole state model exists to prevent.** ColorAtStrength
	// scales toward black, so on an off-white card a "dimmed" border comes out darker
	// than the surface and reads as louder than the live card beside it. That exact
	// mistake put the Resolution pane's idle rows in front of its lit one.
	//
	// "Quieter" on a light ground means *closer to the surface*, so that is what is
	// measured — not luminance, which would call the darker border the dimmer one.
	st := Hand
	mid := st.Height / 2

	for _, e := range Elements() {
		live := strike(e)
		dead := strike(e)
		dead.Enabled = false

		liveBorder := render(t, live, st).RGBAAt(st.BorderWidth/2, mid)
		deadBorder := render(t, dead, st).RGBAAt(st.BorderWidth/2, mid)

		if d, l := distance(deadBorder, SurfaceDisabled), distance(liveBorder, Surface); d >= l {
			t.Errorf("%s: disabled border is %d from its surface, resting is %d — disabled is not quieter",
				e, d, l)
		}
	}
}

func TestSelectedIsTheColorNamedInTheSource(t *testing.T) {
	// A widget names the color it wants at full strength and its other states scale
	// down from it. Selected is that full strength, so it must be BorderOf exactly.
	st := Hand
	for _, e := range Elements() {
		s := strike(e)
		s.Selected = true
		got := render(t, s, st).RGBAAt(st.BorderWidth/2, st.Height/2)
		if got != borderBase(e, "") {
			t.Errorf("%s selected border is %v, want the named color %v", e, got, borderBase(e, ""))
		}
	}
}

func TestSelectedIsLouderThanResting(t *testing.T) {
	st := Hand
	mid := st.Height / 2
	for _, e := range Elements() {
		rest := strike(e)
		sel := strike(e)
		sel.Selected = true

		r := render(t, rest, st).RGBAAt(st.BorderWidth/2, mid)
		s := render(t, sel, st).RGBAAt(st.BorderWidth/2, mid)
		if distance(s, Surface) <= distance(r, Surface) {
			t.Errorf("%s: selected border is no further from the surface than resting", e)
		}
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	st := Hand
	s := strike(Lightning)
	a, b := render(t, s, st), render(t, s, st)
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			t.Fatalf("two renders of the same card differ at byte %d", i)
		}
	}
}

func TestRenderRejectsWhatItCannotDraw(t *testing.T) {
	if _, err := Render(strike(Fire), Hand, nil); err == nil {
		t.Error("rendering with no fonts succeeded, want an error")
	}
	if _, err := Render(strike(Fire), Style{}, faces(t)); err == nil {
		t.Error("rendering into a zero-sized style succeeded, want an error")
	}
}

func TestUnknownElementFallsBackRatherThanPanicking(t *testing.T) {
	// Element is an int and nothing stops a caller inventing one. A card drawn in the
	// wrong color is recoverable; a crash in the middle of a duel is not.
	if got := BorderOf(Element(99)); got != BorderOf(Basic) {
		t.Errorf("out-of-range element gave %v, want the Basic border %v", got, BorderOf(Basic))
	}
	if got := Element(99).String(); got != "?" {
		t.Errorf("out-of-range element is named %q, want %q", got, "?")
	}
}

// distance is how far apart two colors are, summed per channel. Used instead of
// luminance because "quieter" on a light card means closer to the surface, in whichever
// direction that happens to be.
func distance(a, b color.RGBA) int {
	return abs(int(a.R)-int(b.R)) + abs(int(a.G)-int(b.G)) + abs(int(a.B)-int(b.B))
}

func TestTheFormMarkStaysCornerSized(t *testing.T) {
	// **The bound is the invariant, not the number.** How big the corner mark is stays a design
	// choice, but one that grows past a fifth of the card's width is a full-size picture in a
	// corner slot, and it walks into both the tick stack under it and the text column beside it.
	//
	// The floor matters as much as the ceiling: below about 16 pixels a drawn mark has nothing
	// left to read.
	//
	// **It was measured against the generated glyphs' canvas until 2026-09-16** — a reasonable
	// yardstick while the mark was one of them, and a dangling reference to a deleted constant once
	// it was not. The card's own width is the honest measure: what the bound is about is how much of
	// the face the corner may take.
	for name, st := range map[string]Style{"hand": Hand, "mini": Mini} {
		if !st.ShowForm {
			continue
		}
		if st.FormSize < 16 || st.FormSize > st.Width/5 {
			t.Errorf("%s form box is %dpx, want between 16 and a fifth of the card's %d",
				name, st.FormSize, st.Width)
		}
	}
}

func TestDraggingKeepsItsBorderAndGhostsItsFace(t *testing.T) {
	// Dragging and disabled must not look alike: one is the card you are acting on and
	// the other is a card you cannot act on. The border is what separates them — full
	// strength while dragged, dimmed while disabled.
	st := Hand
	mid := st.Height / 2
	probe := image.Point{X: st.Width - st.BorderWidth - 4, Y: st.Height - st.BorderWidth - 4}

	drag := strike(Fire)
	drag.Dragging = true
	dead := strike(Fire)
	dead.Enabled = false

	dragImg, deadImg := render(t, drag, st), render(t, dead, st)

	if got := dragImg.RGBAAt(st.BorderWidth/2, mid); got != borderBase(Fire, "") {
		t.Errorf("dragged border is %v, want full strength %v", got, borderBase(Fire, ""))
	}
	if dragImg.RGBAAt(st.BorderWidth/2, mid) == deadImg.RGBAAt(st.BorderWidth/2, mid) {
		t.Error("dragged and disabled draw the same border — they mean opposite things")
	}

	// The face does ghost, or nothing would say the card is in the air.
	if got := dragImg.RGBAAt(probe.X, probe.Y); got == Surface {
		t.Error("a dragged card's face is the resting surface — nothing marks it as lifted")
	}
}

func duelist(life, of int) Spec {
	s := Spec{Name: "Duelist", Element: Basic, Life: life, MaxLife: of, Enabled: true}
	s.Stats[0] = StatLine{Label: "DMG", Value: "10"}
	s.Stats[1] = StatLine{Label: "AP", Value: "6"}
	s.Stats[2] = StatLine{Label: "Vitae", Value: "5"}
	return s
}

func TestTheTwoFighterCardsShareTheirHealthGeometry(t *testing.T) {
	// **They face each other across the screen.** A bar at a different height on each would
	// turn comparing them into an act of measurement, which is the one thing a bar exists to
	// avoid.
	for _, c := range []struct {
		what             string
		duelist, enemyAt int
	}{
		{"bar top", DuelistStyle.HealthBarTop, EnemyStyle.HealthBarTop},
		{"bar height", DuelistStyle.HealthBarHeight, EnemyStyle.HealthBarHeight},
		{"bar inset", DuelistStyle.HealthBarInset, EnemyStyle.HealthBarInset},
		{"fraction top", DuelistStyle.HealthTextTop, EnemyStyle.HealthTextTop},
		{"width", DuelistStyle.Width, EnemyStyle.Width},
		{"height", DuelistStyle.Height, EnemyStyle.Height},
	} {
		if c.duelist != c.enemyAt {
			t.Errorf("%s is %d on the duelist card and %d on the enemy's", c.what, c.duelist, c.enemyAt)
		}
	}
	if DuelistStyle.HealthTextSize != EnemyStyle.HealthTextSize {
		t.Errorf("the fraction is %gpt on the duelist card and %gpt on the enemy's",
			DuelistStyle.HealthTextSize, EnemyStyle.HealthTextSize)
	}
}

func TestStatusBadgesClearTheHealthTextAndTheBorder(t *testing.T) {
	// The badges live in whatever the fraction leaves above the bottom border, which is about
	// twenty pixels. Both ends matter and neither is obvious by eye at that size: a badge over
	// the fraction makes the life total unreadable, and one through the border squares off the
	// card's corner. Moving `HealthTextTop` to make room would move it on the *duelist* card
	// too — see the twins rule — so this failing is a decision, not a nudge.
	st := EnemyStyle
	f := faces(t)
	face, err := f.at(st.HealthTextSize)
	if err != nil {
		t.Fatal(err)
	}
	m := face.Metrics()

	// The fraction is digits and a slash, so its ink stops at the baseline: drawText places the
	// string's top at HealthTextTop and drops the dot by the ascent.
	textBottom := st.HealthTextTop + m.Ascent.Ceil()
	if st.EffectTop < textBottom {
		t.Errorf("the badges start at y=%d, %dpx into the hit points ending at y=%d",
			st.EffectTop, textBottom-st.EffectTop, textBottom)
	}

	if bottom, inside := st.EffectTop+st.EffectSize, st.Height-st.BorderWidth; bottom > inside {
		t.Errorf("the badges end at y=%d, %dpx into the bottom border at y=%d",
			bottom, bottom-inside, inside)
	}

	// And a full row fits across the card between its side borders.
	span := MaxEffects*st.EffectSize + (MaxEffects-1)*st.EffectGap
	if usable := st.Width - 2*st.BorderWidth; span > usable {
		t.Errorf("%d badges span %dpx across a card %dpx wide inside its borders",
			MaxEffects, span, usable)
	}
}

func TestAnEffectRowIsCenteredAndClosesUpAsItEmpties(t *testing.T) {
	// Nil entries are skipped rather than drawn as holes, so one status sits in the middle of
	// the card. What this checks is that the *drawn* row is centered for every count — the
	// failure it guards is a row laid out against MaxEffects, which leaves a single badge
	// hard left with three empty slots beside it.
	st := EnemyStyle
	f := faces(t)

	solid := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			solid.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}

	for count := 1; count <= MaxEffects; count++ {
		spec := Spec{Name: "Goblin", Element: Basic, Life: 10, MaxLife: 10, Enabled: true}
		for i := 0; i < count; i++ {
			spec.Effects[i] = solid
		}

		img, err := Render(spec, st, f)
		if err != nil {
			t.Fatal(err)
		}

		band := inkBounds(subImage(img, image.Rect(0, st.EffectTop, st.Width, st.EffectTop+st.EffectSize)))
		if band.Empty() {
			t.Fatalf("%d badges drew nothing in the effect band", count)
		}

		// Centered: the margin either side of the drawn row has to match within a pixel, which
		// is all integer division can promise.
		leftGap, rightGap := band.Min.X, st.Width-band.Max.X
		if diff := leftGap - rightGap; diff > 1 || diff < -1 {
			t.Errorf("%d badges sit %dpx from the left and %dpx from the right",
				count, leftGap, rightGap)
		}
	}
}

// subImage copies a rectangle of a card out for measuring, since inkBounds walks a whole image.
func subImage(src *image.RGBA, r image.Rectangle) *image.RGBA {
	out := image.NewRGBA(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			out.SetRGBA(x, y, src.RGBAAt(x, y))
		}
	}
	return out
}

func TestStatRowsClearTheHealthBar(t *testing.T) {
	// MaxStatLines is a *layout* cap, so this is what makes it one: the full ladder has to
	// finish above the bar. Raising the constant without moving the bar fails here rather
	// than printing "Vitae" through the player's health.
	st := DuelistStyle
	f := faces(t)
	face, err := f.at(st.StatSize)
	if err != nil {
		t.Fatal(err)
	}
	m := face.Metrics()
	rowHeight := m.Ascent.Ceil() + m.Descent.Ceil()

	// **statRowTop rather than the pitch**, so the rule's extra gap is counted — the arithmetic
	// here disagreeing with the renderer's is exactly how a guard comes to pass while the last row
	// is drawn through the bar.
	bottom := statRowTop(st, MaxStatLines-1) + rowHeight
	if bottom > st.HealthBarTop {
		t.Errorf("%d stat rows end at y=%d, %dpx into the health bar at y=%d",
			MaxStatLines, bottom, bottom-st.HealthBarTop, st.HealthBarTop)
	}

	// And they start inside the card rather than on its top border. **There is no name to clear
	// any more** — the check that used to be here went with it; see
	// TestTheEnemyNamesItselfAboveItsPortrait, which holds the absence.
	if st.StatsTop < st.BorderWidth {
		t.Errorf("the first stat row is at y=%d, on the %dpx border", st.StatsTop, st.BorderWidth)
	}
}

// The rule is what stops five rows reading as one list, so where it lands is the whole of it: in
// the air between the two groups, touching neither.
func TestTheStatRuleSitsBetweenTheTwoGroups(t *testing.T) {
	st := DuelistStyle
	if st.StatRuleAfter <= 0 || st.StatRuleGap <= 0 {
		t.Fatal("the duelist card has no rule between its stat groups")
	}
	if st.StatRuleAfter >= MaxStatLines {
		t.Fatalf("the rule sits after row %d of %d, so it is under every row rather than between two",
			st.StatRuleAfter, MaxStatLines)
	}

	f := faces(t)
	face, err := f.at(st.StatSize)
	if err != nil {
		t.Fatal(err)
	}
	m := face.Metrics()
	rowHeight := m.Ascent.Ceil() + m.Descent.Ceil()

	above := statRowTop(st, st.StatRuleAfter-1) + rowHeight
	below := statRowTop(st, st.StatRuleAfter)

	if below-above < statRuleHeight+2 {
		t.Errorf("the gap the rule sits in is %dpx, too tight for a %dpx rule and any air",
			below-above, statRuleHeight)
	}

	// The rows either side of it are still at the plain pitch, so the gap is the rule's alone.
	if plain := statRowTop(st, 1) - statRowTop(st, 0); below-statRowTop(st, st.StatRuleAfter-1) <= plain {
		t.Errorf("the rule's row gap is %dpx against a plain pitch of %dpx; it opened no air",
			below-statRowTop(st, st.StatRuleAfter-1), plain)
	}
}

func TestStatRowsAreDrawnAsALabelLeftAndAFigureRight(t *testing.T) {
	// The figures line up as a column whatever their width, which is what right-aligning
	// them buys and the reason they are not simply drawn after the label.
	st := DuelistStyle
	img := render(t, duelist(74, 120), st)

	for i := 0; i < 3; i++ {
		top := st.StatsTop + i*st.StatRowPitch
		left, right := st.Width, 0
		for y := top; y < top+st.StatRowPitch-4; y++ {
			for x := st.BorderWidth; x < st.Width-st.BorderWidth; x++ {
				if img.RGBAAt(x, y) != Surface {
					if x < left {
						left = x
					}
					if x > right {
						right = x
					}
				}
			}
		}
		if right <= left {
			t.Fatalf("stat row %d drew nothing", i)
		}
		if left != st.TextLeft {
			t.Errorf("stat row %d starts at x=%d, want the left margin at x=%d", i, left, st.TextLeft)
		}
		// Within a pixel of the right margin: the glyph's own bearing can leave one column
		// clear, and demanding an exact hit would be measuring the font rather than the layout.
		if want := st.Width - st.TextLeft; want-right > 2 {
			t.Errorf("stat row %d ends at x=%d, %dpx short of the right margin at x=%d",
				i, right, want-right, want)
		}
	}
}

func TestAStyleWithNoStatsDrawsNone(t *testing.T) {
	// Every other style leaves StatRowPitch at zero, and a Spec carrying stats must not put
	// them on one. The hand card is the case that matters: its left column is exactly where
	// the rows would land.
	s := strike(Fire)
	s.Stats[0] = StatLine{Label: "DMG", Value: "10"}

	plain, withStats := render(t, strike(Fire), Hand), render(t, s, Hand)
	for i := range plain.Pix {
		if plain.Pix[i] != withStats.Pix[i] {
			t.Fatalf("stats on a Spec changed a hand card, which draws none (byte %d)", i)
		}
	}
}

func TestABlankStatRowLeavesItsRowEmpty(t *testing.T) {
	// The rows are a fixed ladder with a health bar placed under them, so a card with a gap
	// in its figures must not close up — otherwise the same card at two moments in a run has
	// its numbers at different heights.
	st := DuelistStyle

	full := duelist(74, 120)
	gapped := full
	gapped.Stats[1] = StatLine{}

	fullImg, gappedImg := render(t, full, st), render(t, gapped, st)

	// The third row is identical in both, which is only true if the blank row held its place.
	top := st.StatsTop + 2*st.StatRowPitch
	for y := top; y < top+st.StatRowPitch; y++ {
		for x := st.BorderWidth; x < st.Width-st.BorderWidth; x++ {
			if fullImg.RGBAAt(x, y) != gappedImg.RGBAAt(x, y) {
				t.Fatalf("the last stat row moved when the one above it was blanked, at (%d,%d)", x, y)
			}
		}
	}
}

func TestTheEnemyNamesItselfAboveItsPortrait(t *testing.T) {
	// **A card that names itself names itself across the top** — the enemy did not until
	// 2026-08-12, and this is the invariant rather than the offset: a card whose name sits
	// somewhere else reads as a different kind of object.
	//
	// **A card whose picture is its whole face names itself nowhere** *(owner's call,
	// 2026-09-11)*, which is the other half of the same rule: a title bar over a full-bleed
	// illustration covers the one thing worth looking at in order to repeat it. The two bleeding
	// styles are checked for the absence, so turning a name back on is a decision rather than an
	// accident.
	for name, st := range map[string]Style{"enemy": EnemyStyle, "hand": Hand} {
		if !st.ShowName || !st.NameCentered {
			t.Errorf("%s does not center a name across its top", name)
		}
	}
	for name, st := range map[string]Style{"relic": RelicStyle, "essence": EssenceStyle} {
		if !st.ArtBleed {
			t.Errorf("%s is no longer full-bleed — this test is checking the wrong styles", name)
		}
		if st.ShowName {
			t.Errorf("%s writes a title across its own picture", name)
		}
	}

	// **The duelist card is the third case and it is neither** *(owner's call, 2026-09-15)*: no
	// name and no picture. It carried "Duelist" until then, which spent the best line on the card
	// repeating what the corner it sits in already says. Its top line is the first stat row.
	if DuelistStyle.ShowName {
		t.Error("the duelist card names itself again; the corner it sits in already says who it is")
	}
	if DuelistStyle.ArtBleed {
		t.Error("this test is checking the wrong style — the duelist card has no picture")
	}
	if EnemyStyle.NameTop >= EnemyStyle.ArtTop {
		t.Errorf("the enemy's name is at y=%d, at or below its portrait at y=%d",
			EnemyStyle.NameTop, EnemyStyle.ArtTop)
	}
	// And the portrait still clears the bar under it.
	if bottom := EnemyStyle.ArtTop + EnemyStyle.ArtMaxH; bottom > EnemyStyle.HealthBarTop {
		t.Errorf("the portrait box ends at y=%d, %dpx into the health bar at y=%d",
			bottom, bottom-EnemyStyle.HealthBarTop, EnemyStyle.HealthBarTop)
	}
}

// The back is one drawing with one job, so there are only three things to pin: that it is
// the same object as a face, that it says nothing about which card it is, and that it does
// not need the things a face needs.

func TestBackHasExactlyTheFaceSilhouette(t *testing.T) {
	// The load-bearing property. A flip swaps one picture for the other mid-animation and a
	// stack of backs sits beside the hand, so a back whose outline differed by even a corner
	// pixel would read as a different kind of object.
	for name, st := range map[string]Style{"hand": Hand, "mini": Mini} {
		face := render(t, strike(Fire), st)
		back := render(t, Spec{FaceDown: true}, st)

		for y := 0; y < st.Height; y++ {
			for x := 0; x < st.Width; x++ {
				if (face.RGBAAt(x, y).A == 0) != (back.RGBAAt(x, y).A == 0) {
					t.Fatalf("%s: silhouette differs at (%d,%d): face alpha %d, back alpha %d",
						name, x, y, face.RGBAAt(x, y).A, back.RGBAAt(x, y).A)
				}
			}
		}
	}
}

func TestBackIsTheSamePictureWhateverTheCard(t *testing.T) {
	// The draw pile is shuffled, so a back that varied with the card under it would hand the
	// player the order. Every field but FaceDown has to reach the drawing as nothing.
	want := render(t, Spec{FaceDown: true}, Hand)

	loud := Spec{
		Name: "Prepare", Form: FormDefend, Cost: 4, Text: "Bank 2 AP",
		Element: Lightning, Enabled: true, Selected: true, FaceDown: true,
	}
	got := render(t, loud, Hand)

	for y := 0; y < Hand.Height; y++ {
		for x := 0; x < Hand.Width; x++ {
			if got.RGBAAt(x, y) != want.RGBAAt(x, y) {
				t.Fatalf("a filled-in Spec changed the back at (%d,%d): %v vs %v",
					x, y, got.RGBAAt(x, y), want.RGBAAt(x, y))
			}
		}
	}
}

func TestBackRendersWithNoFont(t *testing.T) {
	// A back draws no text, and a missing font must not be able to empty the draw pile —
	// which is what a returned error would do, since the caller draws nothing on nil.
	img, err := Render(Spec{FaceDown: true}, Hand, nil)
	if err != nil {
		t.Fatalf("rendering a back without fonts: %v", err)
	}
	if img.RGBAAt(Hand.Width/2, Hand.Height/2) != BackInk {
		t.Errorf("center of the back is %v, want the mark %v",
			img.RGBAAt(Hand.Width/2, Hand.Height/2), BackInk)
	}
}

func TestEffectTextBreaksAtEveryWord(t *testing.T) {
	// **One word to a line** *(owner's call, 2026-09-05)*. The width is deliberately far wider than
	// the text: a measurer that fitted the words onto one line is exactly what this replaced, and a
	// generous column is the case that would hide it.
	f := faces(t)
	lines, err := WrapText(f, EssenceStyle.TextSize, "CARD BECOMES FIRE", 400)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 || lines[0] != "CARD" || lines[1] != "BECOMES" || lines[2] != "FIRE" {
		t.Errorf("the text did not break at every word: got %q", lines)
	}
}

func TestANewlineIsJustASpace(t *testing.T) {
	// A newline used to be an authored break and is now nothing special — every space breaks, so
	// there is nothing left for one to ask for. It must not produce an empty line.
	f := faces(t)
	lines, err := WrapText(f, EssenceStyle.TextSize, "CARD BECOMES\nFIRE", 400)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 {
		t.Errorf("a newline did not read as a space: got %q", lines)
	}
}

// TestTheBorderIsTheSameWhateverTheElement is the other half of the 2026-08-23 swap: the border
// used to be the element signal and is now the state signal, so two cards of different elements
// in the same state must be indistinguishable at the edge. Without this the old behavior could
// come back one element at a time and each card would still look defensible on its own.
func TestTheBorderIsTheSameWhateverTheElement(t *testing.T) {
	st := Hand
	mid := st.Height / 2

	want := render(t, strike(Basic), st).RGBAAt(st.BorderWidth/2, mid)
	for _, e := range Elements() {
		if got := render(t, strike(e), st).RGBAAt(st.BorderWidth/2, mid); got != want {
			t.Errorf("%s border is %v, want the same neutral %v every other element draws", e, got, want)
		}
	}
}

// TestARelicBordersByItsRarity is what replaced TestARelicStillBordersPink *(2026-09-13)*. The
// pink survived the 2026-08-23 swap because that change was about elements and pink was never one;
// this change is about relics, so the color it was holding is the color being spent.
//
// What still has to hold is the thing the pink was *for*: no relic may border like a playable
// card. That is now three colors to check rather than one, which is why this asks the question of
// every rarity rather than of a constant.
func TestARelicBordersByItsRarity(t *testing.T) {
	seen := map[color.RGBA]data.Rarity{}
	for _, r := range data.Rarities() {
		got := borderBase(Relic, r)

		if want, ok := rarityBorders[r]; !ok || got != want {
			t.Errorf("a %s relic borders %v, want %v", r, got, want)
		}
		for _, e := range Elements() {
			if got == borderBase(e, "") {
				t.Errorf("a %s relic borders like a %s card — a card and a relic must not look alike", r, e)
			}
		}
		if other, dup := seen[got]; dup {
			t.Errorf("%s and %s border the same %v — rarity has to be readable off the color", r, other, got)
		}
		seen[got] = r
	}
}

// TestEveryRarityHasABorder is the tripwire for a fourth tier. data.Rarities() is the vocabulary
// and rarityBorders is the palette; a rarity added to one and not the other would fall through to
// the pink below and read as a broken record rather than as a new tier.
func TestEveryRarityHasABorder(t *testing.T) {
	for _, r := range data.Rarities() {
		if _, ok := rarityBorders[r]; !ok {
			t.Errorf("rarity %q has no border color", r)
		}
	}
	if len(rarityBorders) != len(data.Rarities()) {
		t.Errorf("rarityBorders has %d entries for %d rarities — one of them names nothing",
			len(rarityBorders), len(data.Rarities()))
	}
}

// TestAnUnclassifiedRelicFallsBackToThePink pins what the pink means now. A relic whose rarity is
// not one of the three is a record that failed validation or a key naming no record at all — the
// scenario sheet's missingSpec draws exactly that — and it has to look wrong rather than look
// common.
func TestAnUnclassifiedRelicFallsBackToThePink(t *testing.T) {
	if got := borderBase(Relic, ""); got != BorderOf(Relic) {
		t.Errorf("an unclassified relic borders %v, want the pink %v", got, BorderOf(Relic))
	}
	for _, r := range data.Rarities() {
		if borderBase(Relic, r) == BorderOf(Relic) {
			t.Errorf("a %s relic borders in the fallback pink — a real tier and a broken record look alike", r)
		}
	}
}

// TestCommonIsNotWhite is the trap borderColors[Basic] documents, one card format over. The bevel
// is derived by pushing the fill toward white, so a white border has nowhere to climb and goes
// flat; the card also sits on a light table, where a white ring reads as no ring. Bone is the
// answer and this is what stops it drifting back.
func TestCommonIsNotWhite(t *testing.T) {
	c := rarityBorders[data.Common]
	if c.R > 230 && c.G > 230 && c.B > 230 {
		t.Errorf("the common border is %v — a near-white ring has no bevel and no edge against the table", c)
	}
	if lit := systems.ColorToward(c, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 100); lit == c {
		t.Error("the common border cannot be lightened, so its bevel has no light edge")
	}
}

// TestTheFormMarkCarriesTheElement is where the color went. Every pair of elements must draw a
// different mark, which is the property the swap was for — if two elements paint the same corner
// then the border was neutralized and nothing took over from it.
//
// **It asks for difference rather than for a particular color**, because the mark is authored art
// rather than a fill: its pixels are a material shaded in the element's hue, and the brightest of
// them is genuinely not the color in borderColors. What the card owes is that two elements never
// draw the same corner, which is what the swap was for.
func TestTheFormMarkCarriesTheElement(t *testing.T) {
	st := Hand
	box := image.Rect(st.GlyphInset, st.FormTop,
		st.GlyphInset+st.FormSize, st.FormTop+st.FormSize)

	marks := map[Element][]color.RGBA{}
	for _, e := range Elements() {
		img := render(t, strike(e), st)
		var px []color.RGBA
		for y := box.Min.Y; y < box.Max.Y; y++ {
			for x := box.Min.X; x < box.Max.X; x++ {
				px = append(px, img.RGBAAt(x, y))
			}
		}
		marks[e] = px
	}

	same := func(a, b []color.RGBA) bool {
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	for i, e := range Elements() {
		for _, other := range Elements()[i+1:] {
			if same(marks[e], marks[other]) {
				t.Errorf("%s and %s draw an identical form mark — the element is not reaching it",
					e, other)
			}
		}
	}
}

// TestTheTicksAndTheBorderShareOneState is why atState exists. The two are drawn from different
// base colors and must move together: a selected card with a lit border and resting ticks is the
// failure a second copy of the state switch produces, and it looks like a rendering glitch rather
// than like a bug.
func TestTheTicksAndTheBorderShareOneState(t *testing.T) {
	st := Hand
	mid := st.Height / 2

	// The tick at full strength: a selected card, which tickFade leaves alone. Every other state
	// is measured as a distance from this.
	full := strike(Fire)
	full.Cost = 2
	full.Selected = true
	lit := tickCore(render(t, full, st), st, 0)

	for _, tc := range []struct {
		name  string
		shape func(*Spec)
	}{
		{"resting", func(*Spec) {}},
		{"selected", func(s *Spec) { s.Selected = true }},
		{"dragging", func(s *Spec) { s.Dragging = true }},
		{"disabled", func(s *Spec) { s.Enabled = false }},
	} {
		s := strike(Fire)
		s.Cost = 2
		tc.shape(&s)
		img := render(t, s, st)

		gotBorder := img.RGBAAt(st.BorderWidth/2, mid)

		if want := s.atState(borderBase(Fire, "")); gotBorder != want {
			t.Errorf("%s border is %v, want %v", tc.name, gotBorder, want)
		}

		// **The tick is measured as a distance, not as a color.** It is a drawing, so what has to
		// match the border is how far the state moves it toward the surface — tickFade is the one
		// number both halves of the column read, and a second copy of that switch is what this
		// test has always existed to catch.
		ink := tickCore(img, st, 0)
		to := Surface
		if !s.Enabled {
			to = SurfaceDisabled
		}
		if want := systems.ColorToward(lit, to, tickFade(s)); !nearColor(ink, want, 1) {
			t.Errorf("%s tick is %v, want %v — it is not at the border's state", tc.name, ink, want)
		}
	}
}

// **A token says the three things a hand is counted on, and nothing else.** Element is the
// border color of its mark and its ticks, form is the mark, cost is the ticks — so a style that
// stopped drawing one of them would be a row of tokens that cannot be read as a hand.
func TestATokenSaysElementFormAndCost(t *testing.T) {
	st := Token
	if !st.ShowForm {
		t.Error("Token draws no form mark, which is one of the three things it is for")
	}
	if st.ShowName {
		t.Error("Token draws a name, which does not fit 40 pixels and is not counted on")
	}
	if st.TextLineHeight > 0 {
		t.Error("Token draws effect text, which cannot be read at this size")
	}
	if st.DashWidth <= 0 || st.DashHeight <= 0 {
		t.Error("Token draws no cost ticks")
	}
}

// **Four ticks fit inside the border**, because drawDashes drops a tick that would run off the
// card rather than complaining — so a cost the layout cannot hold is a card that silently
// understates what it costs. Every card in the game runs 1..3 today; a fourth is what this holds.
func TestATokenHoldsFourTicks(t *testing.T) {
	st := Token
	bottom := st.DashTop + 3*(st.DashHeight+st.DashGap) + st.DashHeight
	if inside := st.Height - st.BorderWidth; bottom > inside {
		t.Errorf("a four-point token's ticks reach y=%d against an inside edge at %d",
			bottom, inside)
	}
	if top := st.FormTop + st.FormSize; st.DashTop < top {
		t.Errorf("the ticks start at y=%d, inside the form mark that ends at %d", st.DashTop, top)
	}
}

// **The mark and the ticks are centered**, because there is no text column to their right for a
// left-aligned column to line up with — the same reason a relic centers its name.
func TestATokenCentersItsColumn(t *testing.T) {
	st := Token
	if want := (st.Width - st.FormSize) / 2; st.GlyphInset != want {
		t.Errorf("the form mark sits at x=%d against a centered %d", st.GlyphInset, want)
	}
	if want := (st.Width - st.DashWidth) / 2; st.DashLeft != want {
		t.Errorf("the ticks sit at x=%d against a centered %d", st.DashLeft, want)
	}
}

// TestAFigureStaysOnItsUnitsLine is the one exception to breaking at every word: a number and the
// unit it belongs to are one fact, and a number alone on a line says nothing until the eye reaches
// the line under it. **The unit leads whichever way the text was authored** — see unitLines.
func TestAFigureStaysOnItsUnitsLine(t *testing.T) {
	f := faces(t)

	cases := []struct {
		text string
		want []string
	}{
		{"CARD BECOMES EASIER -1 AP", []string{"CARD", "BECOMES", "EASIER", "AP -1"}},
		{"CARD GAINS DMG +50%", []string{"CARD", "GAINS", "DMG +50%"}},
		{"CARD GAINS STRENGTH +1 AP", []string{"CARD", "GAINS", "STRENGTH", "AP +1"}},
		{"CARD BECOMES STAB", []string{"CARD", "BECOMES", "STAB"}},
	}

	for _, c := range cases {
		lines, err := WrapText(f, EssenceStyle.TextSize, c.text, 400)
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) != len(c.want) {
			t.Errorf("%q wrapped to %q, want %q", c.text, lines, c.want)
			continue
		}
		for i := range lines {
			if lines[i] != c.want[i] {
				t.Errorf("%q wrapped to %q, want %q", c.text, lines, c.want)
				break
			}
		}
	}
}

// TestTheBottomBlockFitsInsideTheCard is what the fighter cards' packed lower half is held by.
//
// **The empty space under the fraction is reserved, not spare** *(owner's call, 2026-09-15)*. The
// pip row draws nothing until shields are standing, so a duel that has raised none looks like fifty
// pixels of slack — and a bar nudged down into it would leave a row of pips with nowhere to land
// the first time a Guard goes down. So the block is measured at its *reserved* extent: bar, then
// fraction, then the full pip row, inside the bottom border.
//
// Both fighter cards are checked, because the two carry the bar at identical offsets on purpose —
// they face each other across the table, and a bar at a different height on each would make
// comparing them an act of measurement.
func TestTheBottomBlockFitsInsideTheCard(t *testing.T) {
	f := faces(t)

	for _, c := range []struct {
		name string
		st   Style
	}{{"duelist", DuelistStyle}, {"enemy", EnemyStyle}} {
		face, err := f.at(c.st.HealthTextSize)
		if err != nil {
			t.Fatal(err)
		}
		m := face.Metrics()
		textBottom := c.st.HealthTextTop + m.Ascent.Ceil() + m.Descent.Ceil()

		if bar := c.st.HealthBarTop + c.st.HealthBarHeight; bar > c.st.HealthTextTop {
			t.Errorf("%s: the bar ends at y=%d, under the fraction at y=%d",
				c.name, bar, c.st.HealthTextTop)
		}
		if textBottom > c.st.EffectTop {
			t.Errorf("%s: the fraction ends at y=%d, %dpx into the pip row at y=%d",
				c.name, textBottom, textBottom-c.st.EffectTop, c.st.EffectTop)
		}

		// The pip row is the floor, and it is drawn whether or not anything is in it today.
		inside := c.st.Height - c.st.BorderWidth - 4
		if pips := c.st.EffectTop + c.st.EffectSize; pips > inside {
			t.Errorf("%s: the pip row ends at y=%d, %dpx past the inside of the border at y=%d",
				c.name, pips, pips-inside, inside)
		}
	}

	// And the two agree, which is the property that makes the cards comparable across the table.
	if DuelistStyle.HealthBarTop != EnemyStyle.HealthBarTop ||
		DuelistStyle.HealthTextTop != EnemyStyle.HealthTextTop {
		t.Errorf("the fighter cards carry their bars at different heights: duelist %d/%d, enemy %d/%d",
			DuelistStyle.HealthBarTop, DuelistStyle.HealthTextTop,
			EnemyStyle.HealthBarTop, EnemyStyle.HealthTextTop)
	}
}

// TestNoMarkPunchesAHoleInTheFace holds the card's face opaque under every drawing on it.
//
// A mark's edges are anti-aliased, so writing one straight into the face carried the drawing's own
// transparency into the card — and the table showed through it as a pale square round the corner.
// A pixel of a card is either outside its rounded silhouette and clear, or on the face and opaque;
// nothing in between. See blitGlyph, which composites rather than writes.
func TestNoMarkPunchesAHoleInTheFace(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		for _, form := range []Form{FormStab, FormSlash, FormCrush, FormDefend} {
			s := strike(Fire)
			s.Form, s.Enabled = form, enabled
			img := render(t, s, Hand)

			for y := 0; y < Hand.Height; y++ {
				for x := 0; x < Hand.Width; x++ {
					if a := img.RGBAAt(x, y).A; a != 0 && a != 255 {
						t.Fatalf("%v enabled=%v is %d%% opaque at (%d,%d): a mark's soft edge "+
							"went into the face instead of onto it",
							form, enabled, int(a)*100/255, x, y)
					}
				}
			}
		}
	}
}
