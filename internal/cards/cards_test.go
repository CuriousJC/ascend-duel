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
	return Spec{Name: "Strike", Family: FamilyCrush, Cost: 2, Element: e, Enabled: true}
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
	// **The left column is a glyph over a stack of dashes**, and it used to have a damage badge
	// under that. Adding the category glyph is what made this worth a test — before it there
	// was one badge and a corner, and nothing could overlap anything.
	//
	// Four dashes is the most the rules can produce today. A fifth cost tier grows the stack
	// past the bottom of the card, which is a layout change and not just a bigger number. This
	// fails when that happens rather than rendering it.
	const maxCost = 4

	st := Hand

	// The mark's box is what the layout names, so it is what has to fit. A letter is centred in
	// it and a future glyph is clipped to it, so nothing can be drawn outside it either way.
	familyBottom := st.FamilyTop + st.FamilySize
	dashBottom := st.DashTop + (maxCost-1)*(st.DashHeight+st.DashGap) + st.DashHeight

	if familyBottom > st.DashTop {
		t.Errorf("family mark ends at y=%d, %dpx into the dash stack at y=%d",
			familyBottom, familyBottom-st.DashTop, st.DashTop)
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

func TestTheFamilyMarkClearsTheTextBand(t *testing.T) {
	// **The mark is deliberately wider than the column it sits above**, so what keeps it off
	// the text is height rather than width: it sits in the top-left corner and the text band
	// starts below it. If either moves far enough to overlap, the mark is drawn first and the
	// first line of text lands on top of it.
	st := Hand

	if bottom := st.FamilyTop + st.FamilySize; bottom > st.TextBandTop {
		t.Errorf("the family mark ends at y=%d, %dpx into the text band at y=%d",
			bottom, bottom-st.TextBandTop, st.TextBandTop)
	}
}

func TestAGlyphOffTheCornerDoesNotSquareTheCorner(t *testing.T) {
	// The glyph is placed at a negative offset so it runs off the top-left edge. Cropping it at
	// the image's bounding box rather than at the card's own curve would fill the transparent
	// rounded corner with glyph pixels and the card would read as having one square corner.
	st := Hand
	if st.GlyphInset >= 0 && st.FamilyTop >= 0 {
		t.Skip("the hand's family mark no longer hangs off the corner")
	}

	// A defend card, because the kite shield is the widest glyph and reaches furthest into the
	// corner. Disabled as well as enabled: the fade pass walks the same rectangle.
	for _, enabled := range []bool{true, false} {
		s := Spec{Name: "Defend", Family: FamilyPlan, Cost: 3, Element: Ice, Enabled: enabled}
		img := render(t, s, st)

		// The outermost corner pixel of the bounding box is outside a 12px radius and must stay
		// transparent whatever is drawn over it.
		if got := img.RGBAAt(0, 0); got.A != 0 {
			t.Errorf("enabled=%v: the top-left corner pixel is %v, want transparent", enabled, got)
		}
	}
}

func TestMiniShowsEverythingButTheText(t *testing.T) {
	// **What the overlay can say is set by what fits in the visible strip.** At a third
	// size, 29 pixels showed and only dashes fit; at half with the row widened, 84 of the
	// 90 show and the name, glyph and dashes all fit. This pins that, because the tempting
	// regression is to re-tighten the pitch for a longer row and silently lose the name.
	visible := deckVisibleWidth

	if !Mini.ShowName {
		t.Error("Mini does not show the name, which is the only thing identifying a concept")
	}
	if !Mini.ShowFamily {
		t.Error("Mini does not show the family mark")
	}
	if markRight := Mini.GlyphInset + Mini.FamilySize; markRight > visible {
		t.Errorf("the family mark ends at x=%d but only %d pixels show through the overlap",
			markRight, visible)
	}
	if dashRight := Mini.DashLeft + Mini.DashWidth; dashRight > visible {
		t.Errorf("the dashes end at x=%d but only %d pixels show", dashRight, visible)
	}

	// The effect text is the one thing that does not fit. A mini card is 81 wide, and taking a
	// cost column out of that leaves about 55 pixels of measure — four or five characters a
	// line at any size legible on a card half this one's height.
	if Mini.TextLineHeight > 0 {
		t.Errorf("Mini claims to draw effect text; it has %dpx of measure to do it in",
			Mini.Width-Mini.TextColumnLeft-Mini.TextInset)
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
	s := Spec{Name: "Prepare", Family: FamilyPlan, Cost: 1, Element: Lightning, Enabled: true}
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

func TestNameClearsTheFamilyMark(t *testing.T) {
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
	markRight := st.GlyphInset + st.FamilySize

	for _, n := range deckNames {
		w := font.MeasureString(face, n).Ceil()
		left := (st.Width - w) / 2
		if left <= markRight {
			t.Errorf("%q is %dpx wide, so centred it starts at x=%d and runs into the mark ending at x=%d",
				n, w, left, markRight)
		}
	}
}

func TestFamilyMarkIsDrawnAndDiffers(t *testing.T) {
	// Four families, four marks. If two ever render identically the corner is saying nothing,
	// which is worse than leaving it empty — and it is the risk letters carry that silhouettes do
	// not, since four capitals are far closer to each other than a sword is to a shield.
	st := Hand
	seen := map[string]Family{}

	for _, fam := range Families() {
		spec := strike(Fire)
		spec.Family = fam
		img := render(t, spec, st)

		// Hash the mark's own box rather than the whole card, so the name and dashes
		// cannot mask a mark that failed to draw.
		var sum uint64
		for y := st.FamilyTop; y < st.FamilyTop+st.FamilySize; y++ {
			for x := st.GlyphInset; x < st.GlyphInset+st.FamilySize; x++ {
				p := img.RGBAAt(x, y)
				sum = sum*31 + uint64(p.R)<<16 + uint64(p.G)<<8 + uint64(p.B)
			}
		}
		key := fmt.Sprintf("%x", sum)
		if prev, dup := seen[key]; dup {
			t.Errorf("%s and %s draw the same family mark", prev, fam)
		}
		seen[key] = fam
	}

	// And FamilyNone leaves the slot alone, which is what a ring and the opponent's cards need.
	spec := strike(Fire)
	spec.Family = FamilyNone
	img := render(t, spec, st)
	x := st.GlyphInset + st.FamilySize/2
	y := st.FamilyTop + st.FamilySize/2
	if got := img.RGBAAt(x, y); got != Surface {
		t.Errorf("FamilyNone drew something at the mark slot: %v, want the surface %v", got, Surface)
	}
}

func TestEveryFamilyHasItsOwnLetter(t *testing.T) {
	// The pixel test above would also catch this, slowly and by a hash. This says the actual
	// rule: a family with no letter draws nothing, and two sharing one is the reason Slash is D.
	seen := map[string]Family{}
	for _, fam := range Families() {
		l := fam.Letter()
		if l == "" {
			t.Errorf("%s has no letter, so its corner would be blank", fam)
			continue
		}
		if prev, dup := seen[l]; dup {
			t.Errorf("%s and %s both mark themselves %q", prev, fam, l)
		}
		seen[l] = fam
	}
	if got := FamilyNone.Letter(); got != "" {
		t.Errorf("FamilyNone is marked %q; it must draw nothing", got)
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
	if st.ShowFamily || st.TextLineHeight > 0 {
		t.Error("the ring style claims to draw a family mark or effect text")
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
	s := Spec{Name: "Prepare", Family: FamilyPlan, Cost: 4, Element: Lightning, Enabled: true}
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

func TestTheCostColumnHoldsNothingButDashes(t *testing.T) {
	// **The card carries no damage figure** — what it deals is in the effect text — so below the
	// last dash the column is bare surface. This is the pixel half of the geometry test above:
	// it catches anything drawn into the column that the layout constants do not describe.
	st := Hand
	s := Spec{Name: "Cleave", Family: FamilySlash, Cost: 3, Element: Ice, Enabled: true}
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

func TestTheFamilyMarkStaysCornerSized(t *testing.T) {
	// **The bound is the invariant, not the number.** How big the corner mark is stays a design
	// choice, but one that grows past half of GlyphSize is a full-size shape in a corner slot,
	// and it walks into both the dash stack under it and the text column beside it.
	//
	// The floor matters as much as the ceiling: below about 16 pixels neither a letter nor a
	// derived rim has anything left to read.
	for name, st := range map[string]Style{"hand": Hand, "mini": Mini} {
		if !st.ShowFamily {
			continue
		}
		if st.FamilySize < 16 || st.FamilySize > systems.GlyphSize/2 {
			t.Errorf("%s family box is %dpx, want between 16 and half of %d",
				name, st.FamilySize, systems.GlyphSize)
		}
		if st.FamilyLetterSize <= 0 {
			t.Errorf("%s shows a family mark with no letter size, so it would draw nothing", name)
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
		Name: "Prepare", Family: FamilyPlan, Cost: 4, Text: "Bank 2 AP",
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
