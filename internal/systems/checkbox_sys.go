package systems

// The behavior half of models.Checkbox: a click that asks for the other state, and the box and its
// mark drawn from authored art.
//
// **The art is two layers on one canvas**: a box in a stone and a mark that lands on it, each a
// 256 square with the box at its center (docs/art/checkbox_art_prompt.MD), so the mark is drawn over
// the box at the same place and size with nothing nudged.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// The checkbox art's keys: the box live, the box disabled, and the mark.
const (
	checkboxBoxKey  = "checkbox-jade"
	checkboxDeadKey = "checkbox-disabled"
	checkboxMarkKey = "checkmark-check-inside"
)

// checkboxBoxShare is how much of a checkbox file's canvas the box takes: 164 of 256.
const checkboxBoxShare = 164.0 / 256

// checkboxPopTicks is how long a mark takes to settle after it lands, and checkboxPopSwell how
// large it lands. A mark that appears at its resting size between two frames reads as a redraw
// rather than as a tick.
const (
	checkboxPopTicks = 14
	checkboxPopSwell = 0.35
)

// CheckboxRect is the box itself, the part that takes a click.
func CheckboxRect(c *models.Checkbox) image.Rectangle {
	return image.Rect(c.ScreenX-c.Size/2, c.ScreenY-c.Size/2, c.ScreenX+c.Size/2, c.ScreenY+c.Size/2)
}

// UpdateCheckbox runs one frame of a checkbox. **A click is a press and a release on the box**,
// Button's rule, so a misclick can be taken back by dragging off before letting go.
func UpdateCheckbox(gs *state.GlobalState, c *models.Checkbox) {
	if c.MarkArrived() {
		c.TickAge = 0
	} else if c.TickAge < checkboxPopTicks {
		c.TickAge++
	}
	if c.Disabled {
		c.Hovered, c.PressedInside = false, false
		return
	}
	over := image.Pt(gs.MouseX, gs.MouseY).In(CheckboxRect(c)) && gs.CursorAllowed()
	c.Hovered = over
	if over && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		c.PressedInside = true
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		if over && c.PressedInside && c.OnToggle != nil {
			c.OnToggle(!c.Checked)
		}
		c.PressedInside = false
	}
}

// DrawCheckbox draws the box and, if it is ticked, its mark — popping in large and settling. Hover
// brightens the box and a press darkens it, the button faces' states.
func DrawCheckbox(screen *ebiten.Image, c *models.Checkbox) {
	canvas := float64(c.Size) / checkboxBoxShare
	px := int(canvas + 0.5)

	box := checkboxBoxKey
	if c.Disabled {
		box = checkboxDeadKey
	}
	op := checkboxOp(c, canvas, 1)
	switch {
	case c.PressedInside && c.Hovered:
		op.ColorScale.Scale(0.85, 0.85, 0.85, 1)
	case c.Hovered:
		op.ColorScale.Scale(1.12, 1.12, 1.12, 1)
	}
	if img := ArtMarkImage(box, px, px); img != nil {
		screen.DrawImage(img, op)
	}

	if !c.Checked {
		return
	}
	t := float64(c.TickAge) / checkboxPopTicks
	if t > 1 {
		t = 1
	}
	swell := 1 + checkboxPopSwell*(1-t)*(1-t)
	op = checkboxOp(c, canvas, swell)
	op.ColorScale.ScaleAlpha(float32(min(1, 2*t+0.2)))
	if c.Disabled {
		op.ColorScale.ScaleAlpha(0.5)
	}
	if img := ArtMarkImage(checkboxMarkKey, px, px); img != nil {
		screen.DrawImage(img, op)
	}
}

// checkboxOp places a canvas-sized layer centered on the box, scaled again by swell.
func checkboxOp(c *models.Checkbox, canvas, swell float64) *ebiten.DrawImageOptions {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-canvas/2, -canvas/2)
	op.GeoM.Scale(swell, swell)
	op.GeoM.Translate(float64(c.ScreenX), float64(c.ScreenY))
	return op
}
