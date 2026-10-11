package ui

// particles.go — the elemental particles thrown around a hit's total and trailed behind its figure.
//
// **Every sprite is a still picture and every movement is here.** The art is one small object per
// file — an ember, a shard, a spark — and the code scales it, turns it, moves it and fades it.
//
// **It holds no state.** A particle is its index: when it was born, where, how fast and which
// sprite are all read off a hash of the index and the emitter's seed, and where it is now is a
// function of its age. So a field is drawn from nothing but a clock, nothing has to be ticked or
// pooled, and one hit always throws the same cloud — the crack pattern's rule and the dissolve's,
// derived rather than rolled, so no stream is consumed.
//
// **It decides nothing.** Particles are a picture of a hit, and no clock waits on them.

import (
	"image"
	"image/color"
	"math"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// ParticleSet is which sprites a field is drawn from.
type ParticleSet int

const (
	// ParticleShards are pieces of the element — embers, shards, pebbles — out of
	// `particle-<element>-<n>`.
	ParticleShards ParticleSet = iota
	// ParticleSparks are plain flecks of the element's color out of `spark-<element>-<n>`, one set
	// of shapes in every element.
	ParticleSparks
	// ParticleMixed alternates the two.
	ParticleMixed
)

// ParticleMotion is how a field moves.
type ParticleMotion int

const (
	// MotionElemental gives each element its own physics: fire rises, ice drifts down and spins,
	// lightning hops, earth falls, arcane circles.
	MotionElemental ParticleMotion = iota
	// MotionOrbit circles every element round its center the way arcane does, so the elements
	// differ by sprite and color alone.
	MotionOrbit
)

// Emitter is one field's settings. The zero value of every field but Element is a usable default.
type Emitter struct {
	Element cards.Element
	Set     ParticleSet
	Motion  ParticleMotion

	// Rate is particles born per tick, Life how many ticks each lives and Size its drawn width in
	// pixels at birth.
	Rate float64
	Life int
	Size float64

	// Glow draws each sprite a second time additively, so it brightens in its own color.
	Glow bool

	// Seed separates two fields drawn at once, so neighbors are not the same cloud.
	Seed uint32

	// Sprites, when set, replaces the element's own sprites with these asset keys — ash, rubble,
	// rune motes — while Element still picks the physics. None of them is turned to face its way.
	Sprites []string

	// Tint, when its alpha is not zero, multiplies every sprite.
	Tint color.RGBA
}

func (e Emitter) rate() float64 {
	if e.Rate > 0 {
		return e.Rate
	}
	return 0.5
}

// LifeTicks is how long the longest-lived particle of this field lasts, which is how long a field
// that has stopped emitting has left on screen.
func (e Emitter) LifeTicks() int { return e.life() }

func (e Emitter) life() int {
	if e.Life > 0 {
		return e.Life
	}
	return 48
}

func (e Emitter) size() float64 {
	if e.Size > 0 {
		return e.Size
	}
	return 26
}

// particleArtSize is the size the 256 source is reduced to once. Drawn sizes run 8 to 40 pixels,
// so the GPU only ever scales down from here, and a single reduction from a rich source keeps the
// contour.
const particleArtSize = 64

// DrawParticleWrap draws a field born on an ellipse round `center` — the total sitting on its line.
// `age` is how long the field has been emitting, and `emitUntil` the tick it stops — the moment the
// total sets off — after which the particles already out finish their lives. Negative never stops.
func DrawParticleWrap(screen *ebiten.Image, e Emitter, center image.Point, rx, ry float64, emitUntil, age int) {
	cx, cy := float64(center.X), float64(center.Y)
	e.draw(screen, age, emitUntil, func(i int, born float64) (x, y, dx, dy, ox, oy float64) {
		th := 2 * math.Pi * hash01(e.Seed, i, 1)
		r := 0.85 + 0.3*hash01(e.Seed, i, 2)
		c, s := math.Cos(th), math.Sin(th)
		return cx + rx*r*c, cy + ry*r*s, c, s, cx, cy
	})
}

// DrawParticleTrail draws a field dropped behind a moving point — the figure flying into a card.
// `path` is where the point was at a tick, `emitUntil` is the tick it stops dropping particles
// (the arrival), and `age` is the tick now: the trail outlives the flight by a particle's life.
func DrawParticleTrail(screen *ebiten.Image, e Emitter, path func(t float64) (float64, float64), emitUntil, age int) {
	e.draw(screen, age, emitUntil, func(i int, born float64) (x, y, dx, dy, ox, oy float64) {
		x, y = path(born)
		px, py := path(born - 1)
		vx, vy := x-px, y-py
		n := math.Hypot(vx, vy)
		if n == 0 {
			vx, vy, n = 0, -1, 1
		}
		// Thrown backwards off the path and spread to either side of it.
		bx, by := -vx/n, -vy/n
		side := hash01(e.Seed, i, 2)*2 - 1
		dx, dy = bx-by*side*0.8, by+bx*side*0.8
		m := math.Hypot(dx, dy)
		return x + -by*side*10, y + bx*side*10, dx / m, dy / m, x, y
	})
}

// DrawParticleField is the general case the wrap and the trail are two shapes of: `origin` says where
// particle i was born, given its birth tick, and the direction it is thrown in. A field born along a
// burning edge or out of a crumbling card is one of these.
func DrawParticleField(screen *ebiten.Image, e Emitter, emitUntil, age int,
	origin func(i int, born float64) (x, y, dx, dy float64)) {

	e.draw(screen, age, emitUntil, func(i int, born float64) (x, y, dx, dy, ox, oy float64) {
		x, y, dx, dy = origin(i, born)
		return x, y, dx, dy, x, y
	})
}

// draw walks every particle alive at `age`. `origin` says where particle i was born, the direction
// it is thrown in, and the point it circles when it circles one. emitUntil < 0 never stops.
func (e Emitter) draw(screen *ebiten.Image, age, emitUntil int,
	origin func(i int, born float64) (x, y, dx, dy, ox, oy float64)) {

	rate, life := e.rate(), e.life()
	now := float64(age)
	last := int(math.Floor(now * rate))
	if emitUntil >= 0 {
		last = min(last, int(math.Floor(float64(emitUntil)*rate)))
	}
	first := max(0, int(math.Ceil((now-float64(life))*rate)))

	for i := first; i <= last; i++ {
		born := float64(i) / rate
		a := now - born
		lf := float64(life)
		if e.Element == cards.Lightning && e.Motion == MotionElemental {
			lf *= 0.6
		}
		if a < 0 || a >= lf {
			continue
		}
		x, y, dx, dy, ox, oy := origin(i, born)
		e.drawOne(screen, i, a, lf, x, y, dx, dy, ox, oy)
	}
}

// drawOne places, turns, sizes and fades particle i at age a of a life lf.
func (e Emitter) drawOne(screen *ebiten.Image, i int, a, lf, x, y, dx, dy, ox, oy float64) {
	h := func(k int) float64 { return hash01(e.Seed, i, k) }
	speed := 0.6 + 0.8*h(3)
	var vx, vy float64 // the velocity now, which a pointed sprite faces

	motion := e.Motion
	el := e.Element
	if motion == MotionOrbit {
		el = cards.Arcane
	}

	switch el {
	case cards.Fire:
		// Rises, drifting a little outward, and sways.
		vx = dx*0.4*speed + 0.35*math.Sin(a*0.25+h(4)*6)
		vy = dy*0.4*speed - 1.1*speed
		x += dx*0.4*speed*a + 4*math.Sin(a*0.12+h(4)*6)
		y += dy*0.4*speed*a - 1.1*speed*a
	case cards.Ice:
		// Thrown out gently, then drifts down.
		vx = dx*0.7*speed + 0.2*math.Sin(a*0.1+h(4)*6)
		vy = dy*0.7*speed*math.Exp(-a/20) + 0.45
		x += dx*0.7*speed*20*(1-math.Exp(-a/20)) + 3*math.Sin(a*0.08+h(4)*6)
		y += dy*0.7*speed*20*(1-math.Exp(-a/20)) + 0.45*a
	case cards.Lightning:
		// Hops: holds a spot for a few ticks, then jumps along its direction to the next.
		step := math.Floor(a / 4)
		jx := (hash01(e.Seed, i, 10+int(step))*2 - 1) * 8
		jy := (hash01(e.Seed, i, 40+int(step))*2 - 1) * 8
		x += dx*speed*7*step + jx
		y += dy*speed*7*step + jy
		vx, vy = dx+jx*0.05, dy+jy*0.05
	case cards.Earth:
		// Thrown up and out, then falls.
		const g = 0.09
		ux, uy := dx*1.4*speed, dy*1.4*speed-1.6*speed
		x += ux * a
		y += uy*a + 0.5*g*a*a
		vx, vy = ux, uy+g*a
	default:
		// Circles its center, the radius easing outward as it goes.
		rx, ry := x-ox, y-oy
		r0 := math.Hypot(rx, ry)
		th0 := math.Atan2(ry, rx)
		w := (0.04 + 0.03*h(5)) * dirSign(h(6))
		r := r0 + a*0.35*speed
		if r0 < 1 {
			// A particle born on its own center (a trail) spirals out of it.
			r = 4 + a*0.9*speed
			th0 = math.Atan2(dy, dx)
		}
		th := th0 + w*a
		x, y = ox+r*math.Cos(th), oy+r*math.Sin(th)
		vx, vy = -math.Sin(th)*w, math.Cos(th)*w
	}

	img, pointed := e.sprite(i)
	if img == nil {
		return
	}

	t := a / lf
	size := e.size() * (0.6 + 0.6*h(7)) * (1 - 0.6*t)
	if el == cards.Fire {
		size *= 0.9 + 0.1*math.Sin(a*0.6+h(8)*6)
	}
	alpha := float32(1)
	if t > 0.55 {
		alpha = float32((1 - t) / 0.45)
	}

	var rot float64
	if pointed {
		// The art points up; turn it to face where it is going.
		rot = math.Atan2(vx, -vy)
	} else {
		rot = h(9)*2*math.Pi + a*(h(11)*0.12-0.06)
		if el == cards.Ice {
			rot += a * 0.05 * dirSign(h(12))
		}
	}

	sc := size / float64(particleArtSize)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-particleArtSize/2, -particleArtSize/2)
	op.GeoM.Rotate(rot)
	op.GeoM.Scale(sc, sc)
	op.GeoM.Translate(x, y)
	if e.Tint.A != 0 {
		op.ColorScale.ScaleWithColor(e.Tint)
	}
	op.ColorScale.ScaleAlpha(alpha)
	screen.DrawImage(img, op)

	if e.Glow {
		op.Blend = ebiten.BlendLighter
		op.ColorScale.Reset()
		op.ColorScale.ScaleAlpha(alpha * 0.5)
		screen.DrawImage(img, op)
	}
}

// sprite is particle i's picture, and whether it is one of the pointed ones that faces its way.
func (e Emitter) sprite(i int) (*ebiten.Image, bool) {
	if len(e.Sprites) > 0 {
		key := e.Sprites[int(hash01(e.Seed, i, 13)*float64(len(e.Sprites)))%len(e.Sprites)]
		return systems.ArtMarkImage(key, particleArtSize, particleArtSize), false
	}
	n := 1 + int(hash01(e.Seed, i, 13)*3)
	if n > 3 {
		n = 3
	}
	prefix := "particle"
	switch e.Set {
	case ParticleSparks:
		prefix = "spark"
	case ParticleMixed:
		if i%2 == 1 {
			prefix = "spark"
		}
	}
	// The pointed sprite is 2 among the pieces (shard, flame lick, arc) and 3 among the sparks
	// (the streak).
	pointed := (prefix == "particle" && n == 2) || (prefix == "spark" && n == 3)
	key := prefix + "-" + e.Element.String() + "-" + string(rune('0'+n))
	return systems.ArtMarkImage(key, particleArtSize, particleArtSize), pointed
}

func dirSign(f float64) float64 {
	if f < 0.5 {
		return -1
	}
	return 1
}

// hash01 is a value in [0,1) derived from a seed, a particle and a question about it. Derived, never
// rolled: see the file comment.
func hash01(seed uint32, i, k int) float64 {
	x := seed ^ uint32(i)*0x9E3779B1 ^ uint32(k)*0x85EBCA77
	x ^= x >> 16
	x *= 0x7FEB352D
	x ^= x >> 15
	x *= 0x846CA68B
	x ^= x >> 16
	return float64(x) / (1 << 32)
}
