package screens

import (
	"fmt"
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The action box: one hand of cards laid out in a row across the bottom of the screen.
// Click a card to select it into the round's queue, click it again to take it out, and
// drag a card sideways to move it along the row. Selected cards lift up out of the row,
// so the queue reads as a shape along the top edge of the hand rather than as a second
// list somewhere else.
//
// This is a game widget rather than a UI widget — cards carry an action-point cost and the
// selection validates against a budget — so it lives on CombatScene and is hand-rolled
// rather than reaching for a toolkit. Left click and drag only; there is no right click
// and no keyboard anywhere in this game.
const (
	// The card is a fixed portrait rectangle and nothing inside it may change that.
	// 180x264 is roughly a playing card's proportions, and it is written here as two plain
	// numbers on purpose: cardWidth used to be *derived* from the glyph row, so adding a
	// badge silently widened every card in the game and the layout could not be reasoned
	// about without doing the arithmetic. The contents now fit the card, never the reverse.
	cardWidth  = 180
	cardHeight = 264
	cardGap    = 12

	// The row sits low, with the budget and then the button strip beneath it. handTopPct is
	// the top of an *unselected* card; a selected one rises above it by selectedNudge.
	handTopPct = 59

	// selectedNudge is how far a selected card lifts out of the row. Selection is the only
	// state a card carries, so it gets a whole axis to itself rather than a tint that would
	// have to compete with the affordability dimming. Up rather than right, because the
	// hand is now a horizontal strip and the free axis rotated with it.
	selectedNudge = 26

	dropIndicatorWidth = 4

	// The budget, as offsets *below* the bottom of the row: the AP figure at the row's left
	// edge, the pile counts at its right, and the bar spanning the width between them. The
	// bar is exactly as wide as the hand, so it reads as belonging to the cards rather than
	// floating near them, and it sits under the selection it is reporting on — between the
	// cards and the two buttons that spend them.
	apTextBelow = 10
	apBarBelow  = 36
	apBarHeight = 8

	// The bar is one cell per action point. apBarGap separates them so the cells can be
	// counted at a glance; apBarMinCell is the width below which counting stops working and
	// the bar falls back to an unbroken fill.
	apBarGap     = 4
	apBarMinCell = 3

	// The row never grows past this, whatever the hand holds. Beyond the count that fits at
	// full pitch the cards overlap and the band stays put — see handPitch.
	handBandLeftPct  = 4
	handBandRightPct = 96

	// The card's *interior* — the glyph column, the name and category positions, the
	// corner radius, the cost dashes — no longer lives here. It moved to
	// internal/cards.Hand when card drawing was extracted so a command-line tool could
	// render the same picture; see card_art.go and tools/cardsheet.
	//
	// cardWidth and cardHeight stay, because the *row* is laid out from them — the pitch,
	// the band, the drop indicator and the hit rectangles are all functions of a card's
	// footprint rather than of its contents. They are duplicated from cards.Hand rather
	// than read from it because they are consts and it is a var, and
	// TestCardFootprintMatchesTheRenderer fails the build if the two ever disagree.

	// dragThreshold is how far the cursor has to travel with the button held before a
	// press counts as a drag rather than a click. Without it every click would jitter
	// into a one-pixel reorder and selecting a card would be a coin toss.
	dragThreshold = 4
)

// The card's look — its geometry, colours, rounded corners and cost dashes — lives in
// internal/cards, which draws a card into a plain Go image with no graphics context.
// cards.Hand and cards.Deck are the two styles that used to be cards.Hand and
// deckCardStyle here.
//
// **That split is what lets `go run ./tools/cardsheet` review the art without launching
// the game**, the same reason systems.RenderGlyph is free of Ebitengine. Both this screen
// and that tool call cards.Render and blit the result, so the sheet cannot show something
// the game does not draw. See card_art.go for the caching this screen adds on top.

// paletteCard is one card in the hand: the card itself, and whether it is queued this
// round. Selection is the only thing the hand adds to a card that is not also true of it
// sitting in a pile, which is why the instance is embedded rather than copied out field by
// field — c.action still reads the same everywhere it did before.
type paletteCard struct {
	actionCard
	selected bool
}

// dragState is the press currently in progress. Nil when the button is up.
//
// A press starts inactive and only becomes a drag once the cursor has moved past
// dragThreshold; until then it is still a candidate click. The card leaves the list at
// that moment rather than on release, so the gap closes under the cursor and the drop
// index is measured against the list the card actually lands in.
type dragState struct {
	card        paletteCard
	originIndex int
	active      bool

	pressX, pressY int

	// grabDX/grabDY keep the cursor where it landed on the card, so picking one up does
	// not snap it to the cursor.
	grabDX, grabDY int
}

// planning reports whether the player may edit the queue: only between rounds, and only
// while both duelists are standing.
func (s *CombatScene) planning() bool {
	return s.cursor >= len(s.log) && s.fighter.Alive() && s.enemy.Alive()
}

// selectedCount is how many cards are currently picked.
func (s *CombatScene) selectedCount() int {
	n := 0
	for _, c := range s.hand {
		if c.selected {
			n++
		}
	}
	return n
}

// overBudget reports whether the selection costs more than the fighter can spend. Legal to
// be in — it is how cards are gathered for a discard — but DUEL! will not fire from it.
func (s *CombatScene) overBudget() bool {
	return combat.CostOf(s.fighterActions) > s.fighter.ActionPoints()
}

// syncQueue rebuilds the round's queue from the list: the selected cards, in list order.
// The list is the authority on both membership and order, so this runs after every change
// to either rather than the two being maintained side by side and allowed to disagree.
func (s *CombatScene) syncQueue() {
	s.fighterActions = s.fighterActions[:0]
	for _, c := range s.hand {
		if c.selected {
			s.fighterActions = append(s.fighterActions, c.action)
		}
	}
}

// updateActionBox runs the click and drag lifecycle. Called every tick from Update.
func (s *CombatScene) updateActionBox(gs *state.GlobalState) {
	// A round starting mid-press puts whatever is in hand back rather than letting it
	// land on a list that is no longer editable.
	if !s.planning() {
		s.cancelDrag()
		return
	}

	if s.drag == nil {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			s.beginPress(gs)
		}
		return
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		s.endPress(gs)
		return
	}

	s.promoteToDrag(gs)
}

// beginPress records a press over a card without yet committing to what it means.
//
// Backwards through the hand, because overlapping cards are drawn front to back in index
// order: the last card that covers a point is the one visibly on top of it, and that is the
// one the click has to mean.
func (s *CombatScene) beginPress(gs *state.GlobalState) {
	at := image.Pt(gs.MouseX, gs.MouseY)

	for i := len(s.hand) - 1; i >= 0; i-- {
		slot := s.cardSlot(gs, i)
		if !at.In(slot) {
			continue
		}
		s.drag = &dragState{
			card:        s.hand[i],
			originIndex: i,
			pressX:      gs.MouseX,
			pressY:      gs.MouseY,
			grabDX:      gs.MouseX - slot.Min.X,
			grabDY:      gs.MouseY - slot.Min.Y,
		}
		return
	}
}

// promoteToDrag turns a held press into a drag once the cursor has moved far enough,
// lifting the card out of the list as it does.
func (s *CombatScene) promoteToDrag(gs *state.GlobalState) {
	if s.drag.active {
		return
	}
	if abs(gs.MouseX-s.drag.pressX) < dragThreshold && abs(gs.MouseY-s.drag.pressY) < dragThreshold {
		return
	}

	s.drag.active = true
	s.hand = append(
		append([]paletteCard{}, s.hand[:s.drag.originIndex]...),
		s.hand[s.drag.originIndex+1:]...)

	trace.Logf("drag", "lifted card[%d] %s at %d,%d",
		s.drag.originIndex, cardLabel(s.drag.card.actionCard), gs.MouseX, gs.MouseY)
}

// endPress resolves the press: a drag lands the card at a new position, a click toggles
// whether the card is queued.
func (s *CombatScene) endPress(gs *state.GlobalState) {
	drag := s.drag
	s.drag = nil

	if !drag.active {
		s.toggle(drag.originIndex)
		return
	}

	// Released outside the list, the card goes back where it came from. There is no
	// discard gesture any more — clicking a card off is how it leaves the queue, and that
	// is visible on screen in a way that dragging into empty space never was.
	at := drag.originIndex
	inside := image.Pt(gs.MouseX, gs.MouseY).In(handZone(gs))
	if inside {
		at = s.dropIndex(gs)
	}
	s.insertCard(at, drag.card)

	trace.Logf("drag", "dropped %s at %d,%d inside=%v, index %d -> %d",
		cardLabel(drag.card.actionCard), gs.MouseX, gs.MouseY, inside, drag.originIndex, at)
}

// toggle selects or deselects the card at i. Deselecting always works; selecting is refused
// only when maxSelected cards are already picked, which is the same rule the dimming on
// screen is reporting.
//
// Cost is deliberately not checked. A card the budget will not cover can still be selected,
// because selection is also how a card is chosen for the discard pile — see maxSelected.
func (s *CombatScene) toggle(i int) {
	if i < 0 || i >= len(s.hand) {
		return
	}

	if !s.hand[i].selected && s.selectedCount() >= s.fighter.MaxActions() {
		trace.Logf("input", "select refused: %s, already %d of %d picked",
			cardLabel(s.hand[i].actionCard), s.selectedCount(), s.fighter.MaxActions())
		return
	}

	s.hand[i].selected = !s.hand[i].selected
	s.syncQueue()

	if trace.Enabled() {
		verb := "deselected"
		if s.hand[i].selected {
			verb = "selected"
		}
		trace.Logf("input", "%s card[%d] %s -> %d/%d AP%s  hand %s",
			verb, i, cardLabel(s.hand[i].actionCard),
			combat.CostOf(s.fighterActions), s.fighter.ActionPoints(),
			overSuffix(s), handLabel(s.hand))
	}
}

// overSuffix marks a trace line when the selection has gone past the budget, since that is
// the state DUEL! refuses to fire from and the one worth spotting in a log.
func overSuffix(s *CombatScene) string {
	if over := combat.CostOf(s.fighterActions) - s.fighter.ActionPoints(); over > 0 {
		return fmt.Sprintf(" (+%d OVER)", over)
	}
	return ""
}

// cancelDrag puts any in-flight card back and forgets the press.
func (s *CombatScene) cancelDrag() {
	if s.drag != nil && s.drag.active {
		s.insertCard(s.drag.originIndex, s.drag.card)
	}
	s.drag = nil
}

// insertCard puts a card back into the list at index at, clamped to the list's length.
func (s *CombatScene) insertCard(at int, card paletteCard) {
	if at < 0 {
		at = 0
	}
	if at > len(s.hand) {
		at = len(s.hand)
	}

	s.hand = append(s.hand, paletteCard{})
	copy(s.hand[at+1:], s.hand[at:])
	s.hand[at] = card
	s.syncQueue()
}

// laidOutCount is how many slots the row is drawn to hold. A card in flight still owns
// the slot it left, so the row keeps its width and stays centred while a card is up —
// otherwise the whole hand would slide half a card sideways the moment one was lifted.
func (s *CombatScene) laidOutCount() int {
	n := len(s.hand)
	if s.drag != nil && s.drag.active {
		n++
	}
	return n
}

// handPitch is the horizontal step from one card's left edge to the next's.
//
// A small hand sits side by side at cardWidth+cardGap. Once that would carry the row past
// the band, the pitch shrinks so the row lands exactly on the band's width and the cards
// overlap instead — each one covering the right-hand part of its neighbour, like a hand
// held in one fist. Eight cards already need this: eight at full pitch is 1536 pixels
// against a 1280 screen.
//
// **There is no floor.** The row always fits, so a very large hand compresses until the
// cards are stripes of colour. That is the deliberate choice: a hand that has to be
// scrolled or truncated is worse than one that has to be hovered, and the element colour
// survives compression better than anything else on the card.
func handPitch(gs *state.GlobalState, n int) int {
	full := cardWidth + cardGap
	if n < 2 {
		return full
	}

	band := gs.PctX(handBandRightPct) - gs.PctX(handBandLeftPct)
	if (n-1)*full+cardWidth <= band {
		return full
	}
	return (band - cardWidth) / (n - 1)
}

// handBand is the rectangle a row of n cards occupies, centred on the screen rather than
// pinned to a pane.
//
// This is the single authority on that width. The AP bar spans it, the caption box above
// the hand matches it, and the card slots are cut out of it, so none of them can drift
// apart when the hand size changes.
func handBand(gs *state.GlobalState, n int) image.Rectangle {
	if n < 1 {
		n = 1
	}
	w := (n-1)*handPitch(gs, n) + cardWidth
	left := gs.PctX(50) - w/2
	top := gs.PctY(handTopPct)
	return image.Rect(left, top, left+w, top+cardHeight)
}

// handRowLeft is the x of the first slot.
func (s *CombatScene) handRowLeft(gs *state.GlobalState) int {
	return handBand(gs, s.laidOutCount()).Min.X
}

// dropIndex is the position the cursor is currently pointing between. Measured in pitches
// rather than card widths, so it stays right once the cards overlap; and from the middle of
// a step rather than its left edge, so the card lands where the gap is.
func (s *CombatScene) dropIndex(gs *state.GlobalState) int {
	left := s.handRowLeft(gs)
	pitch := handPitch(gs, s.laidOutCount())
	idx := (gs.MouseX - left + pitch/2) / pitch

	if idx < 0 {
		idx = 0
	}
	if idx > len(s.hand) {
		idx = len(s.hand)
	}
	return idx
}

// handZone is the band across the bottom holding the row, and the region a drag has to be
// released inside for it to count as a reorder. It spans the full width rather than just
// the cards: the row is centred with nothing beside it, so a drop that is merely wide of
// the last card is obviously still a drop into the hand. It reaches up by selectedNudge
// because a selected card does.
func handZone(gs *state.GlobalState) image.Rectangle {
	top := gs.PctY(handTopPct)
	return image.Rect(0, top-selectedNudge, gs.PctX(100), top+cardHeight)
}

// cardSlot is the rectangle the card at index i occupies. Selected cards are lifted up,
// and hit testing reads the same function the drawing does, so the protruding part of a
// card is clickable rather than merely visible.
//
// Slots overlap once the hand is large. Cards are drawn in index order, so a later card
// covers an earlier one, and beginPress walks the hand backwards to match — otherwise a
// click in the overlap would pick the card underneath the one it looks like it hit.
func (s *CombatScene) cardSlot(gs *state.GlobalState, i int) image.Rectangle {
	x := s.handRowLeft(gs) + i*handPitch(gs, s.laidOutCount())
	y := gs.PctY(handTopPct)
	if i < len(s.hand) && s.hand[i].selected {
		y -= selectedNudge
	}
	return image.Rect(x, y, x+cardWidth, y+cardHeight)
}

// drawHandRow draws the budget header and the row of cards. There is no pane behind them:
// with one row, and selection shown by position, a frame was outlining empty space.
func (s *CombatScene) drawHandRow(gs *state.GlobalState, screen *ebiten.Image) {
	band := handBand(gs, s.laidOutCount())
	left, right := float32(band.Min.X), float32(band.Max.X)
	top := float32(gs.PctY(handTopPct))
	below := top + cardHeight

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 14}

	// The budget runs the width of the hand: the exact figure at one end, the piles at the
	// other, and the bar between them. The piles have no cards on screen to count, which is
	// why they are written out at all.
	spent, budget := combat.CostOf(s.fighterActions), s.fighter.ActionPoints()
	label := fmt.Sprintf("%d/%d AP", spent, budget)
	budgetOp := &text.DrawOptions{}
	budgetOp.GeoM.Translate(float64(left), float64(below+apTextBelow))
	if spent > budget {
		label = fmt.Sprintf("%s  +%d over", label, spent-budget)
		budgetOp.ColorScale.ScaleWithColor(apOverColor)
	}
	text.Draw(screen, label, face, budgetOp)

	pilesOp := &text.DrawOptions{}
	pilesOp.GeoM.Translate(float64(right), float64(below+apTextBelow))
	pilesOp.PrimaryAlign = text.AlignEnd
	text.Draw(screen, fmt.Sprintf("deck %d  ·  discard %d", len(s.deck), len(s.discard)),
		face, pilesOp)

	s.drawAPBar(screen, left, below+apBarBelow, right-left)

	for i, c := range s.hand {
		// Dimmed means "you cannot pick this", which now means the selection is full rather
		// than that the card is unaffordable — an unaffordable card is pickable on purpose,
		// and dimming something you can click would be a lie. What it costs you is reported
		// by the bar going red, after the fact. A selected card never dims: clicking it off
		// has to stay open, and it is the way out of an over-allocation.
		enabled := c.selected || (s.planning() && s.selectedCount() < s.fighter.MaxActions())
		drawCard(gs, screen, s.cardSlot(gs, i).Min, cards.Hand,
			c.actionCard, enabled, c.selected, s.fighter.Str)
	}

	if s.drag == nil || !s.drag.active || !image.Pt(gs.MouseX, gs.MouseY).In(handZone(gs)) {
		return
	}

	// A bar standing where the card will drop, full card height so it reads as a slot rather
	// than as a tick. Straddling the slot's left edge rather than sitting a gap to its left,
	// since once the cards overlap there is no gap to sit in.
	slot := s.cardSlot(gs, s.dropIndex(gs))
	vector.DrawFilledRect(screen,
		float32(slot.Min.X)-dropIndicatorWidth/2, top,
		dropIndicatorWidth, cardHeight,
		playerSwatch, false)
}

// drawAPBar draws the action-point budget as a bar. The numeric line above it stays — the
// bar answers "how much room is left" at a glance, the number answers "exactly".
//
// **The bar rescales rather than overflowing.** Its full width is the budget until the
// selection exceeds it, and the whole spend after that, with a tick left standing where the
// budget ends. So the fill never runs off the end and the tick shows how far past it you
// are, in the same picture, at whatever the overspend happens to be. A fixed-scale bar can
// only pin at 100% and say nothing about by how much.
// **One cell per action point.** Action points are whole numbers spent in ones and twos, and
// a continuous fill made the player read the `3/6 AP` line to find out how many were left —
// which is the small text the bar exists to save them from. Segmented, the count is
// countable: three lit cells and three dark ones says "three left" without a number.
//
// The cells also make the budget boundary draw itself. Where blue meets red *is* the edge of
// what can be afforded, so the white tick that used to mark it is gone — it was pointing at
// something the colours now say on their own.
func (s *CombatScene) drawAPBar(screen *ebiten.Image, left, top, width float32) {
	budget := s.fighter.ActionPoints()
	if budget <= 0 {
		return
	}
	spent := combat.CostOf(s.fighterActions)

	// Over budget the bar grows extra cells rather than overflowing, so the overspend is
	// shown at whatever size it happens to be instead of pinning at full.
	cells := budget
	if spent > cells {
		cells = spent
	}

	empty := systems.ColorAtStrength(apBarColor, 20)
	cellWidth := (width - float32(cells-1)*apBarGap) / float32(cells)

	// A cell narrower than a couple of pixels is a smear rather than a count, which a big
	// enough bonus could produce. Fall back to one unbroken bar at that point: it stops
	// being countable either way, and stripes are the worse of the two.
	if cellWidth < apBarMinCell {
		vector.DrawFilledRect(screen, left, top, width, apBarHeight, empty, false)
		filled := width * float32(min(spent, budget)) / float32(cells)
		vector.DrawFilledRect(screen, left, top, filled, apBarHeight, apBarColor, false)
		if spent > budget {
			over := width * float32(spent-budget) / float32(cells)
			vector.DrawFilledRect(screen, left+filled, top, over, apBarHeight, apOverColor, false)
		}
		return
	}

	for i := 0; i < cells; i++ {
		fill := empty
		switch {
		case i >= spent: // still available
		case i < budget:
			fill = apBarColor
		default:
			fill = apOverColor
		}

		vector.DrawFilledRect(screen,
			left+float32(i)*(cellWidth+apBarGap), top,
			cellWidth, apBarHeight, fill, false)
	}
}

// drawDraggedCard draws the card in flight. Called last so it rides over everything.
func (s *CombatScene) drawDraggedCard(gs *state.GlobalState, screen *ebiten.Image) {
	if s.drag == nil || !s.drag.active {
		return
	}

	at := image.Pt(gs.MouseX-s.drag.grabDX, gs.MouseY-s.drag.grabDY)
	drawCard(gs, screen, at, cards.Hand,
		s.drag.card.actionCard, true, s.drag.card.selected, s.fighter.Str)
}

// drawCard draws one action card at `at`, in the given style.
//
// **It no longer draws anything itself.** The picture comes from internal/cards, which
// renders it into a plain Go image; this function's whole job is to turn the screen's
// types into a cards.Spec, get the matching image out of the cache, and blit it.
//
// str is the wielder's Strength, because damage is a property of the pairing rather than
// of the card: the same Strike hits for more in stronger hands. It therefore forms part
// of the cache key — see cardKey.
func drawCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point, st cards.Style,
	c actionCard, enabled, selected bool, str int) {

	img := cardImage(gs, cardSpec(c, enabled, selected, str), st)
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	screen.DrawImage(img, op)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
