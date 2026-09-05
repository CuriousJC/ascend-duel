package game

// The game's chrome: what is drawn around every screen rather than by one.
//
// **There are two things in it, and the bar for a third is the same as it was for these.** A frame is easy to
// grow by accident, and the bar for joining it is high: something that is true for the whole
// session, on every screen, and owned by no scene. The settings qualify — the score is started
// once in main and loops across the title, the tower, the duel and the credits, and the game's
// one clock is the same number on every screen — so the control that opens them does too.
// Anything that belongs to a screen belongs to that screen.
//
// **It was the mute button until 2026-08-27** and is now the door to the settings screen. Mute
// stopped existing when the volume bar arrived: a latch and a bar are two controls over one
// number and would have had to be kept from disagreeing, so zero on the bar is the only silence
// there is. What the corner lost is the one-click silence; what it gained is a place to put the
// game speed, which had no control at all.
//
// **The ledger joined it on 2026-09-02** *(owner's call)*: the run's account of itself is true for
// the whole run, wanted on every screen, and owned by no scene — the same three tests the settings
// pass. It is also the one thing here that *could* not have been a screen: leaving the combat
// screen and coming back re-runs Init, which deals a fresh duel, so a ledger that navigated would
// destroy the fight it was opened to read. See internal/screens/ledger.go.
//
// See CLAUDE.md on the input vocabulary: this is a click on a button. **Escape is the one hotkey
// in the game** *(owner's call, 2026-09-05)*, and it does exactly what the cog does and nothing
// else: it is a second way to press one button, not a second way to do something. It is live only
// while the cog itself is — same predicate, same input gate — so a key cannot reach a screen where
// the button has stood down.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/screens"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"image/color"
)

// The settings button: a small square in the bottom-right corner of the screen, carrying a cog.
//
// **Square and iconic because it is not a move.** DUEL! and Discard are 138x50 with a word on
// them because they are choices inside a duel and have to be found and read; this changes
// nothing about a run, and a control that took the same footprint would be claiming the same
// weight. A glyph is smaller, says one thing, and is what the repo already generates for
// interface art — see internal/systems/glyphs.go.
//
// **Inset from both edges rather than flush**, which is what stops it reading as something that
// fell off the screen.
//
// **It moved corners on 2026-09-04** *(owner's call)*: bottom-left, where it had stood since the
// frame existed, is the draw pile's now. The right-hand corner is under the combat screen's
// control column and below the last button in it, so the cog sits off the end of that stack rather
// than in it — which is what a control that is the program rather than the fight should look like.
//
// **Its right edge is the column's, not the screen's.** A corner inset of its own would put it a
// few pixels off the line every other control on that side stands on, which reads as a mistake
// rather than as a margin. See screens.ControlColumnLeft.
const (
	settingsButtonSize  = 44
	settingsButtonInset = 10

	// **`LEDGER`, spelled out and in caps** *(2026-09-04, owner's call)*. It was `L` — what the
	// fight log's button carried, and what a 44-pixel square can hold. It stands in the combat
	// screen's control column now and pairs with HANDS directly above it, which has been in caps
	// since it was written; the two open a page over the game and are the only two controls on
	// that side that do.
	//
	// **Caps at 18 are checked rather than assumed** — see CLAUDE.md, where VITAE rendered as
	// VITRE at 12. Both words are legible at this size on the contact sheet.
	//
	// The size and the width are the column's own, so the pair is set in one type.
	ledgerButtonLabel = "LEDGER"
)

// settingsButtonColor is the face at full strength: a flat slate, deliberately unlike any control
// that plays the game.
//
// DUEL! is crimson and Discard is yellow because they are choices inside a duel. This is not a
// move — it changes nothing about a run and never becomes unavailable for a game reason — so
// it takes a colour that says "this is the program, not the fight". Same argument as the ring
// row's backing being grey rather than pink, and the settings screen's sliders share it.
var settingsButtonColor = color.RGBA{R: 92, G: 96, B: 108, A: 255}

// openSettings goes to the settings screen, remembering where the player was.
//
// **The run is not touched.** Settings is not a station of a run — see session.Phase — so the
// phase stays exactly where it stood and Back puts the player back on the screen it was drawing.
func (g *Game) openSettings() {
	gs := g.GlobalState
	gs.ReturnScreen = gs.ActiveScreen
	gs.ActiveScreen = state.Settings
	gs.NewScreen = true
}

// toggleLedger opens or closes the run's account. **It changes no phase and no screen** — see
// screens.LedgerPanel, and openSettings above, which makes the same promise for the same reason.
func (g *Game) toggleLedger() { g.ledger.Toggle() }

// ledgerButtonRect is where the ledger's button sits: **the last rung of the combat screen's
// control column**, under the hands button *(2026-09-04, owner's call)*.
//
// **The frame borrows a scene's column, which is the one thing here worth arguing with.** The
// ledger belongs to no screen — that is what makes it chrome — and it is now placed by geometry
// the combat screen owns. What makes it the right trade is that the alternative was a corner of
// its own: a button that stands somewhere else on every screen but combat is a button the player
// has to find twice, and the run's account is wanted most where the run is being fought. The
// column is a fixed set of percentages, so it is a real place on every screen rather than a
// rectangle that only means something during a duel.
func ledgerButtonRect(gs *state.GlobalState) image.Rectangle {
	return screens.ControlColumnSlot(gs, screens.SlotLedger)
}

// settingsButtonRect is where the button sits: the bottom-right corner, its right edge on the
// control column's line and its bottom inset from the screen's edge.
//
// Both figures are read off the live screen dimensions rather than the ScreenWidth and
// ScreenHeight constants, so a change to the internal resolution moves the button rather than
// leaving it stranded in the middle.
func settingsButtonRect(gs *state.GlobalState) image.Rectangle {
	right := screens.ControlColumnLeft(gs) + screens.ControlColumnWidth()
	top := gs.ScreenHeight - settingsButtonInset - settingsButtonSize
	return image.Rect(right-settingsButtonSize, top, right, top+settingsButtonSize)
}

// chromeShowing reports whether the frame is drawn at all this frame.
//
// **It stands down while a scene has a dialog up.** The deck overlay's rule is that everything
// behind it goes dead and the one control that closes it is the only one still lit, because
// there is no Escape key and no right click — a modal has to make its exit the brightest thing
// on screen or it is a trap. Chrome updated and drawn after the scene would sit live on top of
// that by construction.
//
// **And it stands down on the screens a player is looking *at* rather than playing.** Settings is
// the room the cog would be a door into; Achievements and Credits joined it on 2026-09-03. Each
// carries its own Back, so nothing is lost — and a ledger button in the corner of the credits would
// be one part of the game sitting on top of another.
//
// **The end-of-run splash is on that list for a different reason**: the run is gone by the time it
// draws, so the ledger button would be dead anyway and the cog would be a door out of the one page
// that has something to say. It has a Back to Title of its own.
func chromeShowing(gs *state.GlobalState) bool {
	if gs.ModalOpen {
		return false
	}
	switch gs.ActiveScreen {
	case state.Settings, state.Achievements, state.Credits, state.RunOver, state.PostBattle, state.Title:
		return false
	}
	return true
}

// ledgerShowing reports whether the ledger's own button is drawn.
//
// **It stands down wherever the cog does, and also while the ledger itself is up** — the panel
// carries a red X, and an opener that survived its own overlay would be a second way out of a
// dialog whose whole design is that the exit is the brightest thing on screen. That is the rule
// the combat screen's toggles were changed to on 2026-08-24.
func (g *Game) ledgerShowing(gs *state.GlobalState) bool {
	return chromeShowing(gs) && !g.ledger.IsOpen()
}

// updateChrome builds the settings button on first use and runs it.
func (g *Game) updateChrome(gs *state.GlobalState) {
	if g.settingsButton == nil {
		g.settingsButton = models.NewButton(
			settingsButtonSize, settingsButtonSize, "", g.openSettings)
		g.settingsButton.BaseColor = settingsButtonColor
	}

	r := settingsButtonRect(gs)
	g.settingsButton.ScreenX = r.Min.X + r.Dx()/2
	g.settingsButton.ScreenY = r.Min.Y + r.Dy()/2

	// **Never disabled.** The mute button that used to live here was, on a machine whose audio
	// device would not open, because a control that silently did nothing would be worse than one
	// saying it cannot. This one opens a screen, which always works — the volume bar on that
	// screen is what goes dead when there is nothing to hear.
	if !chromeShowing(gs) {
		return
	}
	systems.UpdateButton(gs, g.settingsButton)

	// **Escape presses the cog.** Gated on the tutorial's shield as well as on the button being
	// drawn — the lesson makes one rectangle clickable, and a key that walked past that would be a
	// way out of a step the player is being held on.
	if !gs.InputGated && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.openSettings()
		return
	}

	g.updateLedgerButton(gs)
}

// updateLedgerButton builds the ledger's button on first use and runs it.
func (g *Game) updateLedgerButton(gs *state.GlobalState) {
	if g.ledgerButton == nil {
		g.ledgerButton = models.NewButton(
			screens.ControlButtonWidth, screens.ControlButtonHeight,
			ledgerButtonLabel, g.toggleLedger)
		g.ledgerButton.BaseColor = settingsButtonColor
		g.ledgerButton.TextSize = screens.ControlButtonText
	}

	r := ledgerButtonRect(gs)
	g.ledgerButton.ScreenX = r.Min.X + r.Dx()/2
	g.ledgerButton.ScreenY = r.Min.Y + r.Dy()/2

	// **Dead with no run to account for.** The title screen and the credits have none, and a
	// panel that opened onto "no run" would be a control that works and says nothing.
	setChromeEnabled(g.ledgerButton, gs.Run != nil)
	systems.UpdateButton(gs, g.ledgerButton)
}

// setChromeEnabled is the frame's own enable, matching screens.setEnabled: a disabled button
// ignores its colour and reads as unavailable first. **It only ever leaves a disabled button**,
// never a hovered one, so a control re-enabled under a cursor that has since moved does not come
// back lit.
func setChromeEnabled(b *models.Button, on bool) {
	if !on {
		b.State = models.ButtonStateDisabled
		return
	}
	if b.State == models.ButtonStateDisabled {
		b.State = models.ButtonStateNormal
	}
}

// drawChrome draws the settings button and the cog on its face.
//
// **The glyph is drawn by the frame, over the button, rather than being a field on the
// widget** — the same split drawDiscardsLeft makes on the combat screen, and for the same
// reason. models.Button is shared by every screen and holds one centred string; giving it a
// glyph slot would put this control's needs into all of them. It has to come after
// DrawButton, which blits an opaque cached face.
func (g *Game) drawChrome(gs *state.GlobalState, screen *ebiten.Image) {
	if g.settingsButton == nil || !chromeShowing(gs) {
		return
	}
	systems.DrawButton(gs, screen, g.settingsButton)

	// Centred on the face. Measured rather than assumed — the chrome glyphs are 32 where the
	// card's damage sword is 64, and SizeOf is the authority.
	kind := systems.GlyphGear
	size := systems.SizeOf(kind)
	r := settingsButtonRect(gs)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(
		float64(r.Min.X+(r.Dx()-size)/2),
		float64(r.Min.Y+(r.Dy()-size)/2),
	)

	// Untinted, like every glyph. Scaling a five-value palette collapses the bevel into a
	// flat silhouette, which is the whole thing the palette exists to avoid.
	screen.DrawImage(systems.Glyph(kind, systems.PaletteWhite), op)

	// The ledger's button, beside it. **A letter rather than a glyph**, because there is no drawn
	// mark for "the account of this run" and a generated one at 32 pixels would be a silhouette
	// nobody could name. See CLAUDE.md on what a glyph can carry at that size.
	if g.ledgerButton != nil && g.ledgerShowing(gs) {
		systems.DrawButton(gs, screen, g.ledgerButton)
	}
}
