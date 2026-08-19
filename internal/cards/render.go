package cards

import (
	"fmt"
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/systems"
	xdraw "golang.org/x/image/draw"
)

// State is expressed by moving colours *toward the card's own surface*, not by scaling
// them down.
//
// **This is the one thing about the off-white card that changes how state has to be
// drawn, and getting it wrong is a documented past bug.** `systems.ColorAtStrength`
// scales toward black, which reads as "quieter" only against a dark ground. On an
// off-white card it does the opposite: a border scaled to 42% comes out darker than the
// surface and therefore *louder* than the full-strength one beside it, so the disabled
// cards would step in front of the live ones. `systems.ColorToward` exists precisely for
// this, having been written when the same mistake put the Resolution pane's idle rows in
// front of its lit one.
//
// Selected reaches the colour named in borderColors, which is what full strength means.
// Everything else is some distance from it toward the surface.
const (
	borderRestToward     = 20 // percent of the way to the surface
	borderDisabledToward = 62
	inkDisabledToward    = 58
	glyphDisabledToward  = 55

	// surfaceDraggingToward ghosts the face of a card being dragged, without touching its
	// border. Lower than the disabled figure on purpose: dragging is a lighter state and
	// has to stay clearly apart from unavailable.
	surfaceDraggingToward = 34
)

// SurfaceDisabled is the face of a card the fighter cannot afford. It is duller and a
// shade greyer than Surface rather than darker: a disabled control has to read as
// unavailable first and as itself second, and on a light card "unavailable" is washed
// out, not shadowed.
//
// Written as its own colour rather than derived, because what it needs to be is a
// function of the screen background behind the card, which this package cannot see.
var SurfaceDisabled = color.RGBA{R: 214, G: 213, B: 208, A: 255}

// Render draws one card and returns it as a fresh image.
//
// The image is exactly Style.Width by Style.Height with transparent corners, so callers
// composite it at a position and nothing else. The game caches the result; see
// internal/screens/card_art.go. Rendering is not cheap enough to do per frame — every
// pixel of the shape is written in Go and the text is rasterised — which is the one cost
// of moving card drawing off the GPU.
func Render(s Spec, st Style, f *Faces) (*image.RGBA, error) {
	if st.Width <= 0 || st.Height <= 0 {
		return nil, fmt.Errorf("cards: style has no size (%dx%d)", st.Width, st.Height)
	}
	if f == nil && st.needsFont() && !s.FaceDown {
		return nil, fmt.Errorf("cards: no fonts")
	}

	img := image.NewRGBA(image.Rect(0, 0, st.Width, st.Height))

	// The back is drawn before anything else is considered, and returns. Every other field
	// on the Spec describes the face, so a face-down card that consulted them would be
	// deciding things it does not draw — and the font check above is skipped for the same
	// reason: a back has no text, so requiring a parsed font to draw one would let a
	// missing font empty the draw pile.
	if s.FaceDown {
		drawBack(img, s, st)
		return img, nil
	}

	border, surface, ink := s.colors()
	roundedBorder(img, 0, 0, st.Width, st.Height, st.CornerRadius, st.BorderWidth, border, surface)

	if st.ShowName {
		draw := drawText
		if st.NameCentered {
			// Same signature, so the choice is a value rather than a branch around two
			// near-identical calls. The centred one takes the card width where the other
			// takes a left edge.
			draw = func(dst *image.RGBA, f *Faces, size float64, s string, x, y int, c color.RGBA) error {
				return drawTextHCentered(dst, f, size, s, st.Width, y, c)
			}
		}
		if err := draw(img, f, st.NameSize, s.Name, st.TextLeft, st.NameTop, ink(NameInk)); err != nil {
			return nil, err
		}
	}
	if err := drawStats(img, s, st, f, ink); err != nil {
		return nil, err
	}
	if s.Art != nil {
		drawArt(img, s, st)
	}
	if st.ShowForm {
		if err := drawForm(img, s, st, f, ink); err != nil {
			return nil, err
		}
	}

	drawDashes(img, s, st, border)

	if err := drawEffectText(img, s, st, f, ink); err != nil {
		return nil, err
	}
	if st.HealthBarHeight > 0 {
		if err := drawHealth(img, s, st, f); err != nil {
			return nil, err
		}
	}
	drawEffects(img, s, st)
	return img, nil
}

// drawEffects lays the status badges out in a centred row along the bottom of the card.
//
// **The row is sized to what is actually there.** Nil entries are skipped before the width is
// worked out, so one status sits in the middle of the card and four spread across it — rather
// than four fixed slots of which three are holes, which would say the card is waiting for three
// more statuses it may never take.
//
// **Fitted and resampled like the portrait, not blitted like a glyph.** These are 500-pixel
// drawings landing in a twenty-pixel box; nearest-neighbour at that ratio throws away 24 rows
// out of every 25 and what survives is noise. `drawArt`'s argument applies with more force here,
// and this runs once per distinct card rather than per frame.
//
// Each badge is fitted into its own square without distorting, so a picture that is not square
// keeps its shape and is centred in the slot it was given.
func drawEffects(dst *image.RGBA, s Spec, st Style) {
	if st.EffectSize <= 0 {
		return
	}

	shown := make([]image.Image, 0, len(s.Effects))
	for _, e := range s.Effects {
		if e != nil {
			shown = append(shown, e)
		}
	}
	if len(shown) == 0 {
		return
	}

	span := len(shown)*st.EffectSize + (len(shown)-1)*st.EffectGap
	left := (st.Width - span) / 2

	for i, e := range shown {
		box := image.Rect(0, 0, st.EffectSize, st.EffectSize).
			Add(image.Pt(left+i*(st.EffectSize+st.EffectGap), st.EffectTop))
		fitInto(dst, e, box)
	}
}

// fitInto scales src to fit box without distorting it and centres it there. It is drawArt's
// arithmetic with the box named by the caller rather than read off the style, which is what lets
// a row of small badges and one large portrait share it.
func fitInto(dst *image.RGBA, src image.Image, box image.Rectangle) {
	b := src.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 || box.Empty() {
		return
	}

	scale := float64(box.Dx()) / float64(b.Dx())
	if v := float64(box.Dy()) / float64(b.Dy()); v < scale {
		scale = v
	}
	w, h := int(float64(b.Dx())*scale), int(float64(b.Dy())*scale)
	if w <= 0 || h <= 0 {
		return
	}

	at := image.Rect(0, 0, w, h).
		Add(box.Min).
		Add(image.Pt((box.Dx()-w)/2, (box.Dy()-h)/2))

	xdraw.CatmullRom.Scale(dst, at, src, b, xdraw.Over, nil)
}

// needsFont reports whether this style draws any text at all. A Mini card does not, and
// requiring a parsed font to render one would make the deck overlay depend on something
// it never uses.
// **ShowForm is in the list now** *(2026-08-15)*: the corner mark is a typeset letter until the
// forms have glyphs, so a style that draws one needs a font exactly as a name does.
func (st Style) needsFont() bool {
	return st.ShowName || st.ShowForm ||
		st.HealthBarHeight > 0 || st.StatRowPitch > 0 || st.TextLineHeight > 0
}

// drawStats writes the labelled figures down the face: label against the left margin,
// figure against the right, one per row at the style's pitch.
//
// **A blank entry leaves its row empty rather than closing up.** The rows are a fixed
// ladder the health bar is placed under, so a card with two figures has to put them where a
// card with three puts its first two — otherwise the same card at two moments in a run
// would have its numbers at different heights.
func drawStats(dst *image.RGBA, s Spec, st Style, f *Faces, ink func(color.RGBA) color.RGBA) error {
	if st.StatRowPitch <= 0 {
		return nil
	}
	right := st.Width - st.TextLeft

	for i, line := range s.Stats {
		if line.Label == "" && line.Value == "" {
			continue
		}
		y := st.StatsTop + i*st.StatRowPitch

		if err := drawText(dst, f, st.StatSize, line.Label, st.TextLeft, y, ink(LabelInk)); err != nil {
			return err
		}
		if err := drawTextRightAligned(dst, f, st.StatSize, line.Value, right, y, ink(NumberInk)); err != nil {
			return err
		}
	}
	return nil
}

// colors resolves the card's state into the three things that vary with it: the border,
// the surface, and a function that adjusts any ink for the state.
//
// Ink is a function rather than a colour because the card has more than one, and all of
// them move the same distance toward the same ground. Passing the rule rather than the
// results keeps that a single number.
func (s Spec) colors() (border, surface color.RGBA, ink func(color.RGBA) color.RGBA) {
	base := BorderOf(s.Element)

	switch {
	case !s.Enabled:
		return systems.ColorToward(base, SurfaceDisabled, borderDisabledToward),
			SurfaceDisabled,
			func(c color.RGBA) color.RGBA {
				return systems.ColorToward(c, SurfaceDisabled, inkDisabledToward)
			}
	case s.Dragging:
		// **The border stays at full strength and only the face ghosts.** A card in the
		// air is not unavailable — it is the one thing you are currently doing — so
		// dimming its element would say the opposite. Ghosting the surface alone reads as
		// lifted, and keeps it distinguishable from the disabled card next to it, whose
		// border is dimmed too.
		return base,
			systems.ColorToward(Surface, SurfaceDisabled, surfaceDraggingToward),
			identity
	case s.Selected:
		return base, Surface, identity
	default:
		return systems.ColorToward(base, Surface, borderRestToward), Surface, identity
	}
}

func identity(c color.RGBA) color.RGBA { return c }

// drawForm marks the card's form in the corner above the cost stack: an uppercase letter
// today — S, D, C, P — and a silhouette once the forms have one.
//
// **It replaced the phase glyph** *(2026-08-15)*. The sword, shield and open book named which
// phase a card resolved in, and with the deck rebuilt on three attack forms that fact became
// derivable from the form itself. What the corner says now is the thing a pair is counted on.
//
// **The letters are scaffolding and the glyph path is why they can be.** `Form.glyph()` returns
// nothing for every form, so every card falls through to `Form.Letter()`; drawing the art
// instead is one return value and no change here. See the comment on that method.
//
// **Placed by its ink, not by its canvas** *(2026-08-14)*. Every glyph is drawn on a square
// canvas and none of them fills it: the sword's ink started 7 across and 9 down, the shield's 7
// and 5, and each was a different size inside it. Anchoring the canvas therefore put three
// different shapes in three different places from one pair of constants. A typeset letter is
// worse again — a "C" and a "P" have different bearings and different heights — so the letter
// goes through exactly the same centring.
//
// **Its offsets may be negative, and blitGlyph crops to the card's curve** so a mark placed hard
// into the corner cannot square it off.
func drawForm(dst *image.RGBA, s Spec, st Style, f *Faces, ink func(color.RGBA) color.RGBA) error {
	if s.Form == FormNone || st.FormSize <= 0 {
		return nil
	}

	box := image.Rect(st.GlyphInset, st.FormTop,
		st.GlyphInset+st.FormSize, st.FormTop+st.FormSize)

	if kind, ok := s.Form.glyph(); ok {
		glyph := systems.RenderGlyph(kind, systems.PaletteWhite)
		at := placeInk(dst, glyph, box, st.GlyphScale, st)
		if !s.Enabled && !at.Empty() {
			// Glyphs carry a five-value palette that cannot be dimmed by choosing a duller ink,
			// so a disabled card fades one in place. A letter needs none of that — it is drawn in
			// the state's ink to begin with.
			fadeRegion(dst, at, glyphDisabledToward)
		}
		return nil
	}

	letter := s.Form.Letter()
	if letter == "" || st.FormLetterSize <= 0 {
		return nil
	}
	img, err := letterImage(f, st.FormLetterSize, letter, ink(NameInk), st.FormSize)
	if err != nil {
		return err
	}
	placeInk(dst, img, box, 1, st)
	return nil
}

// letterImage typesets one character onto a transparent square, for placeInk to position.
//
// **The canvas is generous rather than tight**, because where a letter's ink lands inside a line
// box is the font's business: an offset that fits "S" clips the descender off something else.
// placeInk throws the margin away, so the only cost of being generous is a wider bounds scan on
// an image the card cache renders once.
func letterImage(f *Faces, size float64, s string, c color.RGBA, box int) (*image.RGBA, error) {
	canvas := box * 3
	img := image.NewRGBA(image.Rect(0, 0, canvas, canvas))
	if err := drawText(img, f, size, s, box, box, c); err != nil {
		return nil, err
	}
	return img, nil
}

// placeInk composites src so its *inked* bounds come out centred in box, clipped to the card's
// own rounded silhouette. It returns the rectangle the ink landed in, which is what a fade pass
// has to walk — the canvas around it is transparent and fading it would fill in the corner
// blitGlyph just protected.
func placeInk(dst *image.RGBA, src *image.RGBA, box image.Rectangle, scale int, st Style) image.Rectangle {
	ink := inkBounds(src)
	if ink.Empty() {
		return image.Rectangle{}
	}
	ink = image.Rectangle{Min: ink.Min.Mul(scale), Max: ink.Max.Mul(scale)}

	// Where the canvas has to sit for the ink to come out centred in the box.
	at := box.Min.Sub(ink.Min).Add(image.Pt(
		(box.Dx()-ink.Dx())/2,
		(box.Dy()-ink.Dy())/2,
	))

	b := src.Bounds()
	blitGlyph(dst, image.Rectangle{Min: at, Max: at.Add(image.Pt(b.Dx()*scale, b.Dy()*scale))},
		src, scale, st)

	return image.Rectangle{Min: at.Add(ink.Min), Max: at.Add(ink.Max)}
}

// inkBounds is the smallest rectangle holding every non-transparent pixel of a glyph.
//
// It exists because a glyph's canvas says nothing about where its picture is: the generated
// shapes leave whatever margin their spans happen to leave, and the drawn art carries its
// own. Aligning canvases aligns nothing.
func inkBounds(g *image.RGBA) image.Rectangle {
	b := g.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X-1, b.Min.Y-1

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if g.RGBAAt(x, y).A == 0 {
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}

	if maxX < minX || maxY < minY {
		return image.Rectangle{}
	}
	return image.Rect(minX, minY, maxX+1, maxY+1)
}

// drawDashes writes the action-point cost as a stack of short horizontal bars,
// hamburger-style, under the category glyph.
//
// Costs run 1 to 4 today (combat.go), so the stack is short by construction. Nothing here
// caps it — a five-point card simply draws five dashes and grows downward — because
// silently clamping a number the rules produced would be a card that lies about its cost.
// The stack has the rest of the card's height to grow into now that the damage badge is gone,
// and TestLeftColumnDoesNotCollide fails at the point it would run off the bottom.
//
// They are drawn in the border colour, so the two things the card says about itself in
// colour say it in the same colour.
func drawDashes(dst *image.RGBA, s Spec, st Style, c color.RGBA) {
	if s.Cost <= 0 || st.DashWidth <= 0 || st.DashHeight <= 0 {
		return
	}
	for i := 0; i < s.Cost; i++ {
		y := st.DashTop + i*(st.DashHeight+st.DashGap)
		if y+st.DashHeight > st.Height {
			return
		}
		fillRect(dst, st.DashLeft, y, st.DashWidth, st.DashHeight, c)
	}
}

// The health bar's two colours, and the fraction under it.
//
// **Red for what is left, not for what is lost.** The bar is a quantity the reader is
// tracking downward, so the saturated colour has to be the part that shrinks — a bar where
// the red grows as the enemy weakens says the opposite of what it means. The empty part is
// a dim version of the same hue rather than a neutral grey, so the two read as one bar
// partly filled instead of as two bars.
var (
	HealthFull  = color.RGBA{R: 198, G: 46, B: 46, A: 255}
	HealthEmpty = color.RGBA{R: 92, G: 66, B: 66, A: 255}
)

// drawHealth draws the bar and the "42/60" line beneath it.
//
// **The bar is square-cornered where the screen's other one is rounded**, which is a real
// inconsistency and the same one the package header already records: rounding on the screen
// goes through an Ebitengine mask and a GPU blend mode, neither of which exists here. A
// hard-edged bar matches the cost dashes and the card's own corner rasterisation, so it is
// at least consistent within the card.
//
// A zero or negative MaxLife draws the empty bar and no fraction rather than dividing by it.
func drawHealth(dst *image.RGBA, s Spec, st Style, f *Faces) error {
	left, width := st.HealthBarInset, st.Width-2*st.HealthBarInset

	fillRect(dst, left, st.HealthBarTop, width, st.HealthBarHeight, HealthEmpty)
	if s.MaxLife <= 0 {
		return nil
	}

	life := s.Life
	if life < 0 {
		life = 0
	}
	if life > s.MaxLife {
		life = s.MaxLife
	}
	fillRect(dst, left, st.HealthBarTop, width*life/s.MaxLife, st.HealthBarHeight, HealthFull)

	// **The exact number as well as the bar, deliberately** — the same argument the character
	// block settled for the player on 2026-08-07. A bar says roughly how hurt something is,
	// and a duel decided in whole points wants the figure. The block writes the player's as a
	// fraction, so the enemy's is written the same way rather than inventing a second form.
	return drawTextHCentered(dst, f, st.HealthTextSize,
		fmt.Sprintf("%d/%d", life, s.MaxLife), st.Width, st.HealthTextTop, NumberInk)
}

// drawEffectText writes what the card does, as a block centred in the space the left column
// leaves.
//
// **Centred both ways, against the name being centred on the whole card.** The name spans the
// card and the text does not — it owns everything right of the cost column — so centring it on
// the card would push every line left, under the dashes. It is centred on its own column
// instead, and vertically inside the band, so a one-line card and a five-line card are the
// same design rather than two.
//
// It is set in LabelInk rather than NameInk, the same distinction the stat rows make with
// colour: the name stays the loudest thing on a card that is now mostly words.
//
// **Every wrapped line is drawn, including one past the band.** Clamping to TextLines() would
// hide an overlong string, and a card that quietly drops half its rules text is the picture
// that lies this repo keeps refusing to draw. TestEveryCardTextFitsItsBand is what catches it
// first, and the sheet shows the overrun if it ever gets past that.
func drawEffectText(dst *image.RGBA, s Spec, st Style, f *Faces, ink func(color.RGBA) color.RGBA) error {
	if st.TextLineHeight <= 0 || s.Text == "" {
		return nil
	}

	width := st.Width - st.TextColumnLeft - st.TextInset
	lines, err := WrapText(f, st.TextSize, s.Text, width)
	if err != nil {
		return err
	}

	// Centred in the band, and never above it: text that overruns grows downward off the card
	// where it is visible, rather than upward into the name where it would look like a
	// different bug.
	top := st.TextBandTop + (st.TextBandBottom-st.TextBandTop-len(lines)*st.TextLineHeight)/2
	if top < st.TextBandTop {
		top = st.TextBandTop
	}

	for i, line := range lines {
		y := top + i*st.TextLineHeight
		if err := drawTextCenteredIn(dst, f, st.TextSize, line,
			st.TextColumnLeft, width, y, ink(LabelInk)); err != nil {
			return err
		}
	}
	return nil
}

// The mark is as wide as the card allows, and centred on it.
//
// **Derived rather than a Style field, unlike every other measurement here.** The face
// cannot work that way — its glyphs are 1:1 pixel art with a one-pixel rim, so a smaller card
// has to *drop* what will not fit rather than shrink it, and each size states its own
// numbers. A triangle has no rim to lose and no detail to fall below, so one proportion
// draws the same back at every size and cannot drift between them. If a size ever wants
// its own mark, this is one line to promote into Style.
//
// **It was 40% of the width until 2026-08-11.** At the draw pile's 44x64 that came out as a
// 17-pixel mark floating in the middle of a dark card, which read as a speck rather than as
// a back. It now spans the card between its rims, so the pile reads as a stack of the same
// object at any size — which is the whole reason the back is one proportion rather than a
// per-Style number.
//
// It stays vertically centred. Hanging the apex from the top row was tried the same day and
// left the whole mark sitting high with a band of empty card under it; an equilateral
// triangle is wider than it is tall in a portrait rectangle, so the space it leaves has to
// be split rather than pushed to one end. Heights, for the three sizes that exist:
// **154 on a hand card, 76 on a mini, 36 on the draw pile's.**
//
// backMarkAspect is sqrt(3)/2 in integer thousandths — the height of an equilateral
// triangle against its base.
//
// backMarkPct trims it back off the rims by a further 5%, so the base does not run the whole
// width of the card and the mark reads as sitting on the back rather than as the back's own
// shape.
const (
	backMarkAspect = 866
	backMarkPct    = 95

	// chevronThicknessPct is how much of the triangle's base each arm of the chevron keeps.
	// The rest is cut away as a smaller triangle sharing the base, so the outer silhouette
	// is identical to MarkTriangle's and only the middle is missing.
	chevronThicknessPct = 22
)

// backMarkWidth is the mark's base: the card less its rims, scaled, and then nudged to the
// card's own parity.
//
// **The parity step is what makes "centred" exact rather than nearly.** A row is placed at
// `(Width-span)/2`, so a base whose width differs in parity from the card leaves the extra
// pixel on one side and the whole triangle leans. Rounding the width by one is invisible;
// the lean is not, at the draw pile's 44 pixels.
func backMarkWidth(st Style) int {
	w := (st.Width - 2*backRimWidth) * backMarkPct / 100
	if (st.Width-w)%2 != 0 {
		w--
	}
	return w
}

// backRimWidth is one pixel at every size, and does not come from Style.BorderWidth.
//
// The face's border is 6 pixels because it carries the element and has to be seen across a
// table of cards; this is a hairline whose only job is to separate one back from the back
// behind it. Scaling it with the card would make the pile's edges heavier exactly where the
// cards are smallest and the separation matters most.
const backRimWidth = 1

// drawBack draws the back of a card: the same silhouette as a face, filled dark, with a
// pale triangle centred on it.
//
// **The silhouette has to match the face exactly** — same footprint, same corner radius —
// because these are the same object seen from the other side. A back with its own shape
// would change outline halfway through a flip, and a stack of them would read as a
// different kind of thing sitting next to the hand rather than as the cards in it.
//
// The edge is a thin neutral rim rather than the face's thick coloured border. The face's
// border is where the element is said, so a back carrying one would name the card under it;
// a hueless one-pixel edge says only "this is where the card stops", which the draw pile
// needs — see BackRim.
func drawBack(dst *image.RGBA, s Spec, st Style) {
	roundedBorder(dst, 0, 0, st.Width, st.Height, st.CornerRadius, backRimWidth,
		BackRim, BackSurface)

	w := backMarkWidth(st)
	h := w * backMarkAspect / 1000
	left, top := (st.Width-w)/2, (st.Height-h)/2

	// **Every mark is built from the same box**, so they are the same weight on the card and
	// swapping one for another cannot change how heavy a pile looks. Only the shape inside it
	// differs.
	switch s.Back {
	case MarkDiamond:
		// Two triangles base to base, each half the height, so the diamond fills the same
		// box rather than being a triangle with something added under it.
		half := h / 2
		fillTriangleUp(dst, left, top, w, half, BackInk)
		fillTriangleDown(dst, left, top+half, w, h-half, BackInk)
	case MarkChevron:
		// The triangle with a triangle taken out of it: a solid one, then the surface
		// painted back over a smaller one sharing its base. Cutting rather than drawing two
		// arms is what keeps the outer edge identical to MarkTriangle's.
		fillTriangleUp(dst, left, top, w, h, BackInk)

		inset := w * chevronThicknessPct / 100
		cutW := w - 2*inset
		if (w-cutW)%2 != 0 {
			cutW--
		}
		cutH := cutW * backMarkAspect / 1000
		fillTriangleUp(dst, left+(w-cutW)/2, top+h-cutH, cutW, cutH, BackSurface)
	default:
		fillTriangleUp(dst, left, top, w, h, BackInk)
	}
}

// blitGlyph composites a generated glyph, untinted, clipped to the card's own silhouette.
//
// Scaling its five-value palette toward anything collapses the bevel into a flat
// silhouette, which is the whole thing the palette exists to avoid; a disabled card fades
// it in place instead, so the shading survives and only the weight changes.
//
// **The clip is what lets a glyph hang off the corner.** The category glyph is placed at a
// negative offset so it runs off the top-left edge, and the card image is exactly the card, so
// the overhang is cropped either way — but cropping at the *bounding box* would paint the
// transparent rounded corner solid and square the card off. Clipping to the rounded shape
// crops it along the curve instead, which is the whole point of hanging it there.
//
// One loop rather than a draw.Draw fast path at scale 1: the clip has to be tested per pixel,
// and a 32x32 glyph once per distinct card is not worth two code paths.
func blitGlyph(dst *image.RGBA, at image.Rectangle, glyph *image.RGBA, scale int, st Style) {
	b := glyph.Bounds()
	radius := clampRadius(st.Width, st.Height, st.CornerRadius)

	for y := 0; y < b.Dy()*scale; y++ {
		for x := 0; x < b.Dx()*scale; x++ {
			c := glyph.RGBAAt(b.Min.X+x/scale, b.Min.Y+y/scale)
			if c.A == 0 {
				continue
			}
			px, py := at.Min.X+x, at.Min.Y+y
			if !insideRounded(st.Width, st.Height, radius, px, py) {
				continue
			}
			dst.SetRGBA(px, py, c)
		}
	}
}

// drawArt scales Spec.Art to fit the style's art box and centres it there.
//
// **Smoothly resampled, unlike everything else here.** The generated glyphs are pixel art
// and must never be filtered; a ring is a photograph-like asset several times the size of
// the box it lands in, and nearest-neighbour downsampling one of those drops every other
// row and produces visible stair-stepping. CatmullRom is the expensive option and this
// runs once per distinct card, not per frame.
func drawArt(dst *image.RGBA, s Spec, st Style) {
	fitInto(dst, s.Art, image.Rect(
		st.ArtInset, st.ArtTop,
		st.Width-st.ArtInset, st.ArtTop+st.ArtMaxH))
}

// fadeRegion moves every pixel of a rectangle pct of the way to the disabled surface.
//
// Used on the glyphs, which are drawn from their own five-value palette and so cannot be
// dimmed by choosing a duller ink the way the text can. Fading in place preserves the
// relative steps of the bevel and only reduces its weight.
//
// **Transparent pixels are left alone**, which matters now that a glyph's rectangle can hang
// off the corner of the card: fading a transparent pixel would give it an alpha and fill in
// the rounded corner the clip in blitGlyph just protected.
func fadeRegion(dst *image.RGBA, r image.Rectangle, pct int) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			c := dst.RGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			dst.SetRGBA(x, y, systems.ColorToward(c, SurfaceDisabled, pct))
		}
	}
}
