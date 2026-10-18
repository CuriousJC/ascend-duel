package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// TestTheSettingsColumnClearsTheBottomRow holds the settings column above Compendium and Back. **The
// column is full**, and a row inserted into it moves everything under it down: without this the one
// that is missed is EXIT drawn over BACK, on a screen no other test can see.
func TestTheSettingsColumnClearsTheBottomRow(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}
	rows := settingsRows(gs)

	exitBottom := rows.exit + ui.ButtonLarge/2
	bottomTop := rows.bottom - ui.ButtonLarge/2
	if exitBottom >= bottomTop {
		t.Errorf("EXIT reaches y=%d and the bottom row starts at y=%d; the column has run into it",
			exitBottom, bottomTop)
	}

	// And down the column, each row below the one above it.
	order := []struct {
		name string
		y    int
	}{
		{"music", rows.music}, {"sounds", rows.sounds}, {"speed", rows.speed},
		{"strip", rows.strip.Min.Y}, {"full screen", rows.full}, {"rule", rows.rule},
		{"abandon", rows.abandon}, {"exit", rows.exit},
	}
	for i := 1; i < len(order); i++ {
		if order[i].y <= order[i-1].y {
			t.Errorf("%s (y=%d) is not below %s (y=%d)", order[i].name, order[i].y, order[i-1].name, order[i-1].y)
		}
	}
}
