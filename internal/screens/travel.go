package screens

import "image"

// **How anything on any screen gets from one place to another.**
//
// `travel` is the clock every mover in the game shares: a delay before it sets off, an age, a
// duration, and an eased progress between them. A card flying to the discard, a damage figure
// crossing the table, a won card settling into the middle of the post-battle screen and — when
// they exist — a bought ring leaving the shop are all one struct read at different speeds.
//
// **It was declared in combat_flight.go until the refactor of 2026-08-21**, which meant the
// post-battle screen borrowed the combat screen's animation machinery by reaching into its
// file. There is nothing about a journey that belongs to a duel, so it is here, beside the
// ground both screens stand on.
//
// The rules that go with it, all of them from CLAUDE.md and none of them local to one screen:
//
//   - **Cards fly; they never appear.** Anything that changes where it is on screen travels
//     there, with a start, a duration and an eased arrival.
//   - **Ease out, so a thing leaves quickly and lands gently.** That is what makes an arrival
//     read as landing rather than as stopping.
//   - **A flight is raised after the model has already moved**, so it is a ghost of something
//     that has happened rather than a thing in progress.
//   - **It may never change an outcome.** A flight is something to look at.

// travel is the clock every moving card on this screen shares: hold for `delay`, then run for
// `ticks`. Three things embed it — a card flying to or from the draw pile, one of the player's
// cards going to its seat on the table, and one of the opponent's arriving at theirs.
//
// **This is deliberately the clock and not the journey** *(extracted 2026-08-12, when the
// opponent's row became the fourth mover)*. The obvious unification is a struct holding a start
// and an end point, and it would be a regression here: **no mover stores its endpoints**. Every
// one of them recomputes both every frame from a layout function — `slotAt`, `playedSeatAt`,
// `enemySeatAt`, `deckStackRect` — which is what makes a flight survive the row re-laying out
// underneath it and survive the window being resized. Caching two coordinates to share a struct
// would trade that away for nothing.
//
// What the four genuinely have in common is a delay, an age, a duration and an eased progress,
// and that is exactly what is here. What they do *not* share is the gesture: the discard
// accelerates away while lifting, turning and shrinking; the deal scales up out of the pile and
// flips face up on the way; the two table rows travel flat. Those are three different drawings
// and folding them into one parameterised one would be a bigger function than the three.
type travel struct {
	// delay holds a card on the launch pad so a handful set off in sequence rather than as a
	// single sheet. age counts from zero *including* the delay, so one counter is the whole
	// clock and there is no second field to keep in step.
	age, delay, ticks int
}

func newTravel(delay, ticks int) travel { return travel{delay: delay, ticks: ticks} }

// waiting reports whether this card is still on the launch pad.
func (t travel) waiting() bool { return t.age < t.delay }

// done reports whether it has arrived.
func (t travel) done() bool { return t.age >= t.delay+t.ticks }

// progress is 0 at the start of the journey and 1 at the end, before easing.
func (t travel) progress() float64 {
	if t.ticks <= 0 {
		return 1
	}
	p := float64(t.age-t.delay) / float64(t.ticks)
	switch {
	case p < 0:
		return 0
	case p > 1:
		return 1
	}
	return p
}

// tick advances the clock, and stops once the card has landed so a seated card costs one
// comparison a frame rather than growing a counter forever.
func (t *travel) tick() {
	if !t.done() {
		t.age++
	}
}

// lerpPoint walks between two points. Integer output because a resolved card is drawn at
// full size and scale 1 — there is no resampling to hide sub-pixel stepping, and none is
// needed at this speed.
func lerpPoint(from, to image.Point, t float64) image.Point {
	return image.Pt(
		from.X+int(float64(to.X-from.X)*t),
		from.Y+int(float64(to.Y-from.Y)*t),
	)
}

// easeOut decelerates into the destination: fast off the pile, settling into the slot.
func easeOut(t float64) float64 { return 1 - (1-t)*(1-t) }

// easeIn accelerates away: a card tossed aside picks up speed rather than drifting off at a
// constant rate, which reads as being thrown instead of as sliding.
func easeIn(t float64) float64 { return t * t }

// flyingTo is where a card sits part-way between two seats. **Eased out**, so it leaves quickly
// and arrives gently — the same shape the combat screen's flights use, because a card decelerating
// into its seat is what makes it read as landing rather than as stopping.
func flyingTo(from, to image.Rectangle, t travel) image.Point {
	p := easeOut(t.progress())
	return image.Pt(
		from.Min.X+int(float64(to.Min.X-from.Min.X)*p),
		from.Min.Y+int(float64(to.Min.Y-from.Min.Y)*p),
	)
}
