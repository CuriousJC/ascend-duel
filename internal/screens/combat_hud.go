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

// The character block, **a strip across the top** *(moved there 2026-08-07)*.
//
// It was a tall box at 4%,12% for as long as the 15–39% column beside it was empty. Action
// Flow claimed that column, and 268px of box starting at 51px ran straight through a pane
// starting at 192px — the block was drawn on top, eating the pane's title and its first rows.
//
// **The band above the panes was dead space and this is what it is for.** Laid out
// horizontally rather than squeezed narrower: three labelled figures side by side, which reads
// at a glance in a way a stack of rows in a thin column would not. It still replaces the
// fighter's sprite and health bar for the same reason as before — a bar says roughly how hurt
// you are, and a duel decided in whole points wants the exact number.
//
// The right edge lines up with Action Flow's, so the strip caps the left column rather than
// floating over the middle of the screen.
const (
	blockLeftPct  = 4
	blockRightPct = 39
	blockTopPct   = 2

	blockHeight = 88

	blockLabelTop = 38 // small caption above each figure
	blockValueTop = 56 // the figure itself
)

// lifeColor is the red the life fraction is written in. It is the one place on the screen
// that has to be found without being looked for, so it gets a colour nothing else uses.
var lifeColor = color.RGBA{R: 225, G: 65, B: 65, A: 255}

// drawFighterBlock draws what the player is: life and vitae.
//
// **Discards left was the middle figure until 2026-08-11**, and moved onto the Discard button
// itself — see drawDiscardsLeft. It was the one figure in the strip that answers a question
// asked *while looking somewhere else*: it ticks down as the button is pressed, at the bottom
// of the screen, and the strip is at the top. Life and vitae are read between rounds and stay.
//
// Vitae is a placeholder reading a fixed 5 — it has no rule behind it yet. It is drawn
// anyway so the block has its real shape while the rest of the character's state is
// decided, rather than being retrofitted into a box already sized without it.
func (s *CombatScene) drawFighterBlock(gs *state.GlobalState, screen *ebiten.Image) {
	left, right := gs.PctX(blockLeftPct), gs.PctX(blockRightPct)
	top := gs.PctY(blockTopPct)
	r := image.Rect(left, top, right, top+blockHeight)

	drawBox(gs, screen, r, playerSwatch, duelistName)

	small := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 12}
	value := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 24}

	// Evenly spaced columns, each a small caption over its figure. Life is the only one that
	// takes a colour: it is the number the whole duel is about, and what is beside it is a
	// budget rather than a stake. The spacing is derived from len(cells), so dropping Discards
	// re-centred the remaining two rather than leaving a hole where it stood.
	cells := []struct {
		label string
		value string
		tint  *color.RGBA
	}{
		// **Title case, not caps.** The strip started out shouting HEALTH / DISCARDS / VITAE
		// and `VITAE` rendered as `VITRE` — kubasta's uppercase A at 12px carries a diagonal
		// that reads as an R when there is no lowercase around it to set the shape. The old
		// tall block used title case and never had the problem. Caps are not worth a label
		// the player reads as a different word.
		{"Health", fmt.Sprintf("%d / %d", s.fighter.CurrentLife, s.fighter.MaxLife), &lifeColor},
		{"Vitae", fmt.Sprintf("%d", s.vitae), nil},
	}

	width := right - left
	for i, c := range cells {
		// Centres at one sixth, one half and five sixths, so the columns sit in their own
		// thirds rather than being packed against the left edge.
		x := float64(left) + float64(width)*(float64(2*i+1)/float64(2*len(cells)))

		labelOp := &text.DrawOptions{}
		labelOp.GeoM.Translate(x, float64(top+blockLabelTop))
		labelOp.PrimaryAlign = text.AlignCenter
		text.Draw(screen, c.label, small, labelOp)

		valueOp := &text.DrawOptions{}
		valueOp.GeoM.Translate(x, float64(top+blockValueTop))
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
