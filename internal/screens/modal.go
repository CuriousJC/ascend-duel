package screens

// **What every modal in this game has in common**, pulled out of the deck panel on 2026-08-24 so
// that the third one did not start life as a copy of the first.
//
// There are three now — the deck, the fight log and the combos ladder — and the parts they share
// are not incidental: the footprint, the scrim, the raised panel, the heading block, the closing
// hint, and the rule that the button which opened a dialog is the button that closes it. **The
// player learns one shape.** Two dialogs at two sizes would read as two kinds of thing, and there
// is no Escape key and no right click, so a modal has to make its exit the brightest thing on
// screen or it is a trap.
//
// What is *not* here is any content. A modal is handed a body that draws inside the panel and a
// heading block saying what it is; everything about which cards, which rows or which rungs belongs
// to the panel itself.

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

// The modal footprint. Nearly the whole screen, stopping above the button band so the control
// that closes a dialog stays outside the panel as well as drawn on top of it.
//
// 92 rather than 86 at the bottom: at 86 the panel stopped short of the hand, so the tops of the
// cards and the whole AP line sat below it, dimmed by the scrim but still visibly outside the
// dialog.
const (
	modalPanelLeftPct   = 4
	modalPanelRightPct  = 96
	modalPanelTopPct    = 4
	modalPanelBottomPct = 92

	// Offsets down from the panel's top edge, and the air kept clear at the bottom.
	modalTitleTop   = 40
	modalCountsTop  = 70
	modalLegendTop  = 92
	modalBodyTop    = 112
	modalBodyBottom = 22
)

// modalPanelRect is the dialog's footprint. Every modal takes it.
func modalPanelRect(gs *state.GlobalState) image.Rectangle {
	return image.Rect(
		gs.PctX(modalPanelLeftPct), gs.PctY(modalPanelTopPct),
		gs.PctX(modalPanelRightPct), gs.PctY(modalPanelBottomPct),
	)
}

// modalScrim darkens everything, so a panel reads as covering the screen rather than floating on
// it, and so the game underneath looks as inert as it now is.
func modalScrim(screen *ebiten.Image) {
	b := screen.Bounds()
	vector.DrawFilledRect(screen, 0, 0, float32(b.Dx()), float32(b.Dy()),
		color.RGBA{A: 190}, false)
}

// modalHead is the words at the top of a panel: what it is, the figure under that, and an optional
// line explaining a state.
//
// **All three are the caller's**, because all three are facts about the screen the panel is
// standing on: a fight has three piles to report and a screen between fights has one number.
//
// **There is no closing hint any more** *(owner's call, 2026-08-24)*. Every panel carries a red X
// in its top-right corner, so the exit is a control rather than a sentence naming a control
// somewhere off the panel — see modalCloser.
type modalHead struct {
	title  string
	counts string
	legend string
}

// drawModalFrame puts up the scrim, the panel and the heading block, and hands back the rectangle
// the body is to be drawn inside.
//
// The returned rectangle is the whole panel; `modalBodyTop` is where content starts under the
// heading, and `modalBodyBottom` is what it has to stop short of at the bottom.
func drawModalFrame(gs *state.GlobalState, screen *ebiten.Image, head modalHead) image.Rectangle {
	modalScrim(screen)

	r := modalPanelRect(gs)

	// Raised, for the reason the fight log's panel is: it is in front of the game, over a scrim.
	systems.BevelRect(screen, r.Min.X, r.Min.Y, r.Dx(), r.Dy(),
		systems.PaneBevelWidth, color.RGBA{R: 30, G: 30, B: 38, A: 255}, false)
	vector.StrokeRect(screen, float32(r.Min.X), float32(r.Min.Y),
		float32(r.Dx()), float32(r.Dy()), 2, apBarColor, false)

	heading := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 28}
	title := &text.DrawOptions{}
	title.GeoM.Translate(float64(r.Min.X+r.Dx()/2), float64(r.Min.Y+modalTitleTop))
	title.PrimaryAlign = text.AlignCenter
	text.Draw(screen, head.title, heading, title)

	if head.counts != "" {
		modalLine(gs, screen, r, modalCountsTop, head.counts)
	}
	// **A legend is only written when there is a state to explain.** A sentence describing
	// something nothing on the panel is in would be the panel describing a screen it is not
	// standing on.
	if head.legend != "" {
		modalLine(gs, screen, r, modalLegendTop, head.legend)
	}
	return r
}

// modalLine writes one centred small line, at a distance down from the panel's top edge.
//
// Hyphens, not em dashes. The kubasta font has no U+2014 and draws a missing-glyph box for it.
func modalLine(gs *state.GlobalState, screen *ebiten.Image, r image.Rectangle, down int, s string) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(r.Min.X+r.Dx()/2), float64(r.Min.Y+down))
	op.PrimaryAlign = text.AlignCenter
	text.Draw(screen, s, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 14}, op)
}

// The close button: a white X on red, in the panel's top-right corner.
//
// **It replaced "press the button again to close" on 2026-08-24** *(owner's call)*. The old rule
// was that the control which opened a dialog closed it, which worked while every opener was
// visible — and stopped working the moment a panel covered its own button. The combos button sits
// above the hand, under the panel, so the only instruction on screen named a control the player
// could not see. An X on the panel itself cannot go missing, and it is the one shape every player
// already knows means "close".
//
// **Red, and the only red control in the game.** Nothing else that closes something is red, so the
// colour is not overloaded, and a dialog's exit is exactly the thing that should be the brightest
// object on a covered screen.
const (
	modalCloseSize  = 34
	modalCloseInset = 12
	modalCloseLabel = "X"
	modalCloseText  = 24
)

// modalCloseColor is the face at full strength. It rests at 65% of this, like every button; see
// the colour rule in CLAUDE.md.
var modalCloseColor = color.RGBA{R: 208, G: 52, B: 58, A: 255}

// modalCloser is the X, and it belongs to whichever panel is up.
//
// **One per dialog rather than one per screen**, except where a screen has dialogs that are not
// modalToggles: the combat screen's deck overlay and fight log share one, because only one dialog
// can ever be up and two would be two buttons in the same corner.
type modalCloser struct {
	button  *models.Button
	pressed bool
}

// update runs the X and reports whether it was pressed this frame.
func (c *modalCloser) update(gs *state.GlobalState) bool {
	if c.button == nil {
		c.button = models.NewButton(modalCloseSize, modalCloseSize, modalCloseLabel,
			func() { c.pressed = true })
		c.button.BaseColor = modalCloseColor
		c.button.TextSize = modalCloseText
	}
	r := modalPanelRect(gs)
	c.button.ScreenX = r.Max.X - modalCloseInset - modalCloseSize/2
	c.button.ScreenY = r.Min.Y + modalCloseInset + modalCloseSize/2

	c.pressed = false
	systems.UpdateButton(gs, c.button)
	return c.pressed
}

// draw puts the X on top of the panel. It is drawn only while a panel is up, because it closes
// that panel and there is nothing else for it to do.
func (c *modalCloser) draw(gs *state.GlobalState, screen *ebiten.Image) {
	if c.button != nil {
		systems.DrawButton(gs, screen, c.button)
	}
}

// The buttons that open a modal: a square carrying one character.
//
// **One character on a square**, exactly as the sort column is, because the button is too small
// for a word. The letters may not collide with each other or with the combat screen's `L`, `$`,
// `T` and `E`.
const (
	// **D is a letter and COMBOS is a word** *(owner's call, 2026-08-24)*. The corner buttons are
	// squares because they stand beside the mute button and the sort column, where there is no
	// room for a word; the combos button stands above the hand with the whole band to itself, and
	// a single `C` there said nothing to anybody who had not already opened it once.
	deckToggleLabel   = "D"
	combosToggleLabel = "COMBOS"

	// combosButtonWidth is what the word needs. Height stays the square buttons' 44, so the two
	// read as the same kind of control at different lengths.
	combosButtonWidth = 116
	combosButtonText  = 18

	// The corner they stand in on a screen with no hand, matching the mute button's inset on the
	// other side so the two bottom corners share a line. See internal/game/chrome.go.
	modalToggleInset = 10

	// The gap between two toggles standing together, wide enough that they do not read as one
	// widget. The same 10 the corner inset is.
	modalToggleGap = 10
)

// modalToggle is a button, whether its panel is up, and the tooltip that panel needs.
//
// **A struct rather than three fields on each scene**, because the three go together and the
// failure of letting them drift apart is silent: a scene that forgets `gs.ModalOpen` leaves the
// frame's mute button live on top of a dialog whose whole design is that one control is lit.
//
// **It knows nothing about what it opens.** The panel's contents reach it as two closures at the
// call site — one to point the tooltip, one to draw — which is what let a second modal be added
// without this growing a second content type.
type modalToggle struct {
	open   bool
	button *models.Button
	tip    models.Tooltip

	// place is where the button's centre goes, asked every frame. Nil means the bottom-right
	// corner; a screen that has somewhere better says so.
	place func(gs *state.GlobalState) image.Point

	// blocked is set by the scene while some *other* dialog is up, and it takes this button out
	// of the frame entirely — neither run nor drawn.
	//
	// **A screen may have exactly one live exit.** Two dialogs each carrying a live button means a
	// player can open the second through the first, and a dialog whose exit is not the brightest
	// thing on screen is a trap.
	blocked bool

	// closer is the X on this toggle's own panel — the only thing that closes it.
	closer modalCloser
}

// block takes the button out of the frame while another dialog is up. Called every tick from the
// scene, never latched, so a panel that closes cannot leave its neighbour dead.
func (t *modalToggle) block(b bool) { t.blocked = b }

// init wires the button. **The button survives a re-entry and the state does not** — a scene's
// Init runs again on every visit, and arriving at a shop with a panel already open would be a
// dialog nobody asked for.
func (t *modalToggle) init(label string, w, h int, textSize float64,
	place func(gs *state.GlobalState) image.Point) {

	if t.button == nil {
		t.button = models.NewButton(w, h, label, func() { t.open = !t.open })
		t.button.BaseColor = sortButtonColor
		t.button.TextSize = textSize
	}
	t.place = place
	t.open = false
	t.tip = models.Tooltip{DwellTicks: tipDwell}
}

// cornerSlot is the bottom-right corner, n places in from it. Slot 0 is the corner itself.
func cornerSlot(n int) func(*state.GlobalState) image.Point {
	return func(gs *state.GlobalState) image.Point {
		step := logButtonSize + modalToggleGap
		return image.Pt(
			gs.PctX(100)-modalToggleInset-logButtonSize/2-n*step,
			gs.PctY(100)-modalToggleInset-logButtonSize/2,
		)
	}
}

// combosCornerPlace is where the combos button stands on a screen with no hand: to the left of the
// deck button, sharing its bottom line. **Measured from the corner rather than from a slot index**,
// because the two buttons are different widths and a slot walk would assume they are not.
func combosCornerPlace(gs *state.GlobalState) image.Point {
	right := gs.PctX(100) - modalToggleInset - logButtonSize - modalToggleGap
	return image.Pt(right-combosButtonWidth/2,
		gs.PctY(100)-modalToggleInset-logButtonSize/2)
}

// update runs the button and, while the panel is up, whatever that panel puts under the cursor.
//
// **It returns whether the panel is covering the screen**, which is the caller's cue to stop
// running everything else: the scene's own rows are still where they were, and a click reaching
// one through a dialog would be a ring bought while reading a deck.
func (t *modalToggle) update(gs *state.GlobalState,
	hover func(at image.Point, tip *models.Tooltip)) bool {

	// **The frame the panel is closed on is still a covered frame.** The press that closes it is
	// the same press the scene's rows would see, so a scene told the panel is down on that frame
	// takes a click the player spent on the exit.
	was := t.open

	if t.blocked {
		return was || t.open
	}

	place := t.place
	if place == nil {
		place = cornerSlot(0)
	}
	c := place(gs)
	t.button.ScreenX, t.button.ScreenY = c.X, c.Y
	t.button.Latched = t.open

	// **While the panel is up the opener is inert and the X is the exit.** The opener stays on
	// screen, latched, so the player can see which control the panel came out of — it simply is
	// not a second way out.
	if t.open {
		if t.closer.update(gs) {
			t.open = false
		}
	} else {
		systems.UpdateButton(gs, t.button)
	}

	// **The tooltip is pointed only while the panel is up.** UpdateTooltip releases it by itself
	// on any frame nothing was pointed at, so a closed panel needs no clearing of its own.
	if t.open {
		gs.ModalOpen = true
		if hover != nil {
			hover(image.Pt(gs.MouseX, gs.MouseY), &t.tip)
		}
	}
	systems.UpdateTooltip(gs, &t.tip)
	return was || t.open
}

// draw puts the panel up if it is open, and the button on top of it either way.
// **The opener is drawn under the panel, never on top of it** *(owner's call, 2026-08-24)*. It was
// drawn last, from when pressing it again was how a dialog closed; now that the X is the exit, a
// button standing over the panel would be a control that looks live and is not.
func (t *modalToggle) draw(gs *state.GlobalState, screen *ebiten.Image, body func()) {
	if !t.blocked {
		systems.DrawButton(gs, screen, t.button)
	}
	if !t.open {
		return
	}
	body()
	if t.blocked {
		return
	}
	t.closer.draw(gs, screen)
	systems.DrawTooltip(gs, screen, &t.tip)
}
