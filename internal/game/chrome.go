package game

// The game's chrome: what is drawn around every screen rather than by one.
//
// **There is exactly one thing in it and that is the point of the file.** A frame is easy to
// grow by accident, and the bar for joining it is high: something that is true for the whole
// session, on every screen, and owned by no scene. The score qualifies — it is started once
// in main and loops across the title, the tower, the duel and the credits — so the control
// that silences it does too. Anything that belongs to a screen belongs to that screen.
//
// See CLAUDE.md on the input vocabulary: this is a click on a button, and there is no hotkey
// for it, because the game has no keyboard.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/music"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"

	"image/color"
)

// The mute button: a small square in the bottom-left corner, carrying a speaker glyph.
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
	muteButtonSize  = 44
	muteButtonInset = 10
)

// muteButtonColor is the face at full strength: a flat slate, deliberately unlike any control
// that plays the game.
//
// DUEL! is crimson and Discard is yellow because they are choices inside a duel. This is not a
// move — it changes nothing about a run and never becomes unavailable for a game reason — so
// it takes a colour that says "this is the program, not the fight". Same argument as the ring
// row's backing being grey rather than pink.
var muteButtonColor = color.RGBA{R: 92, G: 96, B: 108, A: 255}

// mute flips the score between silent and playing. The button reads the state back rather
// than holding its own copy, so the two cannot disagree.
func (g *Game) mute() { music.SetMuted(!music.Muted()) }

// muteButtonRect is where the button sits: the bottom-left corner of the screen, inset.
//
// Bottom-left is measured from the live screen dimensions rather than from the ScreenWidth and
// ScreenHeight constants, so a change to the internal resolution moves it rather than leaving
// it stranded in the middle.
func muteButtonRect(gs *state.GlobalState) image.Rectangle {
	left := muteButtonInset
	top := gs.ScreenHeight - muteButtonInset - muteButtonSize
	return image.Rect(left, top, left+muteButtonSize, top+muteButtonSize)
}

// updateChrome builds the mute button on first use and runs it.
//
// **It stands down while a scene has a dialog up.** The deck overlay's rule is that everything
// behind it goes dead and the one control that closes it is the only one still lit, because
// there is no Escape key and no right click — a modal has to make its exit the brightest thing
// on screen or it is a trap. Chrome updated and drawn after the scene would sit live on top of
// that by construction, so it does neither while state.ModalOpen is set.
func (g *Game) updateChrome(gs *state.GlobalState) {
	if g.muteButton == nil {
		g.muteButton = models.NewButton(muteButtonSize, muteButtonSize, "", g.mute)
		g.muteButton.BaseColor = muteButtonColor
	}

	r := muteButtonRect(gs)
	g.muteButton.ScreenX = r.Min.X + r.Dx()/2
	g.muteButton.ScreenY = r.Min.Y + r.Dy()/2

	// **Disabled when there is no audio device**, which Start is allowed to fail to open —
	// a machine with no sound card still plays the game. A control that silently did nothing
	// would be worse than a control that says it cannot.
	if !music.Available() {
		g.muteButton.State = models.ButtonStateDisabled
		return
	}
	if g.muteButton.State == models.ButtonStateDisabled {
		g.muteButton.State = models.ButtonStateNormal
	}

	if gs.ModalOpen {
		return
	}
	systems.UpdateButton(gs, g.muteButton)
}

// drawChrome draws the mute button and the glyph on its face.
//
// **The glyph is drawn by the frame, over the button, rather than being a field on the
// widget** — the same split drawDiscardsLeft makes on the combat screen, and for the same
// reason. models.Button is shared by every screen and holds one centred string; giving it a
// glyph slot would put this control's needs into all of them. It has to come after
// DrawButton, which blits an opaque cached face.
func (g *Game) drawChrome(gs *state.GlobalState, screen *ebiten.Image) {
	if g.muteButton == nil || gs.ModalOpen {
		return
	}
	systems.DrawButton(gs, screen, g.muteButton)

	// **The glyph says the state, not the action.** A crossed-out speaker means the score is
	// off, exactly as it does everywhere else; a button showing what pressing it would do
	// reads backwards the moment anyone looks at it without pressing it.
	kind := systems.GlyphSound
	if music.Muted() {
		kind = systems.GlyphMuted
	}

	// Centred on the face. Measured rather than assumed — the glyphs are 32 where the card's
	// category glyphs are 32 and its damage sword is 64, and SizeOf is the authority.
	size := systems.SizeOf(kind)
	r := muteButtonRect(gs)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(
		float64(r.Min.X+(r.Dx()-size)/2),
		float64(r.Min.Y+(r.Dy()-size)/2),
	)

	// Untinted, like every glyph. Scaling a five-value palette collapses the bevel into a
	// flat silhouette, which is the whole thing the palette exists to avoid; a disabled
	// button fades its glyph by alpha instead, so the shading survives and only the weight
	// changes.
	if g.muteButton.State == models.ButtonStateDisabled {
		op.ColorScale.ScaleAlpha(disabledGlyphAlpha)
	}
	screen.DrawImage(systems.Glyph(kind, systems.PaletteWhite), op)
}

// disabledGlyphAlpha is how far a glyph fades on a disabled control. Alpha rather than a
// tint, per the rule above.
const disabledGlyphAlpha = 0.35
