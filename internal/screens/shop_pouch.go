package screens

// The pouch: the stones a run is carrying, and the panel the shop offers them back in.
//
// **A stone used to be applied the moment it was owned** *(2026-08-27)*, and that stopped being
// true on 2026-09-02 *(owner's call)*: a rock shower hands over three stones as *consumables*, to
// be spent on a rung or sold. So a run can now be carrying rocks, and there has to be somewhere to
// do something about them.
//
// **It is the shop and not the combat screen**, on one argument: selling is a trade, the shop is
// the only screen that trades, and putting *use* somewhere else would split one row of cards into
// two board pieces answering the same question. A stone raises a rung for the rest of the run, so
// there is nothing urgent about spending one mid-fight.
//
// **It is a panel and not a row, because the screen has no room for one.** The shelf ends at 684
// and the Leave button is centred at 845; a full-size card row wants 224 of the 160 between them.
// The S button is the shop's third corner toggle beside D and C, which is a shape the screen
// already has two of rather than a new kind of thing.
//
// **The tabs are the worn row's confirm gesture.** A stone is armed by a click and two tabs hang
// under it, which is how selling a ring already works — and the reason it works that way applies
// twice over here: spending a stone puts a rung up that cannot be taken back down.

import (
	"fmt"
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	// The button: S, on the square every other corner toggle takes.
	pouchToggleLabel = "S"
	pouchToggleText  = 60

	// Where the stone cards stand inside the panel, as a percentage of its height. **Above
	// centre**, because the two tabs hang below the cards and need the room under them.
	pouchRowPct = 38

	// The panel's own prompt, on the parasite dialog's numbers.
	pouchPromptDrop = 40
	pouchPromptSize = 24

	// The two tabs that hang under an armed stone, side by side. **Narrower than the worn row's
	// single sell tab, because there are two of them** — together they come to about its width.
	pouchTabWidth  = 170
	pouchTabHeight = 30
	pouchTabGap    = 8

	// How far under the card the tabs hang. The shelf's own figure gap.
	pouchTabDrop = 10
)

// pouchAction is what an armed stone's tabs were asked to do, consumed on the next frame for the
// reason the shop's own selling field is: a button's OnClick reaches no global state.
type pouchAction int

const (
	pouchNone pouchAction = iota
	pouchUse
	pouchSell
)

// pouchToggle is the panel behind the S button: which carried stone is armed, and what its tabs
// have been asked to do.
type pouchToggle struct {
	modalToggle

	// armed is the seat in the pouch whose tabs are up, and -1 for none.
	//
	// **A seat rather than a record key**, which is the one place this differs from the worn row's
	// armed field: the pouch may hold two of the same stone and spending one must not be ambiguous
	// about which. It is the argument parasiteToggle.armed is under.
	armed int

	// doing is the tabs' request, consumed on the next frame.
	doing pouchAction

	// use and sell are the two tabs. **Two buttons rather than one moved between two places**,
	// unlike the worn row's single sell tab, because both are up at once whenever a stone is armed.
	use, sell *models.Button
}

// pouchCornerPlace is where the S button stands: one square in from the hands button, sharing the
// bottom line with it and the deck button.
func pouchCornerPlace(gs *state.GlobalState) image.Point {
	right := gs.PctX(100) - modalToggleInset - pileSlotSize - modalToggleGap -
		handsButtonWidth - modalToggleGap
	return image.Pt(right-pileSlotSize/2,
		gs.PctY(100)-modalToggleInset-pileSlotSize/2)
}

// pouchRow is the run's carried stones, resolved, in the order they were acquired.
//
// **A stone the catalogue no longer holds is skipped rather than drawn blank**, which is the belt
// to the braces Carry and Resume already have: a nil record reaching the card renderer is a crash
// where a missing card is a gap.
func pouchRow(gs *state.GlobalState) []session.Stone {
	if gs.Run == nil {
		return nil
	}
	keys := gs.Run.Carried()
	out := make([]session.Stone, 0, len(keys))
	for _, key := range keys {
		if st, ok := session.StoneByKey(key); ok {
			out = append(out, st)
		}
	}
	return out
}

// init wires the button and the two tabs.
func (t *pouchToggle) init() {
	t.modalToggle.init(pouchToggleLabel, pileSlotSize, pileSlotSize, pouchToggleText,
		pouchCornerPlace)
	t.armed, t.doing = -1, pouchNone

	if t.use != nil {
		return
	}
	t.use = models.NewButton(pouchTabWidth, pouchTabHeight, "USE",
		func() { t.doing = pouchUse })
	// **Green, because spending a stone is what the panel is for.** The tab beside it wears the
	// crimson every control on this screen that cannot be taken back wears.
	t.use.BaseColor = color.RGBA{R: 46, G: 150, B: 70, A: 255}
	t.use.TextSize = sellTabTextSize

	t.sell = models.NewButton(pouchTabWidth, pouchTabHeight, "SELL",
		func() { t.doing = pouchSell })
	t.sell.BaseColor = color.RGBA{R: 220, G: 20, B: 60, A: 255}
	t.sell.TextSize = sellTabTextSize
}

// cardRects is where the carried stones stand inside the panel. It reuses the parasite dialog's
// row, which is the one row in the game already written to lay an arbitrary number of cards out
// inside a modal frame.
func (t *pouchToggle) cardRects(gs *state.GlobalState) []image.Rectangle {
	r := modalPanelRect(gs)
	return parasiteCardRects(r, len(pouchRow(gs)), r.Min.Y+r.Dy()*pouchRowPct/100)
}

// tabRects is where the two tabs hang under the armed stone: Use on the left, Sell on the right.
func (t *pouchToggle) tabRects(gs *state.GlobalState) (use, sell image.Rectangle) {
	seats := t.cardRects(gs)
	if t.armed < 0 || t.armed >= len(seats) {
		return image.Rectangle{}, image.Rectangle{}
	}
	seat := seats[t.armed]

	span := pouchTabWidth*2 + pouchTabGap
	left := (seat.Min.X+seat.Max.X)/2 - span/2
	top := seat.Max.Y + pouchTabDrop

	use = image.Rect(left, top, left+pouchTabWidth, top+pouchTabHeight)
	left += pouchTabWidth + pouchTabGap
	sell = image.Rect(left, top, left+pouchTabWidth, top+pouchTabHeight)
	return use, sell
}

// midOf is a rectangle's centre, which is where a button stores itself.
func midOf(r image.Rectangle) (int, int) {
	return (r.Min.X + r.Max.X) / 2, (r.Min.Y + r.Max.Y) / 2
}

// updatePouch runs the button, the panel and the tabs, and reports whether the panel is covering
// the screen.
func (s *ShopScene) updatePouch(gs *state.GlobalState) bool {
	// **The button stands down when the pouch is empty**, which is the rule the parasite bucket's
	// own opener is under: a control lit for something the player cannot do is worse than none.
	s.pouch.block(s.deck.open || s.hands.open || gs.Run == nil || gs.Run.CarryCount() == 0)
	if !s.pouch.open {
		s.pouch.armed, s.pouch.doing = -1, pouchNone
	}

	// **The request is taken before the tabs run again**, so the frame that spent a stone is not
	// also a frame that re-reads a seat the pouch no longer has.
	if s.pouch.doing != pouchNone {
		what, i := s.pouch.doing, s.pouch.armed
		s.pouch.doing, s.pouch.armed = pouchNone, -1
		s.takeStone(gs, what, i)
	}

	// **A stone armed at a seat the pouch no longer has is disarmed**, which is what stops a pair
	// of tabs surviving the sale they asked about.
	if s.pouch.armed >= len(pouchRow(gs)) {
		s.pouch.armed = -1
	}

	if s.pouch.open && s.pouch.armed >= 0 {
		use, sell := s.pouch.tabRects(gs)
		s.pouch.sell.Text = fmt.Sprintf("Sell %d", session.StoneSalePrice)
		s.pouch.use.ScreenX, s.pouch.use.ScreenY = midOf(use)
		s.pouch.sell.ScreenX, s.pouch.sell.ScreenY = midOf(sell)
		systems.UpdateButton(gs, s.pouch.use)
		systems.UpdateButton(gs, s.pouch.sell)
	}

	return s.pouch.modalToggle.update(gs, func(at image.Point, tip *models.Tooltip) {
		s.pouchHover(gs, at, tip)
	})
}

// pouchHover is what the panel does with the cursor: arm a stone, and explain whichever one is
// under it.
//
// **The tabs are not hit-tested here.** They are real buttons and were updated above, so a press
// on one is theirs; this only has to make sure a press on a card arms it.
func (s *ShopScene) pouchHover(gs *state.GlobalState, at image.Point, tip *models.Tooltip) {
	row := pouchRow(gs)
	seats := s.pouch.cardRects(gs)

	for i, seat := range seats {
		if i >= len(row) || !at.In(seat) {
			continue
		}
		tip.Point(seat, row[i].Name, stoneTipLines(gs, row[i]))

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && gs.CursorAllowed() {
			// Clicking the armed stone disarms it, which is the worn row's own gesture.
			if s.pouch.armed == i {
				s.pouch.armed = -1
			} else {
				s.pouch.armed = i
			}
		}
		return
	}
}

// takeStone spends or sells the stone at one seat.
func (s *ShopScene) takeStone(gs *state.GlobalState, what pouchAction, i int) {
	row := pouchRow(gs)
	if i < 0 || i >= len(row) {
		return
	}
	st := row[i]

	switch what {
	case pouchUse:
		if gs.Run.SpendCarried(i) {
			trace.Logf("shop", "used stone %s on rung %s, %d left in the pouch",
				st.Record, st.Hand, gs.Run.CarryCount())
		}
	case pouchSell:
		if gs.Run.SellCarried(i) {
			trace.Logf("shop", "sold stone %s for %d, %d vitae in hand, %d left in the pouch",
				st.Record, session.StoneSalePrice, gs.Run.Vitae(), gs.Run.CarryCount())
		}
	}
	s.tip.Forget()
	saveRun(gs)
}

// drawPouch puts the button and, if it is open, the panel on screen.
func (s *ShopScene) drawPouch(gs *state.GlobalState, screen *ebiten.Image) {
	s.pouch.modalToggle.draw(gs, screen, func() { s.drawPouchPanel(gs, screen) })
}

// drawPouchPanel is the panel: the carried stones, and the two tabs under whichever is armed.
//
// **Every stone is drawn enabled.** Both things that can be done to one are always available —
// there is no price to fail to afford and no rung that can refuse a raise — so a dimmed card here
// would be saying something untrue.
func (s *ShopScene) drawPouchPanel(gs *state.GlobalState, screen *ebiten.Image) {
	r := drawModalFrame(gs, screen, modalHead{})
	row := pouchRow(gs)

	prompt := "Your stones - use one, or sell it"
	if s.pouch.armed >= 0 && s.pouch.armed < len(row) {
		prompt = row[s.pouch.armed].Name + " - use it, or sell it"
	}

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: pouchPromptSize}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(r.Min.X+r.Dx()/2), float64(r.Min.Y+pouchPromptDrop))
	op.PrimaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, prompt, face, op)

	for i, seat := range s.pouch.cardRects(gs) {
		if i >= len(row) {
			break
		}
		drawStoneCard(gs, screen, seat.Min, row[i], true)
	}

	if s.pouch.armed < 0 || s.pouch.armed >= len(row) {
		return
	}
	systems.DrawButton(gs, screen, s.pouch.use)
	systems.DrawButton(gs, screen, s.pouch.sell)
}
