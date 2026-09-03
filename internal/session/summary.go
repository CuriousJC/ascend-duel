package session

// **What a run came to, once it is over.**
//
// A run ends and its `Session` is thrown away — the deck, the rings, the purse, the account. This
// is the handful of numbers that outlive it long enough to be put on a screen: how far up, how many
// fell, how much was dealt, and the code that would deal the whole thing again.
//
// **It is computed from the ledger rather than counted as the run goes.** The account already
// records every fight, its outcome and what the player's blows came to, so a second set of running
// totals on `Session` would be a second arithmetic that can disagree with the first — and the one
// that disagrees silently is always the one nobody is reading. See ledger.go, which owns `dealt`
// for exactly this reason.
//
// **It lives here rather than in `internal/screens` because it is arithmetic, not a picture.** That
// is the same line `internal/pyramid` sits on: a headless caller can ask what a run came to, and a
// test can check the sums without a window.

import "github.com/curiousjc/ascend-duel/internal/seeds"

// How a run finished. **Two ways, and they are not the same fact** — a player who was killed and a
// player who walked away both end a climb, and a summary that called them both "over" would be
// throwing away the only thing that distinguishes the two on the page.
const (
	// EndedInDefeat is the duelist falling. There is no retry; see MECHANICS.md.
	EndedInDefeat = "defeat"

	// EndedByChoice is the player giving the run up from the settings screen.
	EndedByChoice = "abandoned"
)

// RunSummary is a finished run, in numbers.
//
// **Every field is a plain int or string**, so it can be held on the global state and drawn by a
// screen without that screen learning what a `Session` is — the run is gone by the time this is
// read.
type RunSummary struct {
	// Seed is the run code — six Crockford base32 characters, the spelling a player reads off the
	// screen. **The reason the page exists at all**: a run worth telling somebody about is a run
	// they can be given.
	Seed string

	// Ended is EndedInDefeat or EndedByChoice.
	Ended string

	// Floor is how far up the tower the run got, and Rooms is how many duels it entered.
	//
	// **Rooms counts every fight the account opened**, which is the number the player watched go
	// by — not `Fight()`, which is an index into the climb and is one behind after a loss.
	Floor int
	Rooms int

	// Defeated is how many of those duels were won. **Not `Rooms - 1`**: a run can end in the room
	// it was on without that room having been a loss, if it was given up mid-climb.
	Defeated int

	// Dealt is what the player's blows came to across the whole run, and Rounds is how many rounds
	// it took. Both are sums over the account.
	Dealt  int
	Rounds int

	// Vitae is what was left in the purse. **Unspent rather than earned** — the shop is where it
	// goes, so a low figure is a run that bought things rather than a run that earned nothing.
	Vitae int
}

// Summarise adds the run up.
//
// **The seed is passed in rather than held**, exactly as `Snapshot` takes it: a `Session` derives
// everything from the run seed and never stores it, so the one caller that knows it hands it over.
//
// **A run with no fights in it summarises to zeroes rather than refusing.** A player who starts a
// climb and gives it up on the first screen has a real, if short, run — and a summary that returned
// an error would be a screen that has to decide what to draw instead.
func (s *Session) Summarise(runSeed int64, ended string) RunSummary {
	out := RunSummary{
		Seed:  seeds.Code(runSeed),
		Ended: ended,
		Floor: s.Floor(),
		Vitae: s.vitae,
	}

	for _, f := range s.ledger.Fights {
		out.Rooms++
		out.Dealt += f.Dealt()
		out.Rounds += f.RoundCount()
		if f.Outcome == OutcomeWon {
			out.Defeated++
		}

		// **The deepest floor the account saw, rather than the one the run is standing on.** They
		// are the same on a death, and they are not on a run given up after a retreat — and if the
		// climb ever lets a player go back down, the honest answer to "how far did you get" is the
		// high-water mark rather than where they stopped.
		if f.Floor > out.Floor {
			out.Floor = f.Floor
		}
	}
	return out
}
