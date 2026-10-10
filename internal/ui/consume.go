package ui

// consume.go — a consumable being used, as one sequence every kind runs through.
//
// **The reward screen's essence is the pattern**: the card leaves its seat for the middle of the
// screen and the change is shown there, where it can be watched. This is that, generalized. Every
// consumable is used the same way and differs in exactly one beat:
//
//  1. **Lift.** The consumable flies out of its seat in the band to the stage — the middle.
//  2. **Exit.** It is used up there, in its own way: an essence is *absorbed* (it comes apart in
//     light, rising), a rune *fires* (it rattles, a sigil turns out from under it, and it goes the
//     same way), a stone *crumbles* (its pieces fall), a cantrip *burns* (bottom to top).
//  3. **Motes.** What it gives travels from the stage to what it changes, as a particle trail: the
//     cards a rune or an essence was aimed at, the hand ladder a stone raises, the band a cantrip
//     puts its relic into.
//  4. **Change.** Each destination changes as its motes arrive — a morph, the same three shapes the
//     hand and the reward screen already use.
//
// **One clock.** The phases are fractions of the game's beat, so the whole sequence slows down and
// speeds up with the speed slider like every other mover; a Consume is ticked and drawn and decides
// nothing — the run has already changed by the time the first frame is drawn.

import (
	"image"
	"math"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// ConsumeExit is the one beat that differs between consumables: how the card is used up.
type ConsumeExit int

const (
	// ExitAbsorb is an essence: the card comes apart in light, each piece flaring and rising.
	ExitAbsorb ConsumeExit = iota
	// ExitSigil is a rune: a rattle, then a sigil turning out from under it as it is absorbed.
	ExitSigil
	// ExitCrumble is a stone: a tremor, then its pieces fall.
	ExitCrumble
	// ExitBurn is a cantrip: it burns from the bottom up.
	ExitBurn
)

// The sequence's beats.
func consumeLiftTicks() int  { return Beat(1, 1) }
func consumeToastTicks() int { return Beat(3, 5) }
func consumeExitTicks() int  { return Beat(2, 1) }
func consumeMoteTicks() int  { return Beat(1, 1) }

// consumeMotesAt is how far into the exit the motes set off: once the card is visibly going.
const consumeMotesAt = 0.35

// Consume is one consumable being used.
type Consume struct {
	Exit  ConsumeExit
	Face  cards.Spec
	Style cards.Style

	// From is the seat it leaves, Stage the seat it is used in.
	From, Stage image.Rectangle

	// To is every destination, and Changes the morph each one runs as its motes arrive — empty for a
	// destination that is not a card (the hand ladder), which the caller draws and may toast on
	// Arrived.
	To      []image.Rectangle
	Changes []Morph

	// Motes is the trail template: Element picks the physics, Sprites or Set the pictures.
	Motes Emitter

	Seed uint32
	Age  int
}

// NewConsume sets the sequence up and puts every change on the beat its motes arrive.
func NewConsume(c Consume) Consume {
	for i := range c.Changes {
		c.Changes[i] = c.Changes[i].Delayed(c.arrival())
	}
	return c
}

// liftTicks is the lift, or nothing for a card already standing where it is used — a prize on the
// reward screen is used in its own seat.
func (c Consume) liftTicks() int {
	if c.From == c.Stage {
		return 0
	}
	return consumeLiftTicks()
}

func (c Consume) exitStart() int {
	s := c.liftTicks()
	if c.Exit == ExitSigil {
		s += consumeToastTicks()
	}
	return s
}

func (c Consume) depart() int {
	return c.exitStart() + int(consumeMotesAt*float64(consumeExitTicks()))
}

func (c Consume) arrival() int { return c.depart() + consumeMoteTicks() }

// ArrivalTicks is how long after the start the motes reach their destinations, for a caller whose
// destinations change on their own clock — the hand's morphs, delayed to land with the motes.
func (c Consume) ArrivalTicks() int { return c.arrival() }

// Arrived reports whether the motes have reached their destinations — the moment a caller drawing a
// destination that is not a card should mark it.
func (c Consume) Arrived() bool { return c.Age >= c.arrival() }

// ArrivedTravel is a beat-long clock starting at the arrival, for a caller toasting a destination.
func (c Consume) ArrivedTravel() Travel {
	t := NewTravel(c.arrival(), Beat(3, 5))
	t.Age = c.Age
	return t
}

// Ticks is the whole sequence, the last change included.
func (c Consume) Ticks() int {
	end := max(c.exitStart()+consumeExitTicks(), c.arrival())
	if len(c.Changes) > 0 {
		end = c.arrival() + MorphTicks()
	}
	return end + Beat(1, 2)
}

func (c *Consume) Tick() {
	c.Age++
	for i := range c.Changes {
		c.Changes[i].Tick()
	}
}

func (c Consume) Done() bool { return c.Age >= c.Ticks() }

// Draw puts the whole sequence up: the card wherever it has got to, its exit, the motes and the
// changes.
func (c Consume) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	face := CardImage(gs, c.Face, c.Style)
	lift := c.liftTicks()

	switch {
	case c.Age < lift:
		t := NewTravel(0, lift)
		t.Age = c.Age
		BlitCard(gs, screen, FlyingTo(c.From, c.Stage, t), c.Face, c.Style)

	case c.Age < c.exitStart():
		// The rune's toast: a rattle and a turn, lit, on the stage.
		drawConsumeToast(screen, face, c.Stage.Min, c.Age-lift, consumeToastTicks())

	default:
		age, ticks := c.Age-c.exitStart(), consumeExitTicks()
		if age < ticks {
			c.drawExit(screen, face, age, ticks)
		}
	}

	// The motes, out of the middle of the stage into the middle of each destination.
	if c.Age >= c.depart() {
		fly := float64(consumeMoteTicks())
		sx, sy := centerOf(c.Stage)
		for i, to := range c.To {
			tx, ty := centerOf(to)
			path := func(t float64) (float64, float64) {
				p := EaseOut(Clamp01(t / fly))
				// A slight arc, so several destinations fan out rather than overlap.
				lift := math.Sin(p*math.Pi) * 60
				return sx + (tx-sx)*p, sy + (ty-sy)*p - lift
			}
			em := c.Motes
			em.Seed = c.Seed + uint32(i)*97
			DrawParticleTrail(screen, em, path, int(fly), c.Age-c.depart())
		}
	}

	for i, m := range c.Changes {
		if i < len(c.To) {
			DrawMorph(gs, screen, c.To[i].Min, m)
		}
	}
}

func (c Consume) drawExit(screen, face *ebiten.Image, age, ticks int) {
	r := c.Stage
	switch c.Exit {
	case ExitCrumble:
		DrawCrumble(screen, face, r.Min, c.Seed, age, ticks)
		CrumbleDust(screen, FirstDrawn(RubbleSprites, []string{"particle-earth-1", "particle-earth-2"}),
			r, c.Seed+3, age, ticks)
	case ExitBurn:
		DrawBurn(screen, face, r.Min, c.Seed, age, ticks)
		BurnEmbers(screen, r, c.Seed+5, age, ticks)
		BurnAsh(screen, FirstDrawn(AshSprites, []string{"spark-earth-1", "spark-earth-2"}),
			r, c.Seed+9, age, ticks)
	case ExitSigil:
		drawSigil(screen, r, age, ticks)
		DrawAbsorb(screen, face, r.Min, c.Seed, age, ticks)
	default:
		DrawAbsorb(screen, face, r.Min, c.Seed, age, ticks)
	}
}

func centerOf(r image.Rectangle) (float64, float64) {
	return float64(r.Min.X+r.Max.X) / 2, float64(r.Min.Y+r.Max.Y) / 2
}

// drawConsumeToast is a card rattling and rocking in place, brightening as it does — the relic
// toast's gesture for a card that is about to be used rather than one paying into a sum.
func drawConsumeToast(screen, face *ebiten.Image, at image.Point, age, ticks int) {
	if face == nil {
		return
	}
	p := Clamp01(float64(age) / float64(ticks))
	w, h := float64(face.Bounds().Dx()), float64(face.Bounds().Dy())
	shift := math.Sin(p*math.Pi*6) * (1 - p) * 7
	angle := math.Sin(p*math.Pi*2) * (1 - p) * 0.105

	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Rotate(angle)
	op.GeoM.Translate(float64(at.X)+w/2+shift, float64(at.Y)+h/2)
	screen.DrawImage(face, op)

	add := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
	add.GeoM = op.GeoM
	add.ColorScale.ScaleAlpha(float32(0.45 * p))
	screen.DrawImage(face, add)
}

// --- absorbing ------------------------------------------------------------------------------

// DrawAbsorb is a card coming apart in light: the dissolve's patches, each flaring as it goes and
// drifting upward, so the card reads as taken up rather than eaten.
func DrawAbsorb(screen, face *ebiten.Image, at image.Point, seed uint32, age, ticks int) {
	if face == nil || ticks <= 0 {
		return
	}
	w, h := face.Bounds().Dx(), face.Bounds().Dy()
	prog := float64(age) / float64(ticks)
	for _, cell := range dissolveCells(seed, w, h) {
		p := cellProgress(cell.Delay, prog)
		if p >= 1 {
			continue
		}
		r := cell.Rect
		sub := face.SubImage(r).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(at.X+r.Min.X), float64(at.Y+r.Min.Y)-p*p*40)
		op.ColorScale.ScaleAlpha(float32(1 - p))
		screen.DrawImage(sub, op)
		if g := 4 * p * (1 - p); g > 0 {
			add := &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter}
			add.GeoM = op.GeoM
			add.ColorScale.ScaleAlpha(float32(g * 0.9))
			screen.DrawImage(sub, add)
		}
	}
}

// RuneSigilArt is the ring a rune turns out from under itself, and the picture standing in for it
// until it is drawn.
const (
	RuneSigilArt      = "rune-sigil"
	runeSigilFallback = "portal-swirl"
)

// drawSigil is the ring under a firing rune: it turns, grows past the card's edges, and fades,
// drawn additively so it is light rather than a hue.
func drawSigil(screen *ebiten.Image, r image.Rectangle, age, ticks int) {
	key := RuneSigilArt
	if ArtImage(key) == nil {
		key = runeSigilFallback
	}
	const size = 256
	img := artMarkImageSized(key, size)
	if img == nil {
		return
	}
	p := Clamp01(float64(age) / float64(ticks))
	cx, cy := centerOf(r)
	scale := (0.7 + 0.8*EaseOut(p)) * float64(r.Dx()) / size * 1.4
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
	op.GeoM.Translate(-size/2, -size/2)
	op.GeoM.Rotate(p * math.Pi * 1.2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(cx, cy)
	op.ColorScale.ScaleAlpha(float32(math.Min(1, 4*p*(1-p)*1.4)))
	screen.DrawImage(img, op)
}
