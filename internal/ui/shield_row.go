package ui

import (
	"github.com/curiousjc/ascend-duel/internal/cards"
)

// MaxShieldPips is how many pips the row can draw, and it is **a measurement of the card rather
// than a rule** *(2026-09-09)*.
//
// The row sits in the seat the enemy's status badges occupy — 20px squares on a 6px pitch along the
// bottom band of a 162px card — and `cards.MaxEffects` records that six is the last one that fits
// inside the borders. A seventh is a redesign of the band, not a bigger number.
//
// **It was `combat.MaxShields` until the duelist's own clamp came off.** While the engine held a
// duelist to five, one constant could honestly serve as both the cap and the row's width; now that
// a duelist may stand behind ten, they are two questions and this is the one the screen owns. A
// duelist holding more than this draws a full row — the count is on the engine, and the card is
// what is short. See Duelist.raiseShields.
const MaxShieldPips = cards.MaxEffects

// The standing shields on one duelist card, as playback has reached them.
//
// **One list, and the count is its length** *(2026-09-02)*. This replaced four parallel structures
// — a count, a color list, a "has anything spoken this round" flag and a set of seats — three of
// which had to be kept in step by hand at every event. Every shield bug so far was two of them
// disagreeing: a count ahead of the colors drew a white pip, a color list trimmed by something
// that had taken no shield away lost a pip's element, and a set of seats that outlived its round
// gagged the next round's defense so nothing flew and no color was ever recorded.
//
// **So the disagreement is made unrepresentable rather than repaired.** A pip *is* its color;
// there is no second place a count can live. What is left to get wrong is which pips are there,
// which is one question with one answer.
//
// **The whole row is a view.** `combat.Duelist.Shields` is not written until the round's end state
// is adopted, so a row reading the model would fill a whole opposing turn after the card that
// filled it. Nothing here may change an outcome — the round was decided before a frame of it was
// drawn.
type ShieldRow struct {
	// Pips is one element per standing shield, oldest first, and its length is the count. The
	// element is that of the card that raised it, and it is a rule: every shield stops any attack,
	// but one that stops an attack of its own element banks an action point — see
	// combat.Duelist.Surge. So a block takes away a pip of the shield the engine spent.
	//
	// **It was a color until 2026-09-16**, when the shield mark became five authored drawings
	// rather than one drawing tinted five ways. What a pip *is* is now the element, and the color
	// is a thing the drawing already has.
	Pips []cards.Element

	// seen says an event this round has spoken about this side's shields. Until one has, the
	// engine's own figure is the authority — see fitTo.
	seen bool

	// flownFrom is which table seats have already sent their pips, so a defend card scored into a
	// hand does not fly them twice: once with its figure, and again when the defend phase
	// announces the raise. **A seat is a position in one round's table**, so this is forgotten
	// with the round.
	flownFrom map[int]bool
}

// Count is how many shields the row is showing.
func (r *ShieldRow) Count() int { return len(r.Pips) }

// Add appends what one landing flight raised, held to what the row can draw.
func (r *ShieldRow) Add(e cards.Element, n int) {
	for i := 0; i < n && len(r.Pips) < MaxShieldPips; i++ {
		r.Pips = append(r.Pips, e)
	}
	r.seen = true
}

// Hold makes the row exactly n pips, taking the oldest away first and filling any shortfall with
// the newest color it has.
//
// **Oldest out** is a choice the engine does not make for us: it draws no distinction between one
// standing shield and another, so the readout picks the reading that keeps the newest pip the one
// just raised. **Filling repeats** because a pip with no color recorded draws as the bare white
// mark, which reads as a different kind of shield rather than as one nobody watched being raised.
func (r *ShieldRow) Hold(n int, fill cards.Element) {
	r.seen = true
	if n > MaxShieldPips {
		n = MaxShieldPips
	}
	switch {
	case n <= 0:
		r.Pips = nil
	case n < len(r.Pips):
		r.Pips = append([]cards.Element(nil), r.Pips[len(r.Pips)-n:]...)
	case n > len(r.Pips):
		// **The caller's element first, the newest pip second.** An announcement knows the card it
		// is about and hands its element in; a count with no card behind it can only repeat what
		// the row is already wearing. Either beats leaving a pip elementless, which draws as the
		// neutral gray mark.
		if fill == cards.Basic && len(r.Pips) > 0 {
			fill = r.Pips[len(r.Pips)-1]
		}
		for len(r.Pips) < n {
			r.Pips = append(r.Pips, fill)
		}
	}
}

// Spend takes one pip of the given element away — the newest of that element — or the oldest pip
// when the row holds none of it, which is a row that has already drifted from the engine and is
// better one short in the right count than holding a pip that is gone.
func (r *ShieldRow) Spend(e cards.Element) {
	r.seen = true
	for i := len(r.Pips) - 1; i >= 0; i-- {
		if r.Pips[i] == e {
			r.Pips = append(append([]cards.Element(nil), r.Pips[:i]...), r.Pips[i+1:]...)
			return
		}
	}
	if len(r.Pips) > 0 {
		r.Pips = append([]cards.Element(nil), r.Pips[1:]...)
	}
}

// RaiseTo grows the row to a count and never shrinks it.
//
// **A raise is cumulative and late.** It carries what is standing after its own card, and it is
// announced a phase after the pips it describes have already flown and landed — so the first of two
// raises names a smaller number than the row is already showing. Taking it outright made the second
// card's pip vanish and come back a beat later. Only a block or an expiry takes a shield away, so
// only they may lower the row.
func (r *ShieldRow) RaiseTo(n int, fill cards.Element) {
	if n > len(r.Pips) {
		r.Hold(n, fill)
		return
	}
	r.seen = true
}

// FitTo squares the row up with the engine's own figure, for the stretch when no event has spoken.
//
// **This is what makes the planning phase right.** A shield raised at the end of the last round is
// standing while the player builds this one and nothing has announced it, so the model is the only
// thing that knows — and the colors from last round are the honest picture of it.
func (r *ShieldRow) FitTo(model int) {
	if r.seen {
		return
	}
	if model != len(r.Pips) {
		r.Hold(model, cards.Basic)
		r.seen = false
	}
}

// Flew reports whether a seat has already sent its pips this round.
func (r *ShieldRow) Flew(seat int) bool { return r.flownFrom[seat] }

// NoteFlight records the seat a flight left from.
func (r *ShieldRow) NoteFlight(seat int) {
	if r.flownFrom == nil {
		r.flownFrom = map[int]bool{}
	}
	r.flownFrom[seat] = true
}

// EndRound hands authority back to the model and forgets this round's seats. **The pips stay**:
// the shields themselves survive the round, and their colors are the only account of what raised
// them.
func (r *ShieldRow) EndRound() {
	r.seen = false
	r.flownFrom = nil
}
