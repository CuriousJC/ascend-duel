package models

import (
	"image"
	"testing"
)

// The tooltip's handshake: a scene points at something every tick, and a tick with nothing pointed
// hides the panel. Everything here is arithmetic over two ints and a string.

// tick is what systems.UpdateTooltip does, written out here so this package can test the behaviour
// without importing the one that drives it. **If the two ever disagree, this test is the one that is
// wrong** — the driver is the real thing.
func tick(t *Tooltip) {
	if !t.Pointed() {
		t.Forget()
		return
	}
	t.Release()
	if t.Dwell < t.DwellTicks {
		t.Dwell++
	}
}

// aTitle is a plain one-run title, which is what a caller that never thinks about colour builds.
func aTitle(s string) TipLine { return TipLine{{Text: s}} }

func aSeat(x int) image.Rectangle { return image.Rect(x, 0, x+100, 200) }

func TestThePanelWaitsForTheDwell(t *testing.T) {
	// **A delay rather than an instant panel**: a cursor crossing a row of eight cards on its way to
	// a button would otherwise strobe eight tooltips.
	tip := &Tooltip{DwellTicks: 3}

	for i := 0; i < 3; i++ {
		tip.Point(aSeat(0), aTitle("Bash"), oneLine("12 DMG"))
		if tip.Showing() {
			t.Fatalf("the panel showed after %d ticks, want 3", i)
		}
		tick(tip)
	}

	tip.Point(aSeat(0), aTitle("Bash"), oneLine("12 DMG"))
	if !tip.Showing() {
		t.Error("the panel never appeared")
	}
}

func TestATickWithNothingPointedHidesIt(t *testing.T) {
	// The scene never has to remember to hide one, which is the whole reason the handshake is this
	// way round — the same shape state.ModalOpen takes.
	tip := &Tooltip{}
	tip.Point(aSeat(0), aTitle("Bash"), nil)
	tick(tip)

	if !tip.Showing() {
		t.Fatal("a zero dwell should show at once")
	}

	tick(tip) // nothing pointed this time
	if tip.Showing() {
		t.Error("the panel outlived the thing it was describing")
	}
}

func TestMovingToAnotherCardRestartsTheDwell(t *testing.T) {
	// Sliding along a row must not carry one card's waiting over to the next, or the second card
	// would pop instantly and every card after it too.
	tip := &Tooltip{DwellTicks: 2}

	tip.Point(aSeat(0), aTitle("Bash"), nil)
	tick(tip)
	tip.Point(aSeat(0), aTitle("Bash"), nil)
	tick(tip)
	if !tip.Showing() {
		t.Fatal("the first card never showed")
	}

	tip.Point(aSeat(200), aTitle("Slash"), nil)
	if tip.Showing() {
		t.Error("the next card inherited the first card's dwell")
	}
}

func TestTheSameCardSayingSomethingNewKeepsItsDwell(t *testing.T) {
	// A card whose figures change while the cursor rests on it — a relic bought, a status expiring —
	// is still the card being looked at, so the panel must not flicker off and back on.
	tip := &Tooltip{DwellTicks: 1}

	tip.Point(aSeat(0), aTitle("Bash"), oneLine("12 DMG"))
	tick(tip)
	tip.Point(aSeat(0), aTitle("Bash"), oneLine("24 DMG"))

	if !tip.Showing() {
		t.Error("changing the lines restarted the wait")
	}
	if len(tip.Lines) != 1 || len(tip.Lines[0]) != 1 || tip.Lines[0][0].Text != "24 DMG" {
		t.Errorf("the panel is still saying %v", tip.Lines)
	}
}

func TestForgetHidesItImmediately(t *testing.T) {
	// For a scene that has just done the thing the panel was describing — a relic bought out from
	// under the cursor.
	tip := &Tooltip{}
	tip.Point(aSeat(0), aTitle("Keen"), oneLine("doubles slashes"))
	tick(tip)

	tip.Forget()
	if tip.Showing() {
		t.Error("Forget left the panel up")
	}
}

// oneLine is a tooltip line in one colour, which is what a caller that never thinks about colour
// builds.
func oneLine(text string) []TipLine {
	return []TipLine{{{Text: text}}}
}
