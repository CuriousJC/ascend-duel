package screens

// The HUD: the things around the round rather than the round itself. The character strip, the
// caption box, the generic framed box every panel is built from, the enemy's sprite and both
// health bars.
//
// Split out of combat.go on 2026-08-07. It is the most self-contained group on the screen —
// nothing here reads the log, the deck or the queue, so it is the part that can be moved
// without any of the rest being understood.

import (
	"fmt"
	"image"

	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
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

// The caption box sits between the Resolution pane and the hand, and takes its width from
// the hand rather than from the pane above it — the same band the AP bar spans, so the two
// line up on both edges however many cards are held.
const (
	captionTopPct = 48
	captionHeight = 56
)

// drawCaptionBox says what to do next. It used to narrate the round one event at a time as
// well; that job moved to the Resolution pane on 2026-08-07, leaving this with the plan line,
// its action-point cost, and what to press when a fight ends.
//
// It keeps the slot directly above the hand because that is what it is about: the cards you
// are choosing and what they cost. The pane above reports the past, this proposes the future.
func (s *CombatScene) drawCaptionBox(gs *state.GlobalState, screen *ebiten.Image) {
	band := handBand(gs, s.laidOutCount())
	top := gs.PctY(captionTopPct)
	r := image.Rect(band.Min.X, top, band.Max.X, top+captionHeight)

	drawBox(gs, screen, r, resolutionPane.color, "")

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(r.Min.X+r.Dx()/2), float64(r.Min.Y+r.Dy()/2))
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	text.Draw(screen, s.caption(),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 18}, op)
}

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

// drawCombatant draws one duelist and its health bar. Only the enemy uses it now — the
// fighter's sprite and bar became the character block — but it stays shaped for either.
func (s *CombatScene) drawCombatant(gs *state.GlobalState, screen *ebiten.Image, c *entities.Combatant, hPosition, vPosition float64) {
	var cm colorm.ColorM

	op := &colorm.DrawImageOptions{}
	op.GeoM.Translate(-float64(c.Sprite.Bounds().Dx())/2, -float64(c.Sprite.Bounds().Dy())/2) //center our origin
	op.GeoM.Translate(hPosition, vPosition)                                                   //position
	colorm.DrawImage(screen, c.Sprite, cm, op)

	DrawHealthBar(gs, screen, hPosition, vPosition, c.CurrentLife, c.MaxLife)
}

// The health bar's fixed size. Package constants rather than locals so the scratch images
// below can be allocated against them.
const (
	rectWidth  = 150
	rectHeight = 25
)

// The two scratch images DrawHealthBar composites through, allocated once and reused.
//
// **This used to be two `ebiten.NewImage` calls per bar per frame** — 120 new GPU textures a
// second for a picture that changes only when someone takes damage. Both are fully
// overwritten at the top of every call (`Fill` on each, then the mask redrawn), so reuse
// cannot leak last frame's contents.
//
// Package-level mutable state is safe here specifically because Ebitengine calls Draw from
// one goroutine, and because each call finishes compositing into the screen before it
// returns — so two bars drawn in the same frame take turns rather than fighting.
var healthBarMask, healthBarFill *ebiten.Image

// healthBarScratch returns the pair, allocating them on first use. Not built at package
// init: ebiten.NewImage before the game loop is running is a rule worth not testing.
func healthBarScratch() (mask, fill *ebiten.Image) {
	if healthBarMask == nil {
		healthBarMask = ebiten.NewImage(rectWidth, rectHeight)
		healthBarFill = ebiten.NewImage(rectWidth, rectHeight)
	}
	return healthBarMask, healthBarFill
}

func DrawHealthBar(gs *state.GlobalState, screen *ebiten.Image,
	hPositionEntity float64, vPositionEntity float64,
	currentLife int, maxLife int) {

	hPosition := hPositionEntity
	vPosition := vPositionEntity + 100                      //move down 100 px from the position
	rectColor := color.RGBA{A: 255, R: 96, G: 37, B: 37}    // Crimson red
	trackColor := color.RGBA{A: 255, R: 38, G: 22, B: 22}   // the drained portion behind the fill
	maskColor := color.RGBA{A: 255, R: 255, G: 192, B: 203} // mask color
	maskFill := color.RGBA{A: 0, R: 255, G: 255, B: 255}    //fill transparent

	mask, lifeRect := healthBarScratch()

	//Rounded corners mask Image
	mask.Fill(maskFill)
	CreateRoundedRecMask(mask, 0, 0, float32(rectWidth), float32(rectHeight), 10, maskColor)

	//Track plus the red fill, scaled to the current life fraction
	lifeRect.Fill(trackColor)
	if maxLife > 0 && currentLife > 0 {
		fillWidth := float32(rectWidth) * float32(currentLife) / float32(maxLife)
		vector.DrawFilledRect(lifeRect, 0, 0, fillWidth, float32(rectHeight), rectColor, false)
	}

	//Blend the life rectangle into the mask.  The source is the life rectangle
	//  and the mask will only display that portion of the source that is blended into the non-transparent alpha
	rectMaskOp := &ebiten.DrawImageOptions{}
	rectMaskOp.Blend = ebiten.BlendSourceIn
	mask.DrawImage(lifeRect, rectMaskOp) //blend source lifeRect into destination mask
	healthBar := mask                    //mask is now filled with transparent maskFill and the maskColor was overwritten with lifeRect

	healthBarOp := &ebiten.DrawImageOptions{}
	healthBarOp.GeoM.Translate(-float64(rectWidth)/2, -float64(rectHeight)/2) //center our origin
	healthBarOp.GeoM.Translate(hPosition, vPosition)                          //position
	screen.DrawImage(healthBar, healthBarOp)

	// The figure, in the same "60 / 60" shape and the same red the fighter's block uses, so
	// both duelists state their life the same way even though only one of them has a bar.
	// Onto the screen after the bar rather than into the mask before it: the mask is
	// composited with BlendSourceIn, which would eat anything drawn into it first.
	lifeOp := &text.DrawOptions{}
	lifeOp.GeoM.Translate(hPosition, vPosition)
	lifeOp.PrimaryAlign = text.AlignCenter
	lifeOp.SecondaryAlign = text.AlignCenter
	lifeOp.ColorScale.ScaleWithColor(lifeColor)
	text.Draw(screen, fmt.Sprintf("%d / %d", currentLife, maxLife),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}, lifeOp)
}

func CreateRoundedRecMask(mask *ebiten.Image, x, y, width, height, radius float32, maskColor color.Color) {
	//Round Corners
	vector.DrawFilledCircle(mask, x+radius, y+radius, radius, maskColor, false)
	vector.DrawFilledCircle(mask, x+width-radius, y+radius, radius, maskColor, false)
	vector.DrawFilledCircle(mask, x+radius, y+height-radius, radius, maskColor, false)
	vector.DrawFilledCircle(mask, x+width-radius, y+height-radius, radius, maskColor, false)

	//Rectangle Edges
	vector.DrawFilledRect(mask, x+radius, y, width-2*radius, radius, maskColor, false)               //top edge
	vector.DrawFilledRect(mask, x+radius, y+height-radius, width-2*radius, radius, maskColor, false) //bottom edge
	vector.DrawFilledRect(mask, x, y+radius, radius, height-2*radius, maskColor, false)              //left edge
	vector.DrawFilledRect(mask, x+width-radius, y+radius, radius, height-2*radius, maskColor, false) //right edge
}
