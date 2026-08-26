package cards

import (
	"image"
	"image/color"
	"testing"
)

// The accumulator badge: the figure a growing ring carries in the bottom-right corner of its card.

func ringWithCounter(counter string) Spec {
	return Spec{Name: "Enflamed", Element: Ring, Counter: counter, Enabled: true}
}

// differs reports where two renders of the same card are not the same picture.
func differs(a, b *image.RGBA) image.Rectangle {
	out := image.Rectangle{}
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				out = out.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return out
}

// **An empty counter draws nothing at all**, which is what every ring in the file passes today and
// what the shelf passes for all of them. A badge that drew an empty pill would put a mark on
// forty-odd cards that have no number.
func TestARingWithNoCounterIsTheCardThatAlwaysWas(t *testing.T) {
	bare := render(t, ringWithCounter(""), RingStyle)
	same := render(t, Spec{Name: "Enflamed", Element: Ring, Enabled: true}, RingStyle)

	if got := differs(bare, same); !got.Empty() {
		t.Errorf("an empty counter drew something at %v", got)
	}
}

// **The badge stays in its corner.** It is the one thing on the card measured from the right and
// bottom edges, so an over-wide figure is the failure this catches — and a ring's own art box ends
// at ArtTop+ArtMaxH, which the badge must not reach back into.
func TestARingCounterStaysInItsCorner(t *testing.T) {
	st := RingStyle
	bare := render(t, ringWithCounter(""), st)

	for _, counter := range []string{"+5", "1.5x", "12.5x", "+100"} {
		got := differs(bare, render(t, ringWithCounter(counter), st))
		if got.Empty() {
			t.Errorf("counter %q drew nothing", counter)
			continue
		}

		corner := image.Rect(st.Width/2, st.ArtTop+st.ArtMaxH, st.Width-st.BorderWidth,
			st.Height-st.BorderWidth)
		if !got.In(corner) {
			t.Errorf("counter %q drew at %v, which is outside the corner %v", counter, got, corner)
		}
	}
}

// **A style with no counter draws none whatever the Spec says.** Every style but RingStyle is one,
// and a hand card sprouting a badge because a caller filled a field in would be a card saying
// something the game does not mean.
func TestOnlyARingCardDrawsACounter(t *testing.T) {
	for name, st := range map[string]Style{
		"hand": Hand, "mini": Mini, "token": Token,
		"enemy": EnemyStyle, "duelist": DuelistStyle, "worm": WormStyle,
	} {
		bare := strike(Fire)
		withCounter := bare
		withCounter.Counter = "1.5x"

		if got := differs(render(t, bare, st), render(t, withCounter, st)); !got.Empty() {
			t.Errorf("%s drew a counter at %v", name, got)
		}
	}
}

// **There is nothing behind the figure.** It was a filled pill for an afternoon and that is a second
// surface on a card that already has one; what says the counter belongs to the ring is the ink,
// which is the card's own border colour at the card's own state.
//
// The check is that the pixels a counter adds are *ink* and not a block: everything the figure
// changes has to be near the border colour, and the card's surface has to still be showing between
// the characters.
func TestTheCounterIsInkAndNotABadge(t *testing.T) {
	st := RingStyle
	bare := render(t, ringWithCounter(""), st)
	img := render(t, ringWithCounter("1.5x"), st)

	// The border, read off the same card, because a border is drawn at the card's state and an
	// enabled ring rests short of full strength.
	border := img.RGBAAt(st.BorderWidth-1, st.Height/2)

	box := differs(bare, img)
	if box.Empty() {
		t.Fatal("the counter drew nothing")
	}

	changed, surface := 0, 0
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			was, now := bare.RGBAAt(x, y), img.RGBAAt(x, y)
			if was == now {
				surface++
				continue
			}
			changed++
			if !near(now, border) && !between(now, was, border) {
				t.Fatalf("the counter painted %v at (%d,%d), which is neither the surface %v nor the border %v",
					now, x, y, was, border)
			}
		}
	}
	if changed == 0 {
		t.Fatal("the counter changed nothing inside its own bounds")
	}
	if surface == 0 {
		t.Error("the counter filled its whole box, so it is a badge and not ink")
	}
}

// between allows an antialiased edge, where a glyph's pixel is part surface and part ink.
func between(got, a, b color.RGBA) bool {
	within := func(v, lo, hi uint8) bool {
		if lo > hi {
			lo, hi = hi, lo
		}
		return int(v)+8 >= int(lo) && int(v) <= int(hi)+8
	}
	return within(got.R, a.R, b.R) && within(got.G, a.G, b.G) && within(got.B, a.B, b.B)
}

// near allows the rounding the rasteriser does at an edge without letting a different colour past.
func near(got, want color.RGBA) bool {
	d := func(a, b uint8) int {
		if a > b {
			return int(a) - int(b)
		}
		return int(b) - int(a)
	}
	return d(got.R, want.R) <= 8 && d(got.G, want.G) <= 8 && d(got.B, want.B) <= 8
}
