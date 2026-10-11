package ui

// cardfx.go â€” a card destroyed in front of you: a stone crumbling, a cantrip burning.
//
// **Both are the dissolve's grid put to a different use.** A morph cuts the face into
// `cards.DissolveCell` squares and fades each one on its own delay; here the same squares, off the
// same cached lattice, are the pieces the card breaks into. A crumble lets each square go and fall
// as a piece of the card's own picture; a burn takes them bottom row first, charring each one dark
// and flaring it in the fire ramp before it goes. Nothing is authored for the card itself â€” the
// pieces *are* the card â€” and the particles thrown with them are an `Emitter` like any other.
//
// **Derived, never rolled**, and a picture rather than a rule: like the dissolve, a card always
// comes apart the same way, and nothing waits on it unless a caller decides it should.

import (
	"image"
	"image/color"
	"math"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// Placeholder sprite sets until the effect particles are drawn: the keys the art will arrive under
// are below, and a key with no file draws nothing, so these stand in.
var (
	// AshSprites float up off a burning card.
	AshSprites = []string{"ash-1", "ash-2", "ash-3"}
	// RubbleSprites fall off a crumbling one.
	RubbleSprites = []string{"rubble-1", "rubble-2", "rubble-3"}
	// RuneMoteSprites travel from a rune to the cards it changes.
	RuneMoteSprites = []string{"rune-mote-1", "rune-mote-2", "rune-mote-3"}
)

// FirstDrawn is the first of these keys that has art, and the fallback keys when none does â€” so an
// effect wired to sprites nobody has drawn yet still shows something.
func FirstDrawn(keys, fallback []string) []string {
	for _, k := range keys {
		if CardArtExists(k) {
			return keys
		}
	}
	return fallback
}

// standInTint is the tint a stand-in set is drawn in to pass for the set it stands in for, and no
// tint at all once the real set is being drawn — its color is in the art.
func standInTint(sprites, real []string, tint color.RGBA) color.RGBA {
	if len(sprites) > 0 && len(real) > 0 && sprites[0] == real[0] {
		return color.RGBA{}
	}
	return tint
}

// CardArtExists reports whether an asset key names a file.
func CardArtExists(key string) bool { return ArtImage(key) != nil }

// ArtImage is a particle-sized picture for a key, nil if there is none.
func ArtImage(key string) *ebiten.Image {
	return artMarkImageSized(key, particleArtSize)
}

func artMarkImageSized(key string, size int) *ebiten.Image {
	return systems.ArtMarkImage(key, size, size)
}

// --- crumbling ------------------------------------------------------------------------------

// The crumble's shape, as shares of its clock.
const (
	// crumbleShakeShare is the tremor before anything falls.
	crumbleShakeShare = 0.22
	// crumbleSpreadShare is how much of the clock the pieces take to all let go.
	crumbleSpreadShare = 0.45
	// crumbleGravity is pixels per tick per tick.
	crumbleGravity = 0.32
)

// DrawCrumble is a card breaking apart where it stands: a tremor, then its pieces letting go in
// patches and falling, turning and darkening as they go. `face` is the card's finished image, `at`
// its top-left, `age` how far through `ticks` it is.
func DrawCrumble(screen, face *ebiten.Image, at image.Point, seed uint32, age, ticks int) {
	if face == nil || ticks <= 0 {
		return
	}
	w, h := face.Bounds().Dx(), face.Bounds().Dy()
	t := float64(age)
	T := float64(ticks)

	// The tremor grows until the first piece goes, then the card is too broken to shake.
	shakeEnd := crumbleShakeShare * T
	var sx float64
	if t < shakeEnd+crumbleSpreadShare*T*0.3 {
		amp := 3 * math.Min(1, t/shakeEnd)
		sx = math.Sin(t*1.9) * amp
	}

	cx, cy := float64(w)/2, float64(h)/2
	for i, cell := range dissolveCells(seed, w, h) {
		release := shakeEnd + cell.Delay*crumbleSpreadShare*T
		r := cell.Rect
		px := float64(at.X+r.Min.X) + sx
		py := float64(at.Y + r.Min.Y)
		sub := face.SubImage(r).(*ebiten.Image)

		if t < release {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(px, py)
			screen.DrawImage(sub, op)
			continue
		}

		a := t - release
		h1 := hash01(seed, i, 21)
		h2 := hash01(seed, i, 22)
		// Thrown a little outward from the card's middle, then falling.
		ox := (float64(r.Min.X)+float64(r.Dx())/2-cx)/cx*1.2 + (h1-0.5)*1.4
		oy := (float64(r.Min.Y)+float64(r.Dy())/2-cy)/cy*0.6 - 1.2*h2
		x := px + ox*a
		y := py + oy*a + 0.5*crumbleGravity*a*a
		life := (1 - crumbleShakeShare - crumbleSpreadShare) * T
		k := Clamp01(a / life)
		if k >= 1 {
			continue
		}

		hw, hh := float64(r.Dx())/2, float64(r.Dy())/2
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Translate(-hw, -hh)
		op.GeoM.Rotate((h1 - 0.5) * 0.3 * a)
		op.GeoM.Scale(1-0.4*k, 1-0.4*k)
		op.GeoM.Translate(x+hw, y+hh)
		dim := float32(1 - 0.45*k)
		op.ColorScale.Scale(dim, dim, dim, 1)
		op.ColorScale.ScaleAlpha(float32(1 - k*k))
		screen.DrawImage(sub, op)
	}
}

// CrumbleDust is the dust thrown off a crumbling card, as a field: born across the card while the
// pieces let go, falling with the earth's physics.
func CrumbleDust(screen *ebiten.Image, sprites []string, rect image.Rectangle, seed uint32, age, ticks int) {
	T := float64(ticks)
	start := crumbleShakeShare * T
	until := int(start + crumbleSpreadShare*T)
	if age < int(start) {
		return
	}
	em := Emitter{Element: cards.Earth, Sprites: sprites, Rate: 0.9, Life: 40, Size: 18, Seed: seed,
		Tint: standInTint(sprites, RubbleSprites, color.RGBA{R: 150, G: 150, B: 150, A: 255})}
	DrawParticleField(screen, em, until-int(start), age-int(start), func(i int, born float64) (x, y, dx, dy float64) {
		x = float64(rect.Min.X) + hash01(seed, i, 31)*float64(rect.Dx())
		y = float64(rect.Min.Y) + (0.3+0.7*hash01(seed, i, 32))*float64(rect.Dy())
		return x, y, hash01(seed, i, 33)*2 - 1, 0
	})
}

// --- burning --------------------------------------------------------------------------------

// The burn's shape.
const (
	// burnSpreadShare is how much of the clock the edge takes to climb the card.
	burnSpreadShare = 0.8
	// burnRagged is how much of a square's moment comes off the lattice rather than its height:
	// zero is a straight line rising, one is the morph's patches.
	burnRagged = 0.22
)

// burnCharTicks is how long a square chars and glows before it goes.
func burnCharTicks() float64 { return float64(Beat(1, 2)) }

// burnAt is when one square of the card goes: mostly its height, bottom first, plus enough of the
// lattice to make the edge ragged.
func burnAt(cell cards.DissolveCell, h int, T float64) float64 {
	fromBottom := 1 - (float64(cell.Rect.Min.Y)+float64(cell.Rect.Dy())/2)/float64(h)
	d := (1-burnRagged)*fromBottom + burnRagged*cell.Delay
	return burnCharTicks() + d*burnSpreadShare*T
}

// burnGlow is the fire ramp's `light`, which a charring square is flared in.
var burnGlow = color.RGBA{R: 0xFF, G: 0xB2, B: 0x4A, A: 0xFF}

// DrawBurn is a card burning up from the bottom: the edge climbs, each square charring dark and
// flaring orange as it reaches it, then gone.
func DrawBurn(screen, face *ebiten.Image, at image.Point, seed uint32, age, ticks int) {
	if face == nil || ticks <= 0 {
		return
	}
	w, h := face.Bounds().Dx(), face.Bounds().Dy()
	t, T := float64(age), float64(ticks)

	for _, cell := range dissolveCells(seed, w, h) {
		b := burnAt(cell, h, T)
		if t >= b {
			continue
		}
		r := cell.Rect
		sub := face.SubImage(r).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(at.X+r.Min.X), float64(at.Y+r.Min.Y))

		k := Clamp01((t - (b - burnCharTicks())) / burnCharTicks())
		if k > 0 {
			op.ColorScale.Scale(float32(1-0.75*k), float32(1-0.82*k), float32(1-0.9*k), 1)
		}
		screen.DrawImage(sub, op)

		if g := 4 * k * (1 - k); g > 0 {
			add := &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter}
			add.GeoM = op.GeoM
			add.ColorScale.ScaleWithColor(burnGlow)
			add.ColorScale.ScaleAlpha(float32(g))
			screen.DrawImage(sub, add)
		}
	}
}

// burnFront is the height of the burning edge at tick t, from the card's top.
func burnFront(h int, t, T float64) float64 {
	p := Clamp01((t - burnCharTicks()) / (burnSpreadShare * T))
	return float64(h) * (1 - p)
}

// BurnEmbers is the fire thrown off the climbing edge, and BurnAsh the ash that floats off it.
func BurnEmbers(screen *ebiten.Image, rect image.Rectangle, seed uint32, age, ticks int) {
	T := float64(ticks)
	until := int(burnCharTicks() + burnSpreadShare*T)
	em := Emitter{Element: cards.Fire, Set: ParticleMixed, Rate: 1.1, Life: 36, Size: 20, Seed: seed}
	DrawParticleField(screen, em, until, age, func(i int, born float64) (x, y, dx, dy float64) {
		x = float64(rect.Min.X) + hash01(seed, i, 41)*float64(rect.Dx())
		y = float64(rect.Min.Y) + burnFront(rect.Dy(), born, T)
		return x, y, hash01(seed, i, 42)*0.6 - 0.3, -1
	})
}

func BurnAsh(screen *ebiten.Image, sprites []string, rect image.Rectangle, seed uint32, age, ticks int) {
	T := float64(ticks)
	until := int(burnCharTicks() + burnSpreadShare*T)
	em := Emitter{Element: cards.Fire, Sprites: sprites, Rate: 0.5, Life: 60, Size: 16, Seed: seed + 7,
		Tint: standInTint(sprites, AshSprites, color.RGBA{R: 70, G: 66, B: 62, A: 255})}
	DrawParticleField(screen, em, until, age, func(i int, born float64) (x, y, dx, dy float64) {
		x = float64(rect.Min.X) + hash01(seed, i, 51)*float64(rect.Dx())
		y = float64(rect.Min.Y) + burnFront(rect.Dy(), born, T)
		return x, y, hash01(seed, i, 52)*1.0 - 0.5, -0.5
	})
}
