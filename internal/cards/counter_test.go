package cards

import (
	"image"
	"image/color"
	"testing"
)

// The accumulator badge: the figure a growing relic carries in the bottom-right corner of its card.

func relicWithCounter(counter string) Spec {
	return Spec{Name: "Enflamed", Element: Relic, Counter: counter, Enabled: true}
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

// **An empty counter draws nothing at all**, which is what every relic in the file passes today and
// what the shelf passes for all of them. A badge that drew an empty pill would put a mark on
// forty-odd cards that have no number.
func TestARelicWithNoCounterIsTheCardThatAlwaysWas(t *testing.T) {
	bare := render(t, relicWithCounter(""), RelicStyle)
	same := render(t, Spec{Name: "Enflamed", Element: Relic, Enabled: true}, RelicStyle)

	if got := differs(bare, same); !got.Empty() {
		t.Errorf("an empty counter drew something at %v", got)
	}
}

// **The badge stays in its corner.** It is the one thing on the card measured from the right and
// bottom edges, so an over-wide figure is the failure this catches — and a relic's own art box ends
// at ArtTop+ArtMaxH, which the badge must not reach back into.
func TestARelicCounterStaysInItsCorner(t *testing.T) {
	st := RelicStyle
	bare := render(t, relicWithCounter(""), st)

	// **Every shape the label actually takes** — see combat.CounterLabel: a multiplier is always
	// one decimal place, so the widest it reaches is `10.5`, and a flat figure carries a sign. The
	// figure is allowed to outgrow its disc; what it may not do is leave the card.
	for _, counter := range []string{"+5", "1.5", "10.5", "+100"} {
		got := differs(bare, render(t, relicWithCounter(counter), st))
		if got.Empty() {
			t.Errorf("counter %q drew nothing", counter)
			continue
		}

		// **Out to the bleed, not to the card's own edges** *(2026-09-09)*: the badge is centred
		// on the bottom-right corner, so most of it lies outside the card in the room Style.Bleed
		// makes for it. What the box still holds is the half of the card it may not cross, the art
		// box it may not reach back into, and the edge of the image — a figure wider than the
		// bleed is pulled back inside rather than cut, and this is what fails if it stops being.
		corner := image.Rect(st.Width/2, st.ArtTop+st.ArtMaxH,
			st.Width+st.Bleed, st.Height+st.Bleed)
		if !got.In(corner) {
			t.Errorf("counter %q drew at %v, which is outside the corner %v", counter, got, corner)
		}
	}
}

// **A style with no counter draws none whatever the Spec says.** Every style but RelicStyle is one,
// and a hand card sprouting a badge because a caller filled a field in would be a card saying
// something the game does not mean.
func TestOnlyARelicCardDrawsACounter(t *testing.T) {
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

// **There is a disc behind the figure, and the figure is the card’s surface showing through it.**
// That reverses the 2026-08-26 call that the counter should be bare ink; see drawCounter for why.
//
// The check is the inversion, read *inside the circle*, where the badge is the only thing there is:
// every pixel has to be either the card’s border colour or the card’s own surface, and both have
// to be present — all border would be a disc with no figure on it, all surface no disc at all.
func TestTheCounterIsADiscWithTheSurfaceShowingThrough(t *testing.T) {
	st := RelicStyle
	img := render(t, relicWithCounter("1.5"), st)

	// The border, read off the same card, because a border is drawn at the card’s state and an
	// enabled relic rests short of full strength.
	border := img.RGBAAt(st.BorderWidth-1, st.Height/2)

	r := st.CounterRadius
	cx, cy := st.Width-st.CounterRight, st.Height-st.CounterBottom

	// The disc’s left edge, which is inside the circle and clear of a figure centred on the
	// middle of it — so it is the border colour whatever the figure happens to be.
	if got := img.RGBAAt(cx-r+1, cy); !near(got, border) {
		t.Errorf("the disc reads %v at its left edge, want the border %v", got, border)
	}

	disc, surface := 0, 0
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy > (r-1)*(r-1) {
				continue
			}
			switch got := img.RGBAAt(cx+dx, cy+dy); {
			case near(got, border):
				disc++
			case near(got, Surface):
				surface++
			case between(got, Surface, border):
			default:
				t.Fatalf("the badge painted %v at (%d,%d), which is neither the surface %v nor the border %v",
					got, cx+dx, cy+dy, Surface, border)
			}
		}
	}
	if disc == 0 {
		t.Error("nothing inside the circle is the border colour, so there is no disc")
	}
	if surface == 0 {
		t.Error("nothing inside the circle is the card’s surface, so the figure is not showing through")
	}
}

// **Most of the badge lies outside the card**, which is what centring it on the corner means and
// what Style.Bleed exists to make room for. A card image that went back to being exactly its style
// size would clip it to a quarter disc in the corner, and nothing else in the package would notice.
func TestTheBadgeHangsOffTheCard(t *testing.T) {
	st := RelicStyle
	img := render(t, relicWithCounter("1.5"), st)

	if got := img.Bounds(); got.Dx() != st.Width+st.Bleed || got.Dy() != st.Height+st.Bleed {
		t.Fatalf("a relic renders %dx%d, want %dx%d",
			got.Dx(), got.Dy(), st.Width+st.Bleed, st.Height+st.Bleed)
	}

	painted := 0
	for y := st.Height; y < st.Height+st.Bleed; y++ {
		for x := st.Width; x < st.Width+st.Bleed; x++ {
			if img.RGBAAt(x, y).A > 0 {
				painted++
			}
		}
	}
	if painted == 0 {
		t.Error("nothing is drawn past the card's corner, so the badge is not on it")
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
