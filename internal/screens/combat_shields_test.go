package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
)

// shieldScene is a screen with one card on the player's side of the table.
func shieldScene(card combat.Card) *CombatScene {
	s := &CombatScene{}
	s.theatre.resolved = []resolvedCard{{card: card}}
	s.fighter = &entities.Combatant{}
	return s
}

// A defend card's pips are what the card raises; anything else raises none. **The seat is read off
// the table rather than the event**, so the pips and the sum's figure belong to one card.
func TestOnlyAShieldCardRaisesPips(t *testing.T) {
	ward, ok := combat.ConceptByKey("ward")
	if !ok {
		t.Skip("no ward concept in this build")
	}

	s := shieldScene(combat.Card{Concept: ward})
	if got := s.shieldsRaisedBy(combat.SideA, 0); got != combat.Plain(ward).Amount() {
		t.Errorf("a ward raises %d pips, want %d", got, combat.Plain(ward).Amount())
	}

	s = shieldScene(combat.Card{Concept: combat.Strike})
	if got := s.shieldsRaisedBy(combat.SideA, 0); got != 0 {
		t.Errorf("an attack raises %d pips, want none", got)
	}

	// A seat the table does not hold is nothing rather than a panic.
	if got := s.shieldsRaisedBy(combat.SideA, 7); got != 0 {
		t.Errorf("an empty seat raises %d pips", got)
	}
}

// **A flight predicts, so it may never predict past the row.** The engine caps a duelist at
// combat.MaxShields and the pip row draws exactly that many, so a prediction that ignored the
// ceiling would draw a pip with no seat.
func TestPipsNeverPredictPastTheCap(t *testing.T) {
	s := shieldScene(combat.Card{Concept: combat.Strike})

	s.noteShieldFlight(combat.SideA, 0, 3, combat.MaxShields-1)
	if len(s.theatre.shields) != 1 {
		t.Fatalf("no flight was raised")
	}
	if got := s.theatre.shields[0].count; got != 1 {
		t.Errorf("the flight carries %d pips into a row with room for 1", got)
	}

	// A row already full raises nothing at all rather than a flight of zero pips.
	s.theatre.shields = nil
	s.noteShieldFlight(combat.SideA, 0, 3, combat.MaxShields)
	if len(s.theatre.shields) != 0 {
		t.Errorf("a full row still raised %d flights", len(s.theatre.shields))
	}
}

// The pips are paid into the shown count when they arrive, once, and the count then follows the
// events again — a KindRaised sets it outright, which is what corrects a wrong guess.
func TestPipsArePaidInOnArrivalAndOnlyOnce(t *testing.T) {
	s := shieldScene(combat.Card{Concept: combat.Strike})
	s.noteShieldFlight(combat.SideA, 0, 2, 0)

	s.landShields()
	if got := s.shownShields(combat.SideA, 0); got != 0 {
		t.Errorf("pips landed before they arrived: the row shows %d", got)
	}

	for i := 0; i < shieldFlyTicks; i++ {
		s.theatre.tick()
	}
	s.landShields()
	if got := s.shownShields(combat.SideA, 0); got != 2 {
		t.Fatalf("the row shows %d pips after the flight arrived, want 2", got)
	}

	// Held on screen for its fade, the same flight must not pay again.
	s.landShields()
	s.landShields()
	if got := s.shownShields(combat.SideA, 0); got != 2 {
		t.Errorf("the row shows %d pips, so an arrival was paid twice", got)
	}

	// And the announcement is still the authority.
	s.noteShields(combat.Event{Kind: combat.KindRaised, Side: combat.SideA, Amount: 2, Life: 3})
	if got := s.shownShields(combat.SideA, 0); got != 3 {
		t.Errorf("the row shows %d after KindRaised said 3", got)
	}
}

// **A turn of nothing but defences forms no hand**, so there is no sum for the pips to leave with
// — the announcement flies them instead, and the row takes the count the announcement carries when
// they land rather than the moment it is spoken.
func TestAnAnnouncedRaiseFliesItsOwnPips(t *testing.T) {
	s := shieldScene(combat.Card{Concept: combat.Strike})
	s.theatre.firingSeats = []int{0}

	raise := combat.Event{Kind: combat.KindRaised, Side: combat.SideA, Amount: 2, Life: 2}
	if !s.noteShieldRaise(raise) {
		t.Fatal("an announced raise did not fly its pips")
	}
	if got := s.shownShields(combat.SideA, 0); got != 0 {
		t.Errorf("the row shows %d before the pips arrived, want 0", got)
	}

	for i := 0; i < shieldFlyTicks; i++ {
		s.theatre.tick()
	}
	s.landShields()
	if got := s.shownShields(combat.SideA, 0); got != 2 {
		t.Errorf("the row shows %d after the pips landed, want the announced 2", got)
	}
}

// A card that already sent its pips with its figure does not send them again when the defend phase
// announces the raise — that is the same shields being spoken about twice.
func TestPipsAreNotFlownTwiceForOneCard(t *testing.T) {
	s := shieldScene(combat.Card{Concept: combat.Strike})
	s.theatre.firingSeats = []int{0}

	s.noteShieldFlight(combat.SideA, 0, 2, 0)
	if s.noteShieldRaise(combat.Event{Kind: combat.KindRaised, Side: combat.SideA, Amount: 2, Life: 2}) {
		t.Error("the same card's pips flew twice")
	}
	if len(s.theatre.shields) != 1 {
		t.Errorf("%d flights are in the air for one card", len(s.theatre.shields))
	}
}

// **A pip keeps the colour it flew in.** The flight is drawn in its card's element and the row it
// joins has to agree, or the pip changes colour on landing and says the journey meant nothing.
func TestALandedPipKeepsItsColour(t *testing.T) {
	fire := cards.BorderOf(artFor(combat.Fire))

	s := shieldScene(combat.Card{Concept: combat.Strike, Element: combat.Fire})
	s.noteShieldFlight(combat.SideA, 0, 2, 0)
	for i := 0; i < shieldFlyTicks; i++ {
		s.theatre.tick()
	}
	s.landShields()

	inks := s.shownShieldInks(combat.SideA)
	if len(inks) != 2 {
		t.Fatalf("%d pips have a colour, want 2", len(inks))
	}
	for i, ink := range inks {
		if ink != fire {
			t.Errorf("pip %d is %v, want its card's fire %v", i, ink, fire)
		}
	}

	// A shield eaten takes the oldest colour with it, so the row never draws a colour for a pip
	// that is not there.
	s.noteShields(combat.Event{Kind: combat.KindBlocked, Target: combat.SideA, Amount: 1})
	if got := len(s.shownShieldInks(combat.SideA)); got != 1 {
		t.Errorf("%d colours are left for one standing pip", got)
	}

	// An expiry says how many lapsed, not how many are left, so the row it leaves is empty.
	s.noteShields(combat.Event{Kind: combat.KindExpired, Target: combat.SideA, Amount: 1})
	if got := len(s.shownShieldInks(combat.SideA)); got != 0 {
		t.Errorf("%d colours are left for an empty row", got)
	}
}

// **No standing pip is ever drawn without a colour.** A raise is cumulative and arrives a phase
// after the pips it describes have flown, so it can name a count the colour list has not got —
// which used to draw the extra pips as the bare white mark and flicker one into the row between an
// attack and the next. A raise pads and never trims.
func TestARaiseNeverLeavesAPipColourless(t *testing.T) {
	s := shieldScene(combat.Card{Concept: combat.Strike, Element: combat.Fire})
	s.noteShieldFlight(combat.SideA, 0, 1, 0)
	for i := 0; i < shieldFlyTicks; i++ {
		s.theatre.tick()
	}
	s.landShields()

	// The engine announces the second shield the same turn raised; nothing flew for it.
	s.noteShields(combat.Event{Kind: combat.KindRaised, Side: combat.SideA, Amount: 1, Life: 2})
	if got, want := len(s.shownShieldInks(combat.SideA)), 2; got != want {
		t.Fatalf("%d colours for %d standing pips: the rest draw white", got, want)
	}

	// And a raise may not shrink the list: it says what is standing after its own, not instead of
	// what is already there.
	s.noteShields(combat.Event{Kind: combat.KindRaised, Side: combat.SideA, Amount: 1, Life: 1})
	if got := len(s.shownShieldInks(combat.SideA)); got != 2 {
		t.Errorf("a raise trimmed the colours to %d", got)
	}
}

// **A raise may only ever raise the count.** Two defences in one turn announce "1 shield up" and
// then "2 shields up", both after the pips have flown — so a row taking the first outright drops
// the second card's pip and puts it back a beat later.
func TestARaiseNeverLowersTheRow(t *testing.T) {
	s := shieldScene(combat.Card{Concept: combat.Strike, Element: combat.Fire})
	s.noteShieldFlight(combat.SideA, 0, 2, 0)
	for i := 0; i < shieldFlyTicks; i++ {
		s.theatre.tick()
	}
	s.landShields()

	s.noteShields(combat.Event{Kind: combat.KindRaised, Side: combat.SideA, Amount: 1, Life: 1})
	if got := s.shownShields(combat.SideA, 0); got != 2 {
		t.Errorf("the row dropped to %d pips on the first of two raises, want 2", got)
	}

	// A block still takes one away: that is the only thing that can.
	s.noteShields(combat.Event{Kind: combat.KindBlocked, Target: combat.SideA, Amount: 1})
	if got := s.shownShields(combat.SideA, 0); got != 1 {
		t.Errorf("the row shows %d after a block, want 1", got)
	}
}

// **A seat is a position in one round's table.** The record of which seats have sent their pips is
// what stops a card flying them twice, and it has to be forgotten with the round — a defence
// landing in a seat that flew last round would otherwise never fly, and a pip that never flew has
// no colour to land in.
func TestFlownSeatsAreForgottenEachRound(t *testing.T) {
	s := shieldScene(combat.Card{Concept: combat.Strike, Element: combat.Fire})
	s.enemy = &entities.Combatant{}
	s.fighterAfter, s.enemyAfter = s.fighter.Duelist, s.enemy.Duelist
	s.theatre.firingSeats = []int{0}

	s.noteShieldFlight(combat.SideA, 0, 1, 0)
	if !s.row(combat.SideA).flew(0) {
		t.Fatal("a flight did not record the seat it left")
	}

	s.endOfRound()
	if s.row(combat.SideA).flew(0) {
		t.Fatal("last round's seat still counts as flown")
	}
	if !s.noteShieldRaise(combat.Event{Kind: combat.KindRaised, Side: combat.SideA, Amount: 2, Life: 2}) {
		t.Error("this round's defence in the same seat did not fly its pips")
	}
}
