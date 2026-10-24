package screens

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// **Every row of the statistics book fits across its card**, with every figure at a lifetime's size:
// the card is a fixed width, so a longer label or a bigger number is a layout change, not a wrap.
func TestEveryStatRowFitsItsCard(t *testing.T) {
	gs := mathTestState(t)
	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: statRowSize}
	room := float64(statCardW - 2*statCardPadX - statRowGap)

	heading := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: systems.TextLarge}
	for _, row := range statRows(gs, profile.Tally{}, false) {
		if row.heading {
			if w := systems.MeasureText(row.label, heading); w > float64(statCardW-2*statCardPadX) {
				t.Errorf("heading %q is %.0f wide, the card has %d", row.label, w, statCardW-2*statCardPadX)
			}
			continue
		}
		value := strings.ReplaceAll(row.value, "0", "999,999")
		if w := systems.MeasureText(row.label, face) + systems.MeasureText(value, face); w > room {
			t.Errorf("%q with %q is %.0f wide, the card has %.0f", row.label, value, w, room)
		}
	}
}

// **A creature's name stands on its own line and still has to fit**: the longest on the roster,
// indented as the leader rows draw it.
func TestTheLongestCreatureNameFitsItsCard(t *testing.T) {
	gs := mathTestState(t)
	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: statRowSize}
	room := float64(statCardW - 2*statCardPadX - statRowIndent)

	for _, m := range data.LoadMotifs() {
		for _, r := range m.Records {
			if w := systems.MeasureText(r.FullName(), face); w > room {
				t.Errorf("%q is %.0f wide, the card has %.0f", r.FullName(), w, room)
			}
		}
	}
}
