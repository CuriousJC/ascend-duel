package screens

// The draw pile on a between-fights screen: **the deck panel's opener, drawn as the deck**.
//
// **It replaced a lettered square on 2026-09-06** *(owner's call)*. The combat screen has always
// opened the deck panel by clicking the pile of card backs in the duelist's column, and the shop
// stood a 44px `D` beside its other corner controls to do the same job — a second thing to learn
// for one panel, and the only control in the game whose meaning was a letter.
//
// **It stands where the fight's pile stands** *(owner's call)* — the duelist's column, bottom left,
// with its count on the line the chrome's own squares sit on at the other end. The deck is one
// object the player tracks across a whole run, so it is in one place: a pile that changed corners
// between the duel and the shop would be a thing to find again on every screen.
//
// **It is free functions rather than a scene's** *(owner's call, 2026-09-19)*, because three
// between-fights screens draw it now — the shop, the reward screen and the sealed good. Each owns
// its own DeckToggle and its own click, and what they share is where the pile stands and what it
// looks like. A second copy of this geometry is how one screen's pile comes to sit a few pixels off
// the others.
//
// **The count is what the run owns, not a fraction.** A fight's pile writes `n/total`, because the
// numerator is how far through the shuffle the round is; there is no shuffle here and the panel
// behind it shows every card the run owns, so a fraction would have the same number top and bottom.

import (
	"fmt"
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// deckPileRect is the front card of the pile: what is drawn on top, and what a click is tested
// against once the backs behind it are added.
//
// **It is the fight's own rectangle, not a copy of it.** One expression for where the deck sits is
// what makes "the same place on every screen" true by construction; two that agree today are two
// that drift the first time either is nudged.
func deckPileRect(gs *state.GlobalState) image.Rectangle { return deckStackRect(gs) }

// deckPileBounds is the whole pile including the backs drawn up and to the left of the front card,
// which is what the click is tested against.
func deckPileBounds(gs *state.GlobalState) image.Rectangle {
	r := deckPileRect(gs)
	back := (deckStackDepth - 1) * deckStackStep
	return image.Rect(r.Min.X-back, r.Min.Y-back, r.Max.X, r.Max.Y)
}

// deckPileCountRect is the line under the pile, where the count is written left-aligned with it —
// the combat screen's own arrangement.
func deckPileCountRect(gs *state.GlobalState) image.Rectangle {
	pile := deckPileRect(gs)
	top := pile.Max.Y + deckCaptionGap
	return image.Rect(pile.Min.X, top, pile.Max.X, top+deckCountSize)
}

// drawShopPile draws the backs and the count under them.
//
// **The depth is fixed rather than proportional**, exactly as it is in a fight: a pile that visibly
// thinned would be a nice touch and a lie, and here there is nothing for it to thin against at all.
func drawDeckPile(gs *state.GlobalState, screen *ebiten.Image) {
	if gs.Run == nil {
		return
	}

	spec := deckPileBackSpec(gs)
	front := deckPileRect(gs)

	// Back to front, so the front card is the one on top and the one the click tests.
	for i := deckStackDepth - 1; i >= 0; i-- {
		off := i * deckStackStep
		at := image.Pt(front.Min.X-off, front.Min.Y-off)
		if img := ui.CardImage(gs, spec, cards.Stack); img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(at.X), float64(at.Y))
			screen.DrawImage(img, op)
		}
	}

	count := deckPileCountRect(gs)
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(count.Min.X), float64(count.Min.Y))
	op.ColorScale.ScaleWithColor(ui.GroundInk)
	text.Draw(screen, fmt.Sprintf("%d", gs.Run.Size()),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: deckCountSize}, op)
}

// deckPileBackSpec is the card back this run's duelist carries. **Read off the fighter rather than
// defaulted**, so the pile on this screen is the pile the fight will draw from — the same mark, not
// a generic back that happens to look similar.
func deckPileBackSpec(gs *state.GlobalState) cards.Spec {
	fighter := ui.BuildFighter(gs)
	if fighter == nil {
		return cards.Spec{FaceDown: true}
	}
	mark, _ := cards.ParseBackMark(fighter.CardBack)
	return cards.Spec{FaceDown: true, Back: mark}
}
