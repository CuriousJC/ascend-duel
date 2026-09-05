package screens

// A card moving from one seat in a row to another: what a sort looks like, and what a card left
// standing does when the cards around it are spent.
//
// **It belongs to no screen** *(owner's call, 2026-09-05)*. It was `handSlide` on the combat
// screen, which was right while the hand was the only sorted row; the worm screen's offer is
// sorted by the same three tabs, and **a widget that behaves differently on the second screen is
// two widgets**. So the mover moved out here with the tabs, and both rows are re-arranged by one
// piece of code on one clock.
//
// **It stores no coordinates**, for the reason `travel`'s own comment gives: both ends are seat
// lookups made fresh every frame, so a screen that resizes mid-slide does not send a card to where
// its row used to be. What it does store that the other movers do not is a row size at *each* end,
// because the two can differ — at the end of a round a surviving card slides out of the row it was
// in and into the smaller or refilled one that replaced it, and a single count could only describe
// one of those.
//
// **It never travels between a row and anywhere else.** A card going to a pile, to the table or
// back is one of the combat screen's three flights; this is the one that begins and ends in the row.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// cardSlide is one card on its way across a row.
type cardSlide struct {
	travel

	card actionCard

	// lift raises both ends, so a card that is drawn standing proud of its row — the combat
	// screen's queued cards — slides along the raised line it is already on rather than dropping
	// into the row and jumping back out of it. Zero for a row with no such state.
	lift int

	fromIndex, fromCount int
	toIndex, toCount     int
}

// addCardSlide queues one, and drops any slide already heading for the same seat.
//
// **Pressing a second sort button before the first has landed is the case this exists for.**
// Two slides converging on one seat would draw the card twice and leave `slideInto` suppressing
// the row's own copy until the later of them finished. The new arrangement is the true one, so
// the older claim on that seat loses.
func addCardSlide(row []cardSlide, sl cardSlide) []cardSlide {
	kept := row[:0]
	for _, old := range row {
		if old.toIndex != sl.toIndex {
			kept = append(kept, old)
		}
	}
	return append(kept, sl)
}

// slideInto reports whether a card is currently sliding into seat i, so the row can leave that
// seat empty until it lands. Exactly like the combat screen's inboundTo, and hiding a *drawing*
// rather than a card for exactly the same reason: the list is already in its new order.
func slideInto(row []cardSlide, i int) bool {
	for _, sl := range row {
		if sl.toIndex == i {
			return true
		}
	}
	return false
}

// slidesFor is the permutation a sort applied, as slides: for each new position, where the card
// standing there set off from. Cards that did not move do not slide.
//
// **One function so the two screens cannot raise different gestures.** Everything that varies
// between them — which list was sorted, what a card at a seat looks like, whether it stands proud
// of the row — is a parameter; the clock, the easing and the rule that a stationary card stays put
// are not.
func slidesFor(row []cardSlide, order []int, card func(i int) actionCard, lift func(i int) int) []cardSlide {
	n := len(order)
	for to, from := range order {
		if from == to {
			continue
		}
		row = addCardSlide(row, cardSlide{
			travel:    newTravel(0, slideTicks),
			card:      card(to),
			lift:      lift(to),
			fromIndex: from, fromCount: n,
			toIndex: to, toCount: n,
		})
	}
	return row
}

// drawCardSlides draws the cards moving within a row.
//
// **Flat, at full size, with no flip, spin or scale** — and that is the whole gesture. The combat
// screen's other three journeys cross the screen and dramatise it; this one is a card shuffling a
// few inches sideways into place, and anything more would make re-sorting a row look like an event.
//
// seat is where a card sits, given a position and a row size; face is what one looks like. Both are
// the caller's, because a hand and a row of offered deck cards agree about the movement and about
// nothing else.
func drawCardSlides(gs *state.GlobalState, screen *ebiten.Image, row []cardSlide,
	seat func(gs *state.GlobalState, index, count int) image.Point,
	face func(sl cardSlide) cards.Spec) {

	for _, sl := range row {
		if sl.waiting() {
			continue
		}
		t := easeOut(sl.progress())

		from, to := seat(gs, sl.fromIndex, sl.fromCount), seat(gs, sl.toIndex, sl.toCount)
		from.Y -= sl.lift
		to.Y -= sl.lift

		var geo ebiten.GeoM
		geo.Translate(
			float64(from.X)+(float64(to.X)-float64(from.X))*t,
			float64(from.Y)+(float64(to.Y)-float64(from.Y))*t,
		)
		drawFlyingCard(gs, screen, face(sl), cards.Hand, geo)
	}
}
