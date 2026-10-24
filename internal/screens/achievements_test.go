package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/achieve"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// **Every entry fits its third of the page**: a name on one line with room for a progress figure
// beside it, and a description in at most the two lines the entry is deep enough for.
func TestEveryAchievementFitsItsEntry(t *testing.T) {
	gs := mathTestState(t)
	name := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: achievementNameSize}
	line := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: achievementLineSize}
	progress := systems.MeasureText("99,999 / 99,999", line) + achievementRowInset

	for _, a := range achieve.Loaded().All() {
		if w := systems.MeasureText(a.DisplayName, name) + progress; w > achievementTextWidth {
			t.Errorf("%q with its progress is %.0f wide, the entry has %d", a.DisplayName, w, achievementTextWidth)
		}
		if n := len(achievementLines(a.Description, line)); n > 2 {
			t.Errorf("%q wraps to %d lines, the entry holds 2", a.Description, n)
		}
	}
}
