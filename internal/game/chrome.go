package game

// The game's chrome: what is drawn around every screen rather than by one.
//
// **There is exactly one thing in it and that is the point of the file.** A frame is easy to
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
}
