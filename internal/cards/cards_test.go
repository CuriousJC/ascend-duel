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
	"Jab", "Thrust", "Lunge",
	"Cut", "Slash", "Cleave",
	"Bash", "Strike", "Smash",
	"Prepare", "Plan", "Defend",
}

func strike(e Element) Spec {
	return Spec{Name: "Strike", Form: FormCrush, Cost: 2, Element: e, Enabled: true}
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
func TestBorderIsTheNeutralColourAtItsDeclaredWidth(t *testing.T) {
	st := Hand
	mid := st.Height / 2

	for _, e := range Elements() {
		img := render(t, strike(e), st)
		want := systems.ColorToward(borderBase(e), Surface, borderRestToward)

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
// waist is lit and the right edge is shadowed, and neither is the border's own colour — a bevel
// that had quietly become two copies of the fill would draw identically to no bevel at all.
func TestTheCardBorderIsLitOnTheTopLeftAndShadowedOnTheBottomRight(t *testing.T) {
	st := Hand
	img := render(t, strike(Fire), st)
	mid := st.Height / 2

	face := systems.ColorToward(borderBase(Fire), Surface, borderRestToward)
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
	// still the state colour, which is the signal the border exists to carry.
	if got := img.RGBAAt(BorderBevel, mid); got != face {
		t.Errorf("the pixel just inside the bevel is %v, want the border's own %v", got, face)
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

		// The element colour, not the border's: the ticks carry the element and the border does
		// not. See Spec.atState, which is what keeps the two in the same state.
		tick := systems.ColorToward(BorderOf(Fire), Surface, borderRestToward)
		count := 0
		for i := 0; i < 6; i++ {
			y := st.DashTop + i*(st.DashHeight+st.DashGap) + st.DashHeight/2
			if img.RGBAAt(x, y) == tick {
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

	// The mark's box is what the layout names, so it is what has to fit. The mark is centred in
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
	// **The text is centred in what the cost column leaves**, so the column's *width* is now
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
	// **The name is centred on the card, not on the space left beside the mark.** So a
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
			t.Errorf("%q is %dpx wide, so centred it starts at x=%d and runs into the mark ending at x=%d",
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

	// And FormNone leaves the slot alone, which is what a ring and the opponent's cards need.
	spec := strike(Fire)
	spec.Form = FormNone
	img := render(t, spec, st)
	x := st.GlyphInset + st.FormSize/2
	y := st.FormTop + st.FormSize/2
	if got := img.RGBAAt(x, y); got != Surface {
		t.Errorf("FormNone drew something at the mark slot: %v, want the surface %v", got, Surface)
	}
}

func TestEveryFormHasItsOwnGlyph(t *testing.T) {
	// The pixel test above would also catch this, slowly and by a hash. This says the actual
	// rule: every form has a mark, no two forms share one, and FormNone has none at all — a ring
	// and both fighter cards belong to no form and the slot must stay empty for them.
	seen := map[systems.GlyphKind]Form{}
	for _, fam := range Forms() {
		k, ok := fam.Glyph()
		if !ok {
			t.Errorf("%s has no glyph, so its corner would be blank", fam)
			continue
		}
		if prev, dup := seen[k]; dup {
			t.Errorf("%s and %s both mark themselves with glyph %d", prev, fam, k)
		}
		seen[k] = fam
	}
	if _, ok := FormNone.Glyph(); ok {
		t.Error("FormNone has a glyph; it must draw nothing")
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
	if st.ShowForm || st.TextLineHeight > 0 {
		t.Error("the ring style claims to draw a form mark or effect text")
	}
	noCost := Spec{Name: "Fire Ring", Element: Ring, Art: art, Enabled: true, Cost: 0}
	plain := render(t, noCost, st)
	tick := systems.ColorToward(BorderOf(Ring), Surface, borderRestToward)
	for y := 0; y < 40; y++ {
		if plain.RGBAAt(st.DashLeft, y) == tick && st.DashWidth > 0 {
			t.Errorf("a cost dash was drawn on a ring at y=%d", y)
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

	tick := systems.ColorToward(BorderOf(Lightning), Surface, borderRestToward)

	// Every dash must be intact across its full width: no ink from the name in it.
	for i := 0; i < s.Cost; i++ {
		y := st.DashTop + i*(st.DashHeight+st.DashGap) + st.DashHeight/2
		for x := st.DashLeft; x < st.DashLeft+st.DashWidth; x++ {
			if got := img.RGBAAt(x, y); got != tick {
				t.Fatalf("tick %d is broken at x=%d: %v, want the element colour %v — the name is printing over it",
					i, x, got, tick)
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

func TestSelectedIsTheColourNamedInTheSource(t *testing.T) {
	// A widget names the colour it wants at full strength and its other states scale
	// down from it. Selected is that full strength, so it must be BorderOf exactly.
	st := Hand
	for _, e := range Elements() {
		s := strike(e)
		s.Selected = true
		got := render(t, s, st).RGBAAt(st.BorderWidth/2, st.Height/2)
		if got != borderBase(e) {
			t.Errorf("%s selected border is %v, want the named colour %v", e, got, borderBase(e))
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

func TestTheFormMarkStaysCornerSized(t *testing.T) {
	// **The bound is the invariant, not the number.** How big the corner mark is stays a design
	// choice, but one that grows past half of GlyphSize is a full-size shape in a corner slot,
	// and it walks into both the dash stack under it and the text column beside it.
	//
	// The floor matters as much as the ceiling: below about 16 pixels a drawn mark has nothing
	// left to read.
	for name, st := range map[string]Style{"hand": Hand, "mini": Mini} {
		if !st.ShowForm {
			continue
		}
		if st.FormSize < 16 || st.FormSize > systems.GlyphSize/2 {
			t.Errorf("%s form box is %dpx, want between 16 and half of %d",
				name, st.FormSize, systems.GlyphSize)
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

	if got := dragImg.RGBAAt(st.BorderWidth/2, mid); got != borderBase(Fire) {
		t.Errorf("dragged border is %v, want full strength %v", got, borderBase(Fire))
	}
	if dragImg.RGBAAt(st.BorderWidth/2, mid) == deadImg.RGBAAt(st.BorderWidth/2, mid) {
		t.Error("dragged and disabled draw the same border — they mean opposite things")
	}

	// The face does ghost, or nothing would say the card is in the air.
	if got := dragImg.RGBAAt(probe.X, probe.Y); got == Surface {
		t.Error("a dragged card's face is the resting surface — nothing marks it as lifted")
	}
}

func TestAWordPerLineNameClearsWhatIsUnderIt(t *testing.T) {
	// The ring card is the one style that breaks a name across lines, and the thing directly
	// under it is the artwork. A second line is 22 pixels the layout did not previously spend,
	// so this is the join: how many lines the style has room for, measured in ink rather than
	// assumed, against the top of the art box.
	//
	// **It does not know what a ring is called.** Whether the names in `rings.json` fit the room
	// this leaves is `internal/screens`' question, since that is the package allowed to read the
	// file — see TestEveryRingNameFitsItsCard.
	st := RingStyle
	if !st.NameWordPerLine {
		t.Fatal("the ring card is not breaking its name a word to a line")
	}
	if st.NameLinePitch < int(st.NameSize) {
		t.Errorf("a line pitch of %d under a %gpt name overlaps its own lines",
			st.NameLinePitch, st.NameSize)
	}

	f := faces(t)
	face, err := f.at(st.NameSize)
	if err != nil {
		t.Fatal(err)
	}
	m := face.Metrics()
	lineHeight := m.Ascent.Ceil() + m.Descent.Ceil()

	if lines := st.NameLinesAbove(st.ArtTop, lineHeight); lines < 2 {
		t.Errorf("the ring card has room for %d line(s) of name above art at y=%d — a two-word "+
			"ring would draw over its own picture", lines, st.ArtTop)
	}

	// And the pixels agree: a two-word name's ink stops above the art box.
	img := render(t, Spec{Name: "Frozen Lightning", Element: Ring, Enabled: true}, st)
	for y := st.ArtTop; y < st.ArtTop+8; y++ {
		for x := st.BorderWidth; x < st.Width-st.BorderWidth; x++ {
			if img.RGBAAt(x, y) != Surface {
				t.Fatalf("ink at (%d,%d), inside the art box that starts at y=%d", x, y, st.ArtTop)
			}
		}
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

// The duelist card and the enemy card are the two halves of one idea — the same object in
// opposite corners — so what has to be pinned is the things that make them a pair, plus the
// ladder of stat rows the duelist adds.

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

func TestAnEffectRowIsCentredAndClosesUpAsItEmpties(t *testing.T) {
	// Nil entries are skipped rather than drawn as holes, so one status sits in the middle of
	// the card. What this checks is that the *drawn* row is centred for every count — the
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

		// Centred: the margin either side of the drawn row has to match within a pixel, which
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

	bottom := st.StatsTop + (MaxStatLines-1)*st.StatRowPitch + rowHeight
	if bottom > st.HealthBarTop {
		t.Errorf("%d stat rows end at y=%d, %dpx into the health bar at y=%d",
			MaxStatLines, bottom, bottom-st.HealthBarTop, st.HealthBarTop)
	}

	// And they start below the name rather than on it.
	nameFace, err := f.at(st.NameSize)
	if err != nil {
		t.Fatal(err)
	}
	nm := nameFace.Metrics()
	nameBottom := st.NameTop + nm.Ascent.Ceil() + nm.Descent.Ceil()
	if nameBottom > st.StatsTop {
		t.Errorf("the name ends at y=%d, below the first stat row at y=%d", nameBottom, st.StatsTop)
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
	// **Every card in the game carries its name across the top** — Hand, Mini and RingStyle
	// all do, and the enemy did not until 2026-08-12. This is the invariant, not the offset:
	// a card whose name sits somewhere else reads as a different kind of object.
	for name, st := range map[string]Style{
		"enemy": EnemyStyle, "ring": RingStyle, "duelist": DuelistStyle, "hand": Hand,
	} {
		if !st.ShowName || !st.NameCentered {
			t.Errorf("%s does not centre a name across its top", name)
		}
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
		t.Errorf("centre of the back is %v, want the mark %v",
			img.RGBAAt(Hand.Width/2, Hand.Height/2), BackInk)
	}
}

func TestAnAuthoredBreakSurvivesWrapping(t *testing.T) {
	// **A newline is a break the author asked for** *(2026-08-23)*, and the measurer may not undo
	// it: a line that fits is exactly the case width-wrapping would join back up, and it is the
	// case the feature exists for.
	f := faces(t)
	lines, err := WrapText(f, WormStyle.TextSize, "make one card\nFIRE", 400)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0] != "make one card" || lines[1] != "FIRE" {
		t.Errorf("an authored break was not honoured: got %q", lines)
	}
}

func TestAnAuthoredLineStillWraps(t *testing.T) {
	// **A break can only add lines.** Honouring one must not become a way to overrun the band, so
	// an authored line too wide for the column is wrapped like any other.
	f := faces(t)
	lines, err := WrapText(f, Hand.TextSize, "one\nthe second line is far too wide for this", 60)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 3 {
		t.Errorf("an authored line was not wrapped: got %q", lines)
	}
	if lines[0] != "one" {
		t.Errorf("the authored break moved: got %q", lines)
	}
}

// TestTheBorderIsTheSameWhateverTheElement is the other half of the 2026-08-23 swap: the border
// used to be the element signal and is now the state signal, so two cards of different elements
// in the same state must be indistinguishable at the edge. Without this the old behaviour could
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

// TestARingStillBordersPink guards the one colour the swap deliberately kept. Pink was never an
// element — it is the "you cannot play this" signal — so a change that neutralises the four
// element borders must not take it with them.
func TestARingStillBordersPink(t *testing.T) {
	if got := borderBase(Ring); got != BorderOf(Ring) {
		t.Errorf("ring border base is %v, want the pink %v", got, BorderOf(Ring))
	}
	for _, e := range Elements() {
		if borderBase(e) == borderBase(Ring) {
			t.Errorf("%s borders in the ring pink — a card and a ring must not look alike", e)
		}
	}
}

// TestTheFormMarkCarriesTheElement is where the colour went. Every pair of elements must draw a
// different mark, which is the property the swap was for — if two elements paint the same corner
// then the border was neutralised and nothing took over from it.
//
// **It asks for difference rather than for a particular colour**, because the mark's pixels come
// off a ramp between a dark and a light version of the hue and the brightest of them is genuinely
// not the colour in borderColors. Which hue a ramp is built from is tintInk's question and
// TestTintInkFollowsTheHue asks it directly; this one asks the card's.
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

// TestTintInkFollowsTheHue is the direct question: hand the ramp a neutral grey and the colour
// that comes back must be nearer the hue it was built from than any other element's.
//
// A flat grey rather than the real artwork, because the art's own outline and specular are what
// make the rendered mark hard to compare — see the test above. What is being pinned here is that
// tintInk moves a colour toward the hue it was given and not toward some average of the palette.
func TestTintInkFollowsTheHue(t *testing.T) {
	grey := image.NewRGBA(image.Rect(0, 0, 1, 1))
	grey.SetRGBA(0, 0, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	for _, e := range Elements() {
		if e == Basic {
			// Basic is the neutral the border already draws in, so "nearer its own hue than any
			// other" is not a question with an answer for it.
			continue
		}
		got := tintInk(grey, BorderOf(e)).RGBAAt(0, 0)

		mine := distance(atPeakOf(got, BorderOf(e)), BorderOf(e))
		for _, other := range Elements() {
			if other == e || other == Basic {
				continue
			}
			if d := distance(atPeakOf(got, BorderOf(other)), BorderOf(other)); d < mine {
				t.Errorf("a grey tinted %s came back %v, which is nearer %s (%d) than %s (%d)",
					e, got, other, d, e, mine)
			}
		}
	}
}

// atPeakOf rescales c so its brightest channel matches want's, which is what lets two colours of
// different brightness be compared for hue alone. See TestTheFormMarkCarriesTheElement.
func atPeakOf(c, want color.RGBA) color.RGBA {
	peak := max(int(c.R), max(int(c.G), int(c.B)))
	target := max(int(want.R), max(int(want.G), int(want.B)))
	if peak == 0 {
		return c
	}
	scale := func(v uint8) uint8 {
		out := int(v) * target / peak
		if out > 255 {
			out = 255
		}
		return uint8(out)
	}
	return color.RGBA{R: scale(c.R), G: scale(c.G), B: scale(c.B), A: c.A}
}

// TestTheCostTicksCarryTheElement is the tick half of the 2026-08-23 swap. The mark in the corner
// and the ticks under it are the whole of the left column, and both say the element — a column
// where only the top of it is coloured was the first cut and the owner sent it back.
func TestTheCostTicksCarryTheElement(t *testing.T) {
	st := Hand
	at := image.Pt(st.DashLeft+st.DashWidth/2, st.DashTop+st.DashHeight/2)

	for _, e := range Elements() {
		s := strike(e)
		s.Cost = 2
		if got, want := render(t, s, st).RGBAAt(at.X, at.Y), systems.ColorToward(BorderOf(e), Surface, borderRestToward); got != want {
			t.Errorf("%s cost tick is %v, want the element at rest %v", e, got, want)
		}
	}
}

// TestTheTicksAndTheBorderShareOneState is why atState exists. The two are drawn from different
// base colours and must move together: a selected card with a lit border and resting ticks is the
// failure a second copy of the state switch produces, and it looks like a rendering glitch rather
// than like a bug.
func TestTheTicksAndTheBorderShareOneState(t *testing.T) {
	st := Hand
	tick := image.Pt(st.DashLeft+st.DashWidth/2, st.DashTop+st.DashHeight/2)
	mid := st.Height / 2

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
		gotTick := img.RGBAAt(tick.X, tick.Y)

		if want := s.atState(borderBase(Fire)); gotBorder != want {
			t.Errorf("%s border is %v, want %v", tc.name, gotBorder, want)
		}
		if want := s.atState(BorderOf(Fire)); gotTick != want {
			t.Errorf("%s tick is %v, want %v", tc.name, gotTick, want)
		}
	}
}

// **A token says the three things a hand is counted on, and nothing else.** Element is the
// border colour of its mark and its ticks, form is the mark, cost is the ticks — so a style that
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

// **The mark and the ticks are centred**, because there is no text column to their right for a
// left-aligned column to line up with — the same reason a ring centres its name.
func TestATokenCentresItsColumn(t *testing.T) {
	st := Token
	if want := (st.Width - st.FormSize) / 2; st.GlyphInset != want {
		t.Errorf("the form mark sits at x=%d against a centred %d", st.GlyphInset, want)
	}
	if want := (st.Width - st.DashWidth) / 2; st.DashLeft != want {
		t.Errorf("the ticks sit at x=%d against a centred %d", st.DashLeft, want)
	}
}
