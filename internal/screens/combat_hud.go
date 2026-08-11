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
// holding the plan line and its action-point cost; the Resolution feed took the slot, and it
// takes its width from the hand for the same reason this did — the same band the AP bar
// spans, so the two line up on both edges however many cards are held. See combat_panes.go.

// The character block, **back in the top-left corner and stacked vertically**
// *(2026-08-11)*.
//
// It has been three shapes now and each move was forced by what claimed the space beside it.
// It was a tall box at 4%,12% while the 15–39% column was empty; it became a wide strip on
// 2026-08-07 when Action Flow claimed that column and 268px of box ran straight through the
// pane's title. **Action Flow is not drawn and Resolution has left the band entirely** — it is
// a three-line feed above the hand since 2026-08-11 — so the whole 12–46% band is free, and
// what wants it is the ring pane. A strip 35% of the screen wide sitting on top of that band
// was spending the width the rings need.
//
// So: **a narrow column in the corner holding exactly what it holds**, and no more. Three
// things stacked — the duelist's name on the box's own top edge, then Health, then Vitae —
// each a small caption over its figure. The height is derived from the row count rather than
// written down, so adding a fourth figure grows the box instead of overflowing it.
//
// It still replaces the fighter's sprite and health bar for the same reason as before: a bar
// says roughly how hurt you are, and a duel decided in whole points wants the exact number.
const (
	blockLeftPct = 1
	blockTopPct  = 2

	// **A fixed width, not a percentage.** The block is sized by the widest figure it holds —
	// "180 / 180" at 24px — and the ring pane starts where it ends, so a width that moved with
	// the screen would move the rings for no reason anyone could name.
	blockWidth = 158

	blockFirstRow  = 34 // gap from the top edge to the first caption, clearing the title
	blockRowPitch  = 46 // one caption-and-figure pair
	blockValueTop  = 18 // the figure, below its own caption
	blockBottomPad = 10
)

// blockFigure is one labelled number in the block.
//
// A nil tint means the row takes the default ink. Life is the only one that takes a colour:
// it is the number the whole duel is about, and what is beside it is a budget rather than a
// stake.
type blockFigure struct {
	label string
	value string
	tint  *color.RGBA
}

// blockFigures is what the block holds, and **the single thing its height is derived from**.
// A box sized by a constant while this list grew would clip the last figure, which looks like
// a spacing bug rather than a missing row.
//
// **Title case, not caps.** The block started out shouting HEALTH / DISCARDS / VITAE and
// `VITAE` rendered as `VITRE` — kubasta's uppercase A at 12px carries a diagonal that reads as
// an R when there is no lowercase around it to set the shape. Caps are not worth a label the
// player reads as a different word.
//
// **Discards left was the middle figure until 2026-08-11**, and moved onto the Discard button
// itself — see drawDiscardsLeft. It was the one figure here answering a question asked *while
// looking somewhere else*: it ticks down as the button is pressed, at the bottom of the
// screen, and this block is at the top. Life and vitae are read between rounds and stay.
//
// Vitae is a placeholder reading a fixed 5 — it has no rule behind it yet. It is drawn anyway
// so the block has its real shape while the rest of the character's state is decided, rather
// than being retrofitted into a box already sized without it.
func (s *CombatScene) blockFigures() []blockFigure {
	return []blockFigure{
		{"Health", fmt.Sprintf("%d / %d", s.fighter.CurrentLife, s.fighter.MaxLife), &lifeColor},
		{"Vitae", fmt.Sprintf("%d", s.vitae), nil},
	}
}

// blockHeight is the box: its title, one pitch per figure, and a margin under the last.
// Derived so the box cannot claim a height its contents have outgrown.
func blockHeight(rows int) int {
	return blockFirstRow + rows*blockRowPitch + blockBottomPad
}

// fighterBlockRect is where the block sits. **The ring pane starts from its right edge**, so
// this is the one place the block's geometry is written and both read it — see ringPaneRect.
func (s *CombatScene) fighterBlockRect(gs *state.GlobalState) image.Rectangle {
	left, top := gs.PctX(blockLeftPct), gs.PctY(blockTopPct)
	return image.Rect(left, top, left+blockWidth, top+blockHeight(len(s.blockFigures())))
}

// lifeColor is the red the life fraction is written in. It is the one place on the screen
// that has to be found without being looked for, so it gets a colour nothing else uses.
var lifeColor = color.RGBA{R: 225, G: 65, B: 65, A: 255}

// drawFighterBlock draws what the player is: the duelist's name on the box's top edge, then
// life and vitae stacked under it, each a small caption over its figure.
func (s *CombatScene) drawFighterBlock(gs *state.GlobalState, screen *ebiten.Image) {
	r := s.fighterBlockRect(gs)

	drawBox(gs, screen, r, playerSwatch, duelistName)

	small := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 12}
	value := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 24}

	// Centred on the box's own midline. The column is narrow enough that left-aligning the
	// captions and right-aligning the figures would read as two ragged edges rather than as
	// one block.
	x := float64(r.Min.X+r.Max.X) / 2

	for i, c := range s.blockFigures() {
		rowTop := r.Min.Y + blockFirstRow + i*blockRowPitch

		labelOp := &text.DrawOptions{}
		labelOp.GeoM.Translate(x, float64(rowTop))
		labelOp.PrimaryAlign = text.AlignCenter
		text.Draw(screen, c.label, small, labelOp)

		valueOp := &text.DrawOptions{}
		valueOp.GeoM.Translate(x, float64(rowTop+blockValueTop))
		valueOp.PrimaryAlign = text.AlignCenter
		if c.tint != nil {
			valueOp.ColorScale.ScaleWithColor(*c.tint)
		}
		text.Draw(screen, c.value, value, valueOp)
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

// drawEnemyCard draws the opponent in the card format: portrait, name, health bar, and the
// life left as a fraction. Centred on the point it is given, like the sprite it replaced.
//
// **All of it is one cached image from internal/cards**, health bar included, so there is no
// second drawing path for the contact sheet to disagree with. The cost is a re-render on
// every point of damage; see the Life field's comment there for why that is affordable.
//
// Nothing is drawn if the card cannot be built — a missing font, most likely — for the same
// reason drawCard does nothing: a card-shaped hole gets reported, a card in a fallback font
// does not.
func (s *CombatScene) drawEnemyCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point) {
	img := cardImage(gs, enemySpec(gs, s.enemy, s.sideName(combat.SideB)), cards.EnemyStyle)
	if img == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(
		float64(at.X-cards.EnemyStyle.Width/2),
		float64(at.Y-cards.EnemyStyle.Height/2),
	)
	screen.DrawImage(img, op)
}
