package screens

import (
	"image"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// afterLines is the diff read back as the lines the panel would draw.
//
// **The tests assert on the words rather than on the records**, because what they are holding is
// that a take and a cut are told apart in the account a player reads — and the wording is the half
// of that a record cannot check. See internal/ui/ledger_prose.go, which is the translator.
func afterLines(gs *state.GlobalState, before, after session.Holdings) []session.LedgerLine {
	return ui.LedgerLines(afterRecords(gs, before, after))
}

// texts is every line of a diff as plain strings, which is what these tests assert on: the
// coloring is elementSpans' business and is tested where that lives.
func texts(lines []session.LedgerLine) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, l.Text())
	}
	return out
}

func has(t *testing.T, lines []session.LedgerLine, want string) {
	t.Helper()
	for _, got := range texts(lines) {
		if strings.Contains(got, want) {
			return
		}
	}
	t.Fatalf("no line containing %q; got %q", want, texts(lines))
}

// A card taken and a card cut are the two halves of a reward screen, and they have to be told
// apart from each other rather than reported as "the deck changed".
func TestTheAftermathSaysWhatJoinedTheDeckAndWhatLeftIt(t *testing.T) {
	kept := combat.Card{ID: 1, Concept: combat.Jab, Element: combat.Fire}
	cut := combat.Card{ID: 2, Concept: combat.Bash, Element: combat.Ice}
	took := combat.Card{ID: 3, Concept: combat.Thrust, Element: combat.Earth}

	before := session.Holdings{Cards: []combat.Card{kept, cut}}
	after := session.Holdings{Cards: []combat.Card{kept, took}}

	lines := afterLines(nil, before, after)
	if len(lines) != 2 {
		t.Fatalf("want two lines, got %q", texts(lines))
	}
	has(t, lines, "took")
	has(t, lines, "cut")
}

// **A card altered in place is the case the ids exist for.** Without them an essence's work is
// indistinguishable from a cut and a take, which is a pair of lines describing one act.
func TestAnAlteredCardIsOneLineRatherThanTwo(t *testing.T) {
	was := combat.Card{ID: 7, Concept: combat.Jab, Element: combat.Fire}
	now := was
	now.Element = combat.Ice

	lines := afterLines(nil,
		session.Holdings{Cards: []combat.Card{was}},
		session.Holdings{Cards: []combat.Card{now}})

	if len(lines) != 1 {
		t.Fatalf("want one line, got %q", texts(lines))
	}
	has(t, lines, "changed")
	has(t, lines, "into")
}

// An essence that moves a card's form moves nothing the card's name says, so the line has to name it —
// the same gap that had an altered Crush drawing a spear over the word CRUSH.
func TestAnAlteredFormIsNamedInTheLine(t *testing.T) {
	was := combat.Card{ID: 9, Concept: combat.Bash, Element: combat.Fire}
	if was.Form() != combat.FormCrush {
		t.Skipf("bash is no longer a crush; this test needs a card whose form an essence can move")
	}
	now := was
	now.FormOverride = combat.FormStab

	lines := afterLines(nil,
		session.Holdings{Cards: []combat.Card{was}},
		session.Holdings{Cards: []combat.Card{now}})

	has(t, lines, "stab")
}

// Nothing moving writes nothing at all. The watcher runs every frame, so a diff that reported a
// change on a quiet frame would fill a run's account with itself.
func TestAQuietFrameWritesNothing(t *testing.T) {
	held := session.Holdings{
		Cards:  []combat.Card{{ID: 1, Concept: combat.Jab}},
		Relics: []string{"dmg-plus"},
		Stones: map[string]int{"pair": 1},
		Vitae:  5,
	}
	if lines := afterLines(nil, held, held); len(lines) != 0 {
		t.Fatalf("want nothing, got %q", texts(lines))
	}
}

// The relic row, the purse and the ladder are all part of the same account.
func TestTheAftermathAccountsForRelicsStonesAndThePurse(t *testing.T) {
	before := session.Holdings{Relics: []string{"dmg-plus"}, Stones: map[string]int{}, Vitae: 9}
	after := session.Holdings{
		Relics: []string{"dmg-plus", "dmg-earth-crush"},
		Stones: map[string]int{"pair": 1},
		Vitae:  4,
	}

	lines := afterLines(nil, before, after)
	has(t, lines, "wore")
	has(t, lines, "raised")
	has(t, lines, "spent 5 vitae")
}

// A relic sold and a relic bought are the same list read in two directions, so the verbs have to
// differ or a sale reads as a purchase.
func TestASoldRelicIsNotReportedAsAPurchase(t *testing.T) {
	lines := afterLines(nil,
		session.Holdings{Relics: []string{"dmg-plus"}},
		session.Holdings{})

	if len(lines) != 1 {
		t.Fatalf("want one line, got %q", texts(lines))
	}
	has(t, lines, "sold")
}

// The bar and the rows share the panel's right edge, and the rows have to stop first: a heading's
// band is drawn edge to edge inside the border, so an inset that did not cover the bar's whole
// column put a dark band under the one control the panel is scrolled with.
func TestTheLedgersRowsStopBeforeItsScrollbar(t *testing.T) {
	r := image.Rect(60, 55, 1600, 930)

	contentRight := r.Max.X - ledgerPane.RightInset
	if track := ledgerScrollRect(r); contentRight > track.Min.X {
		t.Fatalf("rows reach %d, scrollbar starts at %d", contentRight, track.Min.X)
	}
}

// The bar's track starts under the closing X rather than behind it, for the same reason: the two
// are measured from the same corner and the X is drawn on top.
func TestTheLedgersScrollbarStartsBelowTheClosingX(t *testing.T) {
	r := image.Rect(60, 55, 1600, 930)

	closeBottom := r.Min.Y + ui.ModalCloseInset + ui.ModalCloseSize
	if top := ledgerScrollRect(r).Min.Y; top < closeBottom {
		t.Fatalf("track starts at %d, the X ends at %d", top, closeBottom)
	}
}

// The first row clears the closing X as well as the title. A heading is drawn on a dark band a
// whole pitch tall, so a row that merely starts below the X still paints up behind it.
func TestTheLedgersFirstRowClearsTheClosingX(t *testing.T) {
	closeBottom := ui.ModalCloseInset + ui.ModalCloseSize
	if ledgerPane.FirstRow < closeBottom {
		t.Fatalf("first row at %d, the X ends at %d", ledgerPane.FirstRow, closeBottom)
	}
}
