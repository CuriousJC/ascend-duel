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
	// 162x224 is roughly a playing card's proportions, and it is written here as two plain
	// numbers on purpose: cardWidth used to be *derived* from the glyph row, so adding a
	// badge silently widened every card in the game and the layout could not be reasoned
	// about without doing the arithmetic. The contents now fit the card, never the reverse.
	//
	// **Down a tenth across and 15% down the face on 2026-08-11**, from 180x264, to give the
	// screen back some room. It has to stay equal to cards.Hand's footprint — that package
	// draws the picture and this one places it — and TestCardFootprintMatchesTheRenderer is
	// what says so.
	cardWidth  = 162
	cardHeight = 224
	cardGap    = 12

	// The row sits low, with the budget bar and then the button strip beneath it. handTopPct
	// is the top of an *unselected* card; a selected one rises above it by selectedNudge.
	//
	// **59% until 2026-08-11, 61% until 2026-08-12.** It came down the first time when the AP
	// text line went, so the cards sit directly on top of the bar that fills as they are
	// selected. It came down again when the deck pile was re-hung off the bottom of the screen:
	// that freed the band the pile used to float in, and **66% is the value that puts the
	// action-point figure's top exactly on the Discard button's top**, which is what the owner
	// asked for and also the tightest the strip goes.
	//
	// The arithmetic, because it is a coincidence of five constants and not a round number:
	// PctY(66) is 633, plus a 224px card, plus apBarBelow, apBarHeight and apFigureBelowBar
	// is 887 — and the button strip's centre at PctY(95) less half a 50px button is 887 too.
	// **Nothing enforces that**, so TestTheAPFigureLinesUpWithTheButtonStrip does.
	//
	// The bar's own y is measured from this row and the Resolution feed's bottom edge from it
	// as well, so moving the row moves the whole lower half of the screen together — which is
	// the point: what the drop buys is height between the top row and the cards.
	handTopPct = 66

	// selectedNudge is how far a selected card lifts out of the row. Selection is the only
	// state a card carries, so it gets a whole axis to itself rather than a tint that would
	// have to compete with the affordability dimming. Up rather than right, because the
	// hand is now a horizontal strip and the free axis rotated with it.
	selectedNudge = 26

	dropIndicatorWidth = 4

	// The budget bar, as an offset *below* the bottom of the row. It is exactly as wide as the
	// hand, so it reads as belonging to the cards rather than floating near them, and it sits
	// under the selection it is reporting on — between the cards and the buttons that spend it.
	//
	// **There was a line of text between the two until 2026-08-11**, `3/6 AP` at the left end
	// and `deck 45 · discard 7` at the right, with `apTextBelow = 10` placing it and
	// `apBarBelow = 36` leaving room for it. Both figures found better homes — the bar is
	// segmented and therefore countable, and the draw pile now states its own count — so the
	// line went and the bar came up under the cards. The row moved *down* by the difference
	// rather than the bar moving up: the strip below it holds the deck stack, and its top is
	// measured from the bar.
	apBarBelow  = 14
	apBarHeight = 8

	// The strip under the bar: the AP figure, the two buttons and the deck pile, all on one
	// line. buttonStripPct is that line's centre and every one of them is placed against it.
	buttonStripPct = 95

	// Both buttons on that strip are the same size, and it is named here rather than written
	// at the two NewButton calls because **three other things are placed against it**: the
	// discard badge hangs off a corner, buttonStripSlots divides the space around the pair,
	// and the AP figure's line is checked against their top edge. Those were reading a
	// literal 138 copied into a test, which is the shape of thing that drifts.
	stripButtonWidth  = 138
	stripButtonHeight = 50

	// **The figure came back on 2026-08-11**, having been dropped that same day along with the
	// pile counts it shared a line with. What was wrong with it was where it was — a line of
	// small text wedged between the cards and the bar — not that it was written down. It is
	// now left-aligned under the left end of the bar, on the button line, and the bar is
	// directly under the cards where it belongs.
	//
	// **apFigureReserve is a fixed column width, not the text's measured width**, because the
	// buttons are spaced off its right edge and the text is not a constant length. Measuring
	// would move both buttons the moment the figure went from `9/12 AP` to `10/12 AP`. The
	// reserve holds the normal figure — `12/12 AP` measures 53 pixels at this size — and the
	// `+N over` tail deliberately runs past it, into a gap hundreds of pixels wide.
	//
	// apFigureBelowBar hangs it off the *bar*, not off the button line it started on. The
	// figure is a caption on the bar and belongs against it; the buttons beside it are placed
	// on their own line and the two only have to agree horizontally.
	apFigureSize     = 18
	apFigureReserve  = 110
	apFigureBelowBar = 8

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
// field — c.Action still reads the same everywhere it did before.
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
			s.fighterActions = append(s.fighterActions, c.actionCard)
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
	at := slotAt(gs, i, s.laidOutCount())
	if i < len(s.hand) && s.hand[i].selected {
		at.Y -= selectedNudge
	}
	return image.Rect(at.X, at.Y, at.X+cardWidth, at.Y+cardHeight)
}

// slotAt is where the card at index i sits in a row of n, as a pure function of the two.
//
// **Pulled out of cardSlot so a slot can be located after the row it belonged to is gone.**
// A card flying out of the hand has to start from where it was, and by the time anything is
// drawn the hand has already re-laid out around its absence — so the flight stores the index
// and the count it left behind and asks for that rectangle back. Reading it off the current
// hand would start every discard from wherever the row happens to sit now.
//
// It knows nothing about selection, which is a property of a card and not of a position;
// cardSlot adds the lift.
func slotAt(gs *state.GlobalState, i, n int) image.Point {
	return image.Pt(
		handBand(gs, n).Min.X+i*handPitch(gs, n),
		gs.PctY(handTopPct),
	)
}

// drawHandRow draws the row of cards and the budget bar under it. There is no pane behind
// them: with one row, and selection shown by position, a frame was outlining empty space.
func (s *CombatScene) drawHandRow(gs *state.GlobalState, screen *ebiten.Image) {
	band := handBand(gs, s.laidOutCount())
	left, right := float32(band.Min.X), float32(band.Max.X)
	top := float32(gs.PctY(handTopPct))
	below := top + cardHeight

	s.drawAPBar(screen, left, below+apBarBelow, right-left)
	s.drawAPFigure(gs, screen, band.Min.X, int(below)+apBarBelow+apBarHeight)

	for i, c := range s.hand {
		// A card being dealt is drawn by its flight, somewhere between the pile and here, so
		// the slot it is heading for stays empty until it lands. The card is already *in* the
		// hand — this hides a drawing, not a card, which is why nothing else has to know.
		if s.inboundTo(i) {
			continue
		}

		// A card that has fired this round is drawn by the resolved pile instead — rising,
		// held, or parked in the corner. It has *not* left the hand; it leaves at the end of
		// the round with everything else that was played, which is what keeps the Resolution
		// pane able to narrate from fighterActions while the round runs.
		if s.resolvedInHand(i) {
			continue
		}

		// Dimmed means "you cannot pick this", which now means the selection is full rather
		// than that the card is unaffordable — an unaffordable card is pickable on purpose,
		// and dimming something you can click would be a lie. What it costs you is reported
		// by the bar going red, after the fact. A selected card never dims: clicking it off
		// has to stay open, and it is the way out of an over-allocation.
		enabled := c.selected || (s.planning() && s.selectedCount() < s.fighter.MaxActions())
		drawCard(gs, screen, s.cardSlot(gs, i).Min, cards.Hand,
			c.actionCard, enabled, c.selected)
	}

	s.drawComboPreview(gs, screen)

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

// previewAttack is the blow the current selection would land if DUEL! were pressed now.
//
// **It calls the resolver's own `combat.BlowFor`**, which is what makes a preview trustworthy:
// the combo shown while choosing is the combo that fires, by construction rather than by two
// pieces of code agreeing about what three Strikes are worth. Nothing here knows what a Flurry
// is.
//
// **Only a formed hand previews.** A lone attack carries `HandNone` — the feed writes those as
// an ordinary attack rather than announcing them, and a preview shouting COMBO! over one Strike
// would empty the word before the player ever earned it.
//
// The turn is ordered with an empty opposing queue, because `ResolutionOrder` is the authority
// on which of the player's cards resolve in what order and the opponent's cards do not enter
// their attack phase. Building the slice by hand here would be a second orderer.
func (s *CombatScene) previewAttack() (combat.Blow, bool) {
	if !s.planning() {
		return combat.Blow{}, false
	}
	blow := combat.BlowFor(combat.ResolutionOrder(s.fighterActions, nil))
	if !blow.Formed() {
		return combat.Blow{}, false
	}
	return blow, true
}

// previewHandSlots is where the previewed hand's cards are sitting in the row right now.
//
// **`Blow.Cards` indexes the turn, not the hand**, and the two differ the moment a Prepare is
// queued before a Strike — the turn is in resolution order, the hand is in the order the player
// left it. `handIndexForQueue` is the existing inverse of the walk that builds the queue, so
// this is a translation rather than a second mapping.
func (s *CombatScene) previewHandSlots(blow combat.Blow) []int {
	turn := combat.ResolutionOrder(s.fighterActions, nil)

	out := make([]int, 0, len(blow.Cards))
	for _, i := range blow.Cards {
		if i < 0 || i >= len(turn) {
			continue
		}
		if h, ok := s.handIndexForQueue(turn[i].Index); ok {
			out = append(out, h)
		}
	}
	return out
}

// drawComboPreview rings the cards in the hand that have formed a combo, the instant they form
// it rather than when the round plays out.
//
// **The same ring the table draws**, through the same `drawComboBracket`, so what the player
// sees while choosing and what they see when it fires are one drawing. The words that go with
// it are the Resolution feed's preview line — see resolutionLines — because there is nowhere
// above the row to put them: the feed's bottom edge sits five pixels over the resting cards and
// a selected one already lifts into it.
func (s *CombatScene) drawComboPreview(gs *state.GlobalState, screen *ebiten.Image) {
	blow, ok := s.previewAttack()
	if !ok {
		return
	}

	slots := s.previewHandSlots(blow)
	at := make([]image.Point, 0, len(slots))
	for _, i := range slots {
		// A card still flying in from the pile is not drawn in its slot, so ringing that slot
		// would ring a gap. It cannot be selected either, which makes this belt and braces —
		// kept because the row skips it for the same reason and the two should agree.
		if s.inboundTo(i) {
			continue
		}
		at = append(at, s.cardSlot(gs, i).Min)
	}
	drawComboBracket(screen, at)
}

// drawAPFigure writes the budget as `3/6 AP`, tucked under the left end of the bar — same left
// edge, apFigureBelowBar under its bottom. It reads as a caption on the bar rather than as a
// fourth thing on the button line, which is where it sat for its first hour on 2026-08-11.
//
// It takes the bar's left edge and its bottom rather than deriving them, since its caller has
// just drawn the bar from those two numbers and a second derivation is a second thing to be
// wrong.
//
// **An overspend is said twice on purpose**, here in words and by the bar's red cells. It is
// the one state that stops DUEL! working.
func (s *CombatScene) drawAPFigure(gs *state.GlobalState, screen *ebiten.Image, left, barBottom int) {
	spent, budget := combat.CostOf(s.fighterActions), s.fighter.ActionPoints()

	label := fmt.Sprintf("%d/%d AP", spent, budget)
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(left), float64(barBottom+apFigureBelowBar))
	// One ScaleWithColor, never two — the scale multiplies, so setting the ink and then the
	// warning colour would give a near-black red rather than the red.
	ink := groundInk
	if spent > budget {
		label = fmt.Sprintf("%s  +%d over", label, spent-budget)
		ink = apOverColor
	}
	op.ColorScale.ScaleWithColor(ink)
	text.Draw(screen, label,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: apFigureSize}, op)
}

// apFigureRight is where the figure's reserved column ends, and the left edge of the space the
// buttons are spread across.
func apFigureRight(gs *state.GlobalState) int {
	return handBand(gs, handSize).Min.X + apFigureReserve
}

// buttonStripSlots is where the two buttons sit: three equal gaps between the AP figure's
// column and the deck pile, with a button in the two spaces between them. Returns centres,
// which is what models.Button stores.
//
// **Written as one function of the strip rather than two placements**, because the property
// wanted is a relationship — evenly shared space — and two hand-picked percentages cannot
// hold it when the pile or the figure moves. `TestTheButtonStripSharesItsSpaceEvenly` checks
// the gaps against each other rather than against numbers.
//
// It reads the pile's *bounds*, not its front card: the backs are drawn up and to the left, so
// the front card's edge is not the pile's edge.
func buttonStripSlots(gs *state.GlobalState, discardWidth, duelWidth int) (int, int) {
	left, right := apFigureRight(gs), deckStackBounds(gs).Min.X
	gap := (right - left - discardWidth - duelWidth) / 3

	discardLeft := left + gap
	duelLeft := discardLeft + discardWidth + gap
	return discardLeft + discardWidth/2, duelLeft + duelWidth/2
}

// drawAPBar draws the action-point budget as a bar. The figure beside it answers "exactly";
// the bar answers "how much room is left" without being read.
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

	// **Toward the ground, not toward black.** ColorAtStrength would scale the blue down to
	// {14,26,46}, which on a cream screen is the darkest thing in the bar and reads as the
	// *filled* part — the empty cells stepping in front of the spent ones. See
	// systems.ColorToward.
	empty := systems.ColorToward(apBarColor, screenGround, 80)
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
		s.drag.card.actionCard, true, s.drag.card.selected)
}

// drawCard draws one action card at `at`, in the given style.
//
// **It no longer draws anything itself.** The picture comes from internal/cards, which
// renders it into a plain Go image; this function's whole job is to turn the screen's
// types into a cards.Spec, get the matching image out of the cache, and blit it.
//
// **It takes no Strength any more** *(2026-08-14)*. It used to, because the damage figure on
// the face was `Damage(str)` — the same Strike hits for more in stronger hands, so the wielder
// was part of the cache key. The face carries effect text instead of a figure now, and a
// card's picture is a function of the card alone.
func drawCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point, st cards.Style,
	c actionCard, enabled, selected bool) {

	img := cardImage(gs, cardSpec(c, enabled, selected), st)
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
