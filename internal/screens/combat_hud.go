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

// drawFighterBlock draws what the player is: life, discards left this round, and vitae.
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

	// Three evenly spaced columns, each a small caption over its figure. Life is the only one
	// that takes a colour: it is the number the whole duel is about, and the other two are
	// budgets rather than stakes.
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
		{"Discards", fmt.Sprintf("%d", s.discardsLeft), nil},
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
