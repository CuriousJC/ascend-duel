package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
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
	return &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}
}

func TestDeckStackClearsTheAPBarAndTheScreen(t *testing.T) {
	gs := testState()

	// What the stack has to live below: the hand row, then the AP figure, then the bar.
	barBottom := handTop(gs) + cardHeight + apBarBelow + apBarHeight

	// **The pile is the outermost thing drawn** *(2026-08-24)*. It used to wear a yellow ring
	// while the overlay was open, because the pile was then the only live control on a covered
	// screen; the panel carries a red X now, so the highlight said "press me" about a control
	// that no longer does anything.
	ring := deckStackBounds(gs)

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

func TestTheBottomOfTheScreenIsOneLine(t *testing.T) {
	gs := testState()

	// **Three things hang off the bottom of the screen and they read as one line.** The deck
	// pile and the mute button share an inset exactly; the discard badge lands four pixels
	// lower because it hangs off a button strip placed as a percentage, and chasing that would
	// mean taking the strip off percentages for no other reason.
	//
	// The mute button is chrome and lives in internal/game, which imports this package and so
	// cannot be imported back. Its inset is therefore written down rather than read, and
	// TestTheMuteButtonSitsInTheBottomLeftCornerOnScreen holds the other end of it.
	const muteButtonInset = 10

	pile := gs.ScreenHeight - deckStackRect(gs).Max.Y
	if pile != muteButtonInset {
		t.Errorf("the deck pile sits %dpx off the bottom edge and the mute button %dpx",
			pile, muteButtonInset)
	}

	// The badge is a disc centred on the Discard button's bottom-right corner, so its lowest
	// point is a radius below that corner.
	badgeBottom := buttonStripY(gs) + stripButtonHeight/2 + discardBadgeRadius
	if d := badgeBottom - deckStackRect(gs).Max.Y; d < 0 || d > 6 {
		t.Errorf("the discard badge ends at y=%d and the deck pile at y=%d — %dpx apart, which no longer reads as one line",
			badgeBottom, deckStackRect(gs).Max.Y, d)
	}
}

func TestTheAPFigureLinesUpWithTheButtonStrip(t *testing.T) {
	gs := testState()

	// **The thing the owner asked for on 2026-08-12**: the action-point figure's top on the same
	// line as the Discard button's top.
	//
	// **It was a coincidence of five constants until 2026-09-04 and is now enforced.** handTopPct
	// had been chosen to make it true at 960 tall; at 1080 no integer percentage lands on it, so
	// buttonStripY derives the strip from the figure instead. That makes this check structural
	// rather than a measurement — which is worth saying, because a tautology cannot fail. What it
	// still holds is that the two are placed from one number: it fails the day something goes back
	// to positioning the strip on its own.
	figureTop := apFigureTop(gs)
	buttonTop := buttonStripY(gs) - stripButtonHeight/2

	if figureTop != buttonTop {
		t.Errorf("the AP figure's top is y=%d and the button strip's top is y=%d", figureTop, buttonTop)
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

func TestThePlayedRowFitsOnScreen(t *testing.T) {
	gs := testState()

	// **A card lifted by tableFireLift is the top of the row**, so the check is against that
	// rather than against the row's resting y — the moment a card is firing is the moment it is
	// highest, and a row that only fitted at rest would clip exactly when it mattered.
	//
	// The widest row the rules can produce, asked of the rules rather than written down: a ring
	// raising the action cap must not be able to push the row off the screen quietly.
	//
	// **This measured the yellow hand ring until 2026-08-19**, which stood off the cards and so
	// set the margins. The ring is gone and the cards themselves are what has to fit.
	first := playedSeatAt(gs, 0, combat.Duelist{}.MaxActions(), 0)
	if first.X < 0 {
		t.Errorf("the played row starts at x=%d, off the left of the screen", first.X)
	}
	if top := first.Y - tableFireLift; top < 0 {
		t.Errorf("a firing card reaches y=%d, off the top of the screen", top)
	}

	// And its bottom must clear the band above the hand, where the sum is written — see
	// tableRowTop.
	bottom := first.Y + cardHeight
	bandTop := handTop(gs) - mathBandGapAboveCards - mathBandHeight
	if bottom > bandTop {
		t.Errorf("the played row reaches y=%d, through the band above the hand at y=%d", bottom, bandTop)
	}
}

// The bottom strip is five things on one line — the AP figure, Discard, DUEL!, the Log button
// and the deck pile — and what is wanted of the two buttons is a *relationship*: the space
// between the figure and the things in the corner shared equally, rather than two percentages
// that happened to look right once. That is what this checks, in the only terms it can be
// stated in: the three gaps.
//
// **The right-hand end is the Log button, not the pile** *(2026-08-18)*. It arrived between the
// two, so a strip still measured to the pile would spread its buttons across a span with a
// control standing in it.
func TestTheButtonStripSharesItsSpaceEvenly(t *testing.T) {
	gs := testState()

	const discardWidth, duelWidth = stripButtonWidth, stripButtonWidth
	discardX, duelX := buttonStripSlots(gs, discardWidth, duelWidth)

	figure, corner := apFigureRight(gs), pileSlotRect(gs).Min.X
	before := discardX - discardWidth/2 - figure
	between := duelX - duelWidth/2 - (discardX + discardWidth/2)
	after := corner - (duelX + duelWidth/2)

	if before <= 0 || between <= 0 || after <= 0 {
		t.Fatalf("the buttons do not fit between the AP figure at x=%d and the Log button at x=%d: gaps %d, %d, %d",
			figure, corner, before, between, after)
	}

	// Integer division puts the remainder in the last gap, so the tolerance is the two pixels
	// three-way rounding can lose. Anything larger is a placement, not a division.
	if d := before - between; d > 2 || d < -2 {
		t.Errorf("gap before Discard is %d and the gap between the buttons is %d", before, between)
	}
	if d := before - after; d > 2 || d < -2 {
		t.Errorf("gap before Discard is %d and the gap after DUEL! is %d", before, after)
	}
}
