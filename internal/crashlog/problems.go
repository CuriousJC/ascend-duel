package crashlog

// **The running note of what went wrong without stopping the game.**
//
// A failed save, a corrupt profile, an audio device that would not open: every one of them is a
// `log.Printf` today, which means it is a line in a console nobody who plays this game will ever
// see. Two readers want them. A crash report wants the handful that came before the panic, because
// the thing that finally went wrong is rarely the first thing that did. And the player wants the
// ones that are about *them* — a run that is not being saved is a fact worth knowing before the
// climb rather than after it.
//
// **So there are two verbs.** `Note` records and logs; `Tell` does that and also queues a notice
// for the player. Which of the two a call site wants is a judgment about the player, not about the
// severity: "the score has no device" is a Note, "this run is not being saved" is a Tell.

import (
	"fmt"
	"log"
	"sync"
)

// ringSize is how many problems are kept. **The last several rather than all of them** — the
// interesting ones are the ones nearest the panic, and an unbounded slice on a process that can run
// for hours is a leak with a log in it.
const ringSize = 24

// noticeCap is how many unanswered notices may queue up. A notice is dismissed one at a time by a
// click, so a failure repeating every phase transition would otherwise build a queue the player has
// to click their way out of.
const noticeCap = 4

// Problem is one thing that went wrong and did not stop the game.
type Problem struct {
	// Tick is the simulation tick it happened on, from Tick below. **Not a wall clock**, on the
	// rule internal/trace is under: a tick lines up with a replay of the same seed.
	Tick int `json:"tick"`

	// What it was, already worded. **A sentence rather than an error**, because this is read by a
	// person holding a bug report and there is nothing downstream that would match on a type.
	What string `json:"what"`
}

var (
	mu      sync.Mutex
	ring    []Problem
	notices []string
	tick    int
)

// Tick sets the simulation tick every problem from here on is stamped with.
//
// **Set once a frame rather than passed to each call**, exactly as trace.Tick is, so a call site
// reporting a failed write does not have to be holding the game's state to say when it happened.
func Tick(n int) {
	mu.Lock()
	tick = n
	mu.Unlock()
}

// Note records a failure and logs it.
//
// **It logs as well, rather than instead**, because the console is still where this is read while
// the game is being built and a second place to look is worse than a line printed twice.
func Note(format string, args ...any) {
	what := fmt.Sprintf(format, args...)
	log.Print(what)

	mu.Lock()
	defer mu.Unlock()
	ring = append(ring, Problem{Tick: tick, What: what})
	if len(ring) > ringSize {
		ring = ring[len(ring)-ringSize:]
	}
}

// Tell records a failure, logs it, and queues a notice for the player.
//
// **A repeat of something already queued is not a second notice.** Saving fires at every phase
// transition, so a store that has stopped answering would otherwise put one box in front of the
// player per room for the rest of the run — and the second box says nothing the first did not.
func Tell(format string, args ...any) {
	Note(format, args...)
	what := fmt.Sprintf(format, args...)

	mu.Lock()
	defer mu.Unlock()
	for _, q := range notices {
		if q == what {
			return
		}
	}
	if len(notices) >= noticeCap {
		return
	}
	notices = append(notices, what)
}

// Problems is what has gone wrong so far, oldest first.
func Problems() []Problem {
	mu.Lock()
	defer mu.Unlock()
	return append([]Problem(nil), ring...)
}

// Notice is the problem the player has not been told about yet, or "" if there is none.
//
// **Read rather than popped**, so whatever is drawing it can draw the same box for as many frames
// as it is up without having to hold a copy of what it is saying.
func Notice() string {
	mu.Lock()
	defer mu.Unlock()
	if len(notices) == 0 {
		return ""
	}
	return notices[0]
}

// Dismiss drops the notice at the head of the queue.
func Dismiss() {
	mu.Lock()
	defer mu.Unlock()
	if len(notices) > 0 {
		notices = notices[1:]
	}
}

// forget empties both, for the tests.
func forget() {
	mu.Lock()
	defer mu.Unlock()
	ring, notices, tick = nil, nil, 0
}
