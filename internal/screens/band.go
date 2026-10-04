package screens

// **The band's input, and the same input on every screen that shows it** *(owner's call,
// 2026-10-04)*.
//
// The top third of the combat screen, the reward screen, the shop, a sealed good and the portal is
// one scene: the duelist card, the relics on your fingers, the consumables you carry and the card in
// the opponent's corner. What the player can *do* to it is the same everywhere, which is why it is
// one controller rather than a copy per screen — the deck pile's argument, applied to the row beside
// it. A relic or a consumable that is on the screen can be dragged along its row, clicked to arm a
// sale, and sold through the tab that hangs under it; a consumable on a screen where it can be
// spent hangs a USE tab beside its SELL.
//
// **What differs between screens is a handful of answers, not a second implementation**, and they
// are bandHooks: whether the band is live this frame, what moving or selling a relic also has to
// keep in step on this screen, and whether — and how — a carried card can be used here.

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/curiousjc/ascend-duel/internal/journal"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// bandControls is the band's input state: a drag per row and the one tab.
type bandControls struct {
	relicDrag ui.CardDrag
	heldDrag  ui.CardDrag
	sale      sellTab
}

// bandHooks is what one screen answers about the band, every frame.
type bandHooks struct {
	// live is whether the band takes input at all this frame. A panel covering the screen, a
	// tutorial step holding input elsewhere, or a round playing back makes it false, and both drags
	// cancel and the tab is put away.
	live bool

	// closed is a band that still drags but arms nothing: the combat screen while a round plays
	// back, when the row may be reordered for the next round but nothing may be sold or spent out
	// from under the one already decided.
	closed bool

	// moveRelic is a reorder of the worn row, and nil for the run's own reorder. The combat screen
	// has a live duelist to keep in step as well — see CombatScene.moveRelic.
	moveRelic func(from, to int)

	// sellRelic sells one worn relic, and nil for sellWorn. The shop moves its row as it sells; the
	// combat screen refits the duelist standing in the fight.
	sellRelic func(key string)

	// afterSale runs after any sale, relic or carried card, and may be nil.
	afterSale func()

	// canUse is whether the carried card in one seat can be spent here, now, and nil on a screen
	// where nothing carried is ever spent — which is what decides whether a USE tab hangs at all.
	canUse func(seat int) bool
	use    func(seat int)

	// forget drops the screen's tooltip after a sale, so it does not go on explaining a card that
	// is gone. May be nil.
	forget func()
}

// init puts the band at rest: no drag in progress and nothing armed.
func (b *bandControls) init() {
	b.relicDrag = ui.CardDrag{}
	b.heldDrag = ui.CardDrag{}
	b.sale.initSellTab()
}

// busy is whether a drag is in progress over either row.
func (b *bandControls) busy() bool { return b.relicDrag.Dragging() || b.heldDrag.Dragging() }

// update runs the band for one frame, and reports whether it spent the frame's press — so the
// screen underneath does not also answer it.
func (b *bandControls) update(gs *state.GlobalState, h bandHooks) bool {
	relics := b.relicRow(gs, h)
	held := b.heldRow(gs, h)

	if gs.Run == nil || !h.live || !gs.CursorAllowed() {
		b.relicDrag.Cancel(relics)
		b.heldDrag.Cancel(held)
		b.sale.armed = shopSale{}
		return false
	}

	if h.closed {
		b.sale.armed = shopSale{}
	}
	b.sale.canUse = h.canUse
	if seat, ok := b.sale.takeUse(); ok {
		if h.use != nil {
			h.use(seat)
		}
		return true
	}
	if sale, ok := b.sale.takeSale(); ok {
		b.sell(gs, h, sale)
		return true
	}
	b.sale.updateSellTab(gs)

	pressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	at := image.Pt(gs.MouseX, gs.MouseY)
	b.relicDrag.Update(gs, relics)
	b.heldDrag.Update(gs, held)

	if !pressed {
		return b.busy()
	}
	if b.onBand(gs, at) {
		return true
	}
	// A press anywhere else drops the question.
	b.sale.armed = shopSale{}
	return false
}

// onBand is whether a point is on something the band answers: a worn relic, a carried card, or
// the tab hanging under one.
func (b *bandControls) onBand(gs *state.GlobalState, at image.Point) bool {
	if heldAt(gs, at) >= 0 || b.sale.onTabs(gs, at) {
		return true
	}
	worn := gs.Run.Worn()
	return ui.HoveredSeat(at, len(worn), func(i int) image.Rectangle {
		return relicSlotRect(buildRelicRect(gs), i, len(worn))
	}) >= 0
}

// relicRow is the worn row as the shared drag addresses it. **A click arms a sale**, everywhere.
func (b *bandControls) relicRow(gs *state.GlobalState, h bandHooks) relicRow {
	move := h.moveRelic
	if move == nil {
		move = func(from, to int) { moveWornRelic(gs, from, to) }
	}
	return relicRow{
		rect: buildRelicRect(gs),
		worn: len(wornRelics(gs)),
		click: func(i int) {
			if worn := gs.Run.Worn(); !h.closed && i >= 0 && i < len(worn) {
				b.sale.arm(relicSale(worn[i]))
			}
		},
		move: move,
	}
}

// heldRow is the consumables pane as the shared drag addresses it. **A click arms the card** — and
// with it a SELL tab, plus a USE tab on a screen that can spend it.
func (b *bandControls) heldRow(gs *state.GlobalState, h bandHooks) consumableRow {
	return consumableRow{
		rect:  buildConsumableRect(gs),
		held:  len(heldConsumables(gs)),
		seats: consumableSeats(gs),
		click: func(i int) {
			if !h.closed {
				b.sale.arm(heldSale(i))
			}
		},
		move: func(from, to int) { moveCarried(gs, from, to) },
	}
}

// sell carries out whatever a tab asked for.
func (b *bandControls) sell(gs *state.GlobalState, h bandHooks, a shopSale) {
	switch a.kind {
	case saleRelic:
		if h.sellRelic != nil {
			h.sellRelic(a.key)
		} else {
			sellWorn(gs, a.key)
		}
	case saleHeld:
		sellCarried(gs, a.seat)
	}
	if h.forget != nil {
		h.forget()
	}
	if h.afterSale != nil {
		h.afterSale()
	}
}

// drawHeldGhost draws the carried card riding the cursor, over everything else on the band. The
// relic riding it is the screen's to draw, since the combat screen's counters are the fight's.
func (b *bandControls) drawHeldGhost(gs *state.GlobalState, screen *ebiten.Image, spendable func(session.Consumable) bool) {
	if !b.heldDrag.Dragging() {
		return
	}
	held := heldConsumables(gs)
	if b.heldDrag.Origin() >= len(held) {
		return
	}
	c := held[b.heldDrag.Origin()]
	drawConsumableCard(gs, screen, b.heldDrag.At(gs), c, canSpend(spendable, c), true)
}

// heldSkip is the seat a dragged carried card left, which the pane draws empty.
func (b *bandControls) heldSkip(i int) bool {
	return b.heldDrag.Dragging() && i == b.heldDrag.Origin()
}

// sellWorn takes a worn relic off the run, pays its tier's figure and journals it — the sale every
// screen makes, before whatever its own screen has to move because of it.
func sellWorn(gs *state.GlobalState, key string) bool {
	if gs.Run == nil || !gs.Run.Sell(key) {
		return false
	}
	gs.Journal.Write(journal.Record{
		Kind:   journal.KindSell,
		Key:    key,
		Amount: session.SellValue(key),
	})
	trace.Logf("band", "sold %s for %d, %d vitae in hand, wearing %d",
		key, session.SellValue(key), gs.Run.Vitae(), len(gs.Run.Worn()))
	saveRun(gs)
	return true
}

// moveCarried is a reorder of the consumables pane.
//
// **Only the runes reorder, and only among themselves** *(owner's call, 2026-09-19)*. The row is the
// sack then the pouch, so a seat index is not a sack index past the last rune — dragging across the
// join would reorder by a number that means something else. A rune is ordered because a rune is
// *aimed* and the player reads the row left to right; a stone names its own rung and has nothing to
// be before or after.
func moveCarried(gs *state.GlobalState, from, to int) {
	if gs.Run == nil {
		return
	}
	runes := len(heldRunes(gs))
	if from >= runes || to >= runes {
		return
	}
	if gs.Run.MoveRune(from, to) {
		saveRun(gs)
	}
}

// wornSeat is where one worn relic is sitting in the band's relic row, by key, and whether it is
// worn at all.
func wornSeat(gs *state.GlobalState, key string) (image.Rectangle, bool) {
	worn := gs.Run.Worn()
	for i, k := range worn {
		if k == key {
			return relicSlotRect(buildRelicRect(gs), i, len(worn)), true
		}
	}
	return image.Rectangle{}, false
}

// heldSeat is where one carried card is drawn in the band's consumables pane, and the rectangle it
// is clicked in.
func heldSeat(gs *state.GlobalState, i int) image.Rectangle {
	return consumableSlotRect(buildConsumableRect(gs), i, consumableSeats(gs))
}

// heldAt is which carried card a press landed on, or -1.
//
// **ui.HoveredSeat, like every row**, so the click lands on the card the tooltip just described:
// an over-full pane packs its cards into overlapping seats and the last drawn is on top.
func heldAt(gs *state.GlobalState, at image.Point) int {
	return ui.HoveredSeat(at, len(heldConsumables(gs)), func(i int) image.Rectangle {
		return heldSeat(gs, i)
	})
}
