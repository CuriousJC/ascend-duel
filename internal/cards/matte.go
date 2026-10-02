package cards

// matte.go: a playing card's figure lifted off its own ground, for a card wearing an upgrade.
//
// **Only the pictures painted on an opaque ground need it.** The card prompt asks for a transparent
// background, and a picture authored that way is drawn as it is — see opaque.
//
// **The card pictures are painted on an opaque off-white ground**, and on a plain card that ground
// is the card. An upgrade's art covers the face, so the figure has to go *over* it rather than
// under — and an opaque picture over the upgrade would paint the upgrade out again. So the ground
// is taken back out: every pixel is split into how far it sits from the ground color and in which
// direction, which is the color it would be at full strength and the alpha it is laid down at.
// Laid back over the ground it was painted on, the matte reproduces the picture exactly; laid over
// gold, the smoke of the figure is smoke over gold.
//
// **The ground is measured, not written down.** Each picture's ground is a slightly different cream
// or gray, so it is the median of the picture's own outer ring of pixels. Pixels that differ from it
// by less than the noise a generator leaves in a flat ground are dropped to nothing, so the faint
// grain of the paper does not lay a haze over the upgrade.

import (
	"image"
	"image/color"
	"image/draw"
	"sort"
)

// matteFloor is the alpha, out of 255, below which a pixel is treated as ground. It is the grain
// a flat generated ground carries, and nothing a figure is drawn with.
const matteFloor = 14

// mattes caches each picture's matte, keyed by the picture itself — the same decoded image is the
// same matte, which is the comparison Spec already makes for its cache.
var mattes = map[image.Image]*image.RGBA{}

// matteOf is src with its ground made transparent, premultiplied.
func matteOf(src image.Image) *image.RGBA {
	if m, ok := mattes[src]; ok {
		return m
	}
	b := src.Bounds()
	full := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(full, full.Bounds(), src, b.Min, draw.Src)

	ground := groundOf(full)
	out := image.NewRGBA(full.Bounds())
	for y := 0; y < full.Bounds().Dy(); y++ {
		for x := 0; x < full.Bounds().Dx(); x++ {
			out.SetRGBA(x, y, unground(full.RGBAAt(x, y), ground))
		}
	}
	mattes[src] = out
	return out
}

// groundOf is the median color of the picture's outer ring, channel by channel.
func groundOf(img *image.RGBA) color.RGBA {
	b := img.Bounds()
	var r, g, bl []int
	add := func(x, y int) {
		c := img.RGBAAt(x, y)
		r, g, bl = append(r, int(c.R)), append(g, int(c.G)), append(bl, int(c.B))
	}
	for x := b.Min.X; x < b.Max.X; x++ {
		add(x, b.Min.Y)
		add(x, b.Max.Y-1)
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		add(b.Min.X, y)
		add(b.Max.X-1, y)
	}
	med := func(v []int) uint8 {
		sort.Ints(v)
		return uint8(v[len(v)/2])
	}
	return color.RGBA{R: med(r), G: med(g), B: med(bl), A: 255}
}

// unground is one pixel with the ground taken out of it, premultiplied: the smallest alpha that
// darkens the ground to the pixel, and the color that alpha needs.
func unground(p, ground color.RGBA) color.RGBA {
	pc := [3]int{int(p.R), int(p.G), int(p.B)}
	gc := [3]int{int(ground.R), int(ground.G), int(ground.B)}

	// Alpha out of 255: how far each channel has fallen below the ground, as a share of how far it
	// could have fallen. The largest share is the alpha.
	//
	// **Only darker counts.** The figures are ink and smoke laid on a near-white ground, so every
	// one of them is darker than the ground in some channel; what is *lighter* than the measured
	// ground is the ground's own unevenness — a paler patch, a vignette — and counting it would lay
	// a cream haze over the upgrade.
	alpha := 0
	for i := range pc {
		if d := gc[i] - pc[i]; d > 0 && gc[i] > 0 {
			if a := d * 255 / gc[i]; a > alpha {
				alpha = a
			}
		}
	}
	if alpha < matteFloor {
		return color.RGBA{}
	}
	if alpha > 255 {
		alpha = 255
	}

	// Premultiplied color: p = g + a*(c - g), so a*c = p - g*(1-a).
	var out [3]uint8
	for i := range pc {
		v := pc[i] - gc[i]*(255-alpha)/255
		if v < 0 {
			v = 0
		}
		if v > alpha {
			v = alpha
		}
		out[i] = uint8(v)
	}
	return color.RGBA{R: out[0], G: out[1], B: out[2], A: uint8(alpha)}
}

// opaque is whether a picture still carries its own ground. **A picture authored on a transparent
// background has nothing to lift**, and measuring a ground off its transparent ring would read
// every pixel as ground and erase the figure — so only an opaque picture is matted.
func opaque(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}
	return true
}
