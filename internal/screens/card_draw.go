package screens

import (
	"image"

	"github.com/curiousjc/ascend-duel/data"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// **Putting a card on the screen.** Every scene does it, so it is not the combat screen's.
//
// The split with card_art.go is that this file *blits* and that one *specs*: everything here
// takes a finished `cards.Spec` (or builds one immediately) and puts the picture somewhere,
// and everything there answers "what does a card of this kind look like". Rendering a spec is
// far too slow to do per frame, which is why the cache lives on the spec side and this side is
// nothing but a translate and a DrawImage.
//
// These were four separate near-identical functions in four files before 2026-08-21 — one on
// the hand row, one on the flight path, one on the post-battle screen, one in the art bridge.
// They are wrappers over blitCard now, which is what stops a fifth screen writing a fifth.

// blitCard draws one card's picture with its top-left corner at `at`. Everything else in this
// file goes through it.
//
// A nil image is a card the renderer could not build, and it is silently skipped rather than
// drawn as a hole: the cache logs its own failures, and a missing picture must not take the
// frame with it.
func blitCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point, spec cards.Spec, st cards.Style) {
	img := cardImage(gs, spec, st)
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	screen.DrawImage(img, op)
}

// drawCard draws a card out of a hand or a pile: the run's own card, priced and stateful.
//
// **The pairing is passed rather than derived** — see `held`. What a card costs and what its
// damage figure reads as are both facts about who is holding it, and an enemy's queued card must not
// be drawn through the player's rings.
func drawCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point, st cards.Style,
	c actionCard, h held, enabled, selected bool) {

	blitCard(gs, screen, at, cardSpec(c, h, enabled, selected), st)
}

// drawSpecCard draws a card that is not out of the deck — a prize, a ring, a worm — at hand
// size. The caller has already said what it looks like.
func drawSpecCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point, spec cards.Spec) {
	blitCard(gs, screen, at, spec, cards.Hand)
}

// drawRingCard draws a ring in the card format: the pink border, artwork across the face, and
// neither a cost nor a form. Both rows that hold rings go through it — the combat screen's worn
// row and the shop's shelf — which is what keeps a ring one picture rather than two.
func drawRingCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point,
	r data.RingData, enabled bool) {

	blitCard(gs, screen, at, ringSpec(gs, r, enabled), cards.RingStyle)
}

// drawWormCard draws a worm as the card it is offered as.
func drawWormCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point,
	w session.Worm, enabled bool) {

	blitCard(gs, screen, at, wormSpec(w, enabled), cards.Hand)
}

// drawFlyingCard draws a card mid-journey, under whatever transform the flight has worked out
// — so it takes a GeoM rather than a point, and filters linearly because a card being scaled
// or rotated between two seats is the one time a card is not on a whole pixel.
func drawFlyingCard(gs *state.GlobalState, screen *ebiten.Image, spec cards.Spec, st cards.Style, geo ebiten.GeoM) {
	img := cardImage(gs, spec, st)
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM = geo
	screen.DrawImage(img, op)
}

// drawEmptySeat outlines a place a card is not.
//
// **The exception to "cards fly, they never appear"** is an absence: a card that has been taken
// or removed has nothing to fly, so the seat it would have landed in is drawn empty rather than
// left blank. A blank gap reads as a layout fault; an outlined one reads as a hole where a card
// was.
func drawEmptySeat(screen *ebiten.Image, at image.Rectangle) {
	vector.StrokeRect(screen, float32(at.Min.X), float32(at.Min.Y),
		float32(at.Dx()), float32(at.Dy()), 3, groundInk, false)
}
