package screens

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The slot stands on the pile's caption line, at its left end, with the deck count at the right
// end of the same line. **It was beside the pile until 2026-09-04**, when the pile moved into the
// duelist card's column: there is nothing to the left of it there, and what is to the right at that
// height is the action-point bar.
//
// It is checked rather than eyeballed because every edge of it is derived from the pile, which is
// itself derived from the frame's corner — three constants deep, and nothing in the code says out
// loud that the result has to line up.
func TestThePileSlotStandsOnThePilesCaptionLine(t *testing.T) {
	gs := testState()

	slot, pile, caption := pileSlotRect(gs), deckStackBounds(gs), deckCaptionRect(gs)

	if slot.Min.X != caption.Min.X || slot.Max.Y != caption.Max.Y {
		t.Errorf("the slot %v does not stand on the left end of the caption line %v", slot, caption)
	}
	if slot.Overlaps(pile) {
		t.Errorf("the slot %v overlaps the deck pile %v", slot, pile)
	}
	if gap := pile.Min.Y - caption.Max.Y; gap != deckCaptionGap {
		t.Errorf("the gap between the caption line and the pile is %dpx, not the %dpx it is set to",
			gap, deckCaptionGap)
	}
	if slot.Min.X < 0 || slot.Min.Y < 0 || slot.Max.Y > gs.ScreenHeight {
		t.Errorf("the slot %v is off the screen", slot)
	}
}

// ledgerRun is a run with two fights in its account: one finished, one still being fought.
func ledgerRun() *session.Session {
	run := session.New(panelDeck())

	run.BeginFight(1, "Giant Bat")
	run.RecordRound([]session.LedgerLine{session.Line(session.VoiceYou, "Duelist attacks")}, 40)
	run.EndFight(session.OutcomeWon)

	run.BeginFight(1, "Cave Troll")
	run.RecordRound([]session.LedgerLine{session.Line(session.VoiceFoe, "Cave Troll attacks")}, 12)
	return run
}

// **A finished fight is one line and the fight being fought is opened out.** That is the whole
// shape of the panel at run scale: a thirty-fight climb has to be readable as a list of fights
// before any one of them is opened.
func TestPastFightsAreOneLineAndTheLiveOneIsOpen(t *testing.T) {
	gs := testState()
	gs.Run = ledgerRun()

	rows := ledgerRows(gs, nil)

	var headings, lines []string
	for _, r := range rows {
		if r.fight != 0 || strings.HasPrefix(r.row.text(), "-  Fight") || strings.HasPrefix(r.row.text(), "+  Fight") {
			headings = append(headings, r.row.text())
			continue
		}
		lines = append(lines, r.row.text())
	}

	if len(headings) != 2 {
		t.Fatalf("the ledger has %d fight headings, want 2: %v", len(headings), headings)
	}
	if !strings.Contains(headings[0], "Giant Bat") || !strings.Contains(headings[0], "won in 1") {
		t.Errorf("the finished fight's line is %q, which does not say how it went", headings[0])
	}
	if !strings.Contains(headings[1], "fighting now") {
		t.Errorf("the live fight's line is %q, which does not say it is live", headings[1])
	}

	// The folded fight's own round is not written; the live one's is.
	joined := strings.Join(lines, " | ")
	if strings.Contains(joined, "Duelist attacks") {
		t.Errorf("a folded fight wrote its rounds out: %q", joined)
	}
	if !strings.Contains(joined, "Cave Troll attacks") {
		t.Errorf("the live fight did not write its round out: %q", joined)
	}
}

// A heading the player has opened writes its rounds. **The fight being fought offers no toggle at
// all** — it is already open, and a control that folded away the round the player is in the middle
// of would hide the thing the panel was most likely opened for.
func TestOpeningAPastFightWritesItsRoundsAndTheLiveOneCannotBeFolded(t *testing.T) {
	gs := testState()
	gs.Run = ledgerRun()

	rows := ledgerRows(gs, map[int]bool{1: true})

	var text []string
	live := 0
	for _, r := range rows {
		text = append(text, r.row.text())
		if r.fight == 0 && strings.Contains(r.row.text(), "Cave Troll -") {
			live++
		}
	}
	if live != 1 {
		t.Errorf("the live fight's heading offers a toggle; it must not")
	}
	if !strings.Contains(strings.Join(text, " | "), "Duelist attacks") {
		t.Errorf("an opened fight did not write its rounds: %v", text)
	}
}

// A run with nothing in it says so rather than opening an empty panel. An empty dialog reads as a
// broken one; a sentence reads as a run that has not started.
func TestTheLedgerSaysWhenThereIsNothingInIt(t *testing.T) {
	gs := testState()
	gs.Run = session.New(panelDeck())

	rows := ledgerRows(gs, nil)
	if len(rows) != 1 || rows[0].row.text() == "" {
		t.Errorf("an untouched ledger is %v, want one sentence", rows)
	}

	// And with no run at all — the title screen — it is a sentence rather than a panic.
	if rows := ledgerRows(&state.GlobalState{}, nil); len(rows) != 1 {
		t.Errorf("a runless ledger is %v, want one sentence", rows)
	}
}

// The panel holds whole rows and keeps its last one clear of its own bottom edge. Derived rather
// than written down, so this checks the derivation lands somewhere sane rather than a number.
func TestTheLedgerPanelHoldsEnoughRowsToBeWorthOpening(t *testing.T) {
	gs := testState()

	r := ledgerPanelRect(gs)
	n := ledgerCapacity(r)

	// The panel has to hold a real fight, not a handful of lines: a dialog opened to read a round
	// back is worthless if it holds less than a round. Twenty-five is a floor, not a target — it
	// came down from thirty when the rows went to 20 point, which was the trade being made.
	if n < 25 {
		t.Errorf("the ledger holds only %d rows", n)
	}
	if used := ledgerPane.firstRow + n*ledgerPane.rowHeight + ledgerBottomInset; used > r.Dy() {
		t.Errorf("%d rows plus the heading and the inset need %dpx of a %dpx panel", n, used, r.Dy())
	}
}

// The scrollbar stands inside the panel, clear of the title and of the bottom edge — it is the
// only way to the earlier rows, so a bar drawn over the words or off the panel would be the
// feature failing quietly.
func TestTheScrollbarStandsInsideThePanel(t *testing.T) {
	gs := testState()
	r := ledgerPanelRect(gs)
	bar := ledgerScrollRect(r)

	if !bar.In(r) {
		t.Errorf("the scrollbar %v is not inside the panel %v", bar, r)
	}
	if bar.Min.Y < r.Min.Y+ledgerPane.firstRow {
		t.Errorf("the scrollbar starts at y=%d, above the first row at y=%d",
			bar.Min.Y, r.Min.Y+ledgerPane.firstRow)
	}
	if bar.Dy() < 100 {
		t.Errorf("the scrollbar is only %dpx tall", bar.Dy())
	}
}

// **The working under a blow is what the ledger exists for**, so it is pinned: a line per landing,
// each naming its card and its figure, with the ring that priced it beside it.
func TestABlowWritesItsWorkingOut(t *testing.T) {
	s := &CombatScene{}

	e := combat.Event{
		Kind:          combat.KindHand,
		Side:          combat.SideA,
		HandCardCount: 2,
		Base:          60,
		Multiplier:    150,
		Amount:        90,
	}
	e.HandCards[0], e.HandCards[1] = 0, 1
	e.HandAmounts[0], e.HandAmounts[1] = 20, 40
	// What each landing was worth before its rings: the second card is a 20 the ring doubled.
	e.HandCardBase[0], e.HandCardBase[1] = 20, 20
	e.HandRingScale[1][0] = 200
	e.HandLanding[1][1] = true

	played := []combat.Card{
		{Concept: combat.Strike, Element: combat.Fire},
		{Concept: combat.Jab, Element: combat.Ice},
	}

	lines := s.handTermLines(e, played)
	if len(lines) != 3 {
		t.Fatalf("a two-term blow wrote %d lines, want 2 terms and the sum: %v", len(lines), lines)
	}

	// **The sum is the working's last line and is the one the player watched fly into place** —
	// term by term, not the cards folded into one figure.
	// **The ring's figure stays with the term it priced**, in brackets — folding it into the term
	// hides the ring, and hanging it off the end of the sum would read as multiplying every term
	// and would not come to the total.
	if got, want := lines[2].Text(), "20 + (20 x 2) x 1.5 = 90"; got != want {
		t.Errorf("the sum reads %q, want %q", got, want)
	}
	for i, l := range lines {
		if l.Voice != session.VoiceTerm {
			t.Errorf("term %d is in the %q voice, want %q", i, l.Voice, session.VoiceTerm)
		}
	}
	if !strings.Contains(lines[0].Text(), "20") || !strings.Contains(lines[0].Text(), "fire") {
		t.Errorf("the first term is %q, which does not say what card paid it", lines[0].Text())
	}
	if !strings.Contains(lines[1].Text(), "2x") {
		t.Errorf("the second term is %q, which does not say what the ring did", lines[1].Text())
	}
	if !strings.Contains(lines[1].Text(), "again") {
		t.Errorf("the second term is %q, which does not say a ring landed it again", lines[1].Text())
	}
}

// The arithmetic is set in from the left rather than centred. **A column of figures is the one
// layout centring cannot survive**, and drawPane centres any row with no swatch and no verb — so
// the indent is what keeps the working readable, and it is easy to lose.
func TestTheWorkingIsIndentedRatherThanCentred(t *testing.T) {
	rows := paneRowsFor([]session.LedgerLine{
		session.Line(session.VoiceTerm, "Strike 20"),
		session.Line(session.VoicePlain, "- Round 1 -"),
	})

	if rows[0].indent == 0 {
		t.Error("a term row is not indented, so it will be drawn centred")
	}
	if rows[1].indent != 0 {
		t.Error("a heading is indented, so it will not be centred")
	}
}

// The hand's name leads with the rung and carries its axis in brackets, because the loudest line
// of the round should not open on the least interesting word in it. A name the catalogue does not
// write an axis in front of is left alone.
func TestAHandIsNamedRungFirst(t *testing.T) {
	for _, c := range []struct{ name, want string }{
		{"Form Three of a Kind", "Three of a Kind (Form)"},
		{"Card Pair", "Pair (Card)"},
		{"Elemental Full House", "Full House (Elemental)"},
		{"High Card", "High Card"},
	} {
		if got := axisToBack(c.name); got != c.want {
			t.Errorf("%q reads as %q, want %q", c.name, got, c.want)
		}
	}
}

// The account is coloured the way the screen is: a figure in its card's element, a ring's
// multiplier in the ring pink, the hand's own in the hand's colour. **It is the reason a line is
// runs rather than a string**, and it is the part a refactor would quietly flatten.
func TestTheWorkingIsColouredLikeTheScreen(t *testing.T) {
	s := &CombatScene{}

	e := combat.Event{Kind: combat.KindHand, HandCardCount: 1, Multiplier: 200, Amount: 40}
	e.HandAmounts[0], e.HandCardBase[0] = 20, 20
	e.HandRingScale[0][0] = 200

	lines := s.handTermLines(e, []combat.Card{{Concept: combat.Strike, Element: combat.Fire}})
	rows := paneRowsFor(lines)

	fire := cards.BorderOf(artFor(combat.Fire))
	ring := boostInk

	term := rows[0]
	if term.runs[0].ink != fire {
		t.Errorf("the card's name is in %v, want its element's %v", term.runs[0].ink, fire)
	}
	if last := term.runs[len(term.runs)-1]; last.ink != ring {
		t.Errorf("the ring's note is in %v, want the ring pink %v", last.ink, ring)
	}

	// **Nothing in the sum wears a hue that means something else**, and nothing in it is
	// underlined: hue belongs to the elements and the wheel is full, and an underline mid-sum
	// reads as a typesetting accident. This is the check that catches either coming back.
	sum := rows[len(rows)-1]
	var sawRing bool
	for _, r := range sum.runs {
		sawRing = sawRing || r.ink == ring
		if r.mark {
			t.Errorf("a run of the sum is underlined: %q", r.text)
		}
		if r.ink == cards.BorderOf(cards.Arcane) {
			t.Errorf("a run of the sum is written in the arcane element's colour: %q", r.text)
		}
	}
	if !sawRing {
		t.Error("the ring's figure in the sum is not in the ring's colour")
	}
}

// **A fight is a block with a lid**: its heading on a dark band and everything under it on a tint
// of the same colour, so a panel scrolled past the heading still says which fight is being read.
func TestAnOpenedFightIsBanded(t *testing.T) {
	gs := testState()
	gs.Run = ledgerRun()

	rows := ledgerRows(gs, map[int]bool{1: true})

	// **Consecutive fights alternate between the two blues**, so the walk tracks which pair the
	// current fight is on rather than asserting one colour: what the ledger promises is that a
	// heading and its own rows match and that the next fight's do not.
	heads := 0
	pair := -1
	for _, r := range rows {
		switch {
		case r.fight != 0 || strings.Contains(r.row.text(), "fighting now"):
			heads++
			pair++
			if want := ledgerBands[pair%2]; r.row.band != want {
				t.Errorf("fight %d's heading is on %v, want the band %v", heads, r.row.band, want)
			}
			if r.row.runs[0].ink != ledgerBandInk {
				t.Errorf("a heading on the dark band is written in %v", r.row.runs[0].ink)
			}
		default:
			if want := ledgerGrounds[pair%2]; r.row.band != want {
				t.Errorf("a row of an opened fight is on %v, want the tint %v", r.row.band, want)
			}
		}
	}
	if ledgerBands[0] == ledgerBands[1] || ledgerGrounds[0] == ledgerGrounds[1] {
		t.Error("the two fight colours are the same, so nothing alternates")
	}
	if heads != 2 {
		t.Errorf("%d headings were banded, want 2", heads)
	}
}

// An attack says what it multiplies its owner's damage by, which is the only account of why a
// Giant Rat's gnaw and its maul land such different figures. **The identity is not written** — a
// bracket saying 1x on most swings in the game is a bracket that says nothing — and a defence
// multiplies nothing, so it says nothing either.
func TestAnAttackSaysWhatItWeighs(t *testing.T) {
	for _, id := range []combat.ConceptID{combat.Jab, combat.Strike, combat.Cut} {
		card := combat.Plain(id)
		got, weight := cardWeight(card), card.Amount()

		switch {
		case weight == 100 && got != "":
			t.Errorf("%v is a 1x card and writes %q", combat.ConceptOf(id).Label, got)
		case weight != 100 && got == "":
			t.Errorf("%v is a %dx card and says nothing about it",
				combat.ConceptOf(id).Label, weight)
		}
	}

	ward, ok := combat.ConceptByKey("ward")
	if !ok {
		return
	}
	if got := cardWeight(combat.Plain(ward)); got != "" {
		t.Errorf("a defence writes %q, and multiplies nothing", got)
	}
}
