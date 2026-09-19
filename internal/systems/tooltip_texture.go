package systems

// tooltip_texture.go — a word set in a material rather than in a color.
//
// **The grain shows through the glyph shapes and nothing else on the panel changes.** A form word
// is the only thing in the game drawn this way: a tile out of `assets/texture/` is laid behind the
// word and kept where the letters are, so SLASH is a piece of brushed steel in the shape of the
// word rather than a word in steel gray.
//
// **It cannot live in `internal/cards`**, which is why the span carries a *name* rather than a
// picture. That package draws a card face into a plain Go image with no graphics context, and
// keeping a texture inside a glyph is a blend mode — `BlendSourceIn`, which is a GPU operation.
// So the vocabulary is shared and the compositing is not, exactly as `ArtMark` and `ArtMarkImage`
// already split.
//
// **A word is rendered once and cached, because the panel redraws every frame.** Building the mask
// and tiling the material is several draws for something that never changes, and a tooltip under a
// resting cursor would pay for it sixty times a second.

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// TextureTile is the size the material tiles are committed at, and what they are asked for.
//
// **Asked for at their own size so nothing resamples.** `ArtMarkImage` hands back the file
// untouched when the size matches, and a material reduced again on the way in would lose the grain
// that is the whole point of it.
const TextureTile = 114

// texturedKey is one finished word: the string, the material, and the size it was set at.
//
// **The face's size rather than the face**, because a `*text.GoTextFace` is a pointer and two
// equivalent faces would be two cache entries. The panel sets its type at two sizes and the form
// vocabulary is three words, so what this can hold is bounded by the words that exist.
type texturedKey struct {
	text    string
	texture string
	size    float64
	weight  float64
}

var texturedWords = map[texturedKey]*ebiten.Image{}

// drawTexturedRun draws one run with its material showing through the letters, and reports whether
// it could. **A material that is not there is not an error**: the caller falls back to the run's
// own ink, which is the same contract every optional picture in this codebase has.
func drawTexturedRun(screen *ebiten.Image, s, texture string, face *text.GoTextFace, x, y int) bool {
	img := texturedWord(s, texture, face)
	if img == nil {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(img, op)
	return true
}

// texturedWord is the word as a picture, built once.
//
// **The mask is the same double pass `drawTipLine` makes**, so a textured word on the panel is set
// at the same weight as the plain words either side of it. Anything else would read as a different
// typeface rather than as a different material.
func texturedWord(s, texture string, face *text.GoTextFace) *ebiten.Image {
	key := texturedKey{text: s, texture: texture, size: face.Size, weight: PanelWeight}
	if img, ok := texturedWords[key]; ok {
		return img
	}

	tile := ArtMarkImage(texture, TextureTile, TextureTile)
	if tile == nil {
		texturedWords[key] = nil
		return nil
	}

	w, h := text.Measure(s, face, 0)
	// The second pass of the weight hangs off the right edge, and a glyph's antialiasing can sit a
	// pixel past its advance. A margin costs nothing and a clipped stem is visible.
	iw, ih := int(math.Ceil(w+PanelWeight))+2, int(math.Ceil(h))+2
	if iw <= 0 || ih <= 0 {
		texturedWords[key] = nil
		return nil
	}

	img := ebiten.NewImage(iw, ih)
	offsets := []float64{0}
	if PanelWeight > 0 {
		offsets = append(offsets, PanelWeight)
	}
	for _, dx := range offsets {
		op := &text.DrawOptions{}
		op.GeoM.Translate(dx, 0)
		op.ColorScale.ScaleWithColor(color.White)
		text.Draw(img, s, face, op)
	}

	// **Tiled rather than scaled to fit.** The tiles are seamless at their own size and a word is
	// twenty pixels tall: scaling one down to fit would average the grain away and leave a flat
	// gray. Each tile is drawn with BlendSourceIn, which keeps the material only where the mask
	// already has a letter and leaves every pixel outside its own quad alone — so the tiles compose
	// rather than erasing one another.
	for ty := 0; ty < ih; ty += TextureTile {
		for tx := 0; tx < iw; tx += TextureTile {
			op := &ebiten.DrawImageOptions{}
			op.Blend = ebiten.BlendSourceIn
			op.GeoM.Translate(float64(tx), float64(ty))
			img.DrawImage(tile, op)
		}
	}

	texturedWords[key] = img
	return img
}
