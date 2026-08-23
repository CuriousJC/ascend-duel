package session

// The climb: which room the run is in, and who is standing in it.
//
// **The fight order used to live on the combat screen**, rebuilt on every entry to it, which
// meant the shape of a run was decided by the screen you fight on and was invisible to every
// other screen. It is a run's property — it outlives a fight the way the deck and the purse do —
// so it is here, and the arithmetic behind it is `internal/pyramid`.
//
// **This is what the room choice writes to.** The scene after the shop shapes what comes next; it
// will do that through a method here rather than by reaching into another screen's state.

import (
	"math/rand"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/pyramid"
	"github.com/curiousjc/ascend-duel/internal/seeds"
)

// Start begins a real run: the authored deck, and a climb rolled from the run's own seed.
//
// **It is the constructor the game uses; `New` is the one tests use.** New takes a deck and
// builds a run with no climb in it, which is right for a test about a purse or a worm and wrong
// for a game — an opponent has to come from somewhere. Keeping them separate is what stops a test
// deck quietly becoming a way to start a run.
//
// **The climb is rolled once, here.** A defeat and a retry meet the same opponent again, because
// nothing re-rolls it; see the randomness skill on why the enemy stream is its own.
// **The bosses come in as a second pool rather than merged into the first**, because the climb
// places them differently: a boss stands on a floor's stairway and nowhere else.
func Start(enemies map[string]data.EnemyData, bosses map[string]data.BossData, runSeed int64) *Session {
	s := New(StartingDeck())
	s.climb = pyramid.New(enemies, bosses, rand.New(rand.NewSource(seeds.For(runSeed, seeds.EnemySelect))))
	return s
}

// Enemy is the record key of whoever stands in the room the run is currently in.
//
// Empty on a run with no climb — a test's run, built by New. A caller that gets an empty key has
// been handed a session that was never started, which is a wiring mistake rather than a state the
// game can reach.
func (s *Session) Enemy() string {
	if s.climb == nil {
		return ""
	}
	return s.climb.EnemyAt(s.fight)
}

// Floor is which floor of the tower the run is on, counting from one.
func (s *Session) Floor() int { return pyramid.FloorOf(s.fight) }
