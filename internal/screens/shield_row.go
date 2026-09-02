package screens

import (
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The standing shields on one duelist card, as playback has reached them.
//
// **One list, and the count is its length** *(2026-09-02)*. This replaced four parallel structures
// — a count, a colour list, a "has anything spoken this round" flag and a set of seats — three of
// which had to be kept in step by hand at every event. Every shield bug so far was two of them
// disagreeing: a count ahead of the colours drew a white pip, a colour list trimmed by something
// that had taken no shield away lost a pip's element, and a set of seats that outlived its round
// gagged the next round's defence so nothing flew and no colour was ever recorded.
//
// **So the disagreement is made unrepresentable rather than repaired.** A pip *is* its colour;
// there is no second place a count can live. What is left to get wrong is which pips are there,
// which is one question with one answer.
//
// **The whole row is a view.** `combat.Duelist.Shields` is not written until the round's end state
// is adopted, so a row reading the model would fill a whole opposing turn after the card that
// filled it. Nothing here may change an outcome — the round was decided before a frame of it was
// drawn.
type shieldRow struct {
	// pips is one colour per standing shield, oldest first, and its length is the count. The
	// colour is the element of the card that raised it — cosmetic, per the owner's call: a fire
	// ward and an ice ward stop the same attack.
	pips []color.RGBA

	// seen says an event this round has spoken about this side's shields. Until one has, the
	// engine's own figure is the authority — see fitTo.
	seen bool

	// flownFrom is which table seats have already sent their pips, so a defend card scored into a
	// hand does not fly them twice: once with its figure, and again when the defend phase
	// announces the raise. **A seat is a position in one round's table**, so this is forgotten
	// with the round.
	flownFrom map[int]bool
}

// count is how many shields the row is showing.
func (r *shieldRow) count() int { return len(r.pips) }

// add appends what one landing flight raised, held to the cap the engine holds a duelist to.
func (r *shieldRow) add(ink color.RGBA, n int) {
	for i := 0; i < n && len(r.pips) < combat.MaxShields; i++ {
		r.pips = append(r.pips, ink)
	}
	r.seen = true
}

// hold makes the row exactly n pips, taking the oldest away first and filling any shortfall with
// the newest colour it has.
//
// **Oldest out** is a choice the engine does not make for us: it draws no distinction between one
// standing shield and another, so the readout picks the reading that keeps the newest pip the one
// just raised. **Filling repeats** because a pip with no colour recorded draws as the bare white
// mark, which reads as a different kind of shield rather than as one nobody watched being raised.
func (r *shieldRow) hold(n int, fill color.RGBA) {
	r.seen = true
	if n > combat.MaxShields {
		n = combat.MaxShields
	}
	switch {
	case n <= 0:
		r.pips = nil
	case n < len(r.pips):
		r.pips = append([]color.RGBA(nil), r.pips[len(r.pips)-n:]...)
	case n > len(r.pips):
		// **The caller's colour first, the newest pip second.** An announcement knows the card it
		// is about and hands its element in; a count with no card behind it can only repeat what
		// the row is already wearing. Either beats leaving a pip colourless, which draws as the
		// bare white mark.
		if fill.A == 0 && len(r.pips) > 0 {
			fill = r.pips[len(r.pips)-1]
		}
		for len(r.pips) < n {
			r.pips = append(r.pips, fill)
		}
	}
}

// raiseTo grows the row to a count and never shrinks it.
//
// **A raise is cumulative and late.** It carries what is standing after its own card, and it is
// announced a phase after the pips it describes have already flown and landed — so the first of two
// raises names a smaller number than the row is already showing. Taking it outright made the second
// card's pip vanish and come back a beat later. Only a block or an expiry takes a shield away, so
// only they may lower the row.
func (r *shieldRow) raiseTo(n int, fill color.RGBA) {
	if n > len(r.pips) {
		r.hold(n, fill)
		return
	}
	r.seen = true
}

// fitTo squares the row up with the engine's own figure, for the stretch when no event has spoken.
//
// **This is what makes the planning phase right.** A shield raised at the end of the last round is
// standing while the player builds this one and nothing has announced it, so the model is the only
// thing that knows — and the colours from last round are the honest picture of it.
func (r *shieldRow) fitTo(model int) {
	if r.seen {
		return
	}
	if model != len(r.pips) {
		r.hold(model, color.RGBA{})
		r.seen = false
	}
}

// flew reports whether a seat has already sent its pips this round.
func (r *shieldRow) flew(seat int) bool { return r.flownFrom[seat] }

// noteFlight records the seat a flight left from.
func (r *shieldRow) noteFlight(seat int) {
	if r.flownFrom == nil {
		r.flownFrom = map[int]bool{}
	}
	r.flownFrom[seat] = true
}

// endRound hands authority back to the model and forgets this round's seats. **The pips stay**:
// the shields themselves survive the round, and their colours are the only account of what raised
// them.
func (r *shieldRow) endRound() {
	r.seen = false
	r.flownFrom = nil
}
