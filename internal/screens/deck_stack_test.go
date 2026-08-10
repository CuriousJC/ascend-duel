package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The deck stack's placement, which is arithmetic and therefore checkable without a window
// — the same narrow exception the other tests in this package take. It creates no
// ebiten.Image; a GlobalState is a plain struct and PctX/PctY are integer division.
//
// This exists because the first placement was wrong in a way nobody would have noticed
// until it looked slightly off: at 95% of the screen height the pile sat three pixels
// through the action-point bar. Three pixels is exactly the size of mistake that gets
// looked at, shrugged off, and shipped.

func testState() *state.GlobalState {
	// The fixed internal resolution game.Layout returns. Anything laid out against a
	// percentage is a function of these two numbers and nothing else.
	return &state.GlobalState{ScreenWidth: 1280, ScreenHeight: 960}
}

func TestDeckStackClearsTheAPBarAndTheScreen(t *testing.T) {
	gs := testState()

	// What the stack has to live below: the hand row, then the AP figure, then the bar.
	barBottom := gs.PctY(handTopPct) + cardHeight + apBarBelow + apBarHeight

	// The ring is the outermost thing drawn, and it is what has to fit — the stroke is
	// centred on the rectangle's edge, so it reaches half its width beyond.
	ring := deckStackBounds(gs).Inset(-deckHighlightInset).Inset(-deckHighlightWidth / 2)

	if ring.Min.Y < barBottom {
		t.Errorf("the deck stack's highlight starts at y=%d, which is inside the AP bar ending at y=%d",
			ring.Min.Y, barBottom)
	}
	if ring.Max.Y > gs.ScreenHeight {
		t.Errorf("the deck stack's highlight reaches y=%d, past the bottom of the %d-pixel screen",
			ring.Max.Y, gs.ScreenHeight)
	}
	if ring.Max.X > gs.ScreenWidth {
		t.Errorf("the deck stack's highlight reaches x=%d, past the right of the %d-pixel screen",
			ring.Max.X, gs.ScreenWidth)
	}
}

func TestDeckStackIsTheSizeItIsDrawnAt(t *testing.T) {
	gs := testState()

	// The front card is what a click is tested against, so its rectangle has to be the
	// picture's size or the clickable area and the visible area disagree.
	front := deckStackRect(gs)
	if front.Dx() != cards.Stack.Width || front.Dy() != cards.Stack.Height {
		t.Errorf("the deck stack's hit rectangle is %dx%d, but cards.Stack draws %dx%d",
			front.Dx(), front.Dy(), cards.Stack.Width, cards.Stack.Height)
	}

	// The backs are drawn up and to the left of the front card, so the bounds have to cover
	// them — that rectangle is what the ring is drawn around and what the click uses.
	bounds := deckStackBounds(gs)
	if !front.In(bounds) || bounds.Max != front.Max {
		t.Errorf("bounds %v do not sit around the front card %v", bounds, front)
	}
}

func TestStackStyleKeepsTheCardsProportions(t *testing.T) {
	// The pile has to read as the same object as the cards in the hand, seen smaller. Drift
	// here would make it a differently shaped rectangle that happens to be dark.
	hand := float64(cards.Hand.Width) / float64(cards.Hand.Height)
	stack := float64(cards.Stack.Width) / float64(cards.Stack.Height)

	if d := hand - stack; d > 0.02 || d < -0.02 {
		t.Errorf("cards.Stack is %.3f wide per tall against the hand card's %.3f", stack, hand)
	}
}

func TestTheComboRingFitsOnScreen(t *testing.T) {
	gs := testState()

	// The ring is drawn around the pile, so the first card's inset has to leave room for it.
	// Flush against the margin puts the bracket off the screen, which reads as a highlight
	// with a side missing rather than as a card being too far left.
	first := pileAt(gs, 0)
	left := first.X - comboRingInset - comboRingWidth/2
	if left < 0 {
		t.Errorf("the combo ring starts at x=%d, off the left of the screen", left)
	}

	// And its bottom must clear the action-point figure printed under the hand row.
	bottom := first.Y + cardHeight + comboRingInset + comboRingWidth/2
	apText := gs.PctY(handTopPct) + cardHeight + apTextBelow
	if bottom > apText {
		t.Errorf("the combo ring reaches y=%d, through the AP figure at y=%d", bottom, apText)
	}
}
