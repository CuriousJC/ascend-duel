package screens

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The Log button stands beside the draw pile and shares its bottom edge, so the corner reads as
// two controls on one line rather than as a button that happens to be near a pile.
//
// It is checked rather than eyeballed because both of its edges are derived from the pile, which
// is itself derived from the screen's corner — three constants deep, and nothing in the code says
// out loud that the result has to be level.
func TestTheLogButtonStandsBesideThePileOnTheSameLine(t *testing.T) {
	gs := testState()

	button, pile := logButtonRect(gs), deckStackBounds(gs)

	if button.Max.Y != pile.Max.Y {
		t.Errorf("the Log button's bottom is y=%d and the pile's is y=%d", button.Max.Y, pile.Max.Y)
	}
	if button.Overlaps(pile) {
		t.Errorf("the Log button %v overlaps the deck pile %v", button, pile)
	}
	if gap := pile.Min.X - button.Max.X; gap != logButtonToDeckGap {
		t.Errorf("the gap between the Log button and the pile is %dpx, not the %dpx it is set to",
			gap, logButtonToDeckGap)
	}
	if button.Min.X < 0 || button.Max.Y > gs.ScreenHeight {
		t.Errorf("the Log button %v is off the screen", button)
	}
}

// **A round in the fight log is a round in the fight, in the order it was played**, and the round
// still being played back is included only as far as the player has been shown.
//
// That last part is the whole reason this is worth a test: the dialog can be opened mid-playback,
// the resolved round is sitting in `log` in full, and a log built from all of it would hand the
// player the rest of the round they are watching.
func TestTheFightLogHoldsEveryRoundAndSpoilsNone(t *testing.T) {
	strike := combat.Event{Kind: combat.KindAction, Side: combat.SideA, Action: combat.Strike}
	jab := combat.Event{Kind: combat.KindAction, Side: combat.SideB, Action: combat.Jab}

	s := &CombatScene{
		rounds: [][]combat.Event{{strike}, {jab}},
		log:    []combat.Event{strike, jab},
		cursor: 0, // playback has reached the first event of round three and no further
	}

	rows := s.fightLogRows()

	var headings []string
	for _, r := range rows {
		if strings.HasPrefix(r.prefix, "- Round") {
			headings = append(headings, r.prefix)
		}
	}
	want := []string{"- Round 1 -", "- Round 2 -", "- Round 3 -"}
	if len(headings) != len(want) {
		t.Fatalf("the log has %d round headings, want %d: %v", len(headings), len(want), headings)
	}
	for i := range want {
		if headings[i] != want[i] {
			t.Errorf("heading %d is %q, want %q", i, headings[i], want[i])
		}
	}

	// Round three's second event has not been reached, so nothing it did may be written down.
	// The opponent is the only side acting in it, which is what makes the check a name lookup.
	last := rows[len(rows)-1]
	if strings.Contains(last.prefix+last.suffix, combat.ConceptOf(combat.Jab).Label) {
		t.Errorf("the log's last row is %q%q, which is an event playback has not reached",
			last.prefix, last.suffix)
	}
}

// A fight that has not started says so rather than opening an empty panel. An empty dialog reads
// as a broken one; a sentence reads as a fight that has not started.
func TestTheFightLogSaysWhenThereIsNothingInIt(t *testing.T) {
	s := &CombatScene{}

	rows := s.fightLogRows()
	if len(rows) != 1 || rows[0].prefix == "" {
		t.Errorf("an untouched fight log is %v, want one sentence", rows)
	}
}

// The panel holds whole rows and keeps room for the line telling the player how to close it.
// Derived rather than written down, so this checks the derivation lands somewhere sane rather
// than checking a number.
func TestTheFightLogPanelHoldsEnoughRowsToBeWorthOpening(t *testing.T) {
	gs := testState()

	r := logPanelRect(gs)
	n := logCapacity(r)

	// The panel has to hold a real fight, not a handful of lines: a dialog you open to read back
	// a round is worthless if it holds less than a round. Thirty rows is not a target, it is a
	// floor low enough that only a real mistake trips it.
	if n < 30 {
		t.Errorf("the log panel holds only %d rows", n)
	}
	if used := paneFirstRow + n*paneTextRowHeight + logHintReserve; used > r.Dy() {
		t.Errorf("%d rows plus the heading and the hint need %dpx of a %dpx panel", n, used, r.Dy())
	}
}
