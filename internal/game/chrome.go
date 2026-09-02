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
// See CLAUDE.md on the input vocabulary: this is a click on a button, and there is no hotkey
// for it, because the game has no keyboard.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"

	"image/color"
)

// The settings button: a small square in the bottom-left corner, carrying a cog.
//
// **Square and iconic because it is not a move.** DUEL! and Discard are 138x50 with a word on
// them because they are choices inside a duel and have to be found and read; this changes
// nothing about a run, and a control that took the same footprint would be claiming the same
// weight. A glyph is smaller, says one thing, and is what the repo already generates for
// interface art — see internal/systems/glyphs.go.
//
// **Inset from both edges rather than flush.** Nothing else on any screen is near this corner:
// on the combat screen — the busiest — the resolved-card pile stops at y=809, the action-point
// figure at y=861, and the nearest control on the button strip starts around x=396. The inset
// is what stops it reading as something that fell off the screen.
const (
	settingsButtonSize  = 44
	settingsButtonInset = 10

	// The ledger's button stands beside the cog, in the same corner and on the same line: both are
	// the program rather than the fight, so they read as a pair rather than as one control and a
	// stray. The gap is the inset, which is what the combat screen's own toggles use between two
	// squares standing together.
	ledgerButtonGap = 10

	// **One character on a square**, the size the game's other square buttons carry a letter at.
	// `L` is what the fight log's button was, and the ledger is what became of it.
	ledgerButtonLabel = "L"
	ledgerButtonText  = 30
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

// ledgerButtonRect is where the ledger's button sits: to the right of the cog, bottom edges level.
func ledgerButtonRect(gs *state.GlobalState) image.Rectangle {
	cog := settingsButtonRect(gs)
	left := cog.Max.X + ledgerButtonGap
	return image.Rect(left, cog.Min.Y, left+settingsButtonSize, cog.Max.Y)
}

// settingsButtonRect is where the button sits: the bottom-left corner of the screen, inset.
//
// Bottom-left is measured from the live screen dimensions rather than from the ScreenWidth and
// ScreenHeight constants, so a change to the internal resolution moves it rather than leaving
// it stranded in the middle.
func settingsButtonRect(gs *state.GlobalState) image.Rectangle {
	left := settingsButtonInset
	top := gs.ScreenHeight - settingsButtonInset - settingsButtonSize
	return image.Rect(left, top, left+settingsButtonSize, top+settingsButtonSize)
}

// chromeShowing reports whether the frame is drawn at all this frame.
//
// **It stands down while a scene has a dialog up.** The deck overlay's rule is that everything
// behind it goes dead and the one control that closes it is the only one still lit, because
// there is no Escape key and no right click — a modal has to make its exit the brightest thing
// on screen or it is a trap. Chrome updated and drawn after the scene would sit live on top of
// that by construction.
//
// **And it stands down on the settings screen itself**, which is the one screen where the corner
// would be a door into the room the player is already standing in. That screen carries its own
// Back button, so nothing is lost.
func chromeShowing(gs *state.GlobalState) bool {
	return !gs.ModalOpen && gs.ActiveScreen != state.Settings
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
	g.updateLedgerButton(gs)
}

// updateLedgerButton builds the ledger's button on first use and runs it.
func (g *Game) updateLedgerButton(gs *state.GlobalState) {
	if g.ledgerButton == nil {
		g.ledgerButton = models.NewButton(
			settingsButtonSize, settingsButtonSize, ledgerButtonLabel, g.toggleLedger)
		g.ledgerButton.BaseColor = settingsButtonColor
		g.ledgerButton.TextSize = ledgerButtonText
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
