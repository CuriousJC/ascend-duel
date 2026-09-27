package session

// The journey: which room the run is in, and who is standing in it.
//
// **The fight order is a run's property, not a screen's** — it outlives a fight the way the deck
// and the purse do, and every screen has to be able to ask it — so it is here, and the arithmetic
// behind it is `internal/journey`.
//
// **The offers are the seed's and the pick is the run's.** What a realm offers comes out of the
// journey, a pure function of the run code; which portal the player walked through is `portals`,
// written by TakePortal and saved with the run. Every question about the realm the run is on goes
// through realmTheme, so nothing can read the seed's first offer where the player picked the second.

import (
	"fmt"
	"math/rand"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/journey"
	"github.com/curiousjc/ascend-duel/internal/seeds"
)

// Start begins a real run: the authored deck, and a journey rolled from the run's own seed.
//
// **It is the constructor the game uses; `New` is the one tests use.** New takes a deck and
// builds a run with no journey in it, which is right for a test about a purse or an essence and wrong
// for a game — an opponent has to come from somewhere. Keeping them separate is what stops a test
// deck quietly becoming a way to start a run.
//
// **The journey is rolled once, here.** A defeat and a retry meet the same opponent again, because
// nothing re-rolls it; see the randomness skill on why the enemy stream is its own.
//
// **The journey's shape comes in beside the roster**, because how tall the journey is decides how many
// motifs it has to spend and the roster alone cannot say.
func Start(motifs map[string]data.MotifData, shape data.JourneyData, runSeed int64) *Session {
	deck := StartingDeck()
	if StartingDeckList != nil {
		// A chosen deck, for a fixture or a lesson. Copied, so the caller's slice cannot be
		// aliased by a run that then edits it with an essence.
		deck = append([]combat.Card(nil), StartingDeckList...)
	}
	s := New(deck)
	s.journey = newJourney(motifs, shape, runSeed)
	return s
}

// newJourney is what a run seed offers on every realm.
//
// **One function so a resumed run and a new one cannot roll it differently.** The offers are not
// saved — they are rebuilt from the run code — which is only safe while there is exactly one
// expression that turns a seed into them. The picks are saved; see Snapshot.
func newJourney(motifs map[string]data.MotifData, shape data.JourneyData, runSeed int64) *journey.Journey {
	return journey.New(motifs, shape, rand.New(rand.NewSource(seeds.For(runSeed, seeds.EnemySelect))))
}

// Enemy is the record key of whoever stands in the room the run is currently in.
//
// Empty on a run with no journey — a test's run, built by New. A caller that gets an empty key has
// been handed a session that was never started, which is a wiring mistake rather than a state the
// game can reach.
func (s *Session) Enemy() string {
	// **A taught run's first room is the one the lesson was written against** *(2026-08-25)*. Bob
	// promises a fight ended in one blow, which is a fact about the taught hand's damage against one
	// creature's HP — so the opponent is part of the script, exactly as the seed is. It applies to
	// room zero only: the lesson is over long before room one, and a tutorial that rewrote the whole
	// journey would be teaching a journey nobody else plays.
	//
	// **It is here rather than in the journey** because the journey is a function of the seed and must
	// stay one — see newJourney, and the note in profile/run.go about what depends on that.
	if s.fight == 0 {
		if key := s.tutorial.Enemy(); key != "" {
			return key
		}
	}
	if s.journey == nil {
		return ""
	}
	return s.realmTheme(s.Realm()).Rooms[journey.RoomOf(s.fight)]
}

// Realm is which realm of the journey the run is on, counting from one.
func (s *Session) Realm() int { return journey.RealmOf(s.fight) }

// Element is the element whoever stands in the current room is dealt as, which is the realm's
// theme. Empty on a run with no journey.
//
// **It is a string rather than a combat.Element** for the reason a card back is: this package
// carries what the data file writes, and the parsing belongs where the cards are built.
func (s *Session) Element() string {
	if s.journey == nil {
		return ""
	}
	return s.realmTheme(s.Realm()).Element
}

// Motif is which motif themes the realm the run is on. Empty on a run with no journey.
func (s *Session) Motif() string {
	if s.journey == nil {
		return ""
	}
	return s.realmTheme(s.Realm()).Motif
}

// realmTheme is the theme a realm is fought at: the offer the player walked through, or the realm's
// first offer where no portal was taken — realm one, and a run jumped past a portal by a fixture.
func (s *Session) realmTheme(realm int) journey.Realm {
	offers := s.journey.ChoicesAt(realm)
	if i := realm - 2; i >= 0 && i < len(s.portals) && s.portals[i] != "" {
		for _, f := range offers {
			if f.Motif == s.portals[i] {
				return f
			}
		}
	}
	if len(offers) == 0 {
		return journey.Realm{}
	}
	return offers[0]
}

// PortalOffers is what the portals in front of the run open onto: the offers of the realm it is
// about to enter. Nil on a run with no journey.
func (s *Session) PortalOffers() []journey.Realm {
	if s.journey == nil {
		return nil
	}
	return s.journey.ChoicesAt(s.Realm())
}

// PortalDue reports whether the run is standing in front of an open portal: the room just won was a
// portal room, so the run is at the first room of a realm above the first, and that realm offers
// more than one realm and none has been taken yet.
//
// **More than one**, because a realm the journey wraps back onto past its top is realm one's single
// offer again, and a choice of one is not a choice.
func (s *Session) PortalDue() bool {
	if s.journey == nil || s.fight == 0 || journey.RoomOf(s.fight) != journey.RoomOuter {
		return false
	}
	if len(s.PortalOffers()) < 2 {
		return false
	}
	i := s.Realm() - 2
	return i < 0 || i >= len(s.portals) || s.portals[i] == ""
}

// TakePortal walks the run through one of the open portals, by its index in PortalOffers, and
// reports the motif it chose. Once taken, the realm is fought at that theme and the choice is saved
// with the run.
func (s *Session) TakePortal(offer int) (string, error) {
	offers := s.PortalOffers()
	if offer < 0 || offer >= len(offers) {
		return "", fmt.Errorf("portal %d: realm %d opens onto %d", offer, s.Realm(), len(offers))
	}
	i := s.Realm() - 2
	if i < 0 {
		return "", fmt.Errorf("realm %d is not entered through a portal", s.Realm())
	}
	for len(s.portals) <= i {
		s.portals = append(s.portals, "")
	}
	s.portals[i] = offers[offer].Motif
	return s.portals[i], nil
}

// Portals is the motif taken at each portal so far, in realm order from realm two. A copy.
func (s *Session) Portals() []string { return append([]string(nil), s.portals...) }
