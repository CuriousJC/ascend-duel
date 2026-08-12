package game

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/state"
)

// The mute button's placement, which is arithmetic and needs no window. Nothing here builds
// a models.Button — that allocates an *ebiten.Image and would want a graphics context.
//
// **This package links Ebitengine**, so on Linux the test binary needs a display even though
// no test touches one; CI runs the whole test step under `xvfb-run -a` for exactly that
// reason. See CLAUDE.md.

func TestTheMuteButtonSitsInTheBottomLeftCornerOnScreen(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight}
	r := muteButtonRect(gs)

	if r.Dx() != muteButtonSize || r.Dy() != muteButtonSize {
		t.Errorf("the mute button is %dx%d, want %d square", r.Dx(), r.Dy(), muteButtonSize)
	}

	// Inset from both edges rather than flush: nothing else reaches this corner, and the
	// margin is what stops it reading as something that fell off the screen.
	if r.Min.X != muteButtonInset {
		t.Errorf("the mute button starts at x=%d, want the %dpx inset", r.Min.X, muteButtonInset)
	}
	if got := gs.ScreenHeight - r.Max.Y; got != muteButtonInset {
		t.Errorf("the mute button is %dpx off the bottom edge, want the %dpx inset",
			got, muteButtonInset)
	}

	// **What it actually has to clear is the busiest screen's bottom-left**, and on combat that
	// is the action-point figure, which ends around y=861. internal/screens cannot be reached
	// from here — game imports it, never the reverse — so the number is written down rather
	// than derived, and it is a floor with 45 pixels of slack rather than an exact fit.
	const combatAPFigureBottom = 861
	if r.Min.Y <= combatAPFigureBottom {
		t.Errorf("the mute button starts at y=%d, into the combat screen's action-point figure ending at y=%d",
			r.Min.Y, combatAPFigureBottom)
	}
}

func TestTheMuteButtonStandsDownOverADialog(t *testing.T) {
	// The deck overlay's rule is that the one control closing it is the only lit thing on
	// screen, because there is no Escape key and no right click to fall back on. Chrome drawn
	// after the scene would break that by construction, so both halves check the flag.
	gs := &state.GlobalState{ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight, ModalOpen: true}
	g := &Game{GlobalState: gs}

	g.updateChrome(gs)
	if g.muteButton == nil {
		t.Fatal("updateChrome did not build the button")
	}
	if g.muteButton.PressedInside {
		t.Error("the mute button took a press while a dialog was open")
	}
}
