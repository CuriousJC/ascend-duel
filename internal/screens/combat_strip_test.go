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

// The pile stands at the bottom of the duelist card's column *(2026-09-04, owner's call)*, which
// is a different set of neighbours from the corner it used to sit in: the hand and its bar are
// beside it rather than above it, and the frame's own corner controls are below it.
func TestTheDeckPileStandsInTheDuelistsColumn(t *testing.T) {
	gs := testState()
	s := &CombatScene{}

	// **The pile is the outermost thing drawn** *(2026-08-24)*, so the bounds are what has to fit
	// rather than the front card: the backs are drawn up and to the left of it.
	ring := deckStackBounds(gs)
	card := s.duelistCardRect(gs)

	if deckStackRect(gs).Min.X != card.Min.X {
		t.Errorf("the pile starts at x=%d and the duelist card at x=%d",
			deckStackRect(gs).Min.X, card.Min.X)
	}

	// Clear of the hand, which is what is to its right — the bar included, since that spans the
	// whole band and reaches further down than the cards do.
	if left := handBandLeft(gs); ring.Max.X > left {
		t.Errorf("the pile reaches x=%d, into the hand band starting at x=%d", ring.Max.X, left)
	}

	// The count sits under it, left edges level, and the whole column has to stay on the screen.
	count := deckCountRect(gs)
	if count.Min.X != deckStackRect(gs).Min.X {
		t.Errorf("the count starts at x=%d and the pile at x=%d", count.Min.X, deckStackRect(gs).Min.X)
	}
	if count.Max.Y > gs.ScreenHeight-deckStackBottomInset {
		t.Errorf("the count ends at y=%d, past the %dpx bottom inset", count.Max.Y, deckStackBottomInset)
	}
	if ring.Min.X < 0 || ring.Min.Y < 0 {
		t.Errorf("the pile %v runs off the screen", ring)
	}

	// The bucket button is on the line above it, and that line must clear the played row.
	if top := deckCaptionRect(gs).Min.Y; top < tableRowTop(gs)+cardHeight {
		t.Errorf("the pile's caption line starts at y=%d, inside the played row ending at y=%d",
			top, tableRowTop(gs)+cardHeight)
	}
}

func TestTheBottomOfTheScreenIsOneLine(t *testing.T) {
	gs := testState()

	// **Three things hang off the bottom of the screen and they read as one line**: the deck
	// pile's count on the left, the discard badge in the middle, and the cog at the foot of the
	// control column on the right.
	//
	// The cog is chrome and lives in internal/game, which imports this package and so cannot be
	// imported back. Its inset is therefore written down rather than read, and
	// TestTheSettingsButtonSitsAtTheFootOfTheControlColumn holds the other end of it.
	const cogInset = 10

	if got := gs.ScreenHeight - deckCountRect(gs).Max.Y; got != cogInset {
		t.Errorf("the deck count sits %dpx off the bottom edge and the cog %dpx", got, cogInset)
	}

	// The badge is a disc centred on the Discard button's bottom-right corner, so its lowest
	// point is a radius below that corner.
	badgeBottom := buttonStripY(gs) + stripButtonHeight/2 + discardBadgeRadius
	if d := gs.ScreenHeight - badgeBottom - cogInset; d < -6 || d > 6 {
		t.Errorf("the discard badge ends %dpx off the bottom edge and the cog %dpx — which no longer reads as one line",
			gs.ScreenHeight-badgeBottom, cogInset)
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

	// And its bottom must clear the hand itself. **It no longer has to clear the band the sum is
	// written in** *(2026-09-04, owner's call)*: the arithmetic is an overlay drawn over the row
	// rather than a strip reserved above it, so the only thing under the played cards that they
	// may not reach is the hand — see tableRowTop.
	bottom := first.Y + cardHeight
	if top := handTop(gs) - mathBandGapAboveCards; bottom > top {
		t.Errorf("the played row reaches y=%d, into the hand at y=%d", bottom, top)
	}
}

// The two buttons are one pair centred under the dealt hand *(2026-09-04, owner's call)*, rather
// than spread across the strip or pinned to its right-hand end — both of which described the
// buttons by what was beside them instead of by the cards they act on.
func TestTheButtonsAreCentredUnderTheHand(t *testing.T) {
	gs := testState()

	const discardWidth, duelWidth = stripButtonWidth, stripButtonWidth
	discardX, duelX := buttonStripSlots(gs, discardWidth, duelWidth)

	// Centred as a pair, so the air outside Discard's left edge and outside DUEL!'s right edge is
	// the same. Integer halving costs a pixel.
	left, right := discardX-discardWidth/2, duelX+duelWidth/2
	if d := (left + right) - 2*handRowCentre(gs).X; d > 1 || d < -1 {
		t.Errorf("the pair spans x=%d..%d, which is not centred on the hand at x=%d",
			left, right, handRowCentre(gs).X)
	}

	// **On the row's fixed centre, not the band's**, which narrows as the hand is spent — a pair
	// centred on that would slide sideways mid-round.
	if got, want := right-left, discardWidth+stripButtonGap+duelWidth; got != want {
		t.Errorf("the pair is %dpx wide, want %d", got, want)
	}
	if gap := duelX - duelWidth/2 - (discardX + discardWidth/2); gap != stripButtonGap {
		t.Errorf("the buttons are %dpx apart, want %d", gap, stripButtonGap)
	}

	// Clear of the action-point figure's column, which shares the line with them.
	if figure := apFigureRight(gs); left <= figure {
		t.Errorf("Discard starts at x=%d, inside the action-point figure's column ending at %d",
			left, figure)
	}
}

// The bottom of the screen runs from the ring row's left edge to the control column
// *(2026-09-04, owner's call)*: the rings and the hand start together, and the cards stop a gap
// short of the sort buttons, which stand on the enemy card's left edge.
//
// It is checked at both ends rather than on the width alone: a band of the right size in the
// wrong place would be the same mistake, and the cards are centred on it.
func TestTheHandIsLaidOutBetweenTheFighterCards(t *testing.T) {
	gs := testState()
	s := &CombatScene{}
	left := handBandLeft(gs)

	if want := s.duelistCardRect(gs).Max.X + ringPaneGap; left != want {
		t.Errorf("the band starts at x=%d, want the ring row's own left edge at %d", left, want)
	}

	// The cards stop a gap short of the control column, and are centred on what is left.
	if got, want := cardBandWidth(gs), ControlColumnLeft(gs)-sortColumnGap-left; got != want {
		t.Errorf("the cards are laid out into %dpx, want %dpx", got, want)
	}
	if got, want := handRowCentre(gs).X, left+cardBandWidth(gs)/2; got != want {
		t.Errorf("the hand is centred at x=%d, want %d", got, want)
	}

	// The sort block abuts the cards, and the panel buttons under it stand on the enemy card's
	// left edge — the two groups are anchored to different things, which is the point of the
	// column being two groups. See controlcolumn.go.
	if got, want := sortColumnRect(gs).Min.X, left+cardBandWidth(gs); got != want {
		t.Errorf("the sort block starts at x=%d, want the cards' right edge at %d", got, want)
	}
	if got, want := ControlColumnSlot(gs, SlotHands).Min.X, s.enemyCardRect(gs).Min.X; got != want {
		t.Errorf("the panel buttons start at x=%d, want the enemy card's left edge at %d", got, want)
	}
}
