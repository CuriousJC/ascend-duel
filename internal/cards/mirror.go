package cards

// mirror.go: a mirrored card's picture with its echo — **an offset, softened, see-through copy of the
// figure laid over the figure itself**, inside the card's own frame. The border, the left column and
// the type are the card's as ever, so it still reads as one playing card; only the picture says it
// lands twice.
//
// **The echo is the figure lifted off its ground** (matte.go), because the pictures are painted on
// an opaque off-white: a copy of the whole picture would lay a second ground over the first and
// wash the face out. **It goes over the figure rather than under it** for the same reason — under an
// opaque picture it would never show.
//
// **Drawn into the face rather than over it**, the upgrade's rule rather than the mark's: being
// mirrored is what the card *is*, so it is baked once into the cached face and costs nothing a frame.

import "image"

// The echo's placement and weight. The offset is up and to the right, away from the left column, so
// the echo never lands on the form mark or the ticks; the blur is a box this many pixels each side,
// run twice; the alpha is out of 255.
const (
	mirrorOffsetX = 22
	mirrorOffsetY = -12
	mirrorBlur    = 2
	mirrorAlpha   = 150
)

// drawMirrorEcho lays the echo of a playing card's figure over the face, clipped to the inside of
// its border as the figure is. `fade` is the figure's own state fading, so an
// unaffordable mirrored card fades its echo as far as its figure.
func drawMirrorEcho(dst *image.RGBA, s Spec, st Style, fade int) {
	if !s.Mirrored || s.Art == nil {
		return
	}
	inset := st.BorderWidth
	w, h := st.Width, st.Height
	iw, ih := w-2*inset, h-2*inset
	if iw <= 0 || ih <= 0 {
		return
	}

	figure := s.Art
	if opaque(figure) {
		figure = matteOf(figure)
	}
	echo := image.NewRGBA(image.Rect(0, 0, w, h))
	coverInto(echo, figure, image.Rect(0, 0, w, h))
	boxBlur(echo, mirrorBlur)
	boxBlur(echo, mirrorBlur)

	alpha := mirrorAlpha * (100 - fade) / 100
	radius := clampRadius(iw, ih, st.CornerRadius-inset)
	for y := 0; y < ih; y++ {
		for x := 0; x < iw; x++ {
			if !insideRounded(iw, ih, radius, x, y) {
				continue
			}
			sx, sy := x+inset-mirrorOffsetX, y+inset-mirrorOffsetY
			if sx < 0 || sy < 0 || sx >= w || sy >= h {
				continue
			}
			c := echo.RGBAAt(sx, sy)
			if c.A == 0 {
				continue
			}
			// Premultiplied, so scaling all four channels is scaling the alpha.
			c.R = uint8(int(c.R) * alpha / 255)
			c.G = uint8(int(c.G) * alpha / 255)
			c.B = uint8(int(c.B) * alpha / 255)
			c.A = uint8(int(c.A) * alpha / 255)
			dst.SetRGBA(x+inset, y+inset, over(dst.RGBAAt(x+inset, y+inset), c))
		}
	}
}

// boxBlur softens a premultiplied image in place: a horizontal pass and a vertical one, each the
// mean of the pixels within r. Premultiplied, so a transparent neighbor pulls the alpha down rather
// than darkening the color.
func boxBlur(img *image.RGBA, r int) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	tmp := make([]uint8, len(img.Pix))
	pass := func(src, dst []uint8, n, lines int, at func(line, i int) int) {
		for line := 0; line < lines; line++ {
			for i := 0; i < n; i++ {
				var sum [4]int
				count := 0
				for k := i - r; k <= i+r; k++ {
					if k < 0 || k >= n {
						continue
					}
					p := at(line, k)
					for c := 0; c < 4; c++ {
						sum[c] += int(src[p+c])
					}
					count++
				}
				p := at(line, i)
				for c := 0; c < 4; c++ {
					dst[p+c] = uint8(sum[c] / count)
				}
			}
		}
	}
	stride := img.Stride
	pass(img.Pix, tmp, w, h, func(line, i int) int { return line*stride + 4*i })
	pass(tmp, img.Pix, h, w, func(line, i int) int { return i*stride + 4*line })
}
