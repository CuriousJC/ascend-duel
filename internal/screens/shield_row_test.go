package screens

import (
	"image/color"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

var (
	rowFire = color.RGBA{R: 200, G: 80, B: 40, A: 255}
	rowIce  = color.RGBA{R: 80, G: 160, B: 220, A: 255}
)

// **The count is the list**, which is the whole reason this type exists: there is no second place
// for a count to live, so a pip without a colour cannot be represented and cannot be drawn white.
func TestEveryStandingPipHasAColour(t *testing.T) {
	var r shieldRow
	r.add(rowFire, 1)
	r.add(rowIce, 1)
	r.raiseTo(4, rowIce)
	r.hold(3, rowIce)

	if r.count() != len(r.pips) {
		t.Fatalf("the count is %d and the colours are %d", r.count(), len(r.pips))
	}
	for i, p := range r.pips {
		if p.A == 0 {
			t.Errorf("pip %d has no colour, so it draws as the bare white mark", i)
		}
	}
}

// A raise only ever raises; a block or an expiry is the only thing that takes a shield away, and it
// takes the oldest.
func TestOnlyABlockLowersTheRow(t *testing.T) {
	var r shieldRow
	r.add(rowFire, 1)
	r.add(rowIce, 1)

	r.raiseTo(1, rowFire)
	if r.count() != 2 {
		t.Errorf("a raise of 1 left %d pips, want the 2 already standing", r.count())
	}

	r.hold(1, rowFire)
	if r.count() != 1 {
		t.Fatalf("a block left %d pips, want 1", r.count())
	}
	if r.pips[0] != rowIce {
		t.Errorf("the block ate the newest pip: %v is left, want the ice %v", r.pips[0], rowIce)
	}

	r.hold(0, rowFire)
	if r.count() != 0 {
		t.Errorf("%d pips survive an empty row", r.count())
	}
}

// Nothing may draw more pips than the row on the card has seats for.
func TestTheRowIsHeldToTheCap(t *testing.T) {
	var r shieldRow
	r.add(rowFire, combat.MaxShields+3)
	if r.count() != combat.MaxShields {
		t.Errorf("the row holds %d pips, want the cap of %d", r.count(), combat.MaxShields)
	}
	r.raiseTo(combat.MaxShields+2, rowFire)
	if r.count() != combat.MaxShields {
		t.Errorf("a raise past the cap left %d pips", r.count())
	}
}

// **The model is the authority until an event speaks**, so a shield raised last round is standing
// while the player builds this one — and the moment something is announced, the announcement wins.
func TestTheModelIsTheAuthorityUntilSomethingIsAnnounced(t *testing.T) {
	var r shieldRow
	r.add(rowFire, 2)
	r.endRound()

	r.fitTo(1)
	if r.count() != 1 {
		t.Errorf("the row shows %d against a model of 1", r.count())
	}

	r.raiseTo(3, rowIce)
	r.fitTo(1)
	if r.count() != 3 {
		t.Errorf("the model overrode an announced 3: the row shows %d", r.count())
	}
}

// **An announcement brings a count and the flight brings the colour**, so every pip a raise adds
// wears the element of the card that raised it. A brace announcing two into an empty row landed one
// coloured pip and one bare white mark until the fill was handed in.
func TestAPaddedPipTakesTheColourItWasGiven(t *testing.T) {
	var r shieldRow
	r.raiseTo(2, rowIce)

	if r.count() != 2 {
		t.Fatalf("the row holds %d pips, want the announced 2", r.count())
	}
	for i, p := range r.pips {
		if p != rowIce {
			t.Errorf("pip %d is %v, want the raising card's ice %v", i, p, rowIce)
		}
	}
}
