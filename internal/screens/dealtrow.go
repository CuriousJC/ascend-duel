package screens

// **A row of your own cards, dealt out of the deck, is the same row wherever it is dealt** *(owner's
// call, 2026-10-04)*.
//
// The reward screen deals one to aim an essence at and the vial deals one under its essences, and
// both are what the combat screen's hand is: a handful of the player's cards, overlapping, read the
// same way. So they sort the same way — the three tabs beside the row, the one preference every
// screen shares — and they **reorder by drag** the same way, a card picked up and put down where the
// cursor lets go. A sort re-applies on a deal and on a tab, never every frame, so a card the player
// moved stays where they put it until they ask for an order back; pressing the latched tab is that.
//
// **The row is deck indices, and a selection is row slots**, so a sort and a drag move the
// selection with the cards — the picked card is the same card after the row has moved under it.

import (
	"image"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// dealtRow is one row of dealt cards: what is in it, what is picked, and the controls over it.
type dealtRow struct {
	offer    []int
	selected []int

	mode    ui.HandSort
	sortDue bool
	tabs    *ui.SortTabs
	slides  []ui.CardSlide
	drag    ui.CardDrag

	// top is the row's top edge on this screen. The row is centered on the screen and cut at the
	// hand's own compressing pitch, so this is the one fact a screen supplies.
	top func(gs *state.GlobalState) int
}

// initRow builds the tabs the first time, loads the shared sort preference and empties the row.
func (r *dealtRow) initRow(gs *state.GlobalState, top func(gs *state.GlobalState) int) {
	r.top = top
	if r.tabs == nil {
		r.tabs = ui.NewSortTabs(r.tabRect, func(mode ui.HandSort) { r.mode, r.sortDue = mode, true })
	}
	r.mode = ui.HandSortOf(gs)
	r.offer, r.selected, r.slides = nil, nil, nil
	r.drag = ui.CardDrag{}
}

// deal puts a fresh row up, arranged in the current order and with nothing picked. **Arranged on
// the spot** rather than on the next frame, so a row is never drawn in the order it was dealt.
func (r *dealtRow) deal(gs *state.GlobalState, offer []int) {
	r.offer, r.selected = offer, nil
	r.drag = ui.CardDrag{}
	r.sort(gs)
	r.slides = nil
}

// rowOf is the row for a stated number of cards — what a slide needs, since a row of eight is not
// centered where a row of seven is.
func (r *dealtRow) rowOf(gs *state.GlobalState, n int) image.Rectangle {
	width := (n-1)*handPitch(gs, n) + cardWidth
	left := gs.PctX(50) - width/2
	top := r.top(gs)
	return image.Rect(left, top, left+width, top+cardHeight)
}

// seat is one card's place in a row of `count`, unlifted.
func (r *dealtRow) seat(gs *state.GlobalState, i, count int) image.Point {
	row := r.rowOf(gs, count)
	return image.Pt(row.Min.X+i*handPitch(gs, count), row.Min.Y)
}

// slot is where one card is drawn and clicked. **A picked card lifts out of the row**, the hand's
// own gesture, and the lift is here rather than in the drawing so the part standing proud of the
// row is clickable.
func (r *dealtRow) slot(gs *state.GlobalState, i int) image.Rectangle {
	if i < 0 || i >= len(r.offer) {
		return image.Rectangle{}
	}
	at := r.seat(gs, i, len(r.offer))
	at.Y -= r.lift(i)
	return image.Rect(at.X, at.Y, at.X+cardWidth, at.Y+cardHeight)
}

func (r *dealtRow) lift(i int) int {
	if r.isSelected(i) {
		return offerSelectedNudge
	}
	return 0
}

// isSelected is whether a row slot is one of the picked cards.
func (r *dealtRow) isSelected(i int) bool {
	for _, sel := range r.selected {
		if sel == i {
			return true
		}
	}
	return false
}

// tabRect is the i'th sort tab: one block, no air in it, its top on the row's top, hung off the
// row's right edge — the combat screen's rule for its own block.
func (r *dealtRow) tabRect(gs *state.GlobalState, i int) image.Rectangle {
	row := r.rowOf(gs, len(r.offer))
	left := row.Max.X + ui.SortColumnGap
	top := row.Min.Y + i*ui.ControlButtonHeight
	return image.Rect(left, top, left+ui.ControlColumnWidth(), top+ui.ControlButtonHeight)
}

// update runs the row for one frame: the tabs, any sort they asked for, the drag, and the slides.
// click is a press on a card that never traveled — the screen's own selection.
func (r *dealtRow) update(gs *state.GlobalState, live bool, click func(i int)) {
	r.tabs.Place(gs)
	r.tabs.Update(gs, live)
	ui.SetHandSort(gs, r.mode)
	if r.sortDue {
		r.sortDue = false
		r.sort(gs)
	}

	row := dealtDragRow{r: r, gs: gs, click: click}
	if live && gs.CursorAllowed() {
		r.drag.Update(gs, row)
	} else {
		r.drag.Cancel(row)
	}
	r.slides = ui.Advance(r.slides)
}

// sort arranges the row in the current mode and sends every card that moved sliding to its new
// place, carrying the selection with the cards — so a picked card is lifted at both ends of its
// slide.
//
// **Stable, over the order the row is already in**, so two identical cards keep their places and
// pressing the same tab twice cannot shuffle them.
func (r *dealtRow) sort(gs *state.GlobalState) {
	if gs.Run == nil {
		return
	}
	card := func(deckIndex int) (combat.Card, bool) { return gs.Run.Card(deckIndex) }
	worn := gs.Run.WornRelics()

	order := make([]int, len(r.offer))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, aok := card(r.offer[order[i]])
		b, bok := card(r.offer[order[j]])
		if !aok || !bok {
			return r.offer[order[i]] < r.offer[order[j]]
		}
		return ui.HandLess(r.mode, a, b, worn)
	})
	r.permute(order)

	r.slides = ui.SlidesFor(r.slides, order, func(i int) combat.Card {
		c, _ := card(r.offer[i])
		return c
	}, r.lift, r.lift)
}

// newSlot is where the card that was at `from` sits after a permutation.
func (r *dealtRow) newSlot(order []int, from int) int {
	for to, f := range order {
		if f == from {
			return to
		}
	}
	return from
}

// permute applies a permutation — for each new position, the slot that card came from — to the
// row and to the selection.
func (r *dealtRow) permute(order []int) {
	offer := make([]int, len(r.offer))
	for to, from := range order {
		offer[to] = r.offer[from]
	}
	selected := make([]int, 0, len(r.selected))
	for _, sel := range r.selected {
		selected = append(selected, r.newSlot(order, sel))
	}
	r.offer, r.selected = offer, selected
}

// move is a drag put down: the card at `from` now sits at `to`, everything between shuffled along.
func (r *dealtRow) move(from, to int) {
	if from == to || from < 0 || to < 0 || from >= len(r.offer) || to >= len(r.offer) {
		return
	}
	order := make([]int, 0, len(r.offer))
	for i := range r.offer {
		if i != from {
			order = append(order, i)
		}
	}
	order = append(order[:to], append([]int{from}, order[to:]...)...)
	r.permute(order)
}

// draw puts the row up: every card in its seat, the ones sliding on the mover, the one riding the
// cursor last, and the tabs.
func (r *dealtRow) draw(gs *state.GlobalState, screen *ebiten.Image) {
	if gs.Run == nil {
		return
	}
	for i, deckIndex := range r.offer {
		card, ok := gs.Run.Card(deckIndex)
		if !ok || ui.SlideInto(r.slides, i) || (r.drag.Dragging() && i == r.drag.Origin()) {
			continue
		}
		ui.DrawFloatingCard(gs, screen, r.slot(gs, i).Min, i,
			ui.CardSpec(card, ui.HeldByRun(gs, card), true, r.isSelected(i)), cards.Hand)
	}
	ui.DrawCardSlides(gs, screen, r.slides,
		func(gs *state.GlobalState, i, count int) image.Point {
			p := r.seat(gs, i, count)
			p.Y -= r.lift(i)
			return p
		},
		func(sl ui.CardSlide) cards.Spec {
			return ui.CardSpec(sl.Card, ui.HeldByRun(gs, sl.Card), true, false)
		})
	if r.drag.Dragging() {
		if card, ok := gs.Run.Card(r.offer[r.drag.Origin()]); ok {
			ui.DrawCard(gs, screen, r.drag.At(gs), cards.Hand, card, ui.HeldByRun(gs, card),
				true, r.isSelected(r.drag.Origin()))
		}
	}
	r.tabs.Draw(gs, screen)
}

// dealtDragRow is the row as the shared drag addresses it.
type dealtDragRow struct {
	r     *dealtRow
	gs    *state.GlobalState
	click func(i int)
}

func (d dealtDragRow) RowLen() int { return len(d.r.offer) }

func (d dealtDragRow) RowSlot(gs *state.GlobalState, i int) image.Rectangle { return d.r.slot(gs, i) }

// RowZone is the row itself, grown by a selected card's lift: a drop off the row puts the card back.
func (d dealtDragRow) RowZone(gs *state.GlobalState) image.Rectangle {
	row := d.r.rowOf(gs, len(d.r.offer))
	row.Min.Y -= offerSelectedNudge
	return row
}

// RowDropIndex is which seat the cursor is over, in pitches from the row's left edge and from the
// middle of a step — the hand's arithmetic — clamped to a seat that exists.
func (d dealtDragRow) RowDropIndex(gs *state.GlobalState) int {
	n := len(d.r.offer)
	if n < 2 {
		return 0
	}
	pitch := handPitch(gs, n)
	if pitch < 1 {
		return 0
	}
	idx := (gs.MouseX - d.r.seat(gs, 0, n).X + pitch/2) / pitch
	return min(max(idx, 0), n-1)
}

// RowLift is empty: the card stays in the list while it rides the cursor, and its seat is drawn
// empty — the relic row's shape.
func (d dealtDragRow) RowLift(int) {}

func (d dealtDragRow) RowReturn(from, to int) { d.r.move(from, to) }

func (d dealtDragRow) RowClick(i int) {
	if d.click != nil {
		d.click(i)
	}
}
