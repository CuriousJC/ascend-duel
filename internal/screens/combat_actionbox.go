package screens

import (
	"fmt"
	"image"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
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

	// The glyph column, down the left edge of the card the way a playing card puts its rank
	// and suit in a corner. These have to fit inside cardHeight now rather than setting it:
	// glyphColumnTop + 3*cardGlyphSize + 2*glyphGap = 256, with 8 pixels to spare.
	//
	// The scale lives in systems so the contact sheet draws its actual-size row at exactly
	// what the card uses. See systems.CardGlyphScale.
	cardGlyphScale = systems.CardGlyphScale
	cardGlyphSize  = systems.GlyphSize * cardGlyphScale

	glyphInset     = 12
	glyphGap       = 8
	glyphColumnTop = 48
	glyphNumberGap = 10

	cardNameTop = 14

	// dragThreshold is how far the cursor has to travel with the button held before a
	// press counts as a drag rather than a click. Without it every click would jitter
	// into a one-pixel reorder and selecting a card would be a coin toss.
	dragThreshold = 4
)

// cardStyle is a card's geometry at one size. Two exist: the hand's, and a smaller one for
// the deck overlay, where the point is to see the whole deck at once rather than to read
// any one card closely.
//
// **A glyph cannot be made smaller.** systems.GlyphSize is 64 and CardGlyphScale is 1: the
// art is authored at exactly the size it is displayed, and a fractional scale would drop
// pixels out of a rim that is one pixel thick. glyphScale must therefore stay a whole
// number, and 1 is already the floor.
//
// That makes the glyph column the hard floor on a card's size, not a thing that scales with
// it: three glyphs need 3*64 plus gaps down, and 64 plus a numeral across, whatever else the
// card does. A "small" card is one with less padding and smaller text around the same
// column — which is why deckCardStyle is only a little narrower than the hand's and barely
// shorter at all.
type cardStyle struct {
	width, height int

	nameTop  int
	nameSize float64

	glyphScale     int
	glyphInset     int
	glyphGap       int
	glyphColumnTop int
	glyphNumberGap int
	numberSize     float64

	border float32
}

var (
	// handCardStyle is the card as the hand draws it, and the size every layout constant
	// above is written for.
	handCardStyle = cardStyle{
		width: cardWidth, height: cardHeight,
		nameTop: cardNameTop, nameSize: 20,
		glyphScale: cardGlyphScale, glyphInset: glyphInset, glyphGap: glyphGap,
		glyphColumnTop: glyphColumnTop, glyphNumberGap: glyphNumberGap, numberSize: 26,
		border: 2,
	}

	// deckCardStyle is the card as the deck overlay draws it: the same three glyphs as the
	// hand, in a tighter frame. 138x236 against the hand's 180x264 — all of the saving is
	// padding and text size, because the glyph column underneath is identical and cannot
	// give any back. Its column comes to 34 + 3*64 + 2*4 = 234 inside 236.
	deckCardStyle = cardStyle{
		width: 138, height: 236,
		nameTop: 8, nameSize: 16,
		glyphScale: 1, glyphInset: 10, glyphGap: 4,
		glyphColumnTop: 34, glyphNumberGap: 8, numberSize: 20,
		border: 1,
	}
)

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
	if image.Pt(gs.MouseX, gs.MouseY).In(handZone(gs)) {
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

// handBand is the rectangle a row of n cards occupies: centred on the screen rather than
// pinned to a pane, because the hand is the width of whatever is in it and grows outward.
//
// This is the single authority on that width. The AP bar spans it, the caption box above
// the hand matches it, and the card slots are cut out of it, so none of them can drift
// apart when the hand size changes.
func handBand(gs *state.GlobalState, n int) image.Rectangle {
	if n < 1 {
		n = 1
	}
	w := n*cardWidth + (n-1)*cardGap
	left := gs.PctX(50) - w/2
	top := gs.PctY(handTopPct)
	return image.Rect(left, top, left+w, top+cardHeight)
}

// handRowLeft is the x of the first slot.
func (s *CombatScene) handRowLeft(gs *state.GlobalState) int {
	return handBand(gs, s.laidOutCount()).Min.X
}

// dropIndex is the position the cursor is currently pointing between. Measuring from the
// middle of a slot rather than its left edge means the card lands where the gap is.
func (s *CombatScene) dropIndex(gs *state.GlobalState) int {
	left := s.handRowLeft(gs)
	idx := (gs.MouseX - left + (cardWidth+cardGap)/2) / (cardWidth + cardGap)

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
func (s *CombatScene) cardSlot(gs *state.GlobalState, i int) image.Rectangle {
	x := s.handRowLeft(gs) + i*(cardWidth+cardGap)
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
	budgetOp := &text.DrawOptions{}
	budgetOp.GeoM.Translate(float64(left), float64(below+apTextBelow))
	text.Draw(screen,
		fmt.Sprintf("%d/%d AP", combat.CostOf(s.fighterActions), s.fighter.ActionPoints()),
		face, budgetOp)

	pilesOp := &text.DrawOptions{}
	pilesOp.GeoM.Translate(float64(right), float64(below+apTextBelow))
	pilesOp.PrimaryAlign = text.AlignEnd
	text.Draw(screen, fmt.Sprintf("deck %d  ·  discard %d", len(s.deck), len(s.discard)),
		face, pilesOp)

	s.drawAPBar(screen, left, below+apBarBelow, right-left)

	for i, c := range s.hand {
		// An unselected card the budget will not cover reads as unavailable. A selected
		// one never does — it is already paid for, and clicking it off has to stay open.
		enabled := c.selected || (s.planning() && c.action.Cost() <= s.remainingPoints())
		drawCard(gs, screen, s.cardSlot(gs, i).Min, handCardStyle,
			c.actionCard, enabled, c.selected, s.fighter.Str)
	}

	if s.drag == nil || !s.drag.active || !image.Pt(gs.MouseX, gs.MouseY).In(handZone(gs)) {
		return
	}

	// A bar standing in the gap the card will drop into, full card height so it reads as a
	// slot rather than as a tick.
	slot := s.cardSlot(gs, s.dropIndex(gs))
	vector.DrawFilledRect(screen,
		float32(slot.Min.X-cardGap), top,
		dropIndicatorWidth, cardHeight,
		playerSwatch, false)
}

// drawAPBar draws the action-point budget as a bar: the track is the whole budget, the
// filled part is what the selection currently spends. The numeric line above it stays —
// the bar answers "how much room is left" at a glance, the number answers "exactly".
func (s *CombatScene) drawAPBar(screen *ebiten.Image, left, top, width float32) {
	budget := s.fighter.ActionPoints()
	if budget <= 0 {
		return
	}

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

	at := image.Pt(gs.MouseX-s.drag.grabDX, gs.MouseY-s.drag.grabDY)
	drawCard(gs, screen, at, handCardStyle,
		s.drag.card.actionCard, true, s.drag.card.selected, s.fighter.Str)
}

// cardBadge is one glyph on a card and the number written across it.
type cardBadge struct {
	kind  systems.GlyphKind
	value int
}

// badgesFor is what a card says about itself: what it hits for, how soon it lands, what it
// costs. Damage is omitted when there is none — a sword reading zero on a Guard is worse
// than no sword at all — which is also why the column packs from the top rather than
// filling fixed slots. The card does not change size either way.
func badgesFor(action combat.ActionKind, str int) []cardBadge {
	badges := make([]cardBadge, 0, 3)
	if dmg := action.Damage(str); dmg > 0 {
		badges = append(badges, cardBadge{systems.GlyphDamage, dmg})
	}
	return append(badges,
		cardBadge{systems.GlyphInitiative, action.Initiative()},
		cardBadge{systems.GlyphActionPoints, action.Cost()})
}

// drawCard draws one action card: its name across the top, then a column of glyphs down
// the left edge with each number written beside its glyph — the way a playing card puts
// its rank and suit in a corner and leaves the rest of the face free.
//
// The glyphs replaced the "init 3" line and the bare cost numeral rather than joining
// them. Two of the three numbers were already text, and a card that says everything twice
// is harder to read than one that says it once.
//
// The surface is the card's element at full strength, scaled down for its state — so a
// plain card is grey-white, a fire one burnt orange, and the state still reads the same way
// on both. The pane green it used to be drawn in was vestigial: the colour of a pane
// deleted on 2026-08-02, still filling cards that no longer sat inside anything.
//
// str is the wielder's Strength, since damage is a property of the pairing rather than of
// the card: the same Strike hits for more in stronger hands.
//
// Everything is measured off the style rather than off the package constants, so drawing
// the deck at half size is a struct literal instead of a second copy of this function.
func drawCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point, st cardStyle,
	c actionCard, enabled, selected bool, str int) {

	action, base := c.action, c.element.color()

	fill, border := systems.ColorAtStrength(base, 45), base
	switch {
	case !enabled:
		// 30 rather than the buttons' deeper dimming: white scaled to 20% lands on the same
		// grey as the background and an unaffordable plain card simply vanished.
		fill, border = systems.ColorAtStrength(base, 30), systems.ColorAtStrength(base, 45)
	case selected:
		fill = systems.ColorAtStrength(base, 65)
	}

	x, y := float32(at.X), float32(at.Y)
	w, h := float32(st.width), float32(st.height)
	vector.DrawFilledRect(screen, x, y, w, h, fill, false)
	vector.StrokeRect(screen, x, y, w, h, st.border, border, false)

	nameOp := &text.DrawOptions{}
	nameOp.GeoM.Translate(float64(at.X+st.glyphInset), float64(at.Y+st.nameTop))
	text.Draw(screen, action.String(),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: st.nameSize}, nameOp)

	numberFace := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: st.numberSize}
	pal := systems.PaletteOf(systems.PaletteWhite)
	glyphSize := systems.GlyphSize * st.glyphScale

	for i, badge := range badgesFor(action, str) {
		gx := at.X + st.glyphInset
		gy := at.Y + st.glyphColumnTop + i*(glyphSize+st.glyphGap)

		// Drawn in its own palette, deliberately untinted. Scaling a five-value palette
		// toward the card's colour collapses the bevel back into a flat silhouette, which
		// is the whole thing the palette exists to avoid. A disabled card dims the glyph
		// by alpha instead, so the shading survives and only the weight changes.
		//
		// Scaled by a whole number and nothing else — see cardStyle.glyphScale.
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(st.glyphScale), float64(st.glyphScale))
		op.GeoM.Translate(float64(gx), float64(gy))
		if !enabled {
			op.ColorScale.ScaleAlpha(0.4)
		}
		screen.DrawImage(systems.Glyph(badge.kind, systems.PaletteWhite), op)

		// Beside the glyph rather than across it: the column has width to spare now, and a
		// number sitting in open card is easier to read than one fighting the bevel it is
		// printed on. Drawn in the palette's specular, the lightest value it has, so it
		// still belongs to the art.
		numOp := &text.DrawOptions{}
		numOp.GeoM.Translate(
			float64(gx+glyphSize+st.glyphNumberGap),
			float64(gy+glyphSize/2))
		numOp.SecondaryAlign = text.AlignCenter
		numOp.ColorScale.ScaleWithColor(pal.Specular)
		if !enabled {
			numOp.ColorScale.ScaleAlpha(0.4)
		}
		text.Draw(screen, fmt.Sprintf("%d", badge.value), numberFace, numOp)
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
