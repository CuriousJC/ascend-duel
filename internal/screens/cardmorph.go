package screens

// **One card becoming another, in front of the player.**
//
// A worm eats a card's element and hands back a different card. A parasite overwrites one. A relic
// will turn every ice card in the hand to fire. In all three the deck has changed, and the change
// used to happen between two frames: the card the player was looking at was simply not the card
// that was there next. That is the same failure "cards fly; they never appear" was written against,
// one axis over — a card that changes without being seen to change has to be re-read to find out
// what happened, instead of having been watched happening.
//
// # What it is, and what it deliberately is not
//
// A morph is a **transition between two finished faces**, and it knows nothing about where it is on
// screen or what caused it. The caller owns the rectangle, exactly as it owns a `travel`'s
// endpoints — which is what lets the post-battle screen run one in the middle of the table and the
// combat screen run several in the hand without either learning about the other.
//
// It is **not** a `cards.Mark`. A mark is the card's situation and the card is still itself
// afterwards; a morph ends with a different card, so there is no settled picture for
// `internal/cards` to bake. What that package owns here is the pattern the face comes apart in —
// see cards/dissolve.go — for the reason it owns the crack geometry: one derivation, so a card
// always comes apart the same way.
//
// # Three shapes, and the presence of a face is what picks one
//
// `morphInto` is a card replaced by another. `morphAway` is a card eaten with nothing to follow it.
// `morphIn` is a card arriving out of nothing. There is no style enum, because what a morph does is
// entirely decided by which of the two faces it was handed — an enum would be a second way of
// saying the same thing, and a way for the two to disagree.
//
// # It may never change an outcome
//
// Like every other clock on every screen. The deck is altered by whoever raised the morph, before
// or after it runs; the morph is a picture of a decision already taken. See postbattle.go's
// `applyNow`, which still holds the real change until the stage is over.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// The morph's clock, in the game's own beats so it slows down and speeds up with everything else —
// see `beat` in clock.go.
var (
	// morphWaitTicks is the pause between a card arriving somewhere and starting to change.
	//
	// **The change needs a card to change *from*** *(owner's call, 2026-09-08)*. A dissolve that
	// began on the frame a flight landed would put the one thing worth watching on top of a card
	// the player had not finished reading, so the card lands, is still for a beat, and then goes.
	morphWaitTicks = beat(1, 2)

	// morphTicks is the dissolve itself. Long enough that the ragged edge is seen travelling across
	// the face rather than flickering over it, short enough that it is one beat of a screen rather
	// than a scene of its own.
	morphTicks = beat(5, 4)
)

// morphWindow is how much of the whole clock one square of the grid spends changing.
//
// **Every square takes the same share**, so a late square is as quick as an early one rather than
// being crushed into whatever is left — the same rule `crackProgress` follows for a break. The
// value is what decides how soft the travelling edge is: at 1 every square is changing at once and
// the dissolve is a plain cross-fade, and near 0 it is a hard boundary sweeping across.
const morphWindow = 0.34

// morphGlow is how brightly a square flares as it turns over. **Light rather than a colour**: the
// hue wheel is full (see CLAUDE.md) and a burning edge in a fifth hue would be claiming one. The
// square is simply drawn a second time additively at its peak, so it brightens in its own colours
// and settles back — the same trick `BevelEdges` plays, one step further.
const morphGlow = 0.5

// morph is one face turning into another. **Both faces are optional and which ones are present is
// what the morph does** — see the file comment.
type morph struct {
	before, after       cards.Spec
	hasBefore, hasAfter bool

	// st is the card size both faces are drawn at. A morph is drawn at the card's own size; the
	// caller places it, it does not scale it.
	st cards.Style

	// seed is the pattern this card comes apart in, derived from its name so it is the same every
	// time this card is changed.
	seed uint32

	// t is the wait and then the dissolve: the pause is the travel's delay, so one clock is the
	// whole thing and there is no second counter to keep in step.
	t travel
}

// morphInto is a card replaced by another one.
func morphInto(before, after cards.Spec, st cards.Style) morph {
	return morph{
		before: before, after: after,
		hasBefore: true, hasAfter: true,
		st:   st,
		seed: cards.MarkSeed(before.Name),
		t:    newTravel(morphWaitTicks, morphTicks),
	}
}

// morphAway is a card eaten, with nothing behind it.
func morphAway(before cards.Spec, st cards.Style) morph {
	return morph{
		before: before, hasBefore: true,
		st:   st,
		seed: cards.MarkSeed(before.Name),
		t:    newTravel(morphWaitTicks, morphTicks),
	}
}

// morphIn is a card arriving out of nothing — a copy that was not in the deck a moment ago.
func morphIn(after cards.Spec, st cards.Style) morph {
	return morph{
		after: after, hasAfter: true,
		st:   st,
		seed: cards.MarkSeed(after.Name),
		t:    newTravel(morphWaitTicks, morphTicks),
	}
}

func (m *morph) tick()        { m.t.tick() }
func (m morph) done() bool    { return m.t.done() }
func (m morph) waiting() bool { return m.t.waiting() }

// drawMorph puts a morph on screen with the card's top-left corner at `at`.
//
// **Before it starts it is the old card and after it finishes it is the new one**, drawn plainly
// through the same blit every other card goes through — so the frame the transition begins on and
// the frame it hands over on are both ordinary cards, and there is no seam at either end.
func drawMorph(gs *state.GlobalState, screen *ebiten.Image, at image.Point, m morph) {
	switch {
	case m.waiting():
		if m.hasBefore {
			blitCard(gs, screen, at, m.before, m.st)
		}
		return
	case m.done():
		if m.hasAfter {
			blitCard(gs, screen, at, m.after, m.st)
		}
		return
	}

	var from, to *ebiten.Image
	if m.hasBefore {
		from = cardImage(gs, m.before, m.st)
	}
	if m.hasAfter {
		to = cardImage(gs, m.after, m.st)
	}
	if from == nil && to == nil {
		return
	}

	prog := m.t.progress()
	for _, cell := range dissolveCells(m.seed, m.st.Width, m.st.Height) {
		p := cellProgress(cell.Delay, prog)
		drawCell(screen, from, at, cell.Rect, 1-p)
		drawCell(screen, to, at, cell.Rect, p)

		// The flare rides whichever face is on screen: the arriving one where there is one, and the
		// departing one where the card is simply being eaten.
		if g := morphGlow * 4 * p * (1 - p); g > 0 {
			lit, alpha := to, p
			if lit == nil {
				lit, alpha = from, 1-p
			}
			addCell(screen, lit, at, cell.Rect, g*alpha)
		}
	}
}

// drawCell blits one square of a card face at the alpha its own progress has reached. A nil face —
// a morph with only one side, or a card the renderer could not build — draws nothing, the same
// silence blitCard keeps.
func drawCell(screen *ebiten.Image, face *ebiten.Image, at image.Point, cell image.Rectangle, alpha float64) {
	if face == nil || alpha <= 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(at.X+cell.Min.X), float64(at.Y+cell.Min.Y))
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	screen.DrawImage(face.SubImage(cell).(*ebiten.Image), op)
}

// addCell draws one square again additively, which is what makes a turning square flare.
func addCell(screen *ebiten.Image, face *ebiten.Image, at image.Point, cell image.Rectangle, alpha float64) {
	if face == nil || alpha <= 0 {
		return
	}
	op := &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter}
	op.GeoM.Translate(float64(at.X+cell.Min.X), float64(at.Y+cell.Min.Y))
	op.ColorScale.ScaleAlpha(float32(clamp01(alpha)))
	screen.DrawImage(face.SubImage(cell).(*ebiten.Image), op)
}

// cellProgress is how far one square is through its own change, given when it goes and how far the
// whole morph has run.
//
// The clock is stretched by one window so that every square is untouched at 0 and finished at 1,
// rather than the first squares going before the morph has started and the last ones still going
// after it has ended.
func cellProgress(delay, prog float64) float64 {
	return clamp01((prog*(1+morphWindow) - delay) / morphWindow)
}

// dissolveKey is a pattern for one card at one size.
type dissolveKey struct {
	seed uint32
	w, h int
}

// dissolveCache holds the grids, because working one out is a few hundred squares and a lattice and
// this is asked for every frame of every morph. The pattern is a pure function of its key — the
// same property that makes the card faces themselves cacheable — and the map stays tiny: one entry
// per card that has ever been morphed, per size.
var dissolveCache = map[dissolveKey][]cards.DissolveCell{}

func dissolveCells(seed uint32, w, h int) []cards.DissolveCell {
	key := dissolveKey{seed: seed, w: w, h: h}
	if cells, ok := dissolveCache[key]; ok {
		return cells
	}
	cells := cards.DissolveCells(w, h, seed)
	dissolveCache[key] = cells
	return cells
}
