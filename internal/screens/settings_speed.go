package screens

// The speed strip: a hand of relic cards dealt in, riffled once and thrown off the other side,
// looping under the game-speed bar for as long as the settings screen is open.
//
// **It exists because the speed bar was a number with nothing to look at.** Every other control on
// that screen answers itself — the music bar is audible under the cursor, the fullscreen latch
// changes the window — and the speed bar moved a readout from `1.00x` to `1.70x` and left the
// player to go and start a duel to find out what they had done. This is the same argument that
// keeps a *sounds* bar off the screen: a control that cannot be perceived is a control that lies
// about what it does.
//
// **Cards, because cards are what the speed is mostly spent on.** A duel is a hand dealt, a hand
// sorted and a hand thrown away, over and over; an abstract pulse or a spinning bar would be
// honest about the multiplier and say nothing about the thing being multiplied. Relics rather than
// duelist cards for two reasons — they are full-bleed pictures, so they survive being shrunk in a
// way a card carrying a 1:1 pixel-art form mark and a column of cost ticks does not (see the glyph
// rule in CLAUDE.md), and the catalog is large enough that a different eight every cycle is
// genuinely a different picture.
//
// **It may never change an outcome**, the constraint every animation in this package is under. It
// is drawn on a screen that resolves nothing, reads no run, and touches no duelist.

import (
	"image"
	"math"
	"math/rand"
	"sort"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// speedStripHeight is the band the whole cycle happens in, and it is what the settings column
	// is re-spaced around — see the layout in settings.go.
	speedStripHeight = 125

	// speedStripCards is how many are dealt. Eight is the hand size, which is not a coincidence:
	// the strip is a picture of a hand.
	speedStripCards = 8

	// speedStripScale shrinks RelicStyle's 200x280 to 70x98. **Scaled at draw time rather than
	// rendered small**, which is only safe because a relic card is a picture behind a border: it
	// has no glyph to lose its derived rim and no text to fall under its legible size. Anything
	// carrying either has to be drawn at its own size instead.
	speedStripScale = 0.35

	// speedStripPitch is the gap between one seat and the next. **Narrower than the card**, so the
	// eight overlap and read as a hand being held rather than eight cards standing in a row.
	speedStripPitch = 52

	// speedStripStagger is how much of a phase separates the first card's movement from the last's,
	// as a fraction of that phase. Without it eight cards move as one slab.
	speedStripStagger = 0.06

	// speedStripLift is how high a card arcs as it crosses the hand during the riffle.
	speedStripLift = 26

	// speedStripSeed fixes which relics are dealt.
	//
	// **A fixed seed rather than a stream off the run** *(2026-09-15)*. `internal/seeds` derives
	// every stream from `GlobalState.RunSeed`, and this must not be one of them: the settings
	// screen is reachable from the title, where there is no run at all, and a decoration that
	// advanced a run's cursor would reroll a shop or a shuffle because somebody looked at the
	// speed bar. It is also not a `seeds.Stream`, because the stream table's own question — what
	// would this silently reroll if it shared — has the answer "nothing, in either direction".
	// The catalog is walked in a shuffled order and reshuffled on the wrap, so the cycle shows a
	// different eight each time while the sequence is identical on every launch.
	speedStripSeed = 0x5EED5
)

// The cycle. Each phase reads its length from `beat` **every tick**, which is the whole point of
// the widget — see the note on advance.
type speedPhase int

const (
	speedDealing speedPhase = iota
	speedHolding
	speedRiffling
	speedSettling
	speedClearing
)

// speedInTicks is the deal: eight cards crossing in from the left.
func speedInTicks() int { return beat(6, 5) }

// speedHoldTicks is the beat the hand is held before and after the riffle. **Short**, because a
// hand at rest is the one part of the cycle that says nothing about the speed.
func speedHoldTicks() int { return beat(2, 5) }

// speedRiffleTicks is the shuffle itself.
func speedRiffleTicks() int { return beat(1, 1) }

// speedOutTicks is the throw off the right-hand side.
func speedOutTicks() int { return beat(6, 5) }

// speedStripShuffles are the permutations the strip cycles through, one per pass: where each card
// goes, by seat.
//
// **Three rather than one, and they are deliberately unalike** *(owner's call, 2026-09-15)*. The
// strip ran a single perfect riffle to begin with and read as too uniform — eight cards travelling
// a comparable distance in the same direction at the same moment is one gesture repeated eight
// times, which is a poor thing to judge a *speed* against. What fixes it is not a better
// permutation but a *changing* one: the loop is watched for a while, so the eye starts predicting
// the second pass during the first.
//
//   - a **cut**: the last card crosses the whole hand, everything else shuffles along one seat. One
//     fast thing and seven slow ones in the same beat.
//   - a **riffle**: the bottom half interleaved into the top. Every card moves a middling distance.
//   - a **scatter**: no two cards moving alike, which is the messiest of the three and the one that
//     makes the set read as a shuffle rather than as a rotation.
//
// **Fixed rather than rolled**, which is the repo's default for anything that sounds random: the
// movement is what is being looked at, and a fresh permutation every cycle would be a roll bought
// with nothing. Cycling a short list is indistinguishable from chance to anyone watching and needs
// no stream, no salt and no argument in MECHANICS.md.
//
// **Each row must be a permutation of 0..speedStripCards-1** — every seat used exactly once, or two
// cards land on one seat and a third is never dealt. `TestEverySpeedStripShuffleIsAPermutation` is
// what refuses that, because nothing else would: a bad row draws a plausible-looking hand.
var speedStripShuffles = [][speedStripCards]int{
	{1, 2, 3, 4, 5, 6, 7, 0},
	{0, 2, 4, 6, 1, 3, 5, 7},
	{3, 0, 5, 1, 6, 2, 7, 4},
}

// speedStrip is the widget: which relics are on the table, and where the cycle has got to.
type speedStrip struct {
	keys []string // the eight being shown
	deck []string // the catalog, shuffled, walked eight at a time
	at   int
	rng  *rand.Rand

	phase speedPhase

	// shuffle is which of speedStripShuffles this pass is using. It advances with the hand, so the
	// permutation and the eight pictures change together.
	shuffle int

	// p is how far through the current phase, 0..1. **Progress rather than a tick count**, and
	// that is the one design decision in this file worth defending: a `travel` fixes its duration
	// when it is built, so a speed changed mid-flight would not be seen until the next phase
	// began. Advancing a fraction whose size is read fresh each tick means dragging the bar
	// changes the *rate* under the cursor, with no jump — which is the thing the player is
	// dragging the bar to find out.
	p float64
}

// update advances the cycle by one tick.
func (s *speedStrip) update(gs *state.GlobalState) {
	if s.rng == nil {
		s.rng = rand.New(rand.NewSource(speedStripSeed))
	}
	if len(s.keys) == 0 {
		s.reseat(gs)
		if len(s.keys) == 0 {
			return // no catalog loaded; nothing to draw and nothing to advance
		}
	}

	ticks := s.phaseTicks()
	if ticks < 1 {
		ticks = 1
	}
	s.p += 1 / float64(ticks)
	if s.p < 1 {
		return
	}

	s.p = 0
	switch s.phase {
	case speedDealing:
		s.phase = speedHolding
	case speedHolding:
		s.phase = speedRiffling
	case speedRiffling:
		s.phase = speedSettling
	case speedSettling:
		s.phase = speedClearing
	case speedClearing:
		// A whole new hand and the next permutation, so neither the pictures nor the gesture
		// repeats on consecutive passes.
		s.reseat(gs)
		s.shuffle++
		s.phase = speedDealing
	}
}

// phaseTicks is how long the phase being played lasts, at the speed in force right now.
func (s *speedStrip) phaseTicks() int {
	switch s.phase {
	case speedDealing:
		return speedInTicks()
	case speedRiffling:
		return speedRiffleTicks()
	case speedClearing:
		return speedOutTicks()
	default:
		return speedHoldTicks()
	}
}

// reseat takes the next eight relics, reshuffling the catalog when it runs out.
func (s *speedStrip) reseat(gs *state.GlobalState) {
	if len(gs.Relics) == 0 {
		s.keys = nil
		return
	}
	if s.at+speedStripCards > len(s.deck) {
		s.deck = data.RelicOrder(gs.Relics)
		s.rng.Shuffle(len(s.deck), func(i, j int) { s.deck[i], s.deck[j] = s.deck[j], s.deck[i] })
		s.at = 0
	}

	// A catalog smaller than a hand draws what there is rather than panicking on the slice.
	n := speedStripCards
	if n > len(s.deck) {
		n = len(s.deck)
	}
	s.keys = s.deck[s.at : s.at+n]
	s.at += n
}

// draw paints the cycle inside band.
func (s *speedStrip) draw(gs *state.GlobalState, screen *ebiten.Image, band image.Rectangle) {
	if len(s.keys) == 0 {
		return
	}

	w := float64(cards.RelicStyle.Width) * speedStripScale
	h := float64(cards.RelicStyle.Height) * speedStripScale

	// The row is centered in the band, and the seats are measured from its left edge.
	span := float64(speedStripPitch*(len(s.keys)-1)) + w
	left := float64(band.Min.X+band.Max.X)/2 - span/2
	top := float64(band.Min.Y+band.Max.Y)/2 - h/2

	for _, i := range s.drawOrder() {
		record, ok := gs.Relics[s.keys[i]]
		if !ok {
			continue
		}
		x, y := s.seatOf(i, left, top, w)

		var geo ebiten.GeoM
		geo.Scale(speedStripScale, speedStripScale)
		geo.Translate(x, y)
		drawFlyingCard(gs, screen, relicSpec(gs, record, "", true, false), cards.RelicStyle, geo)
	}
}

// seat is where card i ends the shuffle, under whichever permutation this cycle drew.
func (s *speedStrip) seat(i int) int {
	perm := speedStripShuffles[s.shuffle%len(speedStripShuffles)]
	return perm[i%len(perm)]
}

// moved is how many seats card i travels in the shuffle, which is what makes a permutation *look*
// like one: it sets how high a card arcs and whether it passes over or under its neighbors.
//
// **Derived rather than tabulated** *(2026-09-15)*, so a new permutation needs no second table
// saying which of its cards are the interesting ones — get that pairing wrong and a card slides
// under the very cards it is crossing.
func (s *speedStrip) moved(i int) int {
	d := s.seat(i) - i
	if d < 0 {
		return -d
	}
	return d
}

// drawOrder is which card is painted over which.
//
// **Furthest-travelled last while the shuffle runs**, so the card crossing the hand passes over the
// ones it crosses; **by seat the rest of the time**, so a hand at rest overlaps left-under-right
// like a hand being held. Those are two different questions and the same order cannot answer both:
// a card that ends at seat 0 has to be on top while it flies there and underneath once it lands.
func (s *speedStrip) drawOrder() []int {
	order := make([]int, len(s.keys))
	for i := range order {
		order[i] = i
	}

	if s.phase == speedRiffling {
		sort.SliceStable(order, func(a, b int) bool { return s.moved(order[a]) < s.moved(order[b]) })
		return order
	}
	if s.phase == speedSettling || s.phase == speedClearing {
		sort.SliceStable(order, func(a, b int) bool { return s.seat(order[a]) < s.seat(order[b]) })
	}
	return order
}

// seatOf is where card i sits this frame.
func (s *speedStrip) seatOf(i int, left, top, w float64) (float64, float64) {
	rest := left + float64(speedStripPitch*i)
	shuffled := left + float64(speedStripPitch*s.seat(i))
	offRight := left + float64(speedStripPitch*len(s.keys)) + w*2.5

	switch s.phase {
	case speedDealing:
		// In from off the left edge, the first card leading.
		t := easeOut(s.staggered(i))
		return lerp(left-w*2.5, rest, t), top

	case speedRiffling:
		t := s.staggered(i)

		// **A card arcs in proportion to how far it is going.** One that shuffles along a single
		// seat barely leaves the table; one crossing the whole hand lifts clear of it.
		lift := float64(speedStripLift) * float64(s.moved(i)) / float64(len(s.keys)-1)
		return lerp(rest, shuffled, easeOut(t)), top - math.Sin(t*math.Pi)*lift

	case speedClearing:
		// Off the right, still in their shuffled seats.
		t := easeOut(s.staggered(i))
		return lerp(shuffled, offRight, t), top

	case speedSettling:
		return shuffled, top

	default:
		return rest, top
	}
}

// staggered is card i's own progress through the phase: the strip's progress, shifted so the cards
// do not move as one slab, clamped so nobody starts early or overshoots.
func (s *speedStrip) staggered(i int) float64 {
	span := 1 - speedStripStagger*float64(len(s.keys)-1)
	if span <= 0 {
		return s.p
	}
	t := (s.p - speedStripStagger*float64(i)) / span
	switch {
	case t < 0:
		return 0
	case t > 1:
		return 1
	default:
		return t
	}
}

// lerp is the straight line between two points.
func lerp(a, b, t float64) float64 { return a + (b-a)*t }
