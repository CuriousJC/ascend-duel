package game

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/screens"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The settings button's placement, which is arithmetic and needs no window. Nothing here builds
// a models.Button — that allocates an *ebiten.Image and would want a graphics context.
//
// **This package links Ebitengine**, so on Linux the test binary needs a display even though
// no test touches one; CI runs the whole test step under `xvfb-run -a` for exactly that
// reason. See CLAUDE.md.

func TestTheSettingsButtonSitsInTheBottomRightCorner(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}
	r := settingsButtonRect(gs)

	if r.Dx() != settingsButtonSize || r.Dy() != settingsButtonSize {
		t.Errorf("the settings button is %dx%d, want %d square", r.Dx(), r.Dy(), settingsButtonSize)
	}

	// **The bottom-right corner, on the control column's right-hand line** *(2026-09-04, owner's
	// call)*. It sat in the bottom-left corner until then; that corner is the draw pile's now.
	if want := screens.ControlColumnLeft(gs) + screens.ControlColumnWidth(); r.Max.X != want {
		t.Errorf("the settings button ends at x=%d, want the control column's edge at %d",
			r.Max.X, want)
	}
	if got := gs.ScreenHeight - r.Max.Y; got != settingsButtonInset {
		t.Errorf("the settings button is %dpx off the bottom edge, want the %dpx inset",
			got, settingsButtonInset)
	}

	// The ledger is the last rung of that column, and the cog stands clear below it.
	l := ledgerButtonRect(gs)
	if want := screens.ControlColumnSlot(gs, screens.SlotLedger); l != want {
		t.Errorf("the ledger button is at %v, want the column's last slot %v", l, want)
	}
	if l.Max.Y >= r.Min.Y {
		t.Errorf("the ledger button ends at y=%d, into the cog starting at y=%d", l.Max.Y, r.Min.Y)
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
