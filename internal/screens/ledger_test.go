package screens

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// ledgerRun is a run with two fights in its account: one finished, one still being fought.
func ledgerRun() *session.Session {
	run := session.New(panelDeck())

	run.BeginFight(1, "Outer Room", "Giant Bat")
	run.RecordRound([]session.LedgerRecord{{
		Kind: session.KindAct, Side: session.SideYou, Name: "Duelist",
		Card: "Jab", Verb: session.InkAttack,
	}}, 40, [2]session.LedgerStats{})
	run.EndFight(session.OutcomeWon)

	run.BeginFight(1, "Outer Room", "Cave Troll")
	run.RecordRound([]session.LedgerRecord{{
		Kind: session.KindAct, Side: session.SideFoe, Name: "Cave Troll",
		Card: "Gnaw", Verb: session.InkAttack,
	}}, 12, [2]session.LedgerStats{})
	return run
}

// **A finished fight is one line and the fight being fought is opened out.** That is the whole
// shape of the panel at run scale: a thirty-fight journey has to be readable as a list of fights
// before any one of them is opened.
func TestPastFightsAreOneLineAndTheLiveOneIsOpen(t *testing.T) {
	gs := testState()
	gs.Run = ledgerRun()

	rows := ledgerRows(gs, nil)

	var headings, lines []string
	for _, r := range rows {
		if r.fight != 0 || ledgerHeadingText(r.row.Text(), "-") || ledgerHeadingText(r.row.Text(), "+") {
			headings = append(headings, r.row.Text())
			continue
		}
		lines = append(lines, r.row.Text())
	}

	if len(headings) != 2 {
		t.Fatalf("the ledger has %d fight headings, want 2: %v", len(headings), headings)
	}
	if want := "+  First Realm, Outer Room - Giant Bat - 40 Damage Dealt"; headings[0] != want {
		t.Errorf("the finished fight's line is %q, want %q", headings[0], want)
	}
	if !ledgerHeadingText(headings[1], "-") {
		t.Errorf("the live fight's line is %q, which is not drawn open", headings[1])
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
		text = append(text, r.row.Text())
		if r.fight == 0 && strings.Contains(r.row.Text(), "Cave Troll -") {
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
	if len(rows) != 1 || rows[0].row.Text() == "" {
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
	// back is worthless if it holds less than a round. Twenty-five is a realm, not a target — it
	// came down from thirty when the rows went to 20 point, which was the trade being made.
	if n < 25 {
		t.Errorf("the ledger holds only %d rows", n)
	}
	if used := ledgerPane.FirstRow + n*ledgerPane.RowHeight + ledgerBottomInset; used > r.Dy() {
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
	if bar.Min.Y < r.Min.Y+ledgerPane.FirstRow {
		t.Errorf("the scrollbar starts at y=%d, above the first row at y=%d",
			bar.Min.Y, r.Min.Y+ledgerPane.FirstRow)
	}
	if bar.Dy() < 100 {
		t.Errorf("the scrollbar is only %dpx tall", bar.Dy())
	}
}

// **The working under a blow is what the ledger exists for**, so it is pinned: a line per hit, each
// naming its card and written out as the arithmetic it was, with the relic that priced it beside it,
// and the total of every hit under them.
func TestABlowWritesItsWorkingOut(t *testing.T) {
	e := combat.Event{
		Kind:          combat.KindHand,
		Side:          combat.SideA,
		HandCardCount: 2,
		Multiplier:    150,
		Amount:        90,
	}
	e.HandCards[0], e.HandCards[1] = 0, 1
	e.HandAmounts[0], e.HandAmounts[1] = 20, 40
	e.HitAmounts[0], e.HitAmounts[1] = 30, 60
	// What each landing was worth before its relics: the second card is a 20 the relic doubled.
	e.HandCardBase[0], e.HandCardBase[1] = 20, 20
	// **Built rather than indexed into**: a seat row is a slice as long as the worn row it
	// describes, so a hand-made event has to say how many seats it is talking about. Seat 0
	// doubled the second hit; seat 1 is why that hit exists.
	e.HandRelicScale[1] = []int{200}
	e.HandLanding[1] = []bool{false, true}

	played := []combat.Card{
		{Concept: combat.Bash, Element: combat.Fire},
		{Concept: combat.Jab, Element: combat.Ice},
	}

	// **No relics worn**, so every name falls back to "a relic" — which is what this test is
	// about: the shape of the working and the figures in it, not which record paid them.
	lines := ui.LedgerLines(ui.HandTermRecords(e, nil, played))

	// **Each hit is its card's line and then its working down a column**, one labelled step per
	// figure, ending on what the hit came to. The blow ends on every hit together.
	var hits [][]session.LedgerLine
	for _, l := range lines[:len(lines)-1] {
		if l.Voice == session.VoiceTerm {
			hits = append(hits, nil)
		}
		if len(hits) == 0 {
			t.Fatalf("the working opens on %q, want a hit's card line", l.Text())
		}
		hits[len(hits)-1] = append(hits[len(hits)-1], l)
	}
	if len(hits) != 2 {
		t.Fatalf("a two-hit blow wrote %d hits: %v", len(hits), lines)
	}
	texts := func(ls []session.LedgerLine) []string {
		var out []string
		for _, l := range ls {
			out = append(out, l.Text())
		}
		return out
	}
	has := func(ls []session.LedgerLine, want string) bool {
		for _, s := range texts(ls) {
			if strings.Contains(s, want) {
				return true
			}
		}
		return false
	}

	first, second := hits[0], hits[1]
	if first[0].Text() != "Fire Bash" || !has(first, "x 1.5") || !strings.Contains(first[len(first)-1].Text(), "= 30") {
		t.Errorf("the first hit reads %q, want its card, the hand's 1.5 and its own 30", texts(first))
	}
	if !strings.Contains(second[0].Text(), "Repeats - ") {
		t.Errorf("the second hit opens %q, which does not say a relic made it land again", second[0].Text())
	}
	if !has(second, "x 2") || !has(second, "a relic") || !strings.Contains(second[len(second)-1].Text(), "= 60") {
		t.Errorf("the second hit reads %q, want the relic's x2 as its own labelled step and its 60", texts(second))
	}
	for _, hit := range hits {
		for i, l := range hit[1:] {
			if l.Voice != session.VoiceWork {
				t.Errorf("step %d of a hit is in the %q voice, want %q", i, l.Voice, session.VoiceWork)
			}
		}
	}
	if got := lines[len(lines)-1].Text(); !strings.Contains(got, "90") {
		t.Errorf("the total reads %q, want every hit's 90", got)
	}
}

// The arithmetic is set in from the left rather than centered. **A column of figures is the one
// layout centering cannot survive**, and drawPane centers any row with no swatch and no verb — so
// the indent is what keeps the working readable, and it is easy to lose.
func TestTheWorkingIsIndentedRatherThanCentered(t *testing.T) {
	rows := ui.PaneRowsFor([]session.LedgerLine{
		session.Line(session.VoiceTerm, "Bash 20"),
		session.Line(session.VoicePlain, "- Round 1 -"),
	})

	if rows[0].Indent == 0 {
		t.Error("a term row is not indented, so it will be drawn centered")
	}
	if rows[1].Indent != 0 {
		t.Error("a heading is indented, so it will not be centered")
	}
}

// The account is colored the way the screen is: a figure in its card's element, a relic's
// multiplier in the relic pink, the hand's own in the hand's color. **It is the reason a line is
// runs rather than a string**, and it is the part a refactor would quietly flatten.
func TestTheWorkingIsColoredLikeTheScreen(t *testing.T) {
	e := combat.Event{Kind: combat.KindHand, HandCardCount: 1, Multiplier: 200, Amount: 40}
	e.HandAmounts[0], e.HandCardBase[0], e.HitAmounts[0] = 20, 20, 40
	e.HandRelicScale[0] = []int{200}

	lines := ui.LedgerLines(ui.HandTermRecords(e, nil,
		[]combat.Card{{Concept: combat.Bash, Element: combat.Fire}}))
	rows := ui.PaneRowsFor(lines)

	fire := cards.BorderOf(ui.ArtFor(combat.Fire))
	relic := ui.BoostInk

	if rows[0].Spans[0].Ink != fire {
		t.Errorf("the card's name is in %v, want its element's %v", rows[0].Spans[0].Ink, fire)
	}

	// **The relic's step is pink twice over**: its figure and the name labelling it.
	var sawRelic bool
	for _, row := range rows[1:] {
		if len(row.Spans) == 3 && row.Spans[1].Ink == relic && row.Spans[2].Ink == relic {
			sawRelic = true
		}
	}
	if !sawRelic {
		t.Error("no step of the hit writes the relic's figure and its name in the relic's color")
	}

	// **Nothing in a hit's working wears a hue that means something else**, and nothing in it is
	// underlined: hue belongs to the elements and the wheel is full, and an underline mid-sum
	// reads as a typesetting accident. This is the check that catches either coming back.
	for _, row := range rows {
		for _, r := range row.Spans {
			if r.Mark {
				t.Errorf("a run of the working is underlined: %q", r.Text)
			}
			if r.Ink == cards.BorderOf(cards.Arcane) {
				t.Errorf("a run of the working is written in the arcane element's color: %q", r.Text)
			}
		}
	}
}

// **A fight is a block with a lid**: its heading on a dark band and everything under it on a tint
// of the same color, so a panel scrolled past the heading still says which fight is being read.
func TestAnOpenedFightIsBanded(t *testing.T) {
	gs := testState()
	gs.Run = ledgerRun()

	rows := ledgerRows(gs, map[int]bool{1: true})

	// **Consecutive fights alternate between the two blues**, so the walk tracks which pair the
	// current fight is on rather than asserting one color: what the ledger promises is that a
	// heading and its own rows match and that the next fight's do not.
	heads := 0
	pair := -1
	for _, r := range rows {
		switch {
		case r.fight != 0 || ledgerHeadingText(r.row.Text(), "-"):
			heads++
			pair++
			if want := ledgerBands[pair%2]; r.row.Band != want {
				t.Errorf("fight %d's heading is on %v, want the Band %v", heads, r.row.Band, want)
			}
			if r.row.Spans[0].Ink != ledgerBandInk {
				t.Errorf("a heading on the dark band is written in %v", r.row.Spans[0].Ink)
			}
		default:
			if want := ledgerGrounds[pair%2]; r.row.Band != want {
				t.Errorf("a row of an opened fight is on %v, want the tint %v", r.row.Band, want)
			}
		}
	}
	if ledgerBands[0] == ledgerBands[1] || ledgerGrounds[0] == ledgerGrounds[1] {
		t.Error("the two fight colors are the same, so nothing alternates")
	}
	if heads != 2 {
		t.Errorf("%d headings were banded, want 2", heads)
	}
}

// An attack says what it multiplies its owner's damage by, which is the only account of why a
// Giant Rat's gnaw and its maul land such different figures. **The identity is not written** — a
// bracket saying 1x on most swings in the game is a bracket that says nothing — and a defense
// multiplies nothing, so it says nothing either.
func TestAnAttackSaysWhatItWeighs(t *testing.T) {
	for _, id := range []combat.ConceptID{combat.Jab, combat.Bash, combat.Cut} {
		card := combat.Plain(id)
		got, weight := ui.CardWeight(card), card.Amount()

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
	if got := ui.CardWeight(combat.Plain(ward)); got != "" {
		t.Errorf("a defense writes %q, and multiplies nothing", got)
	}
}

// panelDeck is two lightning cards and a fire one — enough for a flip relic to have something to
// take and something to leave alone.
func panelDeck() []combat.Card {
	return []combat.Card{
		{Concept: combat.Bash, Element: combat.Lightning},
		{Concept: combat.Bash, Element: combat.Lightning},
		{Concept: combat.Bash, Element: combat.Fire},
	}
}

// ledgerHeadingText reports whether a row's text is a fight's heading under marker: the marker,
// then the realm the fight was in.
func ledgerHeadingText(text, marker string) bool {
	return strings.HasPrefix(text, marker+"  ") && strings.Contains(text, "Realm")
}
