package screens

// A sealed good, opened: **a screen, not a dialog** *(owner's call, 2026-09-19)*.
//
// What is on it is a decision about the build — which stone raises which rung, which rune is worth
// carrying, which essence lands on which cards — and every one of those is judged against the
// relics you are wearing and the deck you own. A panel covering the screen to ask the question was
// covering the answer: the relic row went under it, and the deck was two screens away.
//
// **So the build band is up, the draw pile is up, and the deck panel opens over it** exactly as it
// does on the shop and in a fight. What was a modal frame is now the screen's own ground, and the
// cards sit where the reward screen's do.
//
// **There is still no way out but taking a card.** The dialog had no X because the good is already
// paid for, and that survives the move: nothing on this screen leaves it, and the chrome stands
// down the way it does on the reward screen. A good bought and abandoned would be five vitae spent
// on nothing.
//
// **It is not a station of a run.** It never touches `session.Phase` — it is reached from the
// shop's shelf and it goes back to the shop — which is the shape Settings, Achievements, Credits
// and the animation gallery already have. Adding it was the two edits that shape costs: an ordinal
// in `state.ActiveScreen` and an entry in the registry in `internal/game`.

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// openGoods pays nothing and decides nothing: it points the game at the screen that opens whichever
// good the shop has already bought.
//
// **It is here rather than in `internal/actions`** *(2026-09-19)*. That package holds the explicit
// list of screens which may be opened from anywhere, and the list being explicit is what stops a
// *run* screen being entered without its phase being set. This one is reached from exactly one
// place and returns to exactly one place, so the door belongs beside the screen it opens.
func openGoods(gs *state.GlobalState, key string) {
	gs.PendingGood = key
	gs.ActiveScreen = state.Goods
	gs.NewScreen = true
}

// GoodsScene draws one opened sealed good.
//
// **The dialog's own state is embedded rather than rewritten.** What a good holds, what a click on
// one of its cards does and how an essence's alteration is shown were all already written — see
// shop_goods.go — and none of it was about being a modal. What this file adds is the screen around
// it.
type GoodsScene struct {
	goods

	// deck is the panel over the whole deck, opened by clicking the pile. **The same widget the
	// shop and the fight use**, so a player who has learned to click the pile has learned it here.
	deck ui.DeckToggle

	// relicDrag is the press in progress over the worn relic row. **The row is reorderable here
	// like everywhere else** — worn order is a rule, and a screen where a relic is being chosen is
	// a screen where the order it fires in is worth thinking about.
	relicDrag ui.CardDrag
}

// Init opens whatever the shop paid for.
//
// **Re-entered on every visit**, because each good is its own. A visit with nothing pending is a
// screen reached by a route that should not exist, and it leaves rather than drawing an empty
// table.
func (s *GoodsScene) Init(gs *state.GlobalState) {
	s.deck.InitAsPile()
	s.relicDrag = ui.CardDrag{}

	good, ok := session.GoodByKey(gs.PendingGood)
	gs.PendingGood = ""
	if !ok || gs.Run == nil {
		s.reset()
		return
	}
	s.open(gs, good)
}

// Update runs the good, and leaves for the shop once a card has been taken.
func (s *GoodsScene) Update(gs *state.GlobalState) error {
	if gs.Run == nil {
		leaveGoods(gs)
		return nil
	}

	// **The deck panel runs first and swallows the frame**, the shop's own order: while it is up
	// the cards underneath are dead, so a press meant for the panel cannot reach the row behind it.
	if s.deck.Update(gs, ui.OwnedContents(gs)) {
		return nil
	}

	s.updateRelicRow(gs)

	if !s.update(gs, func(at image.Point) bool { return s.clickedPile(gs, at) }) {
		// Nothing is open any more: the card was taken and whatever it did has finished playing.
		leaveGoods(gs)
		return nil
	}

	return nil
}

// Draw puts the screen up: the build the good is being judged against, then the good itself.
func (s *GoodsScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	ui.FillGround(screen)
	if gs.Run == nil {
		return
	}

	// **The build is on screen the whole time**, which is the whole reason this is not a dialog.
	drawBuildBand(gs, screen, gs.Run.Vitae(), &s.relicDrag, s.tip.Showing())
	drawDeckPile(gs, screen)

	heading := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 34}
	small := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 18}
	line := func(y int, face *text.GoTextFace, msg string) {
		if msg == "" {
			return
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(gs.PctX(50)), float64(y))
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(ui.GroundInk)
		text.Draw(screen, msg, face, op)
	}

	// **Under the band rather than at the top of the screen**, the reward screen's rule: the type
	// follows the band the next time the band moves, instead of being absolute pixels written
	// against where it used to end.
	line(offerTitleTop(gs), heading, s.title())
	if s.stage != goodsShowing {
		line(offerHintTop(gs), small, s.hint(gs))
	}

	s.drawCards(gs, screen)
	systems.DrawTooltip(gs, screen, &s.tip)

	// Last, and over everything: the panel covers the screen, so nothing of this one may be drawn
	// on top of it.
	s.deck.Draw(gs, screen, ui.OwnedContents(gs))
}

// updateRelicRow runs the drag over the worn row in the build band.
//
// **A click on a relic does nothing here**, as on the reward screen and in a fight: this screen's
// clicks belong to the good it has opened.
func (s *GoodsScene) updateRelicRow(gs *state.GlobalState) {
	row := buildRelicRow(gs, nil)
	if !gs.CursorAllowed() {
		s.relicDrag.Cancel(row)
		return
	}
	s.relicDrag.Update(gs, row)
}

// leaveGoods puts the player back on the shop they bought the good from.
//
// **The shop rather than `advance`**, because opening a good is not a station of the run: the run
// is standing in the shop the whole time, and it is standing there when the good is finished with.
func leaveGoods(gs *state.GlobalState) {
	trace.Logf("goods", "closed, back to the shop")
	gs.ActiveScreen = state.Shop
	gs.NewScreen = true
}

// clickedPile is the click that opens and closes the deck panel — the shop's own, over the shared
// pile. **It runs whether or not the panel is up**: the X is the exit and this is the opener, so a
// press here while the panel is up must not reach the cards underneath.
func (s *GoodsScene) clickedPile(gs *state.GlobalState, at image.Point) bool {
	if !at.In(deckPileBounds(gs)) {
		return false
	}
	s.deck.Toggle()
	s.tip.Forget()
	return true
}

// **The tutorial does not come here** *(2026-09-19)*. The lesson never buys a sealed good, so this
// scene carries no overlay and answers none of tutorial.go's questions — a step that wanted to
// point at a stone would add the three methods along with the step.
