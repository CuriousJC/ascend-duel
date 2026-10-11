package screens

// combat_particles.go — the elemental particles round a hit's total and behind its figure.
//
// **Two fields, one emitter.** A *wrap* is born on an ellipse round a line's total from the beat the
// total starts landing until the line is thrown; a *trail* is dropped behind the damage figure as it
// flies into its card, a creature's included. Both are `ui.Emitter`s and both wear the element of
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

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// The wrap's ellipse round a total, in pixels.
const (
	wrapRX = 31
	wrapRY = 20
)

// wrapEmitter is the field round a resting total: each element its own physics, pieces and sparks
// together.
func wrapEmitter(el cards.Element, seed uint32) ui.Emitter {
	return ui.Emitter{Element: el, Set: ui.ParticleMixed, Motion: ui.MotionElemental, Seed: seed}
}

// trailEmitter is the field behind a flying figure: denser and a little smaller than the wrap,
// because it is spread along a path rather than gathered round a point.
func trailEmitter(el cards.Element, seed uint32) ui.Emitter {
	return ui.Emitter{Element: el, Set: ui.ParticleMixed, Motion: ui.MotionElemental,
		Rate: 1.2, Size: 22, Seed: seed}
}

// particleField is one wrap or one trail on stage.
type particleField struct {
	em ui.Emitter

	// center is a wrap's total, and col the line it belongs to, so throwing the line can stop it.
	center image.Point
	col    int

	// trail says this field follows hit rather than sitting round center. **The flight is copied,
	// not pointed at**: the figure is dropped from the theater when it fades, and its trail goes on
	// for a particle's life after that.
	trail bool
	hit   hitFlight

	// age is how long the field has been up, and until the tick it stops emitting — negative while a
	// wrap's line is still running.
	age, until int
}

func (f *particleField) Tick() { f.age++ }

// Done is once the last particle born has lived its life.
func (f *particleField) Done() bool {
	return f.until >= 0 && f.age > f.until+f.em.LifeTicks()
}

// raiseWraps starts a wrap round every total that has begun landing and has not got one.
func (s *CombatScene) raiseWraps() {
	b := &s.Theater.mathBox
	for c := range b.columns {
		col := &b.columns[c]
		if !col.begun || col.wrapped || col.unthrown || len(col.items) == 0 || col.at < len(col.items)-1 {
			continue
		}
		col.wrapped = true
		el := s.handCardElement(b.side, col.seat)
		// A defense's line comes to 0 and throws nothing, so it gets nothing to wrap.
		if el == cards.Basic || col.total().text == "0" {
			continue
		}
		s.Theater.particles = append(s.Theater.particles, particleField{
			em:     wrapEmitter(el, uint32(s.round*977+c*131+col.seat+1)),
			center: col.total().at,
			col:    c,
			until:  -1,
		})
	}
}

// closeWrap stops line c's wrap emitting: its total has been thrown. **What is already out finishes
// its life** where it is, so the wrap thins away under the figure leaving it rather than vanishing.
func (s *CombatScene) closeWrap(c int) {
	for i := range s.Theater.particles {
		f := &s.Theater.particles[i]
		if !f.trail && f.col == c && f.until < 0 {
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

// drawParticles draws one kind of field: the wraps under the lines, the trails under the figures.
func (s *CombatScene) drawParticles(gs *state.GlobalState, screen *ebiten.Image, trails bool) {
	for _, f := range s.Theater.particles {
		if f.trail != trails {
			continue
		}
		if !f.trail {
			ui.DrawParticleWrap(screen, f.em, f.center, wrapRX, wrapRY, f.until, f.age)
			continue
		}
		from, ok := s.hitOrigin(gs, f.hit)
		if !ok {
			continue
		}
		to := s.hitTarget(gs, f.hit)
		fly := float64(hitFlyTicks())
		// **The figure's own path**, eased exactly as drawHits eases it, so the trail comes off the
		// number rather than off a line beside it.
		path := func(t float64) (float64, float64) {
			p := ui.EaseOut(ui.Clamp01(t / fly))
			return float64(from.X) + float64(to.X-from.X)*p, float64(from.Y) + float64(to.Y-from.Y)*p
		}
		ui.DrawParticleTrail(screen, f.em, path, f.until, f.age)
	}
}
