package screens

import (
	"image"

	"github.com/curiousjc/ascend-duel/data"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
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

// drawMarkedCard is drawCard with something drawn over the finished face — a break, today, and
// whatever else joins cards.Mark.
//
// **A separate entry rather than a sixth parameter on drawCard**, because a mark is the rare case:
// every row in the game draws unmarked cards and exactly one draws marked ones. The signature that
// nearly every call site uses should not carry a field nearly every call site passes zero for.
func drawMarkedCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point, st cards.Style,
	c actionCard, h held, enabled, selected bool, mark cards.Mark) {

	spec := cardSpec(c, h, enabled, selected)
	spec.Mark = mark
	blitCard(gs, screen, at, spec, st)
}

// drawSpecCard draws a card that is not out of the deck — a prize, a ring, a worm — at hand
// size. The caller has already said what it looks like.
func drawSpecCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point, spec cards.Spec) {
	blitCard(gs, screen, at, spec, cards.Hand)
}

// drawRingCard draws a ring in the card format: the pink border, artwork across the face, and
// neither a cost nor a form. Every row that holds rings goes through it — the combat screen's worn
// row, the build band and the shop's two — which is what keeps a ring one picture rather than four.
//
// **`counter` is the badge in the corner and an empty one draws nothing**, which is what a ring that
// does not grow passes and what the shop's shelf passes for every ring on it: a ring nobody is
// wearing has no accumulator to show. See ringCounter for what the figure means.
// **`lit` is the toast**: a ring drawn while it is firing into a blow's sum. It reuses the card
// format's selected state — a brighter border — because that is already what "this card is the one
// doing something" looks like everywhere else in the game.
func drawRingCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point,
	r data.RingData, counter string, enabled, lit bool) {

	blitCard(gs, screen, at, ringSpec(gs, r, counter, enabled, lit), cards.RingStyle)
}

// drawWormCard draws a worm as the card it is offered as.
func drawWormCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point,
	w session.Worm, enabled bool) {

	blitCard(gs, screen, at, wormSpec(gs, w, enabled), cards.WormStyle)
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

// drawStoneCard draws a stone as the card it is offered as. Same style as a worm — a picture with
// its text under it — because they are the same kind of thing to a player: one card, taken out of
// a set, that changes the run rather than being played in it.
func drawStoneCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point,
	st session.Stone, enabled bool) {

	blitCard(gs, screen, at, stoneSpec(gs, st, enabled), cards.WormStyle)
}

// drawGoodCard draws one of the shop's two sealed goods.
func drawGoodCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point,
	name, line string, art image.Image, enabled bool) {

	blitCard(gs, screen, at, goodSpec(gs, name, line, art, enabled), cards.WormStyle)
}

// marksFor is what a card in this seat is wearing, given where it is drawn.
//
// **It reads `gs.InputFocus`, which is the list the tutorial's spotlight was handed** — so a card
// the lesson has lit and a card the lesson will let you click are the same card by construction,
// rather than by two pieces of code agreeing. That is the rule the whole tutorial overlay is built
// on, applied to the cards themselves now that they are marked rather than framed.
//
// **The seat has to match a focus rectangle, not merely overlap one.** The hand row overlaps when
// it is full, so a card beside a lit one shares pixels with it; asking for containment would light
// its neighbours. See tutorial.Anchor.NamesCards, which is what puts card seats in the list.
func marksFor(gs *state.GlobalState, seat image.Rectangle) cards.Mark {
	if !gs.InputGated {
		return cards.MarkNone
	}
	for _, r := range gs.InputFocus {
		if r == seat {
			return cards.MarkHighlit
		}
	}
	return cards.MarkNone
}
