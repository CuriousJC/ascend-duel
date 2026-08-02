package screens

import (
	"fmt"
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The action box: one list of cards. Click a card to select it into the round's queue,
// click it again to take it out, and drag a card to move it up or down the list. Selected
// cards sit proud of the unselected ones, so the queue reads as a shape down the right
// edge rather than as a second list somewhere else.
//
// This is a game widget rather than a UI widget — cards carry an action-point cost and the
// selection validates against a budget — so it lives on CombatScene and is hand-rolled
// rather than reaching for a toolkit. Left click and drag only; there is no right click
// and no keyboard anywhere in this game.
const (
	cardWidth  = 130
	cardHeight = 68
	cardGap    = 6

	dropIndicatorHeight = 3

	// selectedNudge is how far a selected card sticks out to the right. Selection is the
	// only state a card carries, so it gets a whole axis to itself rather than a tint
	// that would have to compete with the affordability dimming.
	selectedNudge = cardWidth * 30 / 100

	// Offsets from the top of the pane's band. The budget is a header above the list
	// rather than a divider inside it: with one list there is nothing left to divide.
	apTextTop = 8
	apBarTop  = 26
	pilesTop  = 38
	cardsTop  = 52

	apBarHeight = 8
	apBarInset  = 14

	// dragThreshold is how far the cursor has to travel with the button held before a
	// press counts as a drag rather than a click. Without it every click would jitter
	// into a one-pixel reorder and selecting a card would be a coin toss.
	dragThreshold = 4
)

// paletteCard is one row of the list: an action, and whether it is queued this round.
type paletteCard struct {
	action   combat.ActionKind
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

// remainingPoints is the budget left after everything already selected.
func (s *CombatScene) remainingPoints() int {
	return s.fighter.ActionPoints() - combat.CostOf(s.fighterActions)
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
func (s *CombatScene) beginPress(gs *state.GlobalState) {
	at := image.Pt(gs.MouseX, gs.MouseY)

	for i := range s.hand {
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
	if image.Pt(gs.MouseX, gs.MouseY).In(cardsZone(gs)) {
		at = s.dropIndex(gs)
	}
	s.insertCard(at, drag.card)
}

// toggle selects or deselects the card at i. Deselecting always works; selecting is
// refused when the remaining budget will not cover the card, which is the same rule the
// dimming on screen is already reporting.
func (s *CombatScene) toggle(i int) {
	if i < 0 || i >= len(s.hand) {
		return
	}

	if !s.hand[i].selected && s.hand[i].action.Cost() > s.remainingPoints() {
		return
	}

	s.hand[i].selected = !s.hand[i].selected
	s.syncQueue()
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

// dropIndex is the list position the cursor is currently pointing between. Measuring from
// the middle of a slot rather than its top edge means the card lands where the gap is.
func (s *CombatScene) dropIndex(gs *state.GlobalState) int {
	top := gs.PctY(paneTopPct) + cardsTop
	idx := (gs.MouseY - top + (cardHeight+cardGap)/2) / (cardHeight + cardGap)

	if idx < 0 {
		idx = 0
	}
	if idx > len(s.hand) {
		idx = len(s.hand)
	}
	return idx
}

// paneBounds is a pane's rectangle in screen coordinates.
func paneBounds(gs *state.GlobalState, p panePlacement) image.Rectangle {
	return image.Rect(
		gs.PctX(p.leftPct), gs.PctY(paneTopPct),
		gs.PctX(p.rightPct), gs.PctY(paneBottomPct),
	)
}

// cardsZone is the part of the palette's column holding the list, and the region a drag
// has to be released inside for it to count as a reorder. It reaches past the column's
// right edge because a selected card does.
func cardsZone(gs *state.GlobalState) image.Rectangle {
	zone := paneBounds(gs, palettePane)
	zone.Min.Y += cardsTop - cardGap
	zone.Max.X += selectedNudge
	return zone
}

// cardSlot is the rectangle the card at index i occupies. Selected cards are pushed right,
// and hit testing reads the same function the drawing does, so the protruding part of a
// card is clickable rather than merely visible.
func (s *CombatScene) cardSlot(gs *state.GlobalState, i int) image.Rectangle {
	bounds := paneBounds(gs, palettePane)
	x := bounds.Min.X + (bounds.Dx()-cardWidth)/2
	if i < len(s.hand) && s.hand[i].selected {
		x += selectedNudge
	}
	y := bounds.Min.Y + cardsTop + i*(cardHeight+cardGap)
	return image.Rect(x, y, x+cardWidth, y+cardHeight)
}

// drawPalette draws the budget and the list of cards. There is no pane behind them: with
// one list, and selection shown by position, a frame was outlining empty space.
func (s *CombatScene) drawPalette(gs *state.GlobalState, screen *ebiten.Image) {
	bounds := paneBounds(gs, palettePane)
	x, y, w := float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Dx())

	budgetOp := &text.DrawOptions{}
	budgetOp.GeoM.Translate(float64(x+w/2), float64(y+apTextTop))
	budgetOp.PrimaryAlign = text.AlignCenter
	text.Draw(screen,
		fmt.Sprintf("%d/%d AP", combat.CostOf(s.fighterActions), s.fighter.ActionPoints()),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 14}, budgetOp)

	s.drawAPBar(screen, x, y, w)

	// The piles have no cards on screen to count, so they get a line of their own.
	pilesOp := &text.DrawOptions{}
	pilesOp.GeoM.Translate(float64(x+w/2), float64(y+pilesTop))
	pilesOp.PrimaryAlign = text.AlignCenter
	text.Draw(screen, fmt.Sprintf("deck %d  ·  discard %d", len(s.deck), len(s.discard)),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 11}, pilesOp)

	for i, c := range s.hand {
		// An unselected card the budget will not cover reads as unavailable. A selected
		// one never does — it is already paid for, and clicking it off has to stay open.
		enabled := c.selected || (s.planning() && c.action.Cost() <= s.remainingPoints())
		drawCard(gs, screen, s.cardSlot(gs, i), c.action, palettePane.color, enabled, c.selected)
	}

	if s.drag == nil || !s.drag.active || !image.Pt(gs.MouseX, gs.MouseY).In(cardsZone(gs)) {
		return
	}

	slot := s.cardSlot(gs, s.dropIndex(gs))
	vector.DrawFilledRect(screen,
		float32(slot.Min.X), float32(slot.Min.Y-cardGap),
		float32(cardWidth), dropIndicatorHeight,
		palettePane.color, false)
}

// drawAPBar draws the action-point budget as a bar: the track is the whole budget, the
// filled part is what the selection currently spends. The numeric line above it stays —
// the bar answers "how much room is left" at a glance, the number answers "exactly".
func (s *CombatScene) drawAPBar(screen *ebiten.Image, x, y, w float32) {
	budget := s.fighter.ActionPoints()
	if budget <= 0 {
		return
	}

	left, width := x+apBarInset, w-2*apBarInset
	top := y + apBarTop

	vector.DrawFilledRect(screen, left, top, width, apBarHeight,
		systems.ColorAtStrength(apBarColor, 20), false)

	spent := width * float32(combat.CostOf(s.fighterActions)) / float32(budget)
	vector.DrawFilledRect(screen, left, top, spent, apBarHeight, apBarColor, false)

	vector.StrokeRect(screen, left, top, width, apBarHeight, 1,
		systems.ColorAtStrength(apBarColor, 70), false)
}

// drawDraggedCard draws the card in flight. Called last so it rides over everything.
func (s *CombatScene) drawDraggedCard(gs *state.GlobalState, screen *ebiten.Image) {
	if s.drag == nil || !s.drag.active {
		return
	}

	x := gs.MouseX - s.drag.grabDX
	y := gs.MouseY - s.drag.grabDY
	drawCard(gs, screen, image.Rect(x, y, x+cardWidth, y+cardHeight),
		s.drag.card.action, palettePane.color, true, s.drag.card.selected)
}

// drawCard draws one action card: its name, its initiative and its action-point cost.
//
// The lower half is deliberately empty. That is where the glyph row goes — initiative,
// damage, and whatever else the taxonomy settles on — which is what the card is this tall
// for. Until those exist it reads as a card with room in it.
func drawCard(gs *state.GlobalState, screen *ebiten.Image, slot image.Rectangle,
	action combat.ActionKind, base color.RGBA, enabled, selected bool) {

	fill, border := systems.ColorAtStrength(base, 45), base
	switch {
	case !enabled:
		fill, border = systems.ColorAtStrength(base, 20), systems.ColorAtStrength(base, 40)
	case selected:
		fill = systems.ColorAtStrength(base, 65)
	}

	x, y := float32(slot.Min.X), float32(slot.Min.Y)
	vector.DrawFilledRect(screen, x, y, cardWidth, cardHeight, fill, false)
	vector.StrokeRect(screen, x, y, cardWidth, cardHeight, 2, border, false)

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}

	nameOp := &text.DrawOptions{}
	nameOp.GeoM.Translate(float64(slot.Min.X+10), float64(slot.Min.Y+16))
	nameOp.SecondaryAlign = text.AlignCenter
	text.Draw(screen, action.String(), face, nameOp)

	initOp := &text.DrawOptions{}
	initOp.GeoM.Translate(float64(slot.Min.X+10), float64(slot.Min.Y+36))
	initOp.SecondaryAlign = text.AlignCenter
	text.Draw(screen, fmt.Sprintf("init %d", action.Initiative()),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 11}, initOp)

	costOp := &text.DrawOptions{}
	costOp.GeoM.Translate(float64(slot.Max.X-10), float64(slot.Min.Y+16))
	costOp.PrimaryAlign = text.AlignEnd
	costOp.SecondaryAlign = text.AlignCenter
	text.Draw(screen, fmt.Sprintf("%d", action.Cost()), face, costOp)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
