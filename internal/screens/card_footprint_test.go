package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
)

// **This one test stayed with the combat screen when the drawing layer left for internal/ui.**
// What it guards is the hand's own layout constants against the card style they lay cards out in,
// and cardWidth/cardHeight are the hand's — the row, the pitch, the hit rectangles. Everything
// else that was beside it is about a card rather than about the row, and went with the package
// that draws one.

func TestCardFootprintMatchesTheRenderer(t *testing.T) {
	// cardWidth and cardHeight lay out the hand — the pitch, the band, the drop
	// indicator, every hit rectangle. cards.Hand draws the card that sits in those
	// rectangles. They are two copies of one number because one is a const and the other
	// a var field, and nothing but this test stops them drifting.
	//
	// Drift would not crash anything. It would put the cards a few pixels out of their
	// own hit boxes, which reads as "clicking the edge of a card sometimes does nothing"
	// and is miserable to track down.
	if cardWidth != cards.Hand.Width {
		t.Errorf("cardWidth is %d but cards.Hand.Width is %d — the hand would lay out cards at the wrong pitch",
			cardWidth, cards.Hand.Width)
	}
	if cardHeight != cards.Hand.Height {
		t.Errorf("cardHeight is %d but cards.Hand.Height is %d — hit rectangles would not match the art",
			cardHeight, cards.Hand.Height)
	}
}
