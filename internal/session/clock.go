package session

// **The run's round limit.**
//
// Every fight is on a clock — five rounds, and the duelist still standing at the end of the fifth
// dies. `internal/combat` owns what happens; this owns *how many*, because it is a number that
// belongs to the run rather than to any one duel.
//
// **It is a field rather than a constant read at the point of use.** Nothing moves it today, and
// the reason it can be moved is the whole point: a relic or a brand that buys the player a sixth
// round has one place to write, and every fight of the run is on the new number from the moment it
// is worn. See combat.DefaultRoundLimit, which is what a run opens at.

import "github.com/curiousjc/ascend-duel/internal/combat"

// RoundLimit is how many rounds a fight of this run gets.
func (s *Session) RoundLimit() int { return s.roundLimit }

// SetRoundLimit moves the clock, and refuses to stop it.
//
// **A limit below one is clamped up rather than taken as "no clock".** Zero is unlimited inside
// the rules — see combat.Duelist.RoundLimit — and a drawback that reached it here would silently
// turn the mechanic off for the rest of the run instead of making it harsher, which is the one
// direction a bug in this is invisible. Something that genuinely wants to remove the clock should
// say so by its own name rather than by underflowing this.
func (s *Session) SetRoundLimit(rounds int) {
	if rounds < 1 {
		rounds = 1
	}
	s.roundLimit = rounds
}

// resumeRoundLimit is a saved limit read back, with an old save's silence answered.
//
// **A file written before the clock existed carries no number**, and JSON gives that back as zero
// — which means "no clock" everywhere else in the code. Resuming onto it would take the mechanic
// off a run that was playing under it, invisibly, so a figure below one is read as the default
// rather than obeyed. See profile.RunSnapshot.RoundLimit, and CLAUDE.md on why a zero value is
// never a default.
func resumeRoundLimit(saved int) int {
	if saved < 1 {
		return combat.DefaultRoundLimit
	}
	return saved
}
