package screens

// **A theatre is everything a scene has moving on it, and the rules that apply to all of it.**
//
// It is a shared contract rather than a shared struct: what each scene has moving is its own —
// cards flying to a table, a damage figure crossing to a health bar, a won card settling into the
// middle of the post-battle screen, whatever the shop ends up sliding around — but what is true of
// all of it is the same everywhere, and that is what this file holds.
//
// Three rules, and they are the reason this is a type rather than a paragraph:
//
//   - **It is presentation and may never change an outcome.** A whole round is resolved before
//     playback begins. Nothing reachable through a theatre may touch a duelist, a deck or a purse.
//   - **It runs on the game's one speed.** Every duration in it is a `beat` — see clock.go.
//   - **It is taken down all at once.** Anything cleaned up only at the end of a round is assuming
//     every round ends in one, and a settled duel does not. That lesson cost two separate bugs and
//     is written into two file headers; `clear` is what makes it structural instead.
//
// **It lived on the combat screen until 2026-08-21**, as eleven flat fields on `CombatScene` with
// the rules repeated as comments across six files. Moving it out is what lets a between-fight
// screen use the same vocabulary rather than reinventing it, which is how the game ended up with
// two clocks — see clock.go for the other half of that story.

// theatre is what every scene's theatre answers. **Three methods, and a scene that has anything
// moving implements all three or none** — a theatre that can be advanced but not taken down is the
// bug the third rule above exists to prevent.
//
// Nothing takes a `theatre` as a parameter today; it is here so that a second scene's theatre is
// obliged to be the same shape as the first, checked at compile time by a `var _ theatre`
// assertion beside each one.
type theatre interface {
	// tick advances everything by one frame and drops whatever has finished.
	tick()

	// running reports whether anything is still going. **Playback waits on this** — a figure
	// crossing half the screen does not fit inside one event's dwell.
	running() bool

	// clear takes everything down at once, including any view state a mover was feeding.
	clear()
}

// mover is one thing on stage: it advances on the game's clock and eventually finishes.
//
// **The constraint is on the pointer**, because a mover's clock is state it mutates — ticking a
// copy advances nothing. Writing it this way is what lets `advance` be called without spelling out
// any type arguments; the compiler infers the pointer from the slice's element type.
type mover[T any] interface {
	*T
	tick()
	done() bool
}

// advance ticks every mover and returns the ones still going, reusing the slice's own storage.
//
// **This was four near-identical loops** — one each for the card flights, the hand slides, the
// damage figures and the banked points — which is three too many for "advance each, drop the
// finished". The banked points keep their own loop, because they do something on the frame a
// figure *arrives* rather than on the frame it finishes; that is a real difference and it is
// stated where it happens.
func advance[T any, PT mover[T]](on []T) []T {
	if len(on) == 0 {
		return on
	}
	live := on[:0]
	for i := range on {
		m := PT(&on[i])
		m.tick()
		if !m.done() {
			live = append(live, on[i])
		}
	}
	return live
}

// running reports whether any mover in the set is still going.
//
// **It is separate from `advance` rather than a return value of it**, because the two are asked at
// different times: everything is advanced once a frame, and whether anything is still moving is
// asked by whatever is waiting — the playback cursor, or a screen deciding it may leave.
func running[T any, PT mover[T]](on []T) bool {
	for i := range on {
		if !PT(&on[i]).done() {
			return true
		}
	}
	return false
}
