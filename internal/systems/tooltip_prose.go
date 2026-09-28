package systems

// tooltip_prose.go — the tooltip set in the prose glyphs.
//
// **The panel, its placement and its wording are unchanged; only the type is different.** A line is
// wrapped by WrapProse, measured by the pen, and drawn run by run with DrawProse, so a colored word
// sits exactly where the same word in the panel's own ink would.
//
// **Every color is a multiply of the white sheet, and so is every material.** An element's word is
// the sheet times its ink; a form word is the sheet times its tile, so the grain shows in the white
// and the outline stays black. That is the reason the sheet is white — no mask, no second drawing
// of the letters.
//
// **A tooltip with a character the set does not carry is drawn in the font instead**, whole, so a
// panel is never half one typeface and half the other.

import (
	"image/color"
	"math"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The panel's type in the prose glyphs, as capital heights in screen pixels.
const (
	tipProseCap      = 15
	tipProseTitleCap = 20

	// tipProseLead is the air added under each line's own height.
	tipProseLead = 4
)

// tipProseCovers says whether every run of the tooltip can be set in the prose glyphs.
func tipProseCovers(t *models.Tooltip) bool {
	covers := func(line models.TipLine) bool {
		for _, run := range line {
			if run.Text != "" && !ProseCovers(run.Text) {
				return false
			}
		}
		return true
	}
	if !covers(t.Title) {
		return false
	}
	for _, line := range t.Lines {
		if !covers(line) {
			return false
		}
	}
	return true
}

// tipProsePitch is one line's height, baseline to baseline, at a capital height.
func tipProsePitch(capHeight float64) int {
	return int(math.Ceil(ProseLineHeight(capHeight))) + tipProseLead
}

// drawTooltipProse is DrawTooltip's body for a panel set in the glyphs.
func drawTooltipProse(screen *ebiten.Image, t *models.Tooltip, gsW, gsH int) {
	var title, body []models.TipLine
	widest := 0.0
	h := 0
	measure := func(line models.TipLine, capHeight float64) {
		w := 0.0
		for _, run := range line {
			w += proseAdvance(run.Text, capHeight)
		}
		widest = math.Max(widest, w)
	}
	if len(t.Title) > 0 {
		title = WrapProse(t.Title, tipProseTitleCap, tipMaxW)
		for _, line := range title {
			measure(line, tipProseTitleCap)
			h += tipProsePitch(tipProseTitleCap)
		}
	}
	for _, line := range t.Lines {
		for _, wrapped := range WrapProse(line, tipProseCap, tipMaxW) {
			body = append(body, wrapped)
			measure(wrapped, tipProseCap)
			h += tipProsePitch(tipProseCap)
		}
	}
	w, h := int(math.Ceil(widest))+tipPad*2, h+tipPad*2
	at := tipPlaceIn(gsW, gsH, t.Anchor, w, h)

	vector.FillRect(screen, float32(at.X), float32(at.Y), float32(w), float32(h), PanelSurface, false)
	vector.StrokeRect(screen, float32(at.X), float32(at.Y), float32(w), float32(h), 1, PanelEdgeInk, false)

	y := float64(at.Y + tipPad)
	for _, line := range title {
		drawProseRuns(screen, line, tipProseTitleCap, float64(at.X+tipPad), y+proseAscent(tipProseTitleCap), PanelSpeech)
		y += float64(tipProsePitch(tipProseTitleCap))
	}
	for _, line := range body {
		drawProseRuns(screen, line, tipProseCap, float64(at.X+tipPad), y+proseAscent(tipProseCap), PanelInk)
		y += float64(tipProsePitch(tipProseCap))
	}
}

// drawProseRuns draws one line as its runs, left to right from x, on the baseline y. A run with no
// ink takes `plain`; a run naming a material is set in it, and falls back to its ink when the
// material is missing.
func drawProseRuns(screen *ebiten.Image, line models.TipLine, capHeight, x, y float64, plain color.RGBA) {
	for _, run := range line {
		if run.Text == "" {
			continue
		}
		ink := run.Ink
		if ink.A == 0 {
			ink = plain
		}
		if run.Texture == "" || !drawProseTextured(screen, run.Text, run.Texture, capHeight, x, y) {
			DrawProse(screen, run.Text, ink, x, y, capHeight)
		}
		x += proseAdvance(run.Text, capHeight)
	}
}

// proseTexturedKey is one finished word in a material.
type proseTexturedKey struct {
	text, texture string
	capHeight     float64
}

var proseTextured = map[proseTexturedKey]*ebiten.Image{}

// proseTexturedPad is the room round a word's picture, so an outline's antialiasing is not clipped.
const proseTexturedPad = 2

// multiplyBlend multiplies what is drawn into what is already there and leaves the alpha alone: the
// white of a glyph takes the tile's color, the black outline stays black, and the ground stays
// transparent.
var multiplyBlend = ebiten.Blend{
	BlendFactorSourceRGB:        ebiten.BlendFactorDestinationColor,
	BlendFactorSourceAlpha:      ebiten.BlendFactorZero,
	BlendFactorDestinationRGB:   ebiten.BlendFactorZero,
	BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
	BlendOperationRGB:           ebiten.BlendOperationAdd,
	BlendOperationAlpha:         ebiten.BlendOperationAdd,
}

// drawProseTextured draws a word in a material, built once and cached, and reports whether it could.
func drawProseTextured(screen *ebiten.Image, s, texture string, capHeight, x, y float64) bool {
	key := proseTexturedKey{s, texture, capHeight}
	img, ok := proseTextured[key]
	if !ok {
		img = buildProseTextured(s, texture, capHeight)
		proseTextured[key] = img
	}
	if img == nil {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x-proseTexturedPad, y-proseAscent(capHeight)-proseTexturedPad)
	screen.DrawImage(img, op)
	return true
}

func buildProseTextured(s, texture string, capHeight float64) *ebiten.Image {
	tile := ArtMarkImage(texture, TextureTile, TextureTile)
	if tile == nil {
		return nil
	}
	w := int(math.Ceil(proseAdvance(s, capHeight))) + 2*proseTexturedPad
	h := int(math.Ceil(ProseLineHeight(capHeight))) + 2*proseTexturedPad
	if w <= 2*proseTexturedPad {
		return nil
	}
	img := ebiten.NewImage(w, h)
	DrawProse(img, s, color.RGBA{}, proseTexturedPad, proseTexturedPad+proseAscent(capHeight), capHeight)

	// **Tiled at the tile's own size**, never scaled to the word: a scaled tile averages the grain
	// away to a flat gray.
	for ty := 0; ty < h; ty += TextureTile {
		for tx := 0; tx < w; tx += TextureTile {
			op := &ebiten.DrawImageOptions{Blend: multiplyBlend}
			op.GeoM.Translate(float64(tx), float64(ty))
			img.DrawImage(tile, op)
		}
	}
	return img
}
