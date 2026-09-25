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
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"image/color"
)

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
// damage a Bash does in these hands, and the action points the round is bought with — and a
// health bar above the fraction, matching the enemy's at the same offsets so the two can be
// compared across the screen without measuring.
//
// The card's own layout lives in cards.DuelistStyle; what is here is where it sits and what
// goes on it.
const ()

// drawDuelistCard draws what the player is: name, DMG, AP, Vitae, and life as a bar over a
// fraction.
//
// One cached image from internal/cards, like the enemy's, so the contact sheet draws the same
// card the screen does. Nothing is drawn if it cannot be built — a missing font, most likely —
// for the same reason drawCard does nothing.
func (s *CombatScene) drawDuelistCard(gs *state.GlobalState, screen *ebiten.Image) {
	img := ui.CardImage(gs,
		ui.DuelistSpec(gs, s.fighter, s.sideName(combat.SideA),
			s.shownDMG(combat.SideA, s.fighter.DMG),
			s.shownVitae(gs.Run.Vitae()),
			s.shownLife(combat.SideA, s.fighter.CurrentLife),
			s.shownMaxLife(combat.SideA, s.fighter.MaxLife),
			s.fighter.ActionPoints(),
			s.fightIndex,
			s.shownShields(combat.SideA, s.fighter.Shields.Count()),
			s.shownShieldElements(combat.SideA)...),
		cards.DuelistStyle)
	if img == nil {
		return
	}

	r := ui.DuelistCardRect(gs)
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
	// **towerLineGap is the drop from the duelist card's bottom edge to whatever hangs off it**,
	// which is the round timer and nothing else now. The floor and the room stood here — in a
	// column beside the card for a day, then back under it — and moved **onto** the card on
	// 2026-09-15 *(owner's call)*: they are facts about the duelist, and the one place on this
	// screen a fact about the duelist was not written on the duelist was the two lines below it.
	// See cards.DuelistStyle, where they are stat rows now.
	towerLineGap = 10
)

// The discards-left badge: a filled disc centered exactly on the Discard button's bottom-right
// corner, with the count in it.
//
// **Centered on the corner rather than inset from it**, so a quarter of the disc hangs off each
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
// it changes color entirely. `disabledButtonColor` is flat dark gray and deliberately ignores
// BaseColor, so a badge tuned to sit on yellow has nothing to say about gray. Disabled takes a
// mid gray disc: still a disc, plainly not a live one.
var (
	discardBadgeFill         = color.RGBA{R: 245, G: 245, B: 245, A: 255}
	discardBadgeInk          = color.RGBA{R: 25, G: 25, B: 25, A: 255}
	discardBadgeDisabledFill = color.RGBA{R: 110, G: 110, B: 110, A: 255}
	discardBadgeDisabledInk  = color.RGBA{R: 55, G: 55, B: 55, A: 255}
)

// drawDiscardsLeft draws the badge holding the discards remaining this round.
//
// **It is drawn by the scene, over the button, rather than being a second label on the
// widget.** models.Button is a plain struct with one centered string and the count is game
// state that refills at the end of a round; giving the widget a corner-badge field would put
// a rule about this screen into something every screen shares.
//
// It has to be drawn after systems.DrawButton, which blits an opaque cached face.
func (s *CombatScene) drawDiscardsLeft(gs *state.GlobalState, screen *ebiten.Image) {
	b := s.discardButton
	if b == nil {
		return
	}

	// ScreenX/ScreenY are the button's center; both DrawButton and the hit test re-derive the
	// corners from them the same way.
	cx := float64(b.ScreenX) + float64(b.Width)/2
	cy := float64(b.ScreenY) + float64(b.Height)/2

	fill, ink := discardBadgeFill, discardBadgeInk
	if b.State == models.ButtonStateDisabled {
		fill, ink = discardBadgeDisabledFill, discardBadgeDisabledInk
	}

	// Antialiased: this is the only circle on the screen that is not a health-bar corner, and a
	// stepped edge on a disc this size is the first thing the eye finds.
	vector.FillCircle(screen, float32(cx), float32(cy), discardBadgeRadius, fill, true)

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

// drawEnemyCard draws the opponent in the card format: name, portrait, health bar, and the
// life left as a fraction.
//
// **It is in the top-right corner** *(2026-08-12)*, where it was centered at 88%,34% before —
// floating in the middle of the band the relics want, at a height nothing else on the screen
// shared. The corner puts it opposite the player's card and hands the whole band between them
// to the relic row.
//
// **All of it is one cached image from internal/cards**, health bar included, so there is no
// second drawing path for the contact sheet to disagree with. The cost is a re-render on
// every point of damage; see the Life field's comment there for why that is affordable.
//
// Nothing is drawn if the card cannot be built — a missing font, most likely — for the same
// reason drawCard does nothing: a card-shaped hole gets reported, a card in a fallback font
// does not.
func (s *CombatScene) drawEnemyCard(gs *state.GlobalState, screen *ebiten.Image) {
	img := ui.CardImage(gs, ui.EnemySpec(gs, s.enemy, s.sideName(combat.SideB),
		s.shownLife(combat.SideB, s.enemy.CurrentLife)), cards.EnemyStyle)
	if img == nil {
		return
	}

	r := ui.EnemyCardRect(gs)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	screen.DrawImage(img, op)
}

// **The round timer: how much of the fight is left to spend.**
//
// Every duel is five rounds long and the duelist still standing at the end of the fifth dies —
// see MECHANICS.md §The round limit, and `internal/combat/clock.go`, which is what actually ends
// it. This is the readout, and it decides nothing: the bar is a picture of `s.round` and the
// clock is checked inside the resolved round, exactly as the presentation-may-never-change-an-
// outcome rule requires.
//
// **It is under the tower place because it is the same kind of fact.** Where you are and how long
// you have got are both the frame around the fight rather than parts of it, and the left column is
// already one thing: who you are, where you are, what is left to draw. It goes below the two lines
// rather than beside them because the column is only 200 pixels wide.
//
// **It is segments, not a sliding fill, and that is the whole design.** A round is a discrete
// thing the player spends, so what they need read off the bar is a count — three cells dark, two
// left — rather than a proportion they have to convert. It is also what keeps the readout honest
// at a limit a relic has moved: six cells is six rounds, with nothing to rescale.
const (
	// roundTimerGap is the drop from the tower lines to the bar, and roundTimerHeight is how tall
	// it is. **There are 23 pixels between the tower lines and the table row** and these spend 20
	// of them; a taller bar collides, which TestTheRoundTimerFitsUnderTheTowerLines is what says.
	roundTimerGap    = 6
	roundTimerHeight = 14

	// roundTimerCellGap is the bare pixel between one cell and the next. Enough to count them by,
	// and small enough that five of them still read as one bar rather than as five objects.
	roundTimerCellGap = 2
)

// roundTimerRect is the bar's whole footprint: the duelist card's column, hung straight off the
// bottom of the card.
//
// **It used to hang off the floor-and-room lines, which are on the card now** — so the bar moved up
// into the space they left rather than a gap being kept where they were. See towerLineGap.
func (s *CombatScene) roundTimerRect(gs *state.GlobalState) image.Rectangle {
	card := ui.DuelistCardRect(gs)
	top := card.Max.Y + towerLineGap
	return image.Rect(card.Min.X, top, card.Max.X, top+roundTimerHeight)
}

// roundTimerLimit is how many rounds this fight gets, or zero for a fight on no clock at all.
//
// **It is read off the fighter rather than off the run**, because the fighter is what the rules
// will actually check — the run's number reaches a duel through `Equip` and a bar reading the run
// directly would keep drawing five while the duelist fought to some other figure. A run-less
// screen — a test, the scripted demo — draws nothing.
// **A screen with no fighter has no clock**, which is a test or the frame before `Init` rather
// than a state the game plays in — but the tutorial's anchor lookup reaches this from both.
func (s *CombatScene) roundTimerLimit() int {
	if s.fighter == nil {
		return 0
	}
	return s.fighter.RoundLimit
}

// roundTimerSpent is how many rounds of the limit are gone, clamped into the bar.
//
// **`s.round` is rounds *resolved*, so it is zero while the first one is still being planned** —
// which is right: the player has spent nothing yet. It is clamped because a fight can be looked at
// for a frame after the round that timed it out, and a sixth filled cell on a five-cell bar would
// be drawn outside the rectangle.
func (s *CombatScene) roundTimerSpent(limit int) int {
	switch {
	case s.round < 0:
		return 0
	case s.round > limit:
		return limit
	}
	return s.round
}

// drawRoundTimer draws one cell per round, filled for the rounds already spent.
//
// **The last cell is the game's one red** — `modalCloseColor`, the color the destructive answer
// on a confirm dialog takes — and it is lit only once the fight has actually reached it. There is
// no hue left to claim (see CLAUDE.md), and this is not claiming one: it is the existing meaning
// of that red, which is "this ends something", arriving at the moment it becomes true.
func (s *CombatScene) drawRoundTimer(gs *state.GlobalState, screen *ebiten.Image) {
	limit := s.roundTimerLimit()
	if limit < 1 {
		return // a fight on no clock has no bar, rather than a full one or an empty one
	}

	r := s.roundTimerRect(gs)
	spent := s.roundTimerSpent(limit)

	// **The cells are laid out by their own edges rather than by a width times an index**, so the
	// rounding left over from dividing 200 pixels by five lands in the gaps instead of leaving the
	// last cell short of the column it is supposed to end at.
	for i := 0; i < limit; i++ {
		x0 := r.Min.X + i*r.Dx()/limit
		x1 := r.Min.X + (i+1)*r.Dx()/limit - roundTimerCellGap
		if x1 <= x0 {
			continue // a limit high enough that a cell is thinner than its own gap draws nothing
		}

		// An unspent round is the ground's ink at a quarter strength: present enough to be
		// counted, quiet enough not to read as a round already gone. `ColorToward` rather than
		// `ColorAtStrength`, because the table is light — see CLAUDE.md.
		fill := systems.ColorToward(ui.GroundInk, ui.ScreenGround, 75)
		if i < spent {
			fill = ui.GroundInk
			if i == limit-1 {
				fill = ui.ModalCloseColor
			}
		}

		systems.BevelRect(screen, x0, r.Min.Y, x1-x0, r.Dy(),
			systems.PaneBevelWidth, fill, i >= spent)
	}
}
