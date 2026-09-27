package session

// The climb: which room the run is in, and who is standing in it.
//
// **The fight order used to live on the combat screen**, rebuilt on every entry to it, which
// meant the shape of a run was decided by the screen you fight on and was invisible to every
// other screen. It is a run's property — it outlives a fight the way the deck and the purse do —
// so it is here, and the arithmetic behind it is `internal/pyramid`.
//
// **The offers are the seed's and the pick is the run's.** What a floor offers comes out of the
// pyramid, a pure function of the run code; which portal the player walked through is `portals`,
// written by TakePortal and saved with the run. Every question about the floor the run is on goes
// through floorTheme, so nothing can read the seed's first offer where the player picked the second.

import (
	"fmt"
	"math/rand"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/pyramid"
	"github.com/curiousjc/ascend-duel/internal/seeds"
)

// Start begins a real run: the authored deck, and a climb rolled from the run's own seed.
//
// **It is the constructor the game uses; `New` is the one tests use.** New takes a deck and
// builds a run with no climb in it, which is right for a test about a purse or an essence and wrong
// for a game — an opponent has to come from somewhere. Keeping them separate is what stops a test
// deck quietly becoming a way to start a run.
//
// **The climb is rolled once, here.** A defeat and a retry meet the same opponent again, because
// nothing re-rolls it; see the randomness skill on why the enemy stream is its own.
//
// **The tower's shape comes in beside the roster**, because how tall the climb is decides how many
// motifs it has to spend and the roster alone cannot say.
func Start(motifs map[string]data.MotifData, tower data.TowerData, runSeed int64) *Session {
	deck := StartingDeck()
	if StartingDeckList != nil {
		// A chosen deck, for a fixture or a lesson. Copied, so the caller's slice cannot be
		// aliased by a run that then edits it with an essence.
		deck = append([]combat.Card(nil), StartingDeckList...)
	}
	s := New(deck)
	s.climb = newClimb(motifs, tower, runSeed)
	return s
}

// newClimb is what a run seed offers on every floor.
//
// **One function so a resumed run and a new one cannot roll it differently.** The offers are not
// saved — they are rebuilt from the run code — which is only safe while there is exactly one
// expression that turns a seed into them. The picks are saved; see Snapshot.
func newClimb(motifs map[string]data.MotifData, tower data.TowerData, runSeed int64) *pyramid.Pyramid {
	return pyramid.New(motifs, tower, rand.New(rand.NewSource(seeds.For(runSeed, seeds.EnemySelect))))
}

// Enemy is the record key of whoever stands in the room the run is currently in.
//
// Empty on a run with no climb — a test's run, built by New. A caller that gets an empty key has
// been handed a session that was never started, which is a wiring mistake rather than a state the
// game can reach.
func (s *Session) Enemy() string {
	// **A taught run's first room is the one the lesson was written against** *(2026-08-25)*. Bob
	// promises a fight ended in one blow, which is a fact about the taught hand's damage against one
	// creature's HP — so the opponent is part of the script, exactly as the seed is. It applies to
	// room zero only: the lesson is over long before room one, and a tutorial that rewrote the whole
	// climb would be teaching a tower nobody else plays.
	//
	// **It is here rather than in the climb** because the pyramid is a function of the seed and must
	// stay one — see newClimb, and the note in profile/run.go about what depends on that.
	if s.fight == 0 {
		if key := s.tutorial.Enemy(); key != "" {
			return key
		}
	}
	if s.climb == nil {
		return ""
	}
	return s.floorTheme(s.Floor()).Rooms[pyramid.RoomOf(s.fight)]
}

// Floor is which floor of the tower the run is on, counting from one.
func (s *Session) Floor() int { return pyramid.FloorOf(s.fight) }

// Element is the element whoever stands in the current room is dealt as, which is the floor's
// theme. Empty on a run with no climb.
//
// **It is a string rather than a combat.Element** for the reason a card back is: this package
// carries what the data file writes, and the parsing belongs where the cards are built.
func (s *Session) Element() string {
	if s.climb == nil {
		return ""
	}
	return s.floorTheme(s.Floor()).Element
}

// Motif is which motif themes the floor the run is on. Empty on a run with no climb.
func (s *Session) Motif() string {
	if s.climb == nil {
		return ""
	}
	return s.floorTheme(s.Floor()).Motif
}

// floorTheme is the theme a floor is fought at: the offer the player walked through, or the floor's
// first offer where no portal was taken — floor one, and a run jumped past a portal by a fixture.
func (s *Session) floorTheme(floor int) pyramid.Floor {
	offers := s.climb.ChoicesAt(floor)
	if i := floor - 2; i >= 0 && i < len(s.portals) && s.portals[i] != "" {
		for _, f := range offers {
			if f.Motif == s.portals[i] {
				return f
			}
		}
	}
	if len(offers) == 0 {
		return pyramid.Floor{}
	}
	return offers[0]
}

// PortalOffers is what the portals in front of the run open onto: the offers of the floor it is
// about to enter. Nil on a run with no climb.
func (s *Session) PortalOffers() []pyramid.Floor {
	if s.climb == nil {
		return nil
	}
	return s.climb.ChoicesAt(s.Floor())
}

// PortalDue reports whether the run is standing in front of an open portal: the room just won was a
// portal room, so the run is at the first room of a floor above the first, and that floor offers
// more than one realm and none has been taken yet.
//
// **More than one**, because a floor the tower wraps back onto past its top is floor one's single
// offer again, and a choice of one is not a choice.
func (s *Session) PortalDue() bool {
	if s.climb == nil || s.fight == 0 || pyramid.RoomOf(s.fight) != pyramid.RoomOuter {
		return false
	}
	if len(s.PortalOffers()) < 2 {
		return false
	}
	i := s.Floor() - 2
	return i < 0 || i >= len(s.portals) || s.portals[i] == ""
}

// TakePortal walks the run through one of the open portals, by its index in PortalOffers, and
// reports the motif it chose. Once taken, the floor is fought at that theme and the choice is saved
// with the run.
func (s *Session) TakePortal(offer int) (string, error) {
	offers := s.PortalOffers()
	if offer < 0 || offer >= len(offers) {
		return "", fmt.Errorf("portal %d: floor %d opens onto %d", offer, s.Floor(), len(offers))
	}
	i := s.Floor() - 2
	if i < 0 {
		return "", fmt.Errorf("floor %d is not entered through a portal", s.Floor())
	}
	for len(s.portals) <= i {
		s.portals = append(s.portals, "")
	}
	s.portals[i] = offers[offer].Motif
	return s.portals[i], nil
}

// Portals is the motif taken at each portal so far, in floor order from floor two. A copy.
func (s *Session) Portals() []string { return append([]string(nil), s.portals...) }
