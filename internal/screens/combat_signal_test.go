package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// signalScene is a screen with two cards on the player's side of the table and two in the hand, so
// both halves of the deferral have somewhere to fire from.
func signalScene() *CombatScene {
	s := &CombatScene{}
	s.theatre.resolved = []resolvedCard{
		{card: combat.Plain(combat.Strike)},
		{card: combat.Plain(combat.Strike)},
	}
	s.hand = []paletteCard{
		{actionCard: combat.Plain(combat.Strike)},
		{actionCard: combat.Plain(combat.Strike)},
	}
	s.fighter = &entities.Combatant{}
	s.enemy = &entities.Combatant{}
	return s
}

func grantEvent(kind combat.EventKind, rider combat.RiderKind, slot, amount int) combat.Event {
	return combat.Event{
		Kind:   kind,
		Side:   combat.SideA,
		Target: combat.SideA,
		Action: combat.Strike,
		Rider:  rider,
		Slot:   slot,
		Amount: amount,
	}
}

// **A signal is parked, not drawn.** Everything about when it fires is the deferral's business, so
// the event going past must put nothing on stage — see combat_signal.go, where that is argued.
func TestARiderEventIsParkedRatherThanDrawn(t *testing.T) {
	s := signalScene()

	if !s.noteSignal(grantEvent(combat.KindGrantedDMG, combat.RiderGolden, 1, 1)) {
		t.Fatal("a golden card's grant was not taken as a signal")
	}
	if len(s.theatre.pending) != 1 {
		t.Fatalf("%d signals parked, want 1", len(s.theatre.pending))
	}
	if len(s.theatre.signals) != 0 {
		t.Errorf("%d signals went straight to the stage; a played card fires when it scores",
			len(s.theatre.signals))
	}
	if s.theatre.pending[0].seat != 1 {
		t.Errorf("the signal parked against seat %d, want the event's own Slot of 1",
			s.theatre.pending[0].seat)
	}
}

// **The two riders that pay vitae are different signals, and Event.Rider is the only thing that
// tells them apart.** A played silver card fires from its seat on the table; a held one fires from
// its seat in the hand. Getting this wrong is what the fight log did for a fortnight.
func TestAHeldPaymentFiresFromTheHandAndAPlayedOneFromTheTable(t *testing.T) {
	s := signalScene()
	s.noteSignal(grantEvent(combat.KindVitae, combat.RiderSilver, 0, 4))
	s.noteSignal(grantEvent(combat.KindVitae, combat.RiderVitaeInHand, 0, 3))

	if len(s.theatre.pending) != 2 {
		t.Fatalf("%d signals parked, want 2", len(s.theatre.pending))
	}
	if s.theatre.pending[0].held {
		t.Error("a played silver card was parked as held")
	}
	if !s.theatre.pending[1].held {
		t.Error("a card kept back was parked as played")
	}

	// The held one takes a seat in the hand rather than the Slot it never had.
	if got := s.theatre.pending[1].seat; got != 0 {
		t.Errorf("the held signal sits at hand seat %d, want 0", got)
	}
}

// **Two copies of the same held card claim two different seats.** The event names a kind of card
// rather than one of them, so each match is taken once — otherwise both fireworks come out of the
// same card and the second card in the hand looks like it paid nothing.
func TestTwoHeldCopiesTakeTwoSeats(t *testing.T) {
	s := signalScene()
	s.noteSignal(grantEvent(combat.KindVitae, combat.RiderVitaeInHand, 0, 3))
	s.noteSignal(grantEvent(combat.KindVitae, combat.RiderVitaeInHand, 0, 3))

	if len(s.theatre.pending) != 2 {
		t.Fatalf("%d signals parked, want 2", len(s.theatre.pending))
	}
	if a, b := s.theatre.pending[0].seat, s.theatre.pending[1].seat; a == b {
		t.Errorf("both held signals took hand seat %d", a)
	}
}

// **Every card the turn kept back fires together, and a played card waits for its own seat.** The
// owner's call: what the hand held back is one fact about the turn, and what a played card did is a
// fact about that card.
func TestHeldSignalsGoTogetherAndPlayedOnesGoBySeat(t *testing.T) {
	s := signalScene()
	s.noteSignal(grantEvent(combat.KindVitae, combat.RiderVitaeInHand, 0, 3))
	s.noteSignal(grantEvent(combat.KindVitae, combat.RiderVitaeInHand, 0, 3))
	s.noteSignal(grantEvent(combat.KindGrantedDMG, combat.RiderGolden, 0, 1))
	s.noteSignal(grantEvent(combat.KindGrantedLife, combat.RiderGolden, 1, 5))

	if got := s.releaseHeldSignals(); got != 2 {
		t.Errorf("%d held signals went, want both", got)
	}
	if got := len(s.theatre.pending); got != 2 {
		t.Errorf("%d signals still parked, want the two played ones", got)
	}

	if got := s.releaseSeatSignals(combat.SideA, 0); got != 1 {
		t.Errorf("seat 0 released %d signals, want 1", got)
	}
	if got := s.releaseSeatSignals(combat.SideA, 0); got != 0 {
		t.Errorf("seat 0 released %d signals a second time; a seat fires once", got)
	}
	if got := s.releaseSeatSignals(combat.SideA, 1); got != 1 {
		t.Errorf("seat 1 released %d signals, want 1", got)
	}
	if len(s.theatre.pending) != 0 {
		t.Errorf("%d signals never fired", len(s.theatre.pending))
	}
}

// **A turn that never scores still fires.** A hand of nothing but defences forms no hand, so the
// sum never runs and there is no sequence to hang anything on — the boundary is the fallback, and
// without it a golden card played into a defensive turn would be silent.
func TestATurnThatNeverScoresFiresAtTheBoundary(t *testing.T) {
	s := signalScene()
	s.noteSignal(grantEvent(combat.KindGrantedDMG, combat.RiderGolden, 0, 1))

	// An event of the same side is not a boundary: the turn is still going.
	s.flushSignalsAtBoundary(combat.Event{Kind: combat.KindAction, Side: combat.SideA})
	if len(s.theatre.pending) != 1 {
		t.Fatal("a signal fired inside its own turn")
	}

	s.flushSignalsAtBoundary(combat.Event{Kind: combat.KindAction, Side: combat.SideB})
	if len(s.theatre.pending) != 0 {
		t.Error("a signal survived the acting side changing")
	}
	if len(s.theatre.signals) != 1 {
		t.Error("the signal never reached the stage")
	}

	// The round ending takes everything, whichever side it belonged to.
	s = signalScene()
	s.noteSignal(grantEvent(combat.KindGrantedDMG, combat.RiderGolden, 0, 1))
	s.flushSignalsAtBoundary(combat.Event{Kind: combat.KindRoundEnd, Side: combat.SideA})
	if len(s.theatre.pending) != 0 {
		t.Error("a signal survived the end of the round")
	}
}

// **A signal holds the playback cursor** *(owner's call)*, and stops holding it when it is done.
func TestASignalHoldsPlayback(t *testing.T) {
	s := signalScene()
	s.noteSignal(grantEvent(combat.KindGrantedDMG, combat.RiderGolden, 0, 1))
	s.releaseSeatSignals(combat.SideA, 0)

	if !s.theatre.running() {
		t.Fatal("a signal in the air does not hold the round")
	}
	for i := 0; i < signalFlyTicks+signalHoldTicks+2; i++ {
		s.theatre.tick()
	}
	if s.theatre.running() {
		t.Error("a finished signal is still holding the round")
	}
}

// **The figure it landed on moves when it lands, and not before.** That is the whole point of the
// flight: the number arriving is what changes the number it arrives at. The model does not catch up
// until the round is adopted, so until then the card draws the tally on top of it.
func TestTheFigureMovesOnArrivalAndIsDroppedOnAdoption(t *testing.T) {
	s := signalScene()
	s.noteSignal(grantEvent(combat.KindGrantedDMG, combat.RiderGolden, 0, 3))
	s.noteSignal(grantEvent(combat.KindGrantedLife, combat.RiderGolden, 0, 5))
	s.noteSignal(grantEvent(combat.KindVitae, combat.RiderSilver, 0, 4))
	s.releaseSeatSignals(combat.SideA, 0)

	if got := s.shownDMG(combat.SideA, 10); got != 10 {
		t.Errorf("DMG shows %d while the figure is still in the air, want 10", got)
	}

	for i := 0; i < signalFlyTicks; i++ {
		s.theatre.tick()
	}

	if got := s.shownDMG(combat.SideA, 10); got != 13 {
		t.Errorf("DMG shows %d after the figure landed, want 13", got)
	}
	if got := s.shownVitae(20); got != 24 {
		t.Errorf("vitae shows %d after the figure landed, want 24", got)
	}
	if got := s.shownLife(combat.SideA, 40); got != 45 {
		t.Errorf("life shows %d after the figure landed, want 45", got)
	}
	// **Gold raises the ceiling with the floor**, so a bar that filled does not read as unmoved.
	if got := s.shownMaxLife(combat.SideA, 60); got != 65 {
		t.Errorf("max life shows %d, want 65", got)
	}

	s.theatre.adopted()
	if got := s.shownDMG(combat.SideA, 13); got != 13 {
		t.Errorf("DMG shows %d after adoption, want the model's own 13", got)
	}
	if got := s.shownVitae(24); got != 24 {
		t.Errorf("vitae shows %d after adoption, want the run's own 24", got)
	}
}

// **A heal moves the bar and never the ceiling.** The engine has already capped the amount at the
// maximum that exists, so a heal that raised one would be inventing life.
func TestAHealFillsTheBarWithoutRaisingIt(t *testing.T) {
	s := signalScene()
	s.noteSignal(grantEvent(combat.KindHealed, combat.RiderHealOnPlay, 0, 6))
	s.releaseSeatSignals(combat.SideA, 0)
	for i := 0; i < signalFlyTicks; i++ {
		s.theatre.tick()
	}

	if got := s.shownLife(combat.SideA, 30); got != 36 {
		t.Errorf("life shows %d after a heal landed, want 36", got)
	}
	if got := s.shownMaxLife(combat.SideA, 60); got != 60 {
		t.Errorf("a heal moved the ceiling to %d", got)
	}
}

// **The burst is derived, never rolled**, so a card comes apart the same way through a resize, an
// interruption or a re-entry — the crack pattern's rule and the dissolve's. See the randomness
// skill, which records this as the explicit exception.
func TestABurstIsTheSameBurstEveryTime(t *testing.T) {
	same := func(a, b []float64) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	c := cardSignal{rider: combat.RiderGolden, seat: 2}
	if !same(burstRays(c, 0), burstRays(c, 0)) {
		t.Fatal("two readings of one signal's burst disagree")
	}
	if same(burstRays(c, 0), burstRays(cardSignal{rider: combat.RiderGolden, seat: 3}, 0)) {
		t.Error("two seats throw the same burst; the pattern says nothing about the card")
	}

	// **The two rings are different scatters**, or they line up into spokes — which is the one
	// thing a second ring exists to avoid.
	if same(burstRays(c, 0)[:signalSparks], burstRays(c, 1)) {
		t.Error("both rings throw the same lengths")
	}
	if got := len(burstRays(c, 1)); got != signalSparks {
		t.Errorf("the inner ring has %d arms, want %d", got, signalSparks)
	}

	for ring := 0; ring < 2; ring++ {
		for i, r := range burstRays(c, ring) {
			if r <= 0 || r > 1 {
				t.Errorf("ring %d arm %d reaches %v, outside the fraction of its own length it is "+
					"meant to be", ring, i, r)
			}
		}
	}
}

// **Every rider is accounted for, drawn or deliberately silent.** This is the choreography table's
// tripwire one layer down: a rider added tomorrow that emits an event nobody drew reads exactly
// like one that was left out on purpose, and only a written-down silence tells them apart.
func TestEverySignalRiderIsAccountedFor(t *testing.T) {
	for _, k := range combat.RiderKinds() {
		d, ok := riderDraws[k]
		if !ok {
			t.Errorf("rider %v has no entry in riderDraws - say what it throws, or why it throws "+
				"nothing", k)
			continue
		}

		// A rider that reaches the log at all is fed the kind it actually arrives as: one the
		// table says draws must be taken by noteSignal, and one it says is silent must be
		// declined. The table and the code cannot drift apart while this holds.
		if !d.emits {
			if d.why == "" {
				t.Errorf("rider %v announces nothing and the table gives no reason", k)
			}
			continue
		}
		s := signalScene()
		took := s.noteSignal(grantEvent(d.kind, k, 0, 1))
		if draws := d.why == ""; took != draws {
			t.Errorf("riderDraws says %v draws=%v and noteSignal says %v", k, draws, took)
		}
	}

	for k := range riderDraws {
		if _, ok := upgradeForRider[k]; !ok {
			t.Errorf("riderDraws describes %v, which upgradeForRider has no colour for", k)
		}
	}
}

// **A signal takes the colour of the rider that threw it**, out of the same table the card's own
// face is washed with — so a gold card throws gold and a silver one silver, sparks and figure alike.
//
// **The vitae-in-hand rider is the exception, and it is the game's**: `vitaeInk` is the crimson
// vitae is written in everywhere it is written. Silver is deliberately not swept up in it — what
// that card is saying is that the metal came up.
func TestASignalIsDrawnInItsRidersColour(t *testing.T) {
	for _, k := range []combat.RiderKind{combat.RiderGolden, combat.RiderSilver,
		combat.RiderHealOnPlay} {

		want := systems.UpgradeTint(upgradeForRider[k])
		if got := signalInk(k); got != want {
			t.Errorf("%v signals in %v, want its own upgrade tint %v", k, got, want)
		}
	}

	if got := signalInk(combat.RiderVitaeInHand); got != vitaeInk {
		t.Errorf("a card kept back pays in %v, want the crimson vitae is always written in", got)
	}

	// The three that can share a screen have to be told apart, which is the whole ask.
	gold, silver := signalInk(combat.RiderGolden), signalInk(combat.RiderSilver)
	if gold == silver || gold == vitaeInk || silver == vitaeInk {
		t.Errorf("gold %v, silver %v and vitae %v are not three colours", gold, silver, vitaeInk)
	}
}
