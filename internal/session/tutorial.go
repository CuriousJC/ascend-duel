package session

// **The teaching run, parked on the run it is teaching.**
//
// A tutorial step spans the whole loop — it starts in a duel, waits out a reward and finishes in
// the shop — and no scene outlives a fight, so a cursor kept on the combat screen would restart
// every time the player walked back into a room. That is the rule in CLAUDE.md: working state
// belongs to the narrowest thing that needs it, and something outliving a fight is the run's.
//
// **The run does not know what a tutorial is beyond holding one.** `internal/tutorial` decides
// what the current step is and when it gives way; this is a seat, and it is nil for every run that
// is not being taught — which is all of them until something calls Teach.

import "github.com/curiousjc/ascend-duel/internal/tutorial"

// Teach attaches a script to this run, starting it at its first step.
//
// **Nothing in the game calls it yet.** Whether a given player has already been taught is a
// profile question and there is no profile — see TODO.md — so today the caller is the scenario
// fixture. When a real trigger arrives it calls this and nothing else changes.
func (s *Session) Teach(script tutorial.Script) { s.tutorial = tutorial.NewRun(script) }

// Tutorial is the teaching run, or nil if this run is not being taught. **Nil is the ordinary
// case**, and every method on `*tutorial.Run` tolerates a nil receiver for exactly that reason —
// a screen asks whether the tutorial wants anything without first asking whether there is one.
func (s *Session) Tutorial() *tutorial.Run { return s.tutorial }
