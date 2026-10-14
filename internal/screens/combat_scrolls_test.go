package screens

import (
	"image"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/ui"
)

// The scroll under a card's working, and the one thing about it that has already regressed: it has
// to be drawn over the frame's buttons in the bottom corner, which the table — where the scroll's top
// is drawn, under its card — never can be. **Draw order cannot be checked without a window**, so
// these hold the two pieces that produce it: the scene answers the frame's over-chrome call, and the
// copy drawn there starts exactly where the card ends.

// TestTheWorkingIsDrawnOverTheFrame. The combat scene is an OverChrome, and with nothing of its own
// up it draws the working there.
func TestTheWorkingIsDrawnOverTheFrame(t *testing.T) {
	var scene any = &CombatScene{}
	if _, ok := scene.(ui.OverChrome); !ok {
		t.Fatal("the combat scene draws nothing over the frame, so the working goes under its buttons")
	}
	if !(&CombatScene{}).workingOverChrome(testState()) {
		t.Error("with no dialog and no lesson up, the working is not drawn over the frame")
	}
}

// TestTheScrollOverTheFrameStartsAtItsCardsBottom. The scroll is tucked up under its card, and the
// copy drawn over the frame begins at the card's bottom edge — so it covers the buttons below and
// never the card above.
func TestTheScrollOverTheFrameStartsAtItsCardsBottom(t *testing.T) {
	gs := mathTestState(t)
	card := image.Pt(300, 400)
	bottom := card.Y + cardHeight/2
	stack := []mathItem{{text: "40", size: mathTotalSize, at: image.Pt(card.X, bottom+scrollPadY+60)}}

	m := mathScroll{full: scrollUnder(gs, stack, card)}
	if m.full.Min.Y >= bottom {
		t.Errorf("the scroll starts at y=%d, not tucked under a card ending at y=%d", m.full.Min.Y, bottom)
	}
	if m.full.Dx() != cardWidth {
		t.Errorf("the scroll is %d wide, not the card's %d", m.full.Dx(), cardWidth)
	}

	screen := image.Rect(0, 0, gs.ScreenWidth, gs.ScreenHeight)
	over := m.belowCard(screen)
	if over.Min.Y != bottom {
		t.Errorf("the scroll is drawn over the frame from y=%d, not from the card's bottom at y=%d",
			over.Min.Y, bottom)
	}
	if over.Max.Y != screen.Max.Y || over.Dx() != screen.Dx() {
		t.Errorf("the copy over the frame is clipped to %v, not the screen below the card", over)
	}
}
