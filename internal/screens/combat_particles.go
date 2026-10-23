package screens

// combat_particles.go — the elemental particles round a hit's total and behind its figure.
//
// **Two fields, one emitter.** A *wrap* is born on an ellipse round **a hit's whole line** —
// every term, operator and the total — from the beat the line starts until it is thrown, so the
// whole of the working is carried in its card's element and not only the sum; a *trail* is dropped
// behind the damage figure as it flies into its card, a creature's included, as far as a shield
// stops it. Both are `ui.Emitter`s and both wear the element of
// the card that threw the hit — the color rule the figure itself keeps (see hitInkFor), so an
// elementless card throws none.
//
// **Nothing waits on them.** A field is a mover the theater ticks and drops, and it is deliberately
// absent from `combatTheater.Running`: a particle is decoration on a number that already holds the
// cursor, and a round that waited for the last ember to fade would be paced by a picture.
//
// **It decides nothing**, and it rolls nothing — every particle is derived from its index and the
// field's seed. See internal/ui/particles.go.

import (
	"image"
	"math"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// The wrap's reach past the stack it is round, in pixels, and the ellipse its density is tuned
// against: a field round a bigger stack is born proportionally faster, so the cloud is as thick
// round a card's whole working as it is round a lone total.
const (
	wrapPadX  = 34
	wrapPadY  = 26
	wrapRefRX = 31
	wrapRefRY = 20
)

// wrapRate is how fast a wrap of this size is born.
func wrapRate(rx, ry float64) float64 {
	return 0.5 * math.Max(1, ellipsePerimeter(rx, ry)/ellipsePerimeter(wrapRefRX, wrapRefRY))
}

// ellipsePerimeter is Ramanujan's approximation, which is far closer than this needs.
func ellipsePerimeter(a, b float64) float64 {
	return math.Pi * (3*(a+b) - math.Sqrt((3*a+b)*(a+3*b)))
}

// stackExtent is the centre and half-size of everything a set of items will show — every row of
// every line — measured off each item's own laid-out centre and size, so it holds however many rows
// the working runs to.
func stackExtent(gs *state.GlobalState, items []mathItem) (image.Point, float64, float64) {
	if len(items) == 0 {
		return image.Point{}, 0, 0
	}
	left, top := math.Inf(1), math.Inf(1)
	right, bottom := math.Inf(-1), math.Inf(-1)
	for _, it := range items {
		hw, hh := mathWidth(gs, it.text, it.size)/2, figureHeight(it.size)/2
		x, y := float64(it.at.X), float64(it.at.Y)
		left, right = math.Min(left, x-hw), math.Max(right, x+hw)
		top, bottom = math.Min(top, y-hh), math.Max(bottom, y+hh)
	}
	return image.Pt(int((left+right)/2), int((top+bottom)/2)), (right - left) / 2, (bottom - top) / 2
}

// wrapEmitter is the field round a resting total: each element its own physics, pieces and sparks
// together.
func wrapEmitter(el cards.Element, seed uint32, rate float64) ui.Emitter {
	return ui.Emitter{Element: el, Set: ui.ParticleMixed, Rate: rate, Seed: seed}
}

// trailEmitter is the field behind a flying figure: denser and a little smaller than the wrap,
// because it is spread along a path rather than gathered round a point.
func trailEmitter(el cards.Element, seed uint32) ui.Emitter {
	return ui.Emitter{Element: el, Set: ui.ParticleMixed,
		Rate: 1.2, Size: 22, Seed: seed}
}

// particleField is one wrap or one trail on stage.
type particleField struct {
	em ui.Emitter

	// center is the middle of a wrap's stack and rx, ry the ellipse round it, and seat the played
	// card whose working it wraps, so throwing that card's last line can stop it.
	center image.Point
	rx, ry float64
	seat   int

	// trail says this field follows hit rather than sitting round center. **The flight is copied,
	// not pointed at**: the figure is dropped from the theater when it fades, and its trail goes on
	// for a particle's life after that.
	trail bool
	hit   hitFlight

	// toMeet says the trail follows a stopped hit's figure as far as it is met, ic, rather
	// than a landing one.
	toMeet bool
	ic     intercept

	// age is how long the field has been up, and until the tick it stops emitting — negative while a
	// wrap's line is still running.
	age, until int
}

func (f *particleField) Tick() { f.age++ }

// Done is once the last particle born has lived its life.
func (f *particleField) Done() bool {
	return f.until >= 0 && f.age > f.until+f.em.LifeTicks()
}

// raiseWraps starts one wrap round **a card's whole stack** the first time any of its lines begins:
// every row of every hit that card throws, its echoes' lines included. All of them are laid out
// before anything is shown, so the field is round the space the working is about to fill, as long
// as it runs, rather than growing as it fills it. A card's lines are all its element, so one field
// says it.
func (s *CombatScene) raiseWraps(gs *state.GlobalState) {
	b := &s.Theater.mathBox
	for c := range b.columns {
		col := &b.columns[c]
		if !col.begun || col.wrapped {
			continue
		}
		seat := col.seat
		var stack []mathItem
		worth := false
		for k := range b.columns {
			other := &b.columns[k]
			if other.seat != seat {
				continue
			}
			other.wrapped = true
			if other.unthrown || other.silent || len(other.items) == 0 {
				continue
			}
			stack = append(stack, other.items...)
			// A defense's line comes to 0 and throws nothing, so a stack of nothing else is not wrapped.
			if other.total().text != "0" {
				worth = true
			}
		}
		el := s.handCardElement(b.side, seat)
		if el == cards.Basic || !worth {
			continue
		}
		center, hw, hh := stackExtent(gs, stack)
		rx, ry := hw+wrapPadX, hh+wrapPadY
		s.Theater.particles = append(s.Theater.particles, particleField{
			em:     wrapEmitter(el, uint32(s.round*977+seat*131+1), wrapRate(rx, ry)),
			center: center,
			rx:     rx,
			ry:     ry,
			seat:   seat,
			until:  -1,
		})
	}
}

// closeWrap stops the wrap round line c's card once every line of that card has been thrown.
// **What is already out finishes its life** where it is, so the wrap thins away under the last
// figure leaving it rather than vanishing.
func (s *CombatScene) closeWrap(c int) {
	b := &s.Theater.mathBox
	seat := b.columns[c].seat
	for _, col := range b.columns {
		if col.seat == seat && !col.thrown {
			return
		}
	}
	for i := range s.Theater.particles {
		f := &s.Theater.particles[i]
		if !f.trail && f.seat == seat && f.until < 0 {
			f.until = f.age
		}
	}
}

// closeWraps stops every wrap still emitting, for a box coming down with its lines unthrown.
func (s *CombatScene) closeWraps() {
	for i := range s.Theater.particles {
		f := &s.Theater.particles[i]
		if !f.trail && f.until < 0 {
			f.until = f.age
		}
	}
}

// raiseTrail starts the trail behind a figure that has just set off.
func (s *CombatScene) raiseTrail(h hitFlight) {
	if h.element == cards.Basic {
		return
	}
	seed := uint32(s.round*7919+len(s.Theater.particles)*613) + uint32(h.amount)
	s.Theater.particles = append(s.Theater.particles, particleField{
		em:    trailEmitter(h.element, seed),
		trail: true,
		hit:   h,
		until: hitFlyTicks(),
	})
}

// raiseInterceptTrail starts the trail behind a stopped hit's figure.
func (s *CombatScene) raiseInterceptTrail(ic intercept) {
	if ic.ink == cards.Basic {
		return
	}
	seed := uint32(s.round*7919+len(s.Theater.particles)*613) + uint32(len(ic.figure))
	s.Theater.particles = append(s.Theater.particles, particleField{
		em:     trailEmitter(ic.ink, seed),
		trail:  true,
		toMeet: true,
		ic:     ic,
		until:  ic.meetTicks,
	})
}

// drawParticles draws one kind of field: the wraps under the lines, the trails under the figures.
func (s *CombatScene) drawParticles(gs *state.GlobalState, screen *ebiten.Image, trails bool) {
	for _, f := range s.Theater.particles {
		if f.trail != trails {
			continue
		}
		if !f.trail {
			ui.DrawParticleWrap(screen, f.em, f.center, f.rx, f.ry, f.until, f.age)
			continue
		}
		var from, to image.Point
		fly := float64(hitFlyTicks())
		if f.toMeet {
			// **The figure's real path into its target**, cut where it is met: the trail stops being
			// dropped at the meeting, so it runs only as far as the hit did.
			from, to = s.interceptOrigin(gs, f.ic), fighterCenter(gs, f.ic.target)
			fly = float64(interceptFlyTicks())
		} else {
			var ok bool
			if from, ok = s.hitOrigin(gs, f.hit); !ok {
				continue
			}
			to = s.hitTarget(gs, f.hit)
		}
		// **The figure's own path**, eased exactly as drawHits eases it, so the trail comes off the
		// number rather than off a line beside it.
		path := func(t float64) (float64, float64) {
			p := ui.EaseOut(ui.Clamp01(t / fly))
			return float64(from.X) + float64(to.X-from.X)*p, float64(from.Y) + float64(to.Y-from.Y)*p
		}
		ui.DrawParticleTrail(screen, f.em, path, f.until, f.age)
	}
}
