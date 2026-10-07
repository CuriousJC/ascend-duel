package screens

// The draw pile on a between-fights screen: **the deck panel's opener, drawn as the deck**.
//
// **It replaced a lettered square on 2026-09-06** *(owner's call)*. The combat screen has always
// opened the deck panel by clicking the pile of card backs in the duelist's column, and the shop
// stood a 44px `D` beside its other corner controls to do the same job — a second thing to learn
// for one panel, and the only control in the game whose meaning was a letter.
//
// **It stands in the fight's column** *(owner's call)* — the duelist's, bottom left, with its count
// on the line the chrome's own squares sit on at the other end. The deck is one object the player
// tracks across a whole run, so it is in one corner: a pile that changed corners between the duel
// and the shop would be a thing to find again on every screen. **It is a hand-size card, in the
// hand row, on every screen** — the fight's own pile, so nothing about the corner changes when the
// duel ends.
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
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The pile's geometry is the combat screen's: deckStackRect, deckStackBounds and deckCountRect. **A
// between-fights screen does not get a pile of its own**, because the bottom third is one layout on
// every screen that draws it — the pile at hand size in the duelist's column, level with where a hand
// is dealt, and its count on the bottom line.

// drawShopPile draws the backs and the count under them.
//
// **The depth is fixed rather than proportional**, exactly as it is in a fight: a pile that visibly
// thinned would be a nice touch and a lie, and here there is nothing for it to thin against at all.
func drawDeckPile(gs *state.GlobalState, screen *ebiten.Image) {
	if gs.Run == nil {
		return
	}

	spec := deckPileBackSpec(gs)
	front := deckStackRect(gs)

	// Back to front, so the front card is the one on top and the one the click tests.
	for i := deckStackDepth - 1; i >= 0; i-- {
		off := i * deckStackStep
		at := image.Pt(front.Min.X-off, front.Min.Y-off)
		if img := ui.CardImage(gs, spec, cards.Hand); img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(at.X), float64(at.Y))
			screen.DrawImage(img, op)
		}
	}

	count := deckCountRect(gs)
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(count.Min.X), float64(count.Min.Y))
	op.ColorScale.ScaleWithColor(ui.GroundInk)
	systems.DrawUI(screen, fmt.Sprintf("%d", gs.Run.Size()),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: deckCountSize}, op)
}

// deckPileBackSpec is the back of this run's duelist's deck. **Read off the fighter rather than
// defaulted**, so the pile on this screen is the pile the fight will draw from — the same back, not
// a generic one that happens to look similar.
func deckPileBackSpec(gs *state.GlobalState) cards.Spec {
	fighter := ui.BuildFighter(gs)
	if fighter == nil {
		return cards.Spec{FaceDown: true}
	}
	return ui.BackSpec(gs, fighter.Deck)
}
