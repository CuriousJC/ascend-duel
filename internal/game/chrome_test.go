package game

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/state"
)

// The settings button's placement, which is arithmetic and needs no window. Nothing here builds
// a models.Button — that allocates an *ebiten.Image and would want a graphics context.
//
// **This package links Ebitengine**, so on Linux the test binary needs a display even though
// no test touches one; CI runs the whole test step under `xvfb-run -a` for exactly that
// reason. See CLAUDE.md.

func TestTheSettingsButtonSitsInTheBottomLeftCornerOnScreen(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}
	r := settingsButtonRect(gs)

	if r.Dx() != settingsButtonSize || r.Dy() != settingsButtonSize {
		t.Errorf("the settings button is %dx%d, want %d square", r.Dx(), r.Dy(), settingsButtonSize)
	}

	// Inset from both edges rather than flush: nothing else reaches this corner, and the
	// margin is what stops it reading as something that fell off the screen.
	if r.Min.X != settingsButtonInset {
		t.Errorf("the settings button starts at x=%d, want the %dpx inset", r.Min.X, settingsButtonInset)
	}
	if got := gs.ScreenHeight - r.Max.Y; got != settingsButtonInset {
		t.Errorf("the settings button is %dpx off the bottom edge, want the %dpx inset",
			got, settingsButtonInset)
	}

	// **What it actually has to clear is the busiest screen's bottom-left**, and on combat that
	// is the action-point figure, which ends around y=861. internal/screens cannot be reached
	// from here — game imports it, never the reverse — so the number is written down rather
	// than derived, and it is a floor with 45 pixels of slack rather than an exact fit.
	const combatAPFigureBottom = 861
	if r.Min.Y <= combatAPFigureBottom {
		t.Errorf("the settings button starts at y=%d, into the combat screen's action-point figure ending at y=%d",
			r.Min.Y, combatAPFigureBottom)
	}
}

func TestTheSettingsButtonStandsDownOnTheSettingsScreen(t *testing.T) {
	// The corner would otherwise be a door into the room the player is already standing in. That
	// screen carries its own Back button, so nothing is lost.
	gs := &state.GlobalState{
		ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight,
		ActiveScreen: state.Settings,
	}
	if chromeShowing(gs) {
		t.Error("the chrome is drawn on the settings screen")
	}
}

func TestTheSettingsButtonStandsDownOverADialog(t *testing.T) {
	// The deck overlay's rule is that the one control closing it is the only lit thing on
	// screen, because there is no Escape key and no right click to fall back on. Chrome drawn
	// after the scene would break that by construction, so both halves check the flag.
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight, ModalOpen: true}
	g := &Game{GlobalState: gs}

	g.updateChrome(gs)
	if g.settingsButton == nil {
		t.Fatal("updateChrome did not build the button")
	}
	if g.settingsButton.PressedInside {
		t.Error("the settings button took a press while a dialog was open")
	}
}
