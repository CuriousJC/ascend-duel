package screens

// **The game asking "are you sure", and the third dialog shape in it.**
//
// The first two are the modal panel — the deck, the hands ladder, the bucket, the ledger — and the
// tutorial's bubble. This is a third, and it is deliberate rather than a drift *(2026-09-03)*.
//
// **A confirm is a question, not a view.** Every modalToggle takes the same near-full-screen
// footprint because what it holds is a *page* — fifty-five cards, a ladder of rungs, a run's whole
// account — and a page wants the screen. A dialog that covers the screen to ask six words reads as
// something having gone wrong, and it hides the very thing the player is being asked about. So this
// one is a small box in the middle.
//
// **It stays in the family.** Same scrim, same bevelled panel, same pink stroke, and the
// destructive answer is the modal X's red — the only red in the game, and already the colour of
// "this is the way out of here". What it does *not* borrow is the X itself: an X means "put this
// away", and a question with an X on it has three answers where it should have two.
//
// **Cancel is the safe answer and it is on the left**, where the eye lands first, with the
// destructive one at the right end of the row. Nothing here is destructive by accident: both
// callers are throwing away a climb.

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The box, and what stands in it. Sized to the longest line either caller writes rather than to a
// percentage: a question box that grew with the screen would be a page again.
const (
	confirmWidth  = 620
	confirmHeight = 240

	confirmTitleSize = 28
	confirmBodySize  = 18

	// Offsets down from the box's top edge.
	confirmTitleTop = 58
	confirmBodyTop  = 104

	// The answer row: two buttons, centred as a pair, sitting off the bottom edge.
	confirmButtonWidth  = 200
	confirmButtonHeight = 54
	confirmButtonGap    = 24
	confirmButtonBottom = 40
)

// confirmCancelColor is the safe answer: the same flat slate the chrome and the settings sliders
// take, which is this game's colour for "the program, not the fight".
var confirmCancelColor = color.RGBA{R: 92, G: 96, B: 108, A: 255}

// confirmDialog is a question with two answers.
//
// **A scene owns one and opens it by calling ask.** It builds its own buttons lazily, like every
// other widget here, and it reports through the callback it was handed rather than through a flag
// the caller has to remember to clear.
type confirmDialog struct {
	open    bool
	title   string
	body    string
	confirm string
	onYes   func()

	yes *models.Button
	no  *models.Button
}

// ask puts the question up. The label on the destructive button is the caller's, because "Abandon
// Run" and "Start Over" are different promises and a shared "OK" would make them the same one.
func (d *confirmDialog) ask(title, body, confirm string, onYes func()) {
	d.title, d.body, d.confirm, d.onYes = title, body, confirm, onYes
	d.open = true
}

// isOpen reports whether the question is up. A scene reads this to stop running everything else —
// its own rows are still where they were, and a click reaching one through a dialog would be a
// setting changed while being asked about a run.
func (d *confirmDialog) isOpen() bool { return d.open }

// close puts it away without answering.
func (d *confirmDialog) close() { d.open = false }

// update runs the two answers. It sets gs.ModalOpen while it is up, so the frame's cog and ledger
// stand down exactly as they do under every other dialog.
func (d *confirmDialog) update(gs *state.GlobalState) {
	if !d.open {
		return
	}
	if d.no == nil {
		d.no = models.NewButton(confirmButtonWidth, confirmButtonHeight, "Cancel", nil)
		d.no.BaseColor = confirmCancelColor
		d.yes = models.NewButton(confirmButtonWidth, confirmButtonHeight, "", nil)
		d.yes.BaseColor = modalCloseColor
	}

	// **Rebound every frame, because the answer changes with the question.** One dialog serves two
	// callers, so a callback wired once at build time would run the wrong caller's answer the
	// second time it was opened.
	d.yes.Text = d.confirm
	d.no.OnClick = func() { d.close() }
	yes := d.onYes
	d.yes.OnClick = func() {
		d.close()
		if yes != nil {
			yes()
		}
	}

	box := confirmRect(gs)
	row := box.Max.Y - confirmButtonBottom - confirmButtonHeight/2
	half := (confirmButtonWidth*2 + confirmButtonGap) / 2
	left := box.Min.X + box.Dx()/2 - half

	d.no.ScreenX, d.no.ScreenY = left+confirmButtonWidth/2, row
	d.yes.ScreenX = left + confirmButtonWidth + confirmButtonGap + confirmButtonWidth/2
	d.yes.ScreenY = row

	gs.ModalOpen = true
	systems.UpdateButton(gs, d.no)
	systems.UpdateButton(gs, d.yes)
}

// draw puts the scrim, the box, the words and the two answers on top of whatever the scene drew.
func (d *confirmDialog) draw(gs *state.GlobalState, screen *ebiten.Image) {
	if !d.open || d.no == nil {
		return
	}
	modalScrim(screen)

	box := confirmRect(gs)

	// Raised and pink-stroked, like every panel in front of the game. See modal.go.
	systems.BevelRect(screen, box.Min.X, box.Min.Y, box.Dx(), box.Dy(),
		systems.PaneBevelWidth, color.RGBA{R: 30, G: 30, B: 38, A: 255}, false)
	vector.StrokeRect(screen, float32(box.Min.X), float32(box.Min.Y),
		float32(box.Dx()), float32(box.Dy()), 2, apBarColor, false)

	centre := float64(box.Min.X + box.Dx()/2)

	title := &text.DrawOptions{}
	title.GeoM.Translate(centre, float64(box.Min.Y+confirmTitleTop))
	title.PrimaryAlign = text.AlignCenter
	title.SecondaryAlign = text.AlignCenter
	text.Draw(screen, d.title,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: confirmTitleSize}, title)

	body := &text.DrawOptions{}
	body.GeoM.Translate(centre, float64(box.Min.Y+confirmBodyTop))
	body.PrimaryAlign = text.AlignCenter
	body.SecondaryAlign = text.AlignCenter
	body.ColorScale.ScaleWithColor(color.RGBA{R: 198, G: 198, B: 208, A: 255})
	text.Draw(screen, d.body,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: confirmBodySize}, body)

	systems.DrawButton(gs, screen, d.no)
	systems.DrawButton(gs, screen, d.yes)
}

// confirmRect is the box: centred on the screen, at a fixed size. **Not a percentage** — see the
// note at the top of this file about a question the size of a page.
func confirmRect(gs *state.GlobalState) image.Rectangle {
	left := gs.PctX(50) - confirmWidth/2
	top := gs.PctY(50) - confirmHeight/2
	return image.Rect(left, top, left+confirmWidth, top+confirmHeight)
}
