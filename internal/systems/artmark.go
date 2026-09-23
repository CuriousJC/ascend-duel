package systems

// artmark.go — authored art, fetched by asset key and reduced to the size a caller asks for.
//
// **This is the whole of how art reaches the drawing now** *(2026-09-16)*. There used to be a
// silhouette *generator* beside it — `internal/systems/glyphs.go`, a span language with a derived
// rim and computed shading, keyed by an append-only `GlyphKind` enum. It was written when the only
// interface art in the game was a handful of shapes, and code that draws them is smaller than a
// file for each.
//
// **What retired it was the marks being authored per element.** Four forms times five elements
// plus a neutral set plus six ticks is thirty-one pictures, which is a multiplication rather than
// a list — thirty-one enum values would be thirty-one cache slots naming pictures the rules know
// nothing about. And a generated glyph could not be resized at all, its rim being one derived
// pixel, where a painting reduces from 256 to 32 and to 16 and survives both.
//
// So the key is a string built by the caller — `formslash-fire`, `tick-earth`, `gear` — and the
// art is whatever `assets.LoadImageData` has under it. **A missing key is not fatal**: it comes
// back nil and the caller draws nothing, which is the honest failure for a picture nobody has
// filed.
//
// **ArtMark itself must stay free of Ebitengine**, like everything else internal/cards reaches
// through, so the review sheets can render without a window. ArtMarkImage at the foot of the file
// is the screen's texture-shaped door onto the same cache, and it is the only thing here that
// touches a graphics context.

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"

	xdraw "golang.org/x/image/draw"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/curiousjc/ascend-duel/assets"
)

type markKey struct {
	key  string
	w, h int
}

var markCache = map[markKey]*image.RGBA{}

// ArtMark returns the art filed under key, scaled to w x h, or nil if there is no such art.
//
// **An exact whole-number halving is preferred and anything else is resampled** *(2026-09-16)*.
// The averaging in downsample is what keeps a one-pixel outline alive across a factor of two or
// four, and every size the game actually asks for is one of those: a 64px mark to 32 and 16, a
// 40x16 tick to 20x8 and 10x4. A style that asks for something else — the three-quarter Stack,
// a sheet drawing at an odd size — gets CatmullRom rather than a crash, because a contact sheet
// that cannot render is worse than one pixel of softness on a size nothing ships at.
func ArtMark(key string, w, h int) *image.RGBA {
	if key == "" || w <= 0 || h <= 0 {
		return nil
	}
	ck := markKey{key, w, h}
	if img, ok := markCache[ck]; ok {
		return img
	}

	data := assets.LoadImageData()[key]
	if len(data) == 0 {
		markCache[ck] = nil
		return nil
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		markCache[ck] = nil
		return nil
	}

	// Into an RGBA first: a PNG with alpha decodes to NRGBA and both scalers below are only
	// correct on premultiplied values. Same reason renderArt does it.
	b := src.Bounds()
	full := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(full, full.Bounds(), src, b.Min, draw.Src)

	var out *image.RGBA
	switch {
	case b.Dx() == w && b.Dy() == h:
		out = full
	case b.Dx()%w == 0 && b.Dy()%h == 0 && b.Dx()/w == b.Dy()/h:
		out = downsample(full, b.Dx()/w)
	default:
		out = image.NewRGBA(image.Rect(0, 0, w, h))
		xdraw.CatmullRom.Scale(out, out.Bounds(), full, full.Bounds(), xdraw.Over, nil)
	}

	markCache[ck] = out
	return out
}

// artMarkImages caches the ebiten copy of an ArtMark, keyed the same way.
//
// **Separate from ArtMark because that one must stay windowless**: it hands back a plain Go image
// so internal/cards and the review sheets can call it with no graphics context. A screen wants a
// texture, and creating one needs a context — so the conversion lives here, at the one call site
// that has one, rather than being pushed down into the renderer everything shares.
var artMarkImages = map[markKey]*ebiten.Image{}

// ArtMarkImage is ArtMark as a texture, for a screen to draw. Nil if there is no such art.
func ArtMarkImage(key string, w, h int) *ebiten.Image {
	ck := markKey{key, w, h}
	if img, ok := artMarkImages[ck]; ok {
		return img
	}
	src := ArtMark(key, w, h)
	if src == nil {
		artMarkImages[ck] = nil
		return nil
	}
	img := ebiten.NewImageFromImage(src)
	artMarkImages[ck] = img
	return img
}

// downsample averages each factor x factor block down to one pixel.
//
// **Averaged, not sampled.** Nearest-neighbor at half size keeps every other pixel and
// throws the rest away, which deletes a one-pixel outline wherever it lands on an odd
// column — a drawn shield loses half its rim and reads as torn. Averaging keeps it as a
// darker pixel. This is the opposite of the rule for generated glyphs, and the difference
// is that a painting has interior detail to average whereas a derived rim has nothing
// behind it.
//
// image.RGBA is alpha-premultiplied, so the four channels average independently and a
// transparent neighbor correctly darkens nothing.
func downsample(src *image.RGBA, factor int) *image.RGBA {
	if factor <= 1 {
		return src
	}
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()/factor, b.Dy()/factor))
	n := factor * factor

	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			var r, g, bl, a int
			for dy := 0; dy < factor; dy++ {
				for dx := 0; dx < factor; dx++ {
					c := src.RGBAAt(b.Min.X+x*factor+dx, b.Min.Y+y*factor+dy)
					r += int(c.R)
					g += int(c.G)
					bl += int(c.B)
					a += int(c.A)
				}
			}
			out.SetRGBA(x, y, color.RGBA{
				R: uint8(r / n), G: uint8(g / n), B: uint8(bl / n), A: uint8(a / n),
			})
		}
	}
	return out
}
