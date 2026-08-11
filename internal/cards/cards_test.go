package cards

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"golang.org/x/image/font"
)

// These tests assert *pixels*, not that the code ran.
//
// A contact sheet is the usual way to check drawing here, and it is the right tool for
// "does this look good". It is the wrong tool for "is the border six pixels thick" and
// "did the disabled card come out louder than the live one", because those are true or
// false and nobody should have to squint at a PNG to find out. The corners are
// deliberately hard-edged, so every pixel is exactly one colour and can be compared
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
	"Gather", "Sift", "Guard", "Ritual", "Jab", "Strike",
	"Feint", "Heavy", "Brace", "Dodge", "Riposte", "Mirror",
}

func strike(e Element) Spec {
	return Spec{Name: "Strike", Category: CategoryAttack, Damage: 7, Cost: 2, Element: e, Enabled: true}
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

func TestBorderIsTheElementColourAtItsDeclaredWidth(t *testing.T) {
	st := Hand
	mid := st.Height / 2

	for _, e := range Elements() {
		img := render(t, strike(e), st)
		want := systems.ColorToward(BorderOf(e), Surface, borderRestToward)

		// Walk in from the left edge at the card's waist, where there is no curvature.
		for x := 0; x < st.BorderWidth; x++ {
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

func TestEveryElementBorderIsDistinct(t *testing.T) {
	// Basic is a mid grey rather than the near-white the screen uses as a surface,
	// precisely so it is not the same colour as the card it sits on. If someone
	// "restores" it to {235,235,235} this fails.
	seen := map[color.RGBA]Element{}
	for _, e := range Elements() {
		c := BorderOf(e)
		if prev, dup := seen[c]; dup {
			t.Errorf("%s and %s share the border colour %v", prev, e, c)
		}
		seen[c] = e

		if c == Surface {
			t.Errorf("%s border is the same colour as the card surface — it would be invisible", e)
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

func TestCostDrawsOneDashPerPoint(t *testing.T) {
	st := Hand
	x := st.DashLeft + st.DashWidth/2

	for cost := 0; cost <= 4; cost++ {
		s := strike(Fire)
		s.Cost = cost
		img := render(t, s, st)

		border := systems.ColorToward(BorderOf(Fire), Surface, borderRestToward)
		count := 0
		for i := 0; i < 6; i++ {
			y := st.DashTop + i*(st.DashHeight+st.DashGap) + st.DashHeight/2
			if img.RGBAAt(x, y) == border {
				count++
			}
		}
		if count != cost {
			t.Errorf("cost %d drew %d dashes", cost, count)
		}
	}
}

func TestLeftColumnDoesNotCollide(t *testing.T) {
	// **The left column is a stack of three things now**: category glyph, cost dashes,
	// damage badge, one under the other. Adding the category glyph is what made this
	// worth a test — before it there was one badge and a corner, and nothing could
	// overlap anything.
	//
	// Four dashes is the most the rules can produce today. A fifth cost tier grows the
	// stack into the damage badge, which is a layout change and not just a bigger number.
	// This fails when that happens rather than rendering the collision.
	const maxCost = 4

	st := Hand

	// Measured per glyph, never assumed: the category glyphs are 22 pixels and the damage
	// sword is 64. The tallest category glyph is the one that has to fit.
	categoryGlyph := 0
	for _, c := range Categories() {
		kind, ok := c.glyph()
		if !ok {
			t.Fatalf("%s has no glyph", c)
		}
		if n := systems.SizeOf(kind) * st.GlyphScale; n > categoryGlyph {
			categoryGlyph = n
		}
	}

	categoryBottom := st.CategoryGlyphTop + categoryGlyph
	dashBottom := st.DashTop + (maxCost-1)*(st.DashHeight+st.DashGap) + st.DashHeight
	damageBottom := st.GlyphColumnTop + systems.GlyphSize*st.GlyphScale

	if categoryBottom > st.DashTop {
		t.Errorf("category glyph ends at y=%d, %dpx into the dash stack at y=%d",
			categoryBottom, categoryBottom-st.DashTop, st.DashTop)
	}
	if dashBottom > st.GlyphColumnTop {
		t.Errorf("%d dashes end at y=%d, %dpx into the damage badge at y=%d",
			maxCost, dashBottom, dashBottom-st.GlyphColumnTop, st.GlyphColumnTop)
	}
	if inside := st.Height - st.BorderWidth; damageBottom > inside {
		t.Errorf("damage badge ends at y=%d, past the inside of the bottom border at y=%d",
			damageBottom, inside)
	}
}

func TestMiniShowsEverythingButDamage(t *testing.T) {
	// **What the overlay can say is set by what fits in the visible strip.** At a third
	// size, 29 pixels showed and only dashes fit; at half with the row widened, 84 of the
	// 90 show and the name, glyph and dashes all fit. This pins that, because the tempting
	// regression is to re-tighten the pitch for a longer row and silently lose the name.
	visible := deckVisibleWidth

	if !Mini.ShowName {
		t.Error("Mini does not show the name, which is the only thing identifying a concept")
	}
	if !Mini.ShowCategory {
		t.Error("Mini does not show the category glyph")
	}
	if glyphRight := Mini.GlyphInset + systems.SizeOf(systems.GlyphAttack)*Mini.GlyphScale; glyphRight > visible {
		t.Errorf("the glyph ends at x=%d but only %d pixels show through the overlap",
			glyphRight, visible)
	}
	if dashRight := Mini.DashLeft + Mini.DashWidth; dashRight > visible {
		t.Errorf("the dashes end at x=%d but only %d pixels show", dashRight, visible)
	}

	// The damage badge is the one thing that does not fit, and it is height rather than
	// width that stops it: a name, a glyph, four dashes and a 64-pixel sword do not go
	// into 132 pixels.
	if Mini.ShowDamage {
		used := Mini.DashTop + 4*(Mini.DashHeight+Mini.DashGap) + systems.GlyphSize
		t.Errorf("Mini claims to show a damage badge; the column would run to y=%d in a %dpx card",
			used, Mini.Height)
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
	s := Spec{Name: "Riposte", Category: CategoryDefend, Cost: 3, Element: Lightning, Enabled: true}
	img := render(t, s, st)

	ink := 0
	for y := st.BorderWidth; y < st.Height-st.BorderWidth; y++ {
		for x := deckVisibleWidth; x < st.Width-st.BorderWidth; x++ {
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

func TestMiniIsHalfTheHandCard(t *testing.T) {
	// Same proportions, half the size. The deck overlay's row arithmetic assumes it, and
	// a mini card at a different aspect would read as a different object rather than a
	// smaller one.
	for _, c := range []struct {
		name       string
		mini, hand int
	}{
		{"width", Mini.Width, Hand.Width},
		{"height", Mini.Height, Hand.Height},
	} {
		if want := c.hand / 2; c.mini != want {
			t.Errorf("mini %s is %d, want %d (half of %d)", c.name, c.mini, want, c.hand)
		}
	}
}

func TestNameClearsTheCategoryGlyph(t *testing.T) {
	// **The name is centred on the card, not on the space left beside the glyph.** So a
	// long enough name reaches back into the corner the glyph now occupies. Every concept
	// in the deck is checked, because the failure is invisible on "Jab" and obvious on the
	// longest one.
	st := Hand
	f := faces(t)
	face, err := f.at(st.NameSize)
	if err != nil {
		t.Fatal(err)
	}
	glyphRight := st.GlyphInset + systems.SizeOf(systems.GlyphAttack)*st.GlyphScale

	for _, n := range deckNames {
		w := font.MeasureString(face, n).Ceil()
		left := (st.Width - w) / 2
		if left <= glyphRight {
			t.Errorf("%q is %dpx wide, so centred it starts at x=%d and runs into the glyph ending at x=%d",
				n, w, left, glyphRight)
		}
	}
}

func TestCategoryGlyphIsDrawnAndDiffers(t *testing.T) {
	// Three categories, three silhouettes. If two ever render identically the glyph slot
	// is saying nothing, which is worse than leaving it empty.
	st := Hand
	seen := map[string]Category{}

	for _, c := range Categories() {
		s := strike(Fire)
		s.Category = c
		img := render(t, s, st)

		kind, ok := c.glyph()
		if !ok {
			t.Fatalf("%s has no glyph", c)
		}
		size := systems.SizeOf(kind)

		// Hash the glyph's own box rather than the whole card, so the name and dashes
		// cannot mask a glyph that failed to draw.
		var sum uint64
		for y := st.CategoryGlyphTop; y < st.CategoryGlyphTop+size; y++ {
			for x := st.GlyphInset; x < st.GlyphInset+size; x++ {
				p := img.RGBAAt(x, y)
				sum = sum*31 + uint64(p.R)<<16 + uint64(p.G)<<8 + uint64(p.B)
			}
		}
		key := fmt.Sprintf("%x", sum)
		if prev, dup := seen[key]; dup {
			t.Errorf("%s and %s draw the same category glyph", prev, c)
		}
		seen[key] = c
	}

	// And CategoryNone leaves the slot alone, which is what a ring needs.
	s := strike(Fire)
	s.Category = CategoryNone
	img := render(t, s, st)
	x := st.GlyphInset + systems.SizeOf(systems.GlyphAttack)/2
	y := st.CategoryGlyphTop + systems.SizeOf(systems.GlyphAttack)/2
	if got := img.RGBAAt(x, y); got != Surface {
		t.Errorf("CategoryNone drew something at the glyph slot: %v, want the surface %v", got, Surface)
	}
}

func TestRingBorderIsUnmistakable(t *testing.T) {
	// The one thing that must never happen is reaching for a ring thinking it is a card
	// you can play, so the pink has to be a long way from every element border.
	const minDistance = 120

	pink := BorderOf(Ring)
	for _, e := range Elements() {
		if d := distance(pink, BorderOf(e)); d < minDistance {
			t.Errorf("the ring border is only %d from %s's — too close to tell apart at a glance", d, e)
		}
	}
	// And it is not in Elements(), because anything iterating elements means cards.
	for _, e := range Elements() {
		if e == Ring {
			t.Error("Ring is in Elements(); a ring is not an element a card can have")
		}
	}
}

func TestRingDrawsArtAndNoCardFurniture(t *testing.T) {
	st := RingStyle
	art := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for i := range art.Pix {
		art.Pix[i] = 255 // opaque white block, easy to find
	}

	s := Spec{Name: "Fire Ring", Element: Ring, Art: art, Enabled: true}
	img := render(t, s, st)

	// The art lands in the middle of its box.
	cx := st.Width / 2
	cy := st.ArtTop + st.ArtMaxH/2
	if got := img.RGBAAt(cx, cy); got.A == 0 || got == Surface {
		t.Errorf("nothing drawn at the centre of the art box (%d,%d): %v", cx, cy, got)
	}

	// And none of the card's own furniture is on it: a ring has no cost and no phase, so
	// a stray dash or glyph would be claiming something untrue about it.
	if st.ShowCategory || st.ShowDamage {
		t.Error("the ring style claims to draw a category glyph or damage badge")
	}
	noCost := Spec{Name: "Fire Ring", Element: Ring, Art: art, Enabled: true, Cost: 0}
	plain := render(t, noCost, st)
	border := systems.ColorToward(BorderOf(Ring), Surface, borderRestToward)
	for y := 0; y < 40; y++ {
		if plain.RGBAAt(st.DashLeft, y) == border && st.DashWidth > 0 {
			t.Errorf("a cost dash was drawn on a ring at y=%d", y)
		}
	}
}

func TestDashesDoNotOverprintTheName(t *testing.T) {
	// The geometry test above proves the columns do not overlap. This proves it in
	// pixels, on the longest name in the deck at the widest cost — the case where a
	// mistake would actually show.
	st := Hand
	s := Spec{Name: "Riposte", Category: CategoryDefend, Damage: 6, Cost: 4, Element: Lightning, Enabled: true}
	img := render(t, s, st)

	border := systems.ColorToward(BorderOf(Lightning), Surface, borderRestToward)

	// Every dash must be intact across its full width: no ink from the name in it.
	for i := 0; i < s.Cost; i++ {
		y := st.DashTop + i*(st.DashHeight+st.DashGap) + st.DashHeight/2
		for x := st.DashLeft; x < st.DashLeft+st.DashWidth; x++ {
			if got := img.RGBAAt(x, y); got != border {
				t.Fatalf("dash %d is broken at x=%d: %v, want the border colour %v — the name is printing over it",
					i, x, got, border)
			}
		}
	}
}

func TestNoDamageBadgeWhenThereIsNoDamage(t *testing.T) {
	// A sword reading zero is worse than no sword. The column should be untouched
	// surface where the glyph would have been.
	st := Hand
	s := Spec{Name: "Guard", Category: CategoryDefend, Damage: 0, Cost: 3, Element: Ice, Enabled: true}
	img := render(t, s, st)

	x := st.GlyphInset + systems.GlyphSize/2
	y := st.GlyphColumnTop + systems.GlyphSize/2
	if got := img.RGBAAt(x, y); got != Surface {
		t.Errorf("middle of the glyph column is %v on a card with no damage, want the surface %v", got, Surface)
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

func TestSelectedIsTheColourNamedInTheSource(t *testing.T) {
	// A widget names the colour it wants at full strength and its other states scale
	// down from it. Selected is that full strength, so it must be BorderOf exactly.
	st := Hand
	for _, e := range Elements() {
		s := strike(e)
		s.Selected = true
		got := render(t, s, st).RGBAAt(st.BorderWidth/2, st.Height/2)
		if got != BorderOf(e) {
			t.Errorf("%s selected border is %v, want the named colour %v", e, got, BorderOf(e))
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
	// wrong colour is recoverable; a crash in the middle of a duel is not.
	if got := BorderOf(Element(99)); got != BorderOf(Basic) {
		t.Errorf("out-of-range element gave %v, want the Basic border %v", got, BorderOf(Basic))
	}
	if got := Element(99).String(); got != "?" {
		t.Errorf("out-of-range element is named %q, want %q", got, "?")
	}
}

// distance is how far apart two colours are, summed per channel. Used instead of
// luminance because "quieter" on a light card means closer to the surface, in whichever
// direction that happens to be.
func distance(a, b color.RGBA) int {
	return abs(int(a.R)-int(b.R)) + abs(int(a.G)-int(b.G)) + abs(int(a.B)-int(b.B))
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestCategoryGlyphsAreSmallerThanTheDamageBadge(t *testing.T) {
	// **The relationship is the invariant, not the number.** How big a category glyph is
	// stays a design choice — 22 for the generated book, 32 for the drawn sword and shield
	// — but a category glyph that reaches the damage badge's size means someone put a 64px
	// shape in the corner slot, and the left column silently overlaps again.
	//
	// The floor matters as much as the ceiling: below about 16 pixels neither a painting
	// nor a derived rim has anything left to read.
	for _, c := range Categories() {
		kind, ok := c.glyph()
		if !ok {
			t.Fatalf("%s has no glyph", c)
		}
		got := systems.SizeOf(kind)
		if got < 16 || got > systems.GlyphSize/2 {
			t.Errorf("%s glyph is %dpx, want between 16 and half of %d",
				c, got, systems.GlyphSize)
		}
	}

	// The damage sword stays big. It is not a category glyph and was not asked to shrink.
	if got := systems.SizeOf(systems.GlyphDamage); got != systems.GlyphSize {
		t.Errorf("the damage sword is %dpx, want %d", got, systems.GlyphSize)
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

	if got := dragImg.RGBAAt(st.BorderWidth/2, mid); got != BorderOf(Fire) {
		t.Errorf("dragged border is %v, want full strength %v", got, BorderOf(Fire))
	}
	if dragImg.RGBAAt(st.BorderWidth/2, mid) == deadImg.RGBAAt(st.BorderWidth/2, mid) {
		t.Error("dragged and disabled draw the same border — they mean opposite things")
	}

	// The face does ghost, or nothing would say the card is in the air.
	if got := dragImg.RGBAAt(probe.X, probe.Y); got == Surface {
		t.Error("a dragged card's face is the resting surface — nothing marks it as lifted")
	}
}

func TestRingNameIsCentered(t *testing.T) {
	// A ring has no glyph column, so a left-aligned name has nothing to line up with and
	// reads as having slipped. Measured by ink: the name's pixels must straddle the
	// card's midline with roughly equal margins.
	st := RingStyle
	s := Spec{Name: "Fire Ring", Element: Ring, Enabled: true}
	img := render(t, s, st)

	left, right := st.Width, 0
	for y := st.NameTop; y < st.NameTop+int(st.NameSize)+6; y++ {
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
		t.Fatal("no name drawn on the ring")
	}

	leftMargin := left - st.BorderWidth
	rightMargin := (st.Width - st.BorderWidth) - right
	if d := leftMargin - rightMargin; d < -6 || d > 6 {
		t.Errorf("ring name spans x=%d..%d: margins %d and %d, not centred",
			left, right, leftMargin, rightMargin)
	}

	// Action cards are centred too now. The glyph moved into the top-left corner, so a
	// left-aligned name would print straight over it — see TestNameClearsTheCategoryGlyph.
	if !Hand.NameCentered {
		t.Error("the hand card is not centring its name, which would put it under the glyph")
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
		Name: "Riposte", Category: CategoryDefend, Damage: 9, Cost: 4,
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
		t.Errorf("centre of the back is %v, want the mark %v",
			img.RGBAAt(Hand.Width/2, Hand.Height/2), BackInk)
	}
}
