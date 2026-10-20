package screens

// **The band's input, and the same input on every screen that shows it** *(owner's call,
// 2026-10-04)*.
//
// The top third of the combat screen, the reward screen, the shop, a sealed good and the portal is
// one scene: the duelist card, the relics on your fingers, the consumables you carry and the card in
// the opponent's corner. What the player can *do* to it is the same everywhere, which is why it is
// one controller rather than a copy per screen — the deck pile's argument, applied to the row beside
// it. A relic or a consumable that is on the screen can be dragged along its row, clicked to arm a
// sale, and sold through the tab that hangs under it. **A consumable hangs USE beside its SELL on
// every screen**, enabled only where it can be spent right now — so a rune aimed at cards shows the
// same two tabs it shows in a fight, with USE dark, rather than a different card with one tab.
//
// **What differs between screens is a handful of answers, not a second implementation**, and they
// are bandHooks: whether the band is live this frame, what moving or selling a relic also has to
// keep in step on this screen, and whether — and how — a carried card can be used here.

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/curiousjc/ascend-duel/internal/cards"
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

	// uses are carried cards spent off the table: lifted to the middle of the screen and used up
	// there, their motes sent to whatever they changed. Nothing waits on them. See useAnywhere.
	uses []ui.Consume

	// morphs are chimeras turning into the rune they copy, in their own seats in the pane. See
	// becomeLastRune.
	morphs []paneMorph
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

	// canUse is whether the carried card in one seat can be spent here, now — which decides whether
	// its USE tab is lit, never whether it hangs. **Nil is the band's own answer, usableAnywhere**:
	// a freestanding card — a stone, or a rune naming no cards — needs nothing on screen to do what
	// it does, so it is usable wherever the band is; everything else wants a fight or a selection,
	// which only the combat screen has. use is how, and nil is useAnywhere.
	canUse func(seat int) bool
	use    func(seat int)

	// forget drops the screen's tooltip after a sale, so it does not go on explaining a card that
	// is gone. May be nil.
	forget func()

	// extraRelics is how many relics the row draws beside the run's own: the cantrip-relics cast this
	// fight, on the combat screen, and zero everywhere else. They are laid out with the run's — the
	// row's pitch is a function of how many cards are in it — and dragged like them.
	extraRelics int

	// relicKeyAt is which of the run's relics stands in one seat of the row, and false for a seat
	// holding something the run does not own and cannot sell. Nil means the row is the run's alone,
	// seat for seat.
	relicKeyAt func(seat int) (string, bool)

	// relicSeat is where one of the run's relics stands in the row, for the sale tab to hang under.
	// Nil means the run's own seat.
	relicSeat func(key string) (image.Rectangle, bool)

	// relicMotion is how far one of the run's relics is displaced from its seat right now — its
	// float, or its weightless drift and turn — so the sale tab moves with it. Nil means the run's
	// own row, seat for seat. See bandMotion.
	relicMotion func(key string) (dx, dy, turn float64)
}

// init puts the band at rest: no drag in progress and nothing armed.
func (b *bandControls) init() {
	b.relicDrag = ui.CardDrag{}
	b.heldDrag = ui.CardDrag{}
	b.uses = nil
	b.morphs = nil
	b.sale.initSellTab()
}

// usableAnywhere is the band's own answer to whether a carried card can be used: a freestanding one
// yes, anywhere — see session.Consumable.Freestanding — and anything else, not off the table.
func usableAnywhere(gs *state.GlobalState, seat int) bool {
	held := heldConsumables(gs)
	return seat >= 0 && seat < len(held) && anywhereSpendable(gs)(held[seat])
}

// anywhereSpendable is usableAnywhere as the pane reads it, so a card off the table is lit exactly
// when its USE tab would be. **A chimera on a run that has spent no rune is dim**, since there is
// nothing for it to become.
func anywhereSpendable(gs *state.GlobalState) func(session.Consumable) bool {
	return func(c session.Consumable) bool {
		if gs.Run == nil || !c.Freestanding() {
			return false
		}
		if c.Kind == session.ConsumableRune && c.Rune.Target == session.RuneChimera {
			_, ok := gs.Run.Echoes(c.Rune)
			return ok
		}
		return true
	}
}

// useAnywhere spends the freestanding card in one seat of the consumables pane.
func (b *bandControls) useAnywhere(gs *state.GlobalState, seat int, h bandHooks) {
	if !usableAnywhere(gs, seat) {
		return
	}
	c := heldConsumables(gs)[seat]
	switch {
	case c.Kind == session.ConsumableStone:
		b.useStone(gs, seat, c)
	case c.Rune.Target == session.RuneChimera:
		b.becomeLastRune(gs, seat, c)
	default:
		b.useRune(gs, seat, c)
	}
	if h.forget != nil {
		h.forget()
	}
	saveRun(gs)
}

// useStone spends a stone onto its rung, for the rest of the run, and crumbles it in the middle of
// the screen.
//
// **Its seat and face are read before it is spent**, since spending takes it out of the pane.
func (b *bandControls) useStone(gs *state.GlobalState, seat int, c session.Consumable) {
	from, _ := b.sale.seatOf(gs, heldSale(seat))
	face := ui.StoneSpec(gs, c.Stone, true)
	if !gs.Run.SpendCarried(c.At) {
		return
	}
	b.uses = append(b.uses, stoneUseFrom(gs, face, from))
	gs.Journal.Write(journal.Record{
		Kind:   journal.KindStone,
		Action: journal.StoneUsed,
		Key:    c.Stone.Record,
		Seat:   c.At,
	})
	trace.Logf("band", "used stone %s on shape %s, %d stones left",
		c.Stone.Record, c.Stone.Shape, gs.Run.CarryCount())
}

// useRune spends a rune that names no cards — a purse filled or a rock shower — and burns its sigil
// in the middle of the screen, the motes going to the purse on the duelist card or to the hands
// button the stones raised.
//
// **The shower draws the stream the combat screen draws**, showerRNG, so a rune spent here rolls
// what it would roll anywhere else in the same run.
func (b *bandControls) useRune(gs *state.GlobalState, seat int, c session.Consumable) {
	from, _ := b.sale.seatOf(gs, heldSale(seat))
	face := ui.RuneSpec(gs, c.Rune, true, false)
	if !gs.Run.ApplyRuneRolling(c.Rune, nil, runeRNG(gs, c.Rune)) {
		return
	}
	gs.Run.Drop(c.At)
	gs.Journal.Write(journal.Record{Kind: journal.KindRune, Key: c.Rune.Record})

	to := buildCardRect(gs)
	if c.Rune.Target == session.RuneStones {
		to = handsTarget(gs)
	}
	b.uses = append(b.uses, runeUseFrom(gs, face, from, to))
	trace.Logf("band", "used rune %s off the table, %d vitae in hand", c.Rune.Record, gs.Run.Vitae())
}

// becomeLastRune turns a chimera into the run's last rune in its own seat, and dissolves it there —
// the combat screen's gesture, on the band. Nothing fires: the rune it becomes waits for a fight.
func (b *bandControls) becomeLastRune(gs *state.GlobalState, seat int, c session.Consumable) {
	before := ui.RuneSpec(gs, c.Rune, true, false)
	if !gs.Run.BecomeLastRune(c.At) {
		return
	}
	gs.Journal.Write(journal.Record{Kind: journal.KindRune, Key: c.Rune.Record})
	if now := heldConsumables(gs); seat < len(now) {
		b.morphs = append(b.morphs, paneMorph{
			seat: seat,
			m:    ui.MorphInto(before, ui.RuneSpec(gs, now[seat].Rune, true, false), cards.EssenceStyle),
		})
	}
	trace.Logf("band", "chimera became %s off the table", gs.Run.LastRune())
}

// tickUses advances every card being used off the table and every chimera turning, and drops the
// finished ones.
func (b *bandControls) tickUses() {
	kept := b.uses[:0]
	for i := range b.uses {
		b.uses[i].Tick()
		if !b.uses[i].Done() {
			kept = append(kept, b.uses[i])
		}
	}
	b.uses = kept

	morphs := b.morphs[:0]
	for i := range b.morphs {
		b.morphs[i].Tick()
		if !b.morphs[i].Done() {
			morphs = append(morphs, b.morphs[i])
		}
	}
	b.morphs = morphs
}

// busy is whether a drag is in progress over either row.
func (b *bandControls) busy() bool { return b.relicDrag.Dragging() || b.heldDrag.Dragging() }

// update runs the band for one frame, and reports whether it spent the frame's press — so the
// screen underneath does not also answer it.
func (b *bandControls) update(gs *state.GlobalState, h bandHooks) bool {
	relics := b.relicRow(gs, h)
	held := b.heldRow(gs, h)
	defer b.sale.tickScroll()
	b.tickUses()

	if h.canUse == nil {
		h.canUse = func(seat int) bool { return usableAnywhere(gs, seat) }
	}
	if h.use == nil {
		h.use = func(seat int) { b.useAnywhere(gs, seat, h) }
	}

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
	b.sale.relicSeat = h.relicSeat
	b.sale.relicMotion = h.relicMotion
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
	if b.onBand(gs, at, h) {
		return true
	}
	// A press anywhere else drops the question.
	b.sale.armed = shopSale{}
	return false
}

// onBand is whether a point is on something the band answers: a worn relic, a carried card, or
// the tab hanging under one.
func (b *bandControls) onBand(gs *state.GlobalState, at image.Point, h bandHooks) bool {
	if heldAt(gs, at) >= 0 || b.sale.onTabs(gs, at) {
		return true
	}
	n := len(gs.Run.Worn()) + h.extraRelics
	return ui.HoveredSeat(at, n, func(i int) image.Rectangle {
		return relicSlotRect(buildRelicRect(gs), i, n)
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
		worn: len(wornRelics(gs)) + h.extraRelics,
		click: func(i int) {
			if h.closed {
				return
			}
			if h.relicKeyAt != nil {
				if key, ok := h.relicKeyAt(i); ok {
					b.sale.arm(relicSale(key))
				}
				return
			}
			if worn := gs.Run.Worn(); i >= 0 && i < len(worn) {
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

// heldSkip is the seat a dragged carried card left, which the pane draws empty, and a seat a
// chimera is turning in, which the band draws itself — see drawBandOverlay.
func (b *bandControls) heldSkip(i int) bool {
	if b.heldDrag.Dragging() && i == b.heldDrag.Origin() {
		return true
	}
	for _, m := range b.morphs {
		if m.seat == i {
			return true
		}
	}
	return false
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
