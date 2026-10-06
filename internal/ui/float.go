package ui

// **A card at rest is not quite still.** The worn relics, the carried consumables and the dealt
// cards drift a few pixels on a slow loop, so a screen nobody is touching still reads as alive.
//
// **`IdleFloat` is the one the rows wear; `FloatSchemes` is the shortlist it was picked from**, and
// the animation gallery draws every scheme side by side on the real cards so a retune is judged
// against its alternatives. Each is a handful of figures rather than a function of its own, so they
// differ only in what moves and how far.
//
// **What floats**: the worn relics, the carried consumables, the combat hand and the dealt rows.
// **What does not**: the card under the cursor (it is raised, and a target that moves while it is
// being read is a target being chased), a card being dragged, a relic firing (its toast is its own
// motion), and anything mid-flight, mid-slide or mid-morph — those are journeys, and a float is
// only ever a card at rest. Hit testing reads the seat and never the float.
//
// **It is an idle oscillation, so it runs on the free clock** — `gs.Count`, the breath's clock —
// and not on `Beat`. Everything `Beat` scales is a duration between two things; tying this to it
// would make a resting card bob faster the faster a round is watched.
//
// **The phase per seat is derived, never rolled**: the seat's index walked round the circle by the
// golden angle, so neighbors never move in step and the same seat always moves the same way. It may
// never change an outcome — it is something to look at.

import (
	"image"
	"math"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// Float is one idle-motion scheme. Every distance is in screen pixels at the card's own size, every
// period in ticks, and a zero amplitude turns that axis off.
type Float struct {
	Name string
	What string

	// Bob is the vertical swing and BobTicks its period.
	Bob      float64
	BobTicks float64

	// Drift is the sideways swing, on a period of its own so the path is a loop rather than a line.
	Drift      float64
	DriftTicks float64

	// Tilt is a rock about the card's center, in degrees either way.
	Tilt      float64
	TiltTicks float64

	// Ripple, when set, phases the seats in order along the row — a wave rolling across it — rather
	// than scattering them by the golden angle.
	Ripple float64
}

// FloatSchemes is the shortlist the gallery compares.
var FloatSchemes = []Float{
	{
		Name: "bob",
		What: "3px up and down, rippling along the row",
		Bob:  3, BobTicks: 210,
		Ripple: 0.55,
	},
	{
		Name: "drift",
		What: "a 0.75 x 1.25px loop, each card its own",
		Bob:  1.25, BobTicks: 235,
		Drift: 0.75, DriftTicks: 320,
	},
	{
		Name: "sway",
		What: "2px bob and a rock under one degree",
		Bob:  2, BobTicks: 250,
		Tilt: 0.7, TiltTicks: 300,
	},
}

// IdleFloat is the scheme every resting card in the rows wears.
var IdleFloat = FloatSchemes[1] // drift

// DrawFloatingCard draws a card resting in seat `seat` of a row, with its top-left at `at` before
// the float. **The seat is the card's index in its own row**, so one row's phases are independent of
// another's.
func DrawFloatingCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point, seat int,
	spec cards.Spec, st cards.Style) {

	DrawFlyingCard(gs, screen, spec, st, IdleFloat.GeoAt(at, st.Width, st.Height, gs.Count, seat))
}

// floatGolden is the golden angle, which is what scatters seat phases so no two neighbors agree.
const floatGolden = math.Pi * (3 - 2.2360679775)

// phase is where in its loop seat `seat` starts.
func (f Float) phase(seat int) float64 {
	if f.Ripple != 0 {
		return -f.Ripple * float64(seat)
	}
	return floatGolden * float64(seat)
}

// Offset is how far the card in `seat` is displaced at tick `count`: x and y in pixels and a turn in
// radians.
func (f Float) Offset(count, seat int) (dx, dy, turn float64) {
	ph := f.phase(seat)
	c := float64(count)
	if f.Bob != 0 && f.BobTicks > 0 {
		dy = f.Bob * math.Sin(2*math.Pi*c/f.BobTicks+ph)
	}
	if f.Drift != 0 && f.DriftTicks > 0 {
		// A second phase off the first, so the loop is not the same ellipse on every card.
		dx = f.Drift * math.Sin(2*math.Pi*c/f.DriftTicks+ph*1.7)
	}
	if f.Tilt != 0 && f.TiltTicks > 0 {
		turn = f.Tilt * math.Pi / 180 * math.Sin(2*math.Pi*c/f.TiltTicks+ph*1.3)
	}
	return dx, dy, turn
}

// GeoAt is the transform that draws a w×h card whose resting top-left is `at`, floated. It turns
// about the card's center, so a rock does not walk the card sideways.
//
// **It puts the card off the pixel grid**, so it is drawn through `DrawFlyingCard`, which filters.
func (f Float) GeoAt(at image.Point, w, h, count, seat int) ebiten.GeoM {
	dx, dy, turn := f.Offset(count, seat)
	var g ebiten.GeoM
	g.Translate(-float64(w)/2, -float64(h)/2)
	if turn != 0 {
		g.Rotate(turn)
	}
	g.Translate(float64(at.X)+float64(w)/2+dx, float64(at.Y)+float64(h)/2+dy)
	return g
}
