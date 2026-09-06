package screens

// The shop's draw pile: **the deck panel's opener, drawn as the deck**.
//
// **It replaced a lettered square on 2026-09-06** *(owner's call)*. The combat screen has always
// opened the deck panel by clicking the pile of card backs in the duelist's column, and the shop
// stood a 44px `D` beside its other corner controls to do the same job — a second thing to learn
// for one panel, and the only control in the game whose meaning was a letter.
//
// **It is deliberately not part of the corner widget.** The column and the bottom line are shared
// because they are the same controls in the same place on every screen; this pile is *not* where
// the combat screen's pile is, and pretending otherwise would mean moving one of them. What is
// shared is the picture and the panel behind it, which is what the player actually recognises.
//
// **The count is what the run owns, not a fraction.** A fight's pile writes `n/total`, because the
// numerator is how far through the shuffle the round is; there is no shuffle here and the panel
// behind it shows every card the run owns, so a fraction would have the same number top and bottom.

import (
	"fmt"
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// shopPileGap is the air between the pile and the control column it stands to the left of.
const shopPileGap = 20

// shopPileRect is the front card of the pile: what is drawn on top, and what a click is tested
// against once the backs behind it are added.
//
// **Both edges come off things that already exist**, never off a percentage — the rule the combat
// screen's own pile is under, and the reason the corner survived being rearranged twice. The right
// edge is the control column's line, so the pile stands beside HANDS and LEDGER rather than under
// them; the bottom is the fight pile's own, so its count lands on the line the cog stands on.
func shopPileRect(gs *state.GlobalState) image.Rectangle {
	w, h := cards.Stack.Width, cards.Stack.Height

	// **The same bottom rule the fight's pile is under** — the screen's own inset, less the line the
	// count is written on — so the count lands on the bottom line the cog and the stones button
	// stand on. Only the left edge differs between the two screens, which is the point: this is the
	// same object in a different corner, not a second thing that looks like it.
	right := ControlColumnLeft(gs) - shopPileGap
	bottom := gs.ScreenHeight - deckStackBottomInset - deckCountSize - deckCaptionGap

	return image.Rect(right-w, bottom-h, right, bottom)
}

// shopPileBounds is the whole pile including the backs drawn up and to the left of the front card,
// which is what the click is tested against.
func shopPileBounds(gs *state.GlobalState) image.Rectangle {
	r := shopPileRect(gs)
	back := (deckStackDepth - 1) * deckStackStep
	return image.Rect(r.Min.X-back, r.Min.Y-back, r.Max.X, r.Max.Y)
}

// shopPileCountRect is the line under the pile, where the count is written left-aligned with it —
// the combat screen's own arrangement.
func shopPileCountRect(gs *state.GlobalState) image.Rectangle {
	pile := shopPileRect(gs)
	top := pile.Max.Y + deckCaptionGap
	return image.Rect(pile.Min.X, top, pile.Max.X, top+deckCountSize)
}

// updateShopPile is the click that opens and closes the panel. **It runs whether or not the panel
// is up**, which is the combat screen's rule for its own pile: the X is the exit, and this is the
// opener, so a press here while the panel is up must not reach the shelf underneath.
func (s *ShopScene) clickedPile(gs *state.GlobalState, at image.Point) bool {
	if !at.In(shopPileBounds(gs)) {
		return false
	}
	s.deck.toggle()
	s.armed = ""
	s.tip.Forget()
	return true
}

// drawShopPile draws the backs and the count under them.
//
// **The depth is fixed rather than proportional**, exactly as it is in a fight: a pile that visibly
// thinned would be a nice touch and a lie, and here there is nothing for it to thin against at all.
func (s *ShopScene) drawShopPile(gs *state.GlobalState, screen *ebiten.Image) {
	if gs.Run == nil {
		return
	}

	spec := shopBackSpec(gs)
	front := shopPileRect(gs)

	// Back to front, so the front card is the one on top and the one the click tests.
	for i := deckStackDepth - 1; i >= 0; i-- {
		off := i * deckStackStep
		at := image.Pt(front.Min.X-off, front.Min.Y-off)
		if img := cardImage(gs, spec, cards.Stack); img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(at.X), float64(at.Y))
			screen.DrawImage(img, op)
		}
	}

	count := shopPileCountRect(gs)
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(count.Min.X), float64(count.Min.Y))
	op.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, fmt.Sprintf("%d", gs.Run.Size()),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: deckCountSize}, op)
}

// shopBackSpec is the card back this run's duelist carries. **Read off the fighter rather than
// defaulted**, so the pile on this screen is the pile the fight will draw from — the same mark, not
// a generic back that happens to look similar.
func shopBackSpec(gs *state.GlobalState) cards.Spec {
	fighter := buildFighter(gs)
	if fighter == nil {
		return cards.Spec{FaceDown: true}
	}
	mark, _ := cards.ParseBackMark(fighter.CardBack)
	return cards.Spec{FaceDown: true, Back: mark}
}
