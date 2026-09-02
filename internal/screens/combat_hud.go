package screens

// The HUD: the things around the round rather than the round itself. The character strip, the
// enemy card, the generic framed box every panel is built from, and the health bars.
//
// Split out of combat.go on 2026-08-07. It is the most self-contained group on the screen —
// nothing here reads the log, the deck or the queue, so it is the part that can be moved
// without any of the rest being understood.

import (
	"fmt"
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/pyramid"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"image/color"
)

// drawBox draws a framed panel: dim fill, full-strength border, and a title centred on the
// top edge. Dim fill and a full-strength border means the box reads as green or pink at a
// glance without drowning what is drawn on top of it.
//
// This takes a rectangle rather than a panePlacement because the boxes along the bottom
// are sized by the hand rather than by a slice of the screen. An empty title draws none.
func drawBox(gs *state.GlobalState, screen *ebiten.Image, r image.Rectangle, c color.RGBA, title string) {
	x, y := float32(r.Min.X), float32(r.Min.Y)
	w, h := float32(r.Dx()), float32(r.Dy())

	vector.DrawFilledRect(screen, x, y, w, h, systems.ColorAtStrength(c, 25), false)
	vector.StrokeRect(screen, x, y, w, h, 2, c, false)

	if title == "" {
		return
	}
	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(float64(x+w/2), float64(y+paneTitleInset))
	titleOp.PrimaryAlign = text.AlignCenter
	text.Draw(screen, title, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}, titleOp)
}

// **The caption box stood here and is gone** *(2026-08-11)*. It was a hand-width box at 48%
// holding the plan line and its action-point cost; the Resolution feed took the slot and has
// since left it too *(2026-08-18)*. What stands there now is the band the hand dialog writes
// the blow's arithmetic across. See combat_mathbox.go.

// **The character block is gone and the player is a card** *(2026-08-12)*.
//
// It had been three shapes — a tall box at 4%,12%, a wide strip on 2026-08-07, a narrow
// corner column on 2026-08-11 — and each move was forced by whatever claimed the space beside
// it. What ended it is not space: it is that the enemy became a card the day before, and the
// player was then the last thing on this screen drawn as furniture rather than as an object
// in the game. Both corners now hold the same format, which is what makes them read as two
// sides of one fight.
//
// What the box held is what the card holds, in the same order: the duelist's name, then the
// figures, then life. What it *gains* is the two figures the block had nowhere to put — the
// damage a Strike does in these hands, and the action points the round is bought with — and a
// health bar above the fraction, matching the enemy's at the same offsets so the two can be
// compared across the screen without measuring.
//
// The card's own layout lives in cards.DuelistStyle; what is here is where it sits and what
// goes on it.
const (
	duelistCardLeftPct = 1

	// **Both cards share this top**, and the ring row between them aligns to it as well —
	// see ringPaneRect. One percentage rather than three, because what is wanted is that the
	// whole band starts on one line, not that each thing happens to be near the top.
	topRowTopPct = 2
)

// lifeColor is the red the life fraction used to be written in.
//
// **Nothing draws it since the character block became a card** — the card's own bar and
// fraction come from cards.HealthFull and cards.NumberInk, which is the point of drawing the
// player the same way as everything else. It is kept because the deck overlay and the flight
// code still describe state in the screen's own colours and this is the one red among them;
// delete it if a second thing has to be said about that.
var lifeColor = color.RGBA{R: 225, G: 65, B: 65, A: 255}

// duelistCardRect is where the player's card sits. **The ring row starts from its right
// edge**, so this is the one place its geometry is written and both read it — see
// ringPaneRect.
func (s *CombatScene) duelistCardRect(gs *state.GlobalState) image.Rectangle {
	left, top := gs.PctX(duelistCardLeftPct), gs.PctY(topRowTopPct)
	return image.Rect(left, top,
		left+cards.DuelistStyle.Width, top+cards.DuelistStyle.Height)
}

// drawDuelistCard draws what the player is: name, DMG, AP, Vitae, and life as a bar over a
// fraction.
//
// One cached image from internal/cards, like the enemy's, so the contact sheet draws the same
// card the screen does. Nothing is drawn if it cannot be built — a missing font, most likely —
// for the same reason drawCard does nothing.
func (s *CombatScene) drawDuelistCard(gs *state.GlobalState, screen *ebiten.Image) {
	img := cardImage(gs,
		duelistSpec(gs, s.fighter, s.sideName(combat.SideA), gs.Run.Vitae(),
			s.shownLife(combat.SideA, s.fighter.CurrentLife),
			s.fighter.ActionPoints(),
			s.shownShields(combat.SideA, s.fighter.Shields),
			s.shownShieldInks(combat.SideA)...),
		cards.DuelistStyle)
	if img == nil {
		return
	}

	r := s.duelistCardRect(gs)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	screen.DrawImage(img, op)
}

// Where in the tower this fight is: **the floor, and which of that floor's three rooms**, in
// two lines under the duelist card.
//
// MECHANICS.md's tower is 8 floors x 3 fights with a choice of door after fights 1 and 2 and a
// choice of stairwell after the boss, so a floor is three rooms deep and the third one is the
// way up. That makes the position entirely a function of how far along the fight order the
// player has got, which is why nothing is stored: `fightIndex` already says it, and it is the
// same stand-in the enemy roster is walked with. `Session` owns both the moment it exists.
//
// **It says where you are, not what is coming.** Naming the third room Stairway is the only
// thing on this screen that says a floor is about to end — the doors and the stairwell are the
// screen that does not exist yet.
const (
	// fightsPerFloor is how many fights a floor holds, and the third of them is its boss.
	//
	// **It is `pyramid.FightsPerFloor` rather than a 3 of this screen's own**, because the ascent
	// curve that grows an enemy per room reads the same number. Two copies would let the label and
	// the difficulty disagree about how deep a floor is.
	fightsPerFloor = pyramid.FightsPerFloor

	towerLineGap   = 10 // gap from the card's bottom edge to the first line
	towerLineSize  = 18
	towerLinePitch = 22 // the same pitch a card sets its own text at
	towerLines     = 2  // the floor, then the room
)

// towerRoomNames is what each of a floor's three fights is called, in order. Indexed by the
// fight's position within its floor, so it must stay fightsPerFloor long — the two are checked
// against each other by TestEveryRoomOnAFloorIsNamed.
var towerRoomNames = [fightsPerFloor]string{"Outer Room", "Inner Room", "Stairway"}

// towerFloor is which floor a fight is on, counting from one.
//
// **It is not capped at the tower's eight.** The fight order is every record in the roster —
// 96 of them, scaffolding for a generator that does not exist — so playing far enough reads
// Floor 9 and beyond. A clamp would be a screen quietly disagreeing with the counter it is
// drawing; the honest fix is the tower, not a maximum here.
func towerFloor(fight int) int { return fight/fightsPerFloor + 1 }

// towerRoom names which of its floor's fights this is.
func towerRoom(fight int) string { return towerRoomNames[fight%fightsPerFloor] }

// towerPlaceRect is what the two lines occupy: the duelist card's column, starting below it.
//
// The width is the card's rather than the text's — the lines are short and left-aligned to the
// card's left edge, and what the rectangle is for is holding the block against what is drawn
// under it. See TestTheTowerLinesFitBetweenTheCardAndTheTable.
func (s *CombatScene) towerPlaceRect(gs *state.GlobalState) image.Rectangle {
	r := s.duelistCardRect(gs)
	top := r.Max.Y + towerLineGap
	return image.Rect(r.Min.X, top, r.Max.X, top+towerLines*towerLinePitch)
}

// drawTowerPlace writes the floor and the room under the duelist card.
//
// Straight onto the ground rather than onto a surface of its own, so it takes `groundInk` — it
// belongs to the card above it and a panel would make it a third object in a row that already
// has three.
func (s *CombatScene) drawTowerPlace(gs *state.GlobalState, screen *ebiten.Image) {
	r := s.towerPlaceRect(gs)
	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: towerLineSize}

	lines := [towerLines]string{
		fmt.Sprintf("Floor %d", towerFloor(s.fightIndex)),
		towerRoom(s.fightIndex),
	}
	for i, line := range lines {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y+i*towerLinePitch))
		op.ColorScale.ScaleWithColor(groundInk)
		text.Draw(screen, line, face, op)
	}
}

// The discards-left badge: a filled disc centred exactly on the Discard button's bottom-right
// corner, with the count in it.
//
// **Centred on the corner rather than inset from it**, so a quarter of the disc hangs off each
// of the two edges and it reads as a counter attached to the button instead of a second thing
// printed inside it. Nothing sits under that corner — the hand row ends well above the button
// strip — so the overhang costs no legibility anywhere else.
//
// **Large on purpose.** It is a number watched rather than read: the point is seeing it tick as
// the button is pressed, and at the character strip's 24px it was a figure you had to go and
// look for, at the far end of the screen from the control that changes it.
// Both were a quarter larger — 34 and 23 — for the first look at it *(2026-08-11)*. A disc
// that size read as a second control stuck to the button rather than as a counter on it.
const (
	discardBadgeSize   = 26
	discardBadgeRadius = 17
)

// The badge's disc and its ink.
//
// **Two pairs rather than one dimmed by state**, because the button underneath does not dim —
// it changes colour entirely. `disabledButtonColor` is flat dark grey and deliberately ignores
// BaseColor, so a badge tuned to sit on yellow has nothing to say about grey. Disabled takes a
// mid grey disc: still a disc, plainly not a live one.
var (
	discardBadgeFill         = color.RGBA{R: 245, G: 245, B: 245, A: 255}
	discardBadgeInk          = color.RGBA{R: 25, G: 25, B: 25, A: 255}
	discardBadgeDisabledFill = color.RGBA{R: 110, G: 110, B: 110, A: 255}
	discardBadgeDisabledInk  = color.RGBA{R: 55, G: 55, B: 55, A: 255}
)

// drawDiscardsLeft draws the badge holding the discards remaining this round.
//
// **It is drawn by the scene, over the button, rather than being a second label on the
// widget.** models.Button is a plain struct with one centred string and the count is game
// state that refills at the end of a round; giving the widget a corner-badge field would put
// a rule about this screen into something every screen shares.
//
// It has to be drawn after systems.DrawButton, which blits an opaque cached face.
func (s *CombatScene) drawDiscardsLeft(gs *state.GlobalState, screen *ebiten.Image) {
	b := s.discardButton
	if b == nil {
		return
	}

	// ScreenX/ScreenY are the button's centre; both DrawButton and the hit test re-derive the
	// corners from them the same way.
	cx := float64(b.ScreenX) + float64(b.Width)/2
	cy := float64(b.ScreenY) + float64(b.Height)/2

	fill, ink := discardBadgeFill, discardBadgeInk
	if b.State == models.ButtonStateDisabled {
		fill, ink = discardBadgeDisabledFill, discardBadgeDisabledInk
	}

	// Antialiased: this is the only circle on the screen that is not a health-bar corner, and a
	// stepped edge on a disc this size is the first thing the eye finds.
	vector.DrawFilledCircle(screen, float32(cx), float32(cy), discardBadgeRadius, fill, true)

	op := &text.DrawOptions{}
	op.GeoM.Translate(cx, cy)
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(ink)
	text.Draw(screen, fmt.Sprintf("%d", s.discardsLeft),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: discardBadgeSize}, op)
}

// enemyCardRightPct is the opponent's right edge, and it mirrors duelistCardLeftPct rather
// than being chosen: the two cards are the same object on opposite sides of the screen, so
// equal margins are the whole of what "in the corners" means.
const enemyCardRightPct = 99

// enemyCardRect is where the opponent's card sits. **The ring row ends at its left edge**,
// the same way it starts at the duelist card's right — see ringPaneRect. That replaced a
// hardcoded 79%, which was a percentage picked to clear a card whose position it could not
// see and would have gone stale the moment either moved.
func (s *CombatScene) enemyCardRect(gs *state.GlobalState) image.Rectangle {
	right, top := gs.PctX(enemyCardRightPct), gs.PctY(topRowTopPct)
	return image.Rect(right-cards.EnemyStyle.Width, top,
		right, top+cards.EnemyStyle.Height)
}

// drawEnemyCard draws the opponent in the card format: name, portrait, health bar, and the
// life left as a fraction.
//
// **It is in the top-right corner** *(2026-08-12)*, where it was centred at 88%,34% before —
// floating in the middle of the band the rings want, at a height nothing else on the screen
// shared. The corner puts it opposite the player's card and hands the whole band between them
// to the ring row.
//
// **All of it is one cached image from internal/cards**, health bar included, so there is no
// second drawing path for the contact sheet to disagree with. The cost is a re-render on
// every point of damage; see the Life field's comment there for why that is affordable.
//
// Nothing is drawn if the card cannot be built — a missing font, most likely — for the same
// reason drawCard does nothing: a card-shaped hole gets reported, a card in a fallback font
// does not.
func (s *CombatScene) drawEnemyCard(gs *state.GlobalState, screen *ebiten.Image) {
	img := cardImage(gs, enemySpec(gs, s.enemy, s.sideName(combat.SideB),
		s.shownLife(combat.SideB, s.enemy.CurrentLife)), cards.EnemyStyle)
	if img == nil {
		return
	}

	r := s.enemyCardRect(gs)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	screen.DrawImage(img, op)
}
