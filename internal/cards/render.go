package cards

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

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
	if f == nil && st.needsFont() {
		return nil, fmt.Errorf("cards: no fonts")
	}

	img := image.NewRGBA(image.Rect(0, 0, st.Width, st.Height))

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
	if s.Art != nil {
		drawArt(img, s, st)
	}
	if st.ShowCategory {
		drawCategory(img, s, st)
	}

	drawDashes(img, s, st, border)

	if st.ShowDamage && s.Damage > 0 {
		if err := drawDamage(img, s, st, f, ink); err != nil {
			return nil, err
		}
	}
	return img, nil
}

// needsFont reports whether this style draws any text at all. A Mini card does not, and
// requiring a parsed font to render one would make the deck overlay depend on something
// it never uses.
func (st Style) needsFont() bool { return st.ShowName || st.ShowDamage }

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

// drawCategory draws the phase glyph above the cost stack: a sword for attack, a shield
// for defend, an open book for prepare.
//
// **It replaced the category word.** The word was the card's most load-bearing fact and
// also the least visible thing on it — small, grey, and set in the same typeface as the
// name directly above it. A silhouette at 64 pixels is read before any text is.
func drawCategory(dst *image.RGBA, s Spec, st Style) {
	kind, ok := s.Category.glyph()
	if !ok {
		return
	}
	// Measured, never assumed: the category glyphs are 22 pixels where the damage sword
	// is 64. Assuming GlyphSize here would centre a small glyph in a large hole.
	size := systems.SizeOf(kind) * st.GlyphScale
	at := image.Rect(st.GlyphInset, st.CategoryGlyphTop,
		st.GlyphInset+size, st.CategoryGlyphTop+size)

	blitGlyph(dst, at, systems.RenderGlyph(kind, systems.PaletteWhite), st.GlyphScale)
	if !s.Enabled {
		fadeRegion(dst, at, glyphDisabledToward)
	}
}

// drawDashes writes the action-point cost as a stack of short horizontal bars,
// hamburger-style, under the category glyph.
//
// Costs run 1 to 4 today (combat.go), so the stack is short by construction. Nothing here
// caps it — a five-point card simply draws five dashes and grows downward — because
// silently clamping a number the rules produced would be a card that lies about its cost.
// **That growth now runs into the damage badge** rather than into empty card, so a fifth
// cost tier is a layout change here and not just a bigger number.
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

// drawDamage draws the sword glyph and the figure beside it, at the bottom of the column.
func drawDamage(dst *image.RGBA, s Spec, st Style, f *Faces, ink func(color.RGBA) color.RGBA) error {
	size := systems.GlyphSize * st.GlyphScale
	gx, gy := st.GlyphInset, st.GlyphColumnTop
	at := image.Rect(gx, gy, gx+size, gy+size)

	blitGlyph(dst, at, systems.RenderGlyph(systems.GlyphDamage, systems.PaletteWhite), st.GlyphScale)
	if !s.Enabled {
		fadeRegion(dst, at, glyphDisabledToward)
	}

	return drawTextVCentered(dst, f, st.NumberSize,
		fmt.Sprintf("%d", s.Damage),
		gx+size+st.GlyphNumberGap, gy+size/2, ink(NumberInk))
}

// blitGlyph composites a generated glyph, untinted.
//
// Scaling its five-value palette toward anything collapses the bevel into a flat
// silhouette, which is the whole thing the palette exists to avoid; a disabled card fades
// it in place instead, so the shading survives and only the weight changes.
func blitGlyph(dst *image.RGBA, at image.Rectangle, glyph *image.RGBA, scale int) {
	if scale == 1 {
		draw.Draw(dst, at, glyph, image.Point{}, draw.Over)
		return
	}
	// Nearest-neighbour, and only ever by a whole number. See Style.GlyphScale.
	b := glyph.Bounds()
	for y := 0; y < b.Dy()*scale; y++ {
		for x := 0; x < b.Dx()*scale; x++ {
			c := glyph.RGBAAt(b.Min.X+x/scale, b.Min.Y+y/scale)
			if c.A == 0 {
				continue
			}
			dst.SetRGBA(at.Min.X+x, at.Min.Y+y, c)
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
	boxW := st.Width - 2*st.ArtInset
	boxH := st.ArtMaxH
	if boxW <= 0 || boxH <= 0 {
		return
	}

	src := s.Art.Bounds()
	if src.Dx() == 0 || src.Dy() == 0 {
		return
	}

	// Fit inside the box without distorting: whichever axis is tighter sets the scale.
	scale := float64(boxW) / float64(src.Dx())
	if v := float64(boxH) / float64(src.Dy()); v < scale {
		scale = v
	}
	w, h := int(float64(src.Dx())*scale), int(float64(src.Dy())*scale)
	if w <= 0 || h <= 0 {
		return
	}

	at := image.Rect(0, 0, w, h).
		Add(image.Pt(st.ArtInset+(boxW-w)/2, st.ArtTop+(boxH-h)/2))

	xdraw.CatmullRom.Scale(dst, at, s.Art, src, xdraw.Over, nil)
}

// fadeRegion moves every pixel of a rectangle pct of the way to the disabled surface.
//
// Used on the glyphs, which are drawn from their own five-value palette and so cannot be
// dimmed by choosing a duller ink the way the text can. Fading in place preserves the
// relative steps of the bevel and only reduces its weight.
func fadeRegion(dst *image.RGBA, r image.Rectangle, pct int) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			dst.SetRGBA(x, y, systems.ColorToward(dst.RGBAAt(x, y), SurfaceDisabled, pct))
		}
	}
}
