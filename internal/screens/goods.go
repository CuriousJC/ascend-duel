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
// screen's SKIP, on the same terms. It is a labelled button at the bottom of the screen
// rather than an X, because an X means "put this away" everywhere else and this forfeits something.
//
// **The bottom corner is the reward screen's**: the cog and the ledger are the frame's and stand up
// here as they do there, and the hands button stands beside the ledger — so a stone taken from a bag
// sends its dust to the ladder it raised, exactly as one spent from the pouch does.
//
// **It is not a station of a run.** It never touches `session.Phase` — it is reached from the
// shop's shelf and it goes back to the shop — which is the shape Settings, Achievements, Credits
// and the animation gallery already have. Adding it was the two edits that shape costs: an ordinal
// in `state.ActiveScreen` and an entry in the registry in `internal/game`.

import (
	"image"

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

	// hands is the hand ladder's panel, the shop's own button in the shop's own slot: a bag of rocks
	// is bought against the ladder, and a stone taken from one sends its dust here.
	hands ui.HandsToggle

	// skipButton leaves without taking anything, and skipping is its request, consumed by Update —
	// a button's OnClick reaches no global state, and leaving the screen needs it.
	skipButton *models.Button
	skipping   bool

	// band is the top third's input, the same on every screen that shows it: the worn row and the
	// consumables pane drag, and a click on either arms the sale. A sack opened over a full pane is
	// the case it matters most for here — the cards are held back until a carried one is sold.
	band bandControls
}

// The skip button: its face, and a size about a third of the shop's LEAVE.
const (
	goodsSkipLabel    = "SKIP"
	goodsSkipWidth    = 140
	goodsSkipHeight   = ui.ButtonSmall
	goodsSkipTextSize = 28
)

// goodsSkipGap is how far under the good's row SKIP stands.
const goodsSkipGap = 32

// goodsTypeDrop is how much lower than the reward screen's the good's title and hint sit, so the
// type stands off the band above it.
const goodsTypeDrop = 15

// goodsTypeLiftPct is how much of the gap between the title and a one-row good's cards the title
// and the hint are moved down by *(owner's call, 2026-10-04)*, so the type reads as the row's own
// heading rather than as the band's caption.
const goodsTypeLiftPct = 40

// goodsTitleTop and goodsHintTop are where the good's two lines are written: the reward screen's
// places, dropped by goodsTypeDrop.
func goodsTitleTop(gs *state.GlobalState) int { return offerTitleTop(gs) + goodsTypeDrop }
func goodsHintTop(gs *state.GlobalState) int  { return offerHintTop(gs) + goodsTypeDrop }

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
		s.skipButton.BaseColor = ui.ButtonJade
	}
	s.skipping = false
	s.band.init()

	s.deck.InitAsPile()
	s.hands.InitInColumn(func(gs *state.GlobalState) image.Point {
		return ControlColumnSlotCenter(gs, SlotHands)
	})

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
	s.deck.Block(s.hands.IsOpen())
	s.hands.Block(s.deck.IsOpen())
	if s.deck.Update(gs, ui.OwnedContents(gs)) {
		return nil
	}
	if s.hands.Update(gs) {
		return nil
	}

	// **The band runs before the cards**, so a press on a relic, a carried card or a tab is spent
	// there and never reaches the row underneath. Live only while the cards are up: once one is
	// taken there is nothing left to make room for.
	if s.band.update(gs, bandHooks{
		live:   s.stage == goodsPick,
		forget: s.tip.Forget,
	}) {
		return nil
	}

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
	ui.FillScreenBackdrop(gs, screen)
	if gs.Run == nil {
		return
	}

	// **The build is on screen the whole time**, which is the whole reason this is not a dialog. It
	// is drawn in its parts so the pane can keep the seat a carried card is flying into empty until
	// it lands.
	drawBuildCard(gs, screen, gs.Run.Vitae())
	drawGuideCard(gs, screen)
	drawBuildRelics(gs, screen, &s.band.relicDrag)
	drawConsumablePane(gs, screen, buildConsumableRect(gs), nil,
		func(i int) bool { return s.landingSeat(i) || s.band.heldSkip(i) }, s.tip.Showing())
	drawDeckPile(gs, screen)

	heading := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 34}
	small := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: systems.TextSmall}
	line := func(y int, face *text.GoTextFace, msg string) {
		if msg == "" {
			return
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(gs.PctX(50)), float64(y))
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(ui.GroundInk)
		systems.DrawText(screen, msg, face, op)
	}

	// **Under the band rather than at the top of the screen**, the reward screen's rule: the type
	// follows the band the next time the band moves, instead of being absolute pixels written
	// against where it used to end.
	line(goodsTitleTop(gs)+s.typeLift(gs), heading, s.title())
	if s.stage == goodsPick {
		line(goodsHintTop(gs)+s.typeLift(gs), small, s.hint(gs))
	}

	s.drawCards(gs, screen)
	if s.stage == goodsPick {
		systems.DrawButton(gs, screen, s.skipButton)
	}
	drawBandOverlay(gs, screen, &s.band)
	systems.DrawTooltip(gs, screen, &s.tip)

	// Last, and over everything: the panel covers the screen, so nothing of this one may be drawn
	// on top of it.
	s.deck.Draw(gs, screen, ui.OwnedContents(gs))
	s.hands.Draw(gs, screen, ui.OwnedHands(gs))
}

// skipSeat is where SKIP stands: **centered under the good's row**, on the line the title and the
// hint are centered on *(owner's call, 2026-10-04)*.
//
// **The vial is the exception**: its second row — the cards an essence is aimed at — is right under
// the first, so SKIP stays at the far right, its bottom level with the essences and its right edge
// the build band's.
func (s *GoodsScene) skipSeat(gs *state.GlobalState) image.Rectangle {
	row := s.slot(gs, 0)
	if s.good.Contains == session.ContentsEssences {
		right, bottom := buildOpponentRect(gs).Max.X, row.Max.Y
		return image.Rect(right-goodsSkipWidth, bottom-goodsSkipHeight, right, bottom)
	}
	left, top := gs.PctX(50)-goodsSkipWidth/2, row.Max.Y+goodsSkipGap
	return image.Rect(left, top, left+goodsSkipWidth, top+goodsSkipHeight)
}

// typeLift is how far the title and the hint are moved down toward a one-row good's cards. **The
// vial keeps its place**, because its second row leaves no gap above the first to move into.
func (s *GoodsScene) typeLift(gs *state.GlobalState) int {
	if s.good.Contains == session.ContentsEssences || s.count() == 0 {
		return 0
	}
	return (s.slot(gs, 0).Min.Y - goodsTitleTop(gs)) * goodsTypeLiftPct / 100
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
	if !at.In(deckStackBounds(gs)) {
		return false
	}
	s.deck.Toggle()
	s.tip.Forget()
	return true
}

// **The tutorial does not come here** *(2026-09-19)*. The lesson never buys a sealed good, so this
// scene carries no overlay and answers none of tutorial.go's questions — a step that wanted to
// point at a stone would add the three methods along with the step.
