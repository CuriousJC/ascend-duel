package screens

// Selling: a worn relic or a carried card, taken off the top row for vitae.
//
// **One gesture and one tab for both panes.** A click on a worn relic or a carried card arms the
// confirm tab under it, a click on the tab sells, and a click on the armed thing again puts it
// away. Nothing is written under a seat until it is asked: the tab says the price and asks the
// question at once, so a standing figure under every card would be the price said before anyone
// wanted it.
//
// **Only one thing on the row is armed at a time**, so `shopSale` is one value naming either kind
// and the scene holds one of it. What differs between a relic and a carried card is where its seat
// is, what it pays and which rule takes it off the run — the three switches below — and nothing
// else.

import (
	"fmt"
	"image"

	"github.com/curiousjc/ascend-duel/internal/journal"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// saleKind is which pane an armed sale stands in. The zero value is nothing armed.
type saleKind int

const (
	saleNone saleKind = iota
	saleRelic
	saleHeld
)

// shopSale is the thing a confirm tab is hanging under. **Comparable**, so arming the armed thing
// again is an equality test.
type shopSale struct {
	kind saleKind

	// key is the worn relic's record key, for saleRelic. A relic is named by key because the row
	// re-centers and reorders under it, and a key survives both.
	key string

	// seat is the carried card's seat in the consumables pane, for saleHeld. **A seat rather than
	// a key**, because the pane may hold two of one record and a sale must say which.
	seat int
}

func relicSale(key string) shopSale { return shopSale{kind: saleRelic, key: key} }
func heldSale(seat int) shopSale    { return shopSale{kind: saleHeld, seat: seat} }

// any is whether something is armed.
func (a shopSale) any() bool { return a.kind != saleNone }

// sellTab is the sale itself: what is armed, the tabs hung under it, and the request each raises.
// **Every screen that shows the band holds one**, through bandControls, so there is one gesture and
// one tab wherever a relic or a carried card is sold.
//
// **A carried card on a screen that can spend it hangs two tabs**, USE over SELL *(owner's call,
// 2026-10-04)* — the stone pouch's shape. canUse is that screen's answer for one seat, and nil where
// nothing carried is ever spent; it is set by the band every frame.
type sellTab struct {
	armed      shopSale
	sellButton *models.Button
	selling    bool

	useButton *models.Button
	using     bool
	canUse    func(seat int) bool
}

// initSellTab builds the tabs the first time and puts any question away.
func (s *sellTab) initSellTab() {
	if s.sellButton == nil {
		s.sellButton = models.NewButton(sellTabWidth, sellTabHeight, "", func() { s.selling = true })
		// **The color a control that commits something wears**, and the same red DUEL! takes. A
		// sale is the only thing on the screen that cannot be taken back.
		s.sellButton.BaseColor = ui.ButtonRed
		s.sellButton.TextSize = sellTabTextSize
	}
	if s.useButton == nil {
		s.useButton = models.NewButton(sellTabWidth, sellTabHeight, "USE", func() { s.using = true })
		s.useButton.BaseColor = ui.ButtonGray
		s.useButton.TextSize = sellTabTextSize
	}
	s.armed, s.selling, s.using = shopSale{}, false, false
}

// arm puts the question under one thing, or takes it away again if that thing is already asking
// it.
//
// **Pulled out of `click` so it can be tested**: the press needs a cursor and a window, and what
// is worth pinning is that arming changes nothing about the run.
func (s *sellTab) arm(a shopSale) {
	if s.armed == a {
		s.armed = shopSale{}
		return
	}
	s.armed = a
}

// takeSale is the sale the tab asked for, once, and clears the question.
func (s *sellTab) takeSale() (shopSale, bool) {
	if !s.selling {
		return shopSale{}, false
	}
	a := s.armed
	s.selling, s.armed = false, shopSale{}
	return a, a.any()
}

// takeUse is the carried seat the USE tab asked to spend, once, and clears the question.
func (s *sellTab) takeUse() (int, bool) {
	if !s.using {
		return 0, false
	}
	a := s.armed
	s.using, s.armed = false, shopSale{}
	if a.kind != saleHeld || s.canUse == nil || !s.canUse(a.seat) {
		return 0, false
	}
	return a.seat, true
}

// showsUse is whether the armed thing hangs a USE tab: a carried card, on a screen that spends one.
func (s *sellTab) showsUse() bool { return s.armed.kind == saleHeld && s.canUse != nil }

// armedSeat is where the armed thing is sitting, and whether it is still there.
func (s *sellTab) armedSeat(gs *state.GlobalState) (image.Rectangle, bool) {
	switch s.armed.kind {
	case saleRelic:
		return wornSeat(gs, s.armed.key)
	case saleHeld:
		if s.armed.seat < 0 || s.armed.seat >= len(heldConsumables(gs)) {
			return image.Rectangle{}, false
		}
		return heldSeat(gs, s.armed.seat), true
	}
	return image.Rectangle{}, false
}

// salePrice is what selling the armed thing pays. **Asked of the run's rules**, never worked out
// here.
func (s *sellTab) salePrice() int {
	if s.armed.kind == saleHeld {
		return session.ConsumableSalePrice
	}
	return session.SellValue(s.armed.key)
}

// updateSellTab positions the tabs under whatever is armed and runs them.
//
// **It disarms a thing that is no longer there**, which is what stops a tab surviving the sale it
// asked about — or a scenario arriving with a key the run does not hold.
func (s *sellTab) updateSellTab(gs *state.GlobalState) {
	if !s.armed.any() {
		return
	}
	if _, ok := s.armedSeat(gs); !ok {
		s.armed = shopSale{}
		return
	}

	s.sellButton.Text = fmt.Sprintf("SELL FOR %d?", s.salePrice())
	place(s.sellButton, s.sellTabRect(gs))
	systems.UpdateButton(gs, s.sellButton)

	if s.showsUse() {
		place(s.useButton, s.useTabRect(gs))
		ui.SetEnabled(s.useButton, s.canUse(s.armed.seat))
		systems.UpdateButton(gs, s.useButton)
	}
}

// place centers a button on a rectangle.
func place(b *models.Button, r image.Rectangle) {
	b.ScreenX, b.ScreenY = (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2
}

// tabTop is where the first tab hangs: under the armed card, centered on it.
func (s *sellTab) tabTop(gs *state.GlobalState) (image.Rectangle, bool) {
	seat, ok := s.armedSeat(gs)
	if !ok {
		return image.Rectangle{}, false
	}
	left := (seat.Min.X+seat.Max.X)/2 - sellTabWidth/2
	top := seat.Max.Y + shopFigureGap
	return image.Rect(left, top, left+sellTabWidth, top+sellTabHeight), true
}

// useTabRect is the USE tab, first under the card when it hangs at all.
func (s *sellTab) useTabRect(gs *state.GlobalState) image.Rectangle {
	r, ok := s.tabTop(gs)
	if !ok || !s.showsUse() {
		return image.Rectangle{}
	}
	return r
}

// sellTabRect is where the SELL tab hangs: under the armed card, or under its USE tab. One
// rectangle, drawn in and hit-tested against.
func (s *sellTab) sellTabRect(gs *state.GlobalState) image.Rectangle {
	r, ok := s.tabTop(gs)
	if !ok {
		return image.Rectangle{}
	}
	if s.showsUse() {
		r = r.Add(image.Pt(0, sellTabHeight+shopFigureGap))
	}
	return r
}

// onTabs is whether a point is on either tab.
func (s *sellTab) onTabs(gs *state.GlobalState, at image.Point) bool {
	if !s.armed.any() {
		return false
	}
	return at.In(s.sellTabRect(gs)) || at.In(s.useTabRect(gs))
}

// drawTab draws the tabs under whatever is armed.
func (s *sellTab) drawTab(gs *state.GlobalState, screen *ebiten.Image) {
	if _, ok := s.armedSeat(gs); !ok {
		return
	}
	if s.showsUse() {
		systems.DrawButton(gs, screen, s.useButton)
	}
	systems.DrawButton(gs, screen, s.sellButton)
}

// sellCarried sells the carried card at one seat of the consumables pane, journals it, and reports
// whether it went. **The one sale both screens make** — the shop's and the goods screen's, where a
// full pane is emptied to make room for what a sack holds.
func sellCarried(gs *state.GlobalState, i int) bool {
	held := heldConsumables(gs)
	if gs.Run == nil || i < 0 || i >= len(held) {
		return false
	}
	c := held[i]
	if !gs.Run.SellConsumable(c) {
		return false
	}

	gs.Journal.Write(journal.Record{
		Kind:   journal.KindSell,
		Key:    consumableKey(c),
		Action: consumableWord(c.Kind),
		Seat:   i,
		Amount: session.ConsumableSalePrice,
	})
	trace.Logf("shop", "sold %s %s for %d, %d vitae in hand, carrying %d",
		consumableWord(c.Kind), consumableKey(c), session.ConsumableSalePrice,
		gs.Run.Vitae(), gs.Run.ConsumableCount())
	return true
}

// consumableKey is the record key of a carried thing, whichever catalog it is from.
func consumableKey(c session.Consumable) string {
	switch c.Kind {
	case session.ConsumableStone:
		return c.Stone.Record
	case session.ConsumableEssence:
		return c.Essence.Record
	case session.ConsumableCantrip:
		return c.Cantrip.Record
	default:
		return c.Rune.Record
	}
}

// consumableWord is the journal's verb for a kind, written as a sale's Action so the line says what
// went over the counter.
func consumableWord(k session.ConsumableKind) string {
	switch k {
	case session.ConsumableStone:
		return journal.KindStone
	case session.ConsumableEssence:
		return journal.KindEssence
	case session.ConsumableCantrip:
		return journal.KindCantrip
	default:
		return journal.KindRune
	}
}
