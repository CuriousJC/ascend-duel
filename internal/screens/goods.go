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
// **SKIP is the one way out that is not a card** *(owner's call, 2026-09-26)*. The good is already
// paid for, so skipping it is the player's own choice to spend those vitae on nothing — the reward
// screen's LET THEM ESCAPE, on the same terms. It is a labelled button at the bottom of the screen
// rather than an X, because an X means "put this away" everywhere else and this forfeits something.
// The chrome still stands down the way it does on the reward screen.
//
// **It is not a station of a run.** It never touches `session.Phase` — it is reached from the
// shop's shelf and it goes back to the shop — which is the shape Settings, Achievements, Credits
// and the animation gallery already have. Adding it was the two edits that shape costs: an ordinal
// in `state.ActiveScreen` and an entry in the registry in `internal/game`.

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/curiousjc/ascend-duel/internal/models"
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

	// skipButton leaves without taking anything, and skipping is its request, consumed by Update —
	// a button's OnClick reaches no global state, and leaving the screen needs it.
	skipButton *models.Button
	skipping   bool
}

// The skip button: its face, and a size about a third of the shop's LEAVE, because it stands in
// the gutter beside the good's row rather than on a line of its own.
const (
	goodsSkipLabel    = "SKIP"
	goodsSkipWidth    = 140
	goodsSkipHeight   = 52
	goodsSkipTextSize = 28
)

// Init opens whatever the shop paid for.
//
// **Re-entered on every visit**, because each good is its own. A visit with nothing pending is a
// screen reached by a route that should not exist, and it leaves rather than drawing an empty
// table.
func (s *GoodsScene) Init(gs *state.GlobalState) {
	if s.skipButton == nil {
		s.skipButton = models.NewButton(goodsSkipWidth, goodsSkipHeight, goodsSkipLabel,
			func() { s.skipping = true })
		s.skipButton.TextSize = goodsSkipTextSize
		s.skipButton.BaseColor = color.RGBA{R: 120, G: 132, B: 150, A: 255}
	}
	s.skipping = false

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

	// **Only while the cards are up.** Once an essence is spent the change is playing and the
	// choice is made, so there is nothing left to skip.
	if s.stage == goodsPick {
		seat := s.skipSeat(gs)
		s.skipButton.ScreenX, s.skipButton.ScreenY = seat.Min.X+seat.Dx()/2, seat.Min.Y+seat.Dy()/2
		systems.UpdateButton(gs, s.skipButton)
		if s.skipping {
			s.skipping = false
			trace.Logf("goods", "skipped %s, took nothing", s.good.Record)
			s.reset()
			leaveGoods(gs)
			return nil
		}
	}

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
	if s.stage == goodsPick {
		systems.DrawButton(gs, screen, s.skipButton)
	}
	systems.DrawTooltip(gs, screen, &s.tip)

	// Last, and over everything: the panel covers the screen, so nothing of this one may be drawn
	// on top of it.
	s.deck.Draw(gs, screen, ui.OwnedContents(gs))
}

// skipSeat is where SKIP stands: **at the far right, its bottom level with the bottom of the row of
// things in the good** — the essences, the stones or the runes. Its right edge is the build band's,
// so the button lines up with the relic row above it rather than with a margin of its own.
//
// **Beside the good's own row rather than under the cards**, because the vial's second row is a
// hand's worth of cards and its compressing pitch runs close to the screen's middle and bottom.
func (s *GoodsScene) skipSeat(gs *state.GlobalState) image.Rectangle {
	right, bottom := gs.PctX(buildBandRightPct), s.slot(gs, 0).Max.Y
	return image.Rect(right-goodsSkipWidth, bottom-goodsSkipHeight, right, bottom)
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
