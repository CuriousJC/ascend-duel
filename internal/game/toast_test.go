package game

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// TestAScreenChangeIsNotHeldBehindAToast is the promise that an achievement earned on the tick a
// screen changes still gets drawn. Draw skips every frame while a screen is waiting for its Init,
// so an overlay that took the frame before that Init ran would leave the window black with an
// invisible toast waiting for a click — which is what winning a first fight did.
func TestAScreenChangeIsNotHeldBehindAToast(t *testing.T) {
	g := NewGame()
	g.GlobalState.Store = profile.At(t.TempDir())
	g.GlobalState.ActiveScreen = state.Credits
	g.GlobalState.NewScreen = true
	g.GlobalState.EarnedThisSession = []string{"first-steps"}

	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	if g.GlobalState.NewScreen {
		t.Error("the incoming screen was never initialized while the toast was up, so nothing can be drawn")
	}
	if len(g.GlobalState.EarnedThisSession) == 0 {
		t.Error("the toast was dropped rather than shown")
	}
}
